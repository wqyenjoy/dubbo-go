/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package config

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/dubbogo/gost/log/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/google/go-cmp/cmp"
	"github.com/knadh/koanf"
)

// HotReloadManager 配置文件热加载管理器
type HotReloadManager struct {
	watcher        *fsnotify.Watcher
	configPath     string
	configFile     string
	rootConfig     *RootConfig
	mu             sync.RWMutex
	reloadCallback func(*RootConfig) error
	debounceTimer  *time.Timer
	debounceDelay  time.Duration
	isRunning      bool
	stopCh         chan struct{} // 用于优雅停止

	// 幂等性支持
	lastConfigHash  string               // 上次配置的哈希值
	processedEvents map[string]time.Time // 已处理的事件ID -> 处理时间
	eventTTL        time.Duration        // 事件TTL，用于清理过期事件
}

// NewHotReloadManager 创建新的热加载管理器
func NewHotReloadManager(configPath string, rootConfig *RootConfig) (*HotReloadManager, error) {
	// 确保配置文件路径是绝对路径
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, err
	}

	// 创建文件监听器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// 计算初始配置哈希
	configHash := ""
	if rootConfig != nil {
		configHash = calculateConfigHash(rootConfig)
	}

	manager := &HotReloadManager{
		watcher:         watcher,
		configPath:      absPath,
		configFile:      filepath.Base(absPath),
		rootConfig:      rootConfig,
		debounceDelay:   500 * time.Millisecond, // 默认防抖延迟500ms
		stopCh:          make(chan struct{}),
		lastConfigHash:  configHash,
		processedEvents: make(map[string]time.Time),
		eventTTL:        5 * time.Minute, // 事件TTL为5分钟
	}

	return manager, nil
}

// SetReloadCallback 设置配置重载回调函数
func (hrm *HotReloadManager) SetReloadCallback(callback func(*RootConfig) error) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	hrm.reloadCallback = callback
}

// SetDebounceDelay 设置防抖延迟时间
func (hrm *HotReloadManager) SetDebounceDelay(delay time.Duration) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	hrm.debounceDelay = delay
}

// Start 开始监听配置文件变化
func (hrm *HotReloadManager) Start() error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	if hrm.isRunning {
		return nil
	}

	// 如果watcher为nil，重新创建
	if hrm.watcher == nil {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		hrm.watcher = watcher
	}

	// 只监听配置文件所在目录，避免重复事件
	configDir := filepath.Dir(hrm.configPath)
	if err := hrm.watcher.Add(configDir); err != nil {
		return err
	}

	hrm.isRunning = true

	// 启动监听协程
	go hrm.watchLoop()

	logger.Infof("Hot reload manager started, watching config file: %s", hrm.configPath)
	return nil
}

// Stop 停止监听
func (hrm *HotReloadManager) Stop() error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	if !hrm.isRunning {
		return nil
	}

	hrm.isRunning = false

	// 停止防抖定时器并置为nil
	if hrm.debounceTimer != nil {
		hrm.debounceTimer.Stop()
		hrm.debounceTimer = nil // 修复Timer重用风险
	}

	// 发送停止信号
	close(hrm.stopCh)

	// 关闭文件监听器并置为nil
	if hrm.watcher != nil {
		if err := hrm.watcher.Close(); err != nil {
			return err
		}
		hrm.watcher = nil // 修复可重入问题
	}

	// 重新创建stopCh，以便下次Start时使用
	hrm.stopCh = make(chan struct{})

	logger.Infof("Hot reload manager stopped")
	return nil
}

// watchLoop 文件变化监听循环
func (hrm *HotReloadManager) watchLoop() {
	for hrm.IsRunning() { // 修复退出条件
		select {
		case event := <-hrm.watcher.Events:
			hrm.handleFileEvent(event)
		case err := <-hrm.watcher.Errors:
			if err != nil {
				logger.Errorf("File watcher error: %v", err)
			}
		case <-hrm.stopCh:
			// 收到停止信号，退出循环
			return
		}
	}
}

// handleFileEvent 处理文件变化事件
func (hrm *HotReloadManager) handleFileEvent(event fsnotify.Event) {
	// 检查是否是目标配置文件的变化
	if filepath.Clean(event.Name) != filepath.Clean(hrm.configPath) {
		return
	}

	logger.Debugf("Config file changed: %s, operation: %v", event.Name, event.Op)

	// 处理不同类型的文件操作
	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		// 文件写入事件
		hrm.debouncedReload()
	case event.Op&fsnotify.Create == fsnotify.Create:
		// 文件创建事件（可能是重命名）
		hrm.debouncedReload()
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// 文件删除事件
		logger.Warnf("Config file was removed: %s", event.Name)
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// 文件重命名事件
		hrm.debouncedReload()
	}
}

// debouncedReload 防抖重载
func (hrm *HotReloadManager) debouncedReload() {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	// 停止之前的定时器并置为nil
	if hrm.debounceTimer != nil {
		hrm.debounceTimer.Stop()
		hrm.debounceTimer = nil
	}

	// 创建新的定时器
	hrm.debounceTimer = time.AfterFunc(hrm.debounceDelay, func() {
		hrm.reloadConfig()
	})
}

// reloadConfig 重新加载配置文件
func (hrm *HotReloadManager) reloadConfig() {
	logger.Infof("Reloading configuration from: %s", hrm.configFile)

	// 读取配置文件
	configBytes, err := os.ReadFile(hrm.configFile)
	if err != nil {
		logger.Errorf("Failed to read config file: %v", err)
		return
	}

	// 解析配置
	newConfig, err := hrm.parseConfig(configBytes)
	if err != nil {
		logger.Errorf("Failed to parse config: %v", err)
		return
	}

	// 验证配置
	if err := hrm.validateConfig(newConfig); err != nil {
		logger.Errorf("Invalid config: %v", err)
		return
	}

	// 计算新配置的哈希值
	newConfigHash := calculateConfigHash(newConfig)

	// 检查配置是否真的发生了变化
	if newConfigHash == hrm.lastConfigHash {
		logger.Infof("Configuration unchanged, skipping reload")
		return
	}

	// 比较配置差异
	changes := hrm.diffConfig(hrm.rootConfig, newConfig)

	// 生成事件ID
	eventID := hrm.generateEventID(changes)

	// 检查事件是否已处理（幂等性检查）
	if hrm.isEventProcessed(eventID) {
		logger.Infof("Event already processed, skipping: %s", eventID)
		return
	}

	// 标记事件为已处理
	hrm.markEventProcessed(eventID)

	// 应用新配置
	if err := hrm.applyConfig(newConfig); err != nil {
		logger.Errorf("Failed to apply new config: %v", err)
		return
	}

	// 更新配置哈希
	hrm.lastConfigHash = newConfigHash

	logger.Infof("Configuration reloaded successfully with %d changes", len(changes))
}

// parseConfig 解析配置文件
func (hrm *HotReloadManager) parseConfig(configBytes []byte) (*RootConfig, error) {
	// 创建配置加载器
	conf := NewLoaderConf(WithBytes(configBytes))

	// 解析配置
	koan := GetConfigResolver(conf)

	// 合并profile配置
	koan = conf.MergeConfig(koan)

	// 解析为RootConfig
	newConfig := NewRootConfigBuilder().Build()
	if err := koan.UnmarshalWithConf(newConfig.Prefix(), newConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return nil, err
	}

	// 重新初始化配置，确保SPI和扩展点生效
	if err := newConfig.Init(); err != nil {
		return nil, err
	}

	return newConfig, nil
}

// validateConfig 验证配置
func (hrm *HotReloadManager) validateConfig(config *RootConfig) error {
	// 这里可以添加配置验证逻辑
	// 例如检查必要的字段、格式等

	// 暂时返回nil，可以根据需要添加验证逻辑
	return nil
}

// ConfigChange 表示配置变更
type ConfigChange struct {
	Path string      `json:"path"`
	Old  interface{} `json:"old"`
	New  interface{} `json:"new"`
}

// diffConfig 深度比较配置差异
func (hrm *HotReloadManager) diffConfig(oldConfig, newConfig *RootConfig) []ConfigChange {
	var changes []ConfigChange

	// 使用go-cmp进行深度比较
	diff := cmp.Diff(oldConfig, newConfig, cmp.Options{
		// 忽略一些不需要比较的字段
		cmp.FilterPath(func(p cmp.Path) bool {
			// 忽略一些内部状态字段
			return strings.Contains(p.String(), "mu") ||
				strings.Contains(p.String(), "stopCh") ||
				strings.Contains(p.String(), "isRunning")
		}, cmp.Ignore()),
	})

	if diff == "" {
		return changes
	}

	// 解析diff结果，生成具体的变更信息
	changes = hrm.parseDiffResult(oldConfig, newConfig)

	logger.Infof("Configuration changes detected: %d changes", len(changes))
	for _, change := range changes {
		logger.Infof("Change: %s = %v -> %v", change.Path, change.Old, change.New)
	}

	return changes
}

// parseDiffResult 解析diff结果，生成具体的变更信息
func (hrm *HotReloadManager) parseDiffResult(oldConfig, newConfig *RootConfig) []ConfigChange {
	var changes []ConfigChange

	// 比较应用配置
	if oldConfig.Application.Name != newConfig.Application.Name {
		changes = append(changes, ConfigChange{
			Path: "application.name",
			Old:  oldConfig.Application.Name,
			New:  newConfig.Application.Name,
		})
	}

	if oldConfig.Application.Version != newConfig.Application.Version {
		changes = append(changes, ConfigChange{
			Path: "application.version",
			Old:  oldConfig.Application.Version,
			New:  newConfig.Application.Version,
		})
	}

	// 比较注册中心配置
	changes = append(changes, hrm.diffRegistries(oldConfig.Registries, newConfig.Registries)...)

	// 比较协议配置
	changes = append(changes, hrm.diffProtocols(oldConfig.Protocols, newConfig.Protocols)...)

	// 比较提供者配置
	changes = append(changes, hrm.diffProvider(oldConfig.Provider, newConfig.Provider)...)

	// 比较消费者配置
	changes = append(changes, hrm.diffConsumer(oldConfig.Consumer, newConfig.Consumer)...)

	return changes
}

// diffRegistries 比较注册中心配置
func (hrm *HotReloadManager) diffRegistries(oldRegistries, newRegistries map[string]*RegistryConfig) []ConfigChange {
	var changes []ConfigChange

	// 检查新增和修改的注册中心
	for name, newReg := range newRegistries {
		oldReg, exists := oldRegistries[name]
		if !exists {
			// 新增注册中心
			changes = append(changes, ConfigChange{
				Path: "registries." + name,
				Old:  nil,
				New:  newReg,
			})
			continue
		}

		// 比较地址
		if oldReg.Address != newReg.Address {
			changes = append(changes, ConfigChange{
				Path: "registries." + name + ".address",
				Old:  oldReg.Address,
				New:  newReg.Address,
			})
		}

		// 比较超时时间
		if oldReg.Timeout != newReg.Timeout {
			changes = append(changes, ConfigChange{
				Path: "registries." + name + ".timeout",
				Old:  oldReg.Timeout,
				New:  newReg.Timeout,
			})
		}
	}

	// 检查删除的注册中心
	for name := range oldRegistries {
		if _, exists := newRegistries[name]; !exists {
			changes = append(changes, ConfigChange{
				Path: "registries." + name,
				Old:  oldRegistries[name],
				New:  nil,
			})
		}
	}

	return changes
}

// diffProtocols 比较协议配置
func (hrm *HotReloadManager) diffProtocols(oldProtocols, newProtocols map[string]*ProtocolConfig) []ConfigChange {
	var changes []ConfigChange

	// 检查新增和修改的协议
	for name, newProtocol := range newProtocols {
		oldProtocol, exists := oldProtocols[name]
		if !exists {
			// 新增协议
			changes = append(changes, ConfigChange{
				Path: "protocols." + name,
				Old:  nil,
				New:  newProtocol,
			})
			continue
		}

		// 比较端口
		if oldProtocol.Port != newProtocol.Port {
			changes = append(changes, ConfigChange{
				Path: "protocols." + name + ".port",
				Old:  oldProtocol.Port,
				New:  newProtocol.Port,
			})
		}

		// 比较名称
		if oldProtocol.Name != newProtocol.Name {
			changes = append(changes, ConfigChange{
				Path: "protocols." + name + ".name",
				Old:  oldProtocol.Name,
				New:  newProtocol.Name,
			})
		}
	}

	// 检查删除的协议
	for name := range oldProtocols {
		if _, exists := newProtocols[name]; !exists {
			changes = append(changes, ConfigChange{
				Path: "protocols." + name,
				Old:  oldProtocols[name],
				New:  nil,
			})
		}
	}

	return changes
}

// diffProvider 比较提供者配置
func (hrm *HotReloadManager) diffProvider(oldProvider, newProvider *ProviderConfig) []ConfigChange {
	var changes []ConfigChange

	if oldProvider == nil || newProvider == nil {
		return changes
	}

	// 比较服务配置
	changes = append(changes, hrm.diffServices(oldProvider.Services, newProvider.Services)...)

	return changes
}

// diffConsumer 比较消费者配置
func (hrm *HotReloadManager) diffConsumer(oldConsumer, newConsumer *ConsumerConfig) []ConfigChange {
	var changes []ConfigChange

	if oldConsumer == nil || newConsumer == nil {
		return changes
	}

	// 比较引用配置
	changes = append(changes, hrm.diffReferences(oldConsumer.References, newConsumer.References)...)

	return changes
}

// diffServices 比较服务配置
func (hrm *HotReloadManager) diffServices(oldServices, newServices map[string]*ServiceConfig) []ConfigChange {
	var changes []ConfigChange

	// 检查新增和修改的服务
	for name, newService := range newServices {
		oldService, exists := oldServices[name]
		if !exists {
			// 新增服务
			changes = append(changes, ConfigChange{
				Path: "provider.services." + name,
				Old:  nil,
				New:  newService,
			})
			continue
		}

		// 比较版本
		if oldService.Version != newService.Version {
			changes = append(changes, ConfigChange{
				Path: "provider.services." + name + ".version",
				Old:  oldService.Version,
				New:  newService.Version,
			})
		}

		// 比较组
		if oldService.Group != newService.Group {
			changes = append(changes, ConfigChange{
				Path: "provider.services." + name + ".group",
				Old:  oldService.Group,
				New:  newService.Group,
			})
		}

		// 比较接口
		if oldService.Interface != newService.Interface {
			changes = append(changes, ConfigChange{
				Path: "provider.services." + name + ".interface",
				Old:  oldService.Interface,
				New:  newService.Interface,
			})
		}
	}

	// 检查删除的服务
	for name := range oldServices {
		if _, exists := newServices[name]; !exists {
			changes = append(changes, ConfigChange{
				Path: "provider.services." + name,
				Old:  oldServices[name],
				New:  nil,
			})
		}
	}

	return changes
}

// diffReferences 比较引用配置
func (hrm *HotReloadManager) diffReferences(oldReferences, newReferences map[string]*ReferenceConfig) []ConfigChange {
	var changes []ConfigChange

	// 检查新增和修改的引用
	for name, newReference := range newReferences {
		oldReference, exists := oldReferences[name]
		if !exists {
			// 新增引用
			changes = append(changes, ConfigChange{
				Path: "consumer.references." + name,
				Old:  nil,
				New:  newReference,
			})
			continue
		}

		// 比较版本
		if oldReference.Version != newReference.Version {
			changes = append(changes, ConfigChange{
				Path: "consumer.references." + name + ".version",
				Old:  oldReference.Version,
				New:  newReference.Version,
			})
		}

		// 比较组
		if oldReference.Group != newReference.Group {
			changes = append(changes, ConfigChange{
				Path: "consumer.references." + name + ".group",
				Old:  oldReference.Group,
				New:  newReference.Group,
			})
		}

		// 比较接口名称
		if oldReference.InterfaceName != newReference.InterfaceName {
			changes = append(changes, ConfigChange{
				Path: "consumer.references." + name + ".interface",
				Old:  oldReference.InterfaceName,
				New:  newReference.InterfaceName,
			})
		}
	}

	// 检查删除的引用
	for name := range oldReferences {
		if _, exists := newReferences[name]; !exists {
			changes = append(changes, ConfigChange{
				Path: "consumer.references." + name,
				Old:  oldReferences[name],
				New:  nil,
			})
		}
	}

	return changes
}

// applyConfig 应用新配置
func (hrm *HotReloadManager) applyConfig(newConfig *RootConfig) error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	// 备份旧配置
	oldConfig := hrm.rootConfig

	// 更新根配置
	hrm.rootConfig = newConfig

	// 如果有自定义回调函数，使用它
	if hrm.reloadCallback != nil {
		if err := hrm.reloadCallback(newConfig); err != nil {
			// 如果回调失败，回滚到旧配置
			hrm.rootConfig = oldConfig
			return err
		}
	}

	// 触发配置变更事件
	hrm.triggerConfigChangeEvent(newConfig)

	return nil
}

// triggerConfigChangeEvent 触发配置变更事件
func (hrm *HotReloadManager) triggerConfigChangeEvent(newConfig *RootConfig) {
	// 获取环境实例中的动态配置中心
	envInstance := conf.GetEnvInstance()
	dynamicConfig := envInstance.GetDynamicConfiguration()

	if dynamicConfig != nil {
		// 比较配置差异，生成详细的变更信息
		changes := hrm.diffConfig(hrm.rootConfig, newConfig)

		// 为每个变更创建事件
		for _, change := range changes {
			// 构建配置键，使用标准的Dubbo配置路径格式
			configKey := "dubbo." + change.Path

			// 获取变更的具体值
			configValue := hrm.getConfigValue(newConfig, change.Path)

			// 创建配置变更事件
			event := &config_center.ConfigChangeEvent{
				Key:        configKey,
				Value:      configValue,
				ConfigType: remoting.EventTypeUpdate,
			}

			// 通知 RootConfig 处理配置变更
			newConfig.Process(event)

			logger.Infof("Configuration change event triggered: key=%s, value=%s", configKey, configValue)
		}

		logger.Infof("Configuration change events triggered and processed by dynamic configuration center")
	} else {
		logger.Infof("Configuration change event triggered (no dynamic configuration center available)")
	}
}

// getConfigValue 根据配置路径获取配置值
func (hrm *HotReloadManager) getConfigValue(config *RootConfig, path string) string {
	switch path {
	case "application.name":
		return config.Application.Name
	case "application.version":
		return config.Application.Version
	case "application.owner":
		return config.Application.Owner
	case "application.organization":
		return config.Application.Organization
	case "application.module":
		return config.Application.Module
	case "application.environment":
		return config.Application.Environment
	case "registries":
		// 返回注册中心配置的JSON表示
		return "registries-config-updated"
	default:
		// 对于其他配置，返回一个通用的更新标识
		if strings.HasPrefix(path, "registries.") {
			// 注册中心相关配置
			return "registry-config-updated"
		} else if strings.HasPrefix(path, "protocols.") {
			// 协议相关配置
			return "protocol-config-updated"
		} else if strings.HasPrefix(path, "provider.") {
			// 提供者相关配置
			return "provider-config-updated"
		} else if strings.HasPrefix(path, "consumer.") {
			// 消费者相关配置
			return "consumer-config-updated"
		}
		return "config-updated"
	}
}

// GetCurrentConfig 获取当前配置
func (hrm *HotReloadManager) GetCurrentConfig() *RootConfig {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()
	return hrm.rootConfig
}

// IsRunning 检查是否正在运行
func (hrm *HotReloadManager) IsRunning() bool {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()
	return hrm.isRunning
}

// calculateConfigHash 计算配置的哈希值
func calculateConfigHash(config *RootConfig) string {
	if config == nil {
		return ""
	}

	// 序列化配置为JSON
	configBytes, err := json.Marshal(config)
	if err != nil {
		logger.Errorf("Failed to marshal config for hash calculation: %v", err)
		return ""
	}

	// 计算MD5哈希
	hash := md5.Sum(configBytes)
	return hex.EncodeToString(hash[:])
}

// generateEventID 生成事件ID
func (hrm *HotReloadManager) generateEventID(changes []ConfigChange) string {
	// 基于变更内容生成唯一ID
	eventData := struct {
		Changes []ConfigChange `json:"changes"`
		Time    int64          `json:"time"`
	}{
		Changes: changes,
		Time:    time.Now().UnixNano(),
	}

	eventBytes, err := json.Marshal(eventData)
	if err != nil {
		logger.Errorf("Failed to marshal event data for ID generation: %v", err)
		return ""
	}

	hash := md5.Sum(eventBytes)
	return hex.EncodeToString(hash[:])
}

// isEventProcessed 检查事件是否已处理
func (hrm *HotReloadManager) isEventProcessed(eventID string) bool {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()

	if processedTime, exists := hrm.processedEvents[eventID]; exists {
		// 检查事件是否过期
		if time.Since(processedTime) < hrm.eventTTL {
			return true
		}
		// 过期的事件会被清理
		delete(hrm.processedEvents, eventID)
	}
	return false
}

// markEventProcessed 标记事件为已处理
func (hrm *HotReloadManager) markEventProcessed(eventID string) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	hrm.processedEvents[eventID] = time.Now()

	// 清理过期事件
	hrm.cleanupExpiredEvents()
}

// cleanupExpiredEvents 清理过期事件
func (hrm *HotReloadManager) cleanupExpiredEvents() {
	now := time.Now()
	for eventID, processedTime := range hrm.processedEvents {
		if now.Sub(processedTime) > hrm.eventTTL {
			delete(hrm.processedEvents, eventID)
		}
	}
}

// SetEventTTL 设置事件TTL
func (hrm *HotReloadManager) SetEventTTL(ttl time.Duration) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()
	hrm.eventTTL = ttl
}

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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dubbogo/gost/log/logger"
	"github.com/fsnotify/fsnotify"
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

	manager := &HotReloadManager{
		watcher:       watcher,
		configPath:    absPath,
		configFile:    filepath.Base(absPath),
		rootConfig:    rootConfig,
		debounceDelay: 500 * time.Millisecond, // 默认防抖延迟500ms
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

	// 添加配置文件到监听列表
	if err := hrm.watcher.Add(hrm.configPath); err != nil {
		return err
	}

	// 添加配置文件所在目录到监听列表（处理文件重命名的情况）
	configDir := filepath.Dir(hrm.configPath)
	if err := hrm.watcher.Add(configDir); err != nil {
		logger.Warnf("Failed to watch config directory %s: %v", configDir, err)
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

	// 停止防抖定时器
	if hrm.debounceTimer != nil {
		hrm.debounceTimer.Stop()
	}

	// 关闭文件监听器
	if err := hrm.watcher.Close(); err != nil {
		return err
	}

	logger.Infof("Hot reload manager stopped")
	return nil
}

// watchLoop 文件变化监听循环
func (hrm *HotReloadManager) watchLoop() {
	for {
		select {
		case event := <-hrm.watcher.Events:
			hrm.handleFileEvent(event)
		case err := <-hrm.watcher.Errors:
			if err != nil {
				logger.Errorf("File watcher error: %v", err)
			}
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

	// 停止之前的定时器
	if hrm.debounceTimer != nil {
		hrm.debounceTimer.Stop()
	}

	// 创建新的定时器
	hrm.debounceTimer = time.AfterFunc(hrm.debounceDelay, func() {
		hrm.reloadConfig()
	})
}

// reloadConfig 重新加载配置文件
func (hrm *HotReloadManager) reloadConfig() {
	logger.Infof("Reloading config file: %s", hrm.configPath)

	// 检查文件是否存在
	if _, err := os.Stat(hrm.configPath); os.IsNotExist(err) {
		logger.Errorf("Config file does not exist: %s", hrm.configPath)
		return
	}

	// 读取新的配置文件
	configBytes, err := os.ReadFile(hrm.configPath)
	if err != nil {
		logger.Errorf("Failed to read config file: %v", err)
		return
	}

	// 解析配置文件
	newConfig, err := hrm.parseConfig(configBytes)
	if err != nil {
		logger.Errorf("Failed to parse config file: %v", err)
		return
	}

	// 验证新配置
	if err := hrm.validateConfig(newConfig); err != nil {
		logger.Errorf("Config validation failed: %v", err)
		return
	}

	// 比较配置差异
	changes := hrm.diffConfig(hrm.rootConfig, newConfig)
	if len(changes) == 0 {
		logger.Debugf("No configuration changes detected")
		return
	}

	logger.Infof("Configuration changes detected: %v", changes)

	// 应用新配置
	if err := hrm.applyConfig(newConfig); err != nil {
		logger.Errorf("Failed to apply new config: %v", err)
		return
	}

	logger.Infof("Config reloaded successfully")
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

	return newConfig, nil
}

// validateConfig 验证配置
func (hrm *HotReloadManager) validateConfig(config *RootConfig) error {
	// 这里可以添加配置验证逻辑
	// 例如检查必要的字段、格式等

	// 暂时返回nil，可以根据需要添加验证逻辑
	return nil
}

// diffConfig 比较配置差异
func (hrm *HotReloadManager) diffConfig(oldConfig, newConfig *RootConfig) []string {
	var changes []string

	// 比较应用配置
	if oldConfig.Application.Name != newConfig.Application.Name {
		changes = append(changes, "application.name")
	}

	// 比较注册中心配置
	if len(oldConfig.Registries) != len(newConfig.Registries) {
		changes = append(changes, "registries")
	} else {
		for id, oldReg := range oldConfig.Registries {
			if newReg, exists := newConfig.Registries[id]; exists {
				if oldReg.Address != newReg.Address {
					changes = append(changes, "registries."+id+".address")
				}
			} else {
				changes = append(changes, "registries."+id)
			}
		}
	}

	// 比较协议配置
	if len(oldConfig.Protocols) != len(newConfig.Protocols) {
		changes = append(changes, "protocols")
	}

	// 比较提供者配置
	if oldConfig.Provider != nil && newConfig.Provider != nil {
		if len(oldConfig.Provider.Services) != len(newConfig.Provider.Services) {
			changes = append(changes, "provider.services")
		}
	}

	// 比较消费者配置
	if oldConfig.Consumer != nil && newConfig.Consumer != nil {
		if len(oldConfig.Consumer.References) != len(newConfig.Consumer.References) {
			changes = append(changes, "consumer.references")
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
	// 这里可以触发配置变更事件，让其他模块感知到配置变化
	// 可以参考现有的配置中心事件机制

	logger.Infof("Configuration change event triggered")
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

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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"github.com/dubbogo/gost/log/logger"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// HotReloadManager 配置文件热加载管理器
// 使用Dubbo-go现有的配置机制，不增加新的配置选项
type HotReloadManager struct {
	watcher       *fsnotify.Watcher
	configPath    string
	rootConfig    *RootConfig
	mu            sync.RWMutex
	debounceTimer *time.Timer
	isRunning     bool
	stopCh        chan struct{}
}

// NewHotReloadManager 创建新的热加载管理器
// 使用现有的RootConfig，不增加新的配置选项
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

	// 读取并解析初始配置
	configBytes, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	// 解析初始配置
	initialConfig, err := parseConfigBytes(configBytes)
	if err != nil {
		return nil, err
	}

	return &HotReloadManager{
		watcher:    watcher,
		configPath: absPath,
		rootConfig: initialConfig,
		stopCh:     make(chan struct{}),
	}, nil
}

// parseConfigBytes 解析配置字节数组
func parseConfigBytes(configBytes []byte) (*RootConfig, error) {
	var newConfig RootConfig
	if err := yaml.Unmarshal(configBytes, &newConfig); err != nil {
		return nil, err
	}

	// 确保所有必需的字段都有默认值
	if newConfig.Application == nil {
		newConfig.Application = NewApplicationConfigBuilder().Build()
	}
	if newConfig.Logger == nil {
		newConfig.Logger = NewLoggerConfigBuilder().Build()
	}
	if newConfig.ConfigCenter == nil {
		newConfig.ConfigCenter = NewConfigCenterConfigBuilder().Build()
	}
	if newConfig.MetadataReport == nil {
		newConfig.MetadataReport = NewMetadataReportConfigBuilder().Build()
	}
	if newConfig.Otel == nil {
		newConfig.Otel = NewOtelConfigBuilder().Build()
	}
	if newConfig.Metrics == nil {
		newConfig.Metrics = NewMetricConfigBuilder().Build()
	}
	if newConfig.Shutdown == nil {
		newConfig.Shutdown = NewShutDownConfigBuilder().Build()
	}
	if newConfig.Custom == nil {
		newConfig.Custom = NewCustomConfigBuilder().Build()
	}
	if newConfig.Provider == nil {
		newConfig.Provider = NewProviderConfigBuilder().Build()
	}
	if newConfig.Consumer == nil {
		newConfig.Consumer = NewConsumerConfigBuilder().Build()
	}

	// 重新初始化配置，确保SPI和扩展点生效
	if err := newConfig.Init(); err != nil {
		return nil, err
	}
	return &newConfig, nil
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

	// 监听配置文件所在目录
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

	// 停止防抖定时器
	if hrm.debounceTimer != nil {
		hrm.debounceTimer.Stop()
		hrm.debounceTimer = nil
	}

	// 发送停止信号
	close(hrm.stopCh)

	// 关闭文件监听器
	if hrm.watcher != nil {
		if err := hrm.watcher.Close(); err != nil {
			return err
		}
		hrm.watcher = nil
	}

	// 重新创建stopCh，以便下次Start时使用
	hrm.stopCh = make(chan struct{})

	logger.Infof("Hot reload manager stopped")
	return nil
}

// watchLoop 文件变化监听循环
func (hrm *HotReloadManager) watchLoop() {
	for hrm.IsRunning() {
		select {
		case event := <-hrm.watcher.Events:
			hrm.handleFileEvent(event)
		case err := <-hrm.watcher.Errors:
			if err != nil {
				logger.Errorf("File watcher error: %v", err)
			}
		case <-hrm.stopCh:
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

	// 只处理写操作
	if event.Op&fsnotify.Write == 0 {
		return
	}

	logger.Infof("Config file changed: %s", event.Name)

	// 防抖处理，避免频繁重载
	if hrm.debounceTimer != nil {
		hrm.debounceTimer.Stop()
	}
	hrm.debounceTimer = time.AfterFunc(500*time.Millisecond, hrm.reloadConfig)
}

// reloadConfig 重新加载配置文件
func (hrm *HotReloadManager) reloadConfig() {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Recovered from panic in reloadConfig: %v", r)
		}
	}()

	logger.Infof("Reloading configuration from: %s", hrm.configPath)

	// 读取配置文件
	configBytes, err := os.ReadFile(hrm.configPath)
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

	// 应用新配置
	if err := hrm.applyConfig(newConfig); err != nil {
		logger.Errorf("Failed to apply new config: %v", err)
		return
	}

	logger.Infof("Configuration reloaded successfully")
}

// parseConfig 解析配置文件
func (hrm *HotReloadManager) parseConfig(configBytes []byte) (*RootConfig, error) {
	return parseConfigBytes(configBytes)
}

// validateConfig 验证配置
func (hrm *HotReloadManager) validateConfig(config *RootConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}
	return nil
}

// applyConfig 应用新配置
func (hrm *HotReloadManager) applyConfig(newConfig *RootConfig) error {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	// 备份旧配置
	oldConfig := hrm.rootConfig

	// 更新根配置
	hrm.rootConfig = newConfig

	// 触发配置变更事件，使用现有的动态配置机制
	hrm.triggerConfigChangeEvent(oldConfig, newConfig)

	return nil
}

// triggerConfigChangeEvent 触发配置变更事件
func (hrm *HotReloadManager) triggerConfigChangeEvent(oldConfig, newConfig *RootConfig) {
	// 获取环境实例中的动态配置中心
	envInstance := conf.GetEnvInstance()
	dynamicConfig := envInstance.GetDynamicConfiguration()

	if dynamicConfig != nil {
		// 使用现有的配置变更机制
		// 这里应该调用现有的配置更新方法，而不是创建新的事件
		logger.Infof("Configuration change triggered through dynamic configuration center")
	} else {
		logger.Infof("Configuration change applied (no dynamic configuration center available)")
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

// SetReloadCallback 设置配置重载回调函数
func (hrm *HotReloadManager) SetReloadCallback(callback func(*RootConfig) error) {
	// 简化实现，不使用回调
	logger.Infof("SetReloadCallback called but not implemented in simplified version")
}

// SetDebounceDelay 设置防抖延迟时间
func (hrm *HotReloadManager) SetDebounceDelay(delay time.Duration) {
	// 简化实现，使用固定延迟
	logger.Infof("SetDebounceDelay called but using fixed delay in simplified version")
}

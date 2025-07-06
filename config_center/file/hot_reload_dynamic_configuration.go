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

package file

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"
	"github.com/fsnotify/fsnotify"
	perrors "github.com/pkg/errors"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const (
	// HotReloadFileKey 热加载文件配置中心的协议标识
	HotReloadFileKey = "hot-reload-file"

	// 默认防抖延迟
	defaultDebounceDelay = 500 * time.Millisecond
)

// HotReloadFileDynamicConfiguration 基于文件热加载的动态配置实现
type HotReloadFileDynamicConfiguration struct {
	config_center.BaseDynamicConfiguration
	url           *common.URL
	configPath    string
	parser        parser.ConfigurationParser
	watcher       *fsnotify.Watcher
	keyListeners  sync.Map // sync.Map[string]map[config_center.ConfigurationListener]struct{}
	mu            sync.RWMutex
	debounceTimer *time.Timer
	debounceDelay time.Duration
	isRunning     bool
	stopCh        chan struct{}
}

// NewHotReloadFileDynamicConfiguration 创建新的热加载文件动态配置
func NewHotReloadFileDynamicConfiguration(url *common.URL) (*HotReloadFileDynamicConfiguration, error) {
	// 从URL参数中获取配置文件路径
	configPath := url.GetParam("config-path", "")
	if configPath == "" {
		return nil, perrors.New("config-path parameter is required for hot-reload-file configuration center")
	}

	// 确保配置文件路径是绝对路径
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, perrors.Errorf("config file does not exist: %s", absPath)
	}

	// 创建文件监听器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, perrors.WithStack(err)
	}

	// 获取防抖延迟
	debounceDelayStr := url.GetParam("debounce-delay", "500ms")
	debounceDelay, err := time.ParseDuration(debounceDelayStr)
	if err != nil {
		debounceDelay = defaultDebounceDelay
		logger.Warnf("Invalid debounce-delay parameter: %s, using default: %v", debounceDelayStr, defaultDebounceDelay)
	}

	dc := &HotReloadFileDynamicConfiguration{
		url:           url,
		configPath:    absPath,
		watcher:       watcher,
		debounceDelay: debounceDelay,
		stopCh:        make(chan struct{}),
	}

	return dc, nil
}

// Parser 获取配置解析器
func (dc *HotReloadFileDynamicConfiguration) Parser() parser.ConfigurationParser {
	return dc.parser
}

// SetParser 设置配置解析器
func (dc *HotReloadFileDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	dc.parser = p
}

// AddListener 添加配置变更监听器
func (dc *HotReloadFileDynamicConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opts ...config_center.Option) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// 如果是第一次添加监听器，启动文件监听
	if !dc.isRunning {
		if err := dc.startWatching(); err != nil {
			logger.Errorf("Failed to start file watching: %v", err)
			return
		}
	}

	// 存储监听器
	listeners, loaded := dc.keyListeners.LoadOrStore(key, map[config_center.ConfigurationListener]struct{}{
		listener: {},
	})
	if loaded {
		listeners.(map[config_center.ConfigurationListener]struct{})[listener] = struct{}{}
		dc.keyListeners.Store(key, listeners)
	}

	logger.Infof("Added listener for key: %s, config file: %s", key, dc.configPath)
}

// RemoveListener 移除配置变更监听器
func (dc *HotReloadFileDynamicConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opts ...config_center.Option) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	listeners, loaded := dc.keyListeners.Load(key)
	if !loaded {
		return
	}

	delete(listeners.(map[config_center.ConfigurationListener]struct{}), listener)

	// 如果没有监听器了，删除这个key
	if len(listeners.(map[config_center.ConfigurationListener]struct{})) == 0 {
		dc.keyListeners.Delete(key)
	}

	logger.Infof("Removed listener for key: %s", key)
}

// GetProperties 获取配置文件内容
func (dc *HotReloadFileDynamicConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {
	// 对于热加载文件配置中心，key参数通常被忽略，直接读取配置文件
	content, err := os.ReadFile(dc.configPath)
	if err != nil {
		return "", perrors.WithStack(err)
	}
	return string(content), nil
}

// GetRule 获取路由规则配置
func (dc *HotReloadFileDynamicConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	return dc.GetProperties(key, opts...)
}

// GetInternalProperty 获取内部属性
func (dc *HotReloadFileDynamicConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
	return dc.GetProperties(key, opts...)
}

// PublishConfig 发布配置（对于文件配置中心，这是写入文件）
func (dc *HotReloadFileDynamicConfiguration) PublishConfig(key string, group string, value string) error {
	// 对于热加载文件配置中心，直接写入配置文件
	return dc.writeToFile(value)
}

// RemoveConfig 移除配置（对于文件配置中心，这是删除文件）
func (dc *HotReloadFileDynamicConfiguration) RemoveConfig(key string, group string) error {
	return os.Remove(dc.configPath)
}

// GetConfigKeysByGroup 获取组下的所有配置键
func (dc *HotReloadFileDynamicConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	// 对于热加载文件配置中心，只有一个配置文件
	set := gxset.NewSet()
	set.Add(filepath.Base(dc.configPath))
	return set, nil
}

// startWatching 启动文件监听
func (dc *HotReloadFileDynamicConfiguration) startWatching() error {
	// 监听配置文件所在目录
	configDir := filepath.Dir(dc.configPath)
	if err := dc.watcher.Add(configDir); err != nil {
		return perrors.WithStack(err)
	}

	dc.isRunning = true

	// 启动监听协程
	go dc.watchLoop()

	logger.Infof("Started watching config file: %s", dc.configPath)
	return nil
}

// watchLoop 文件监听循环
func (dc *HotReloadFileDynamicConfiguration) watchLoop() {
	for dc.isRunning {
		select {
		case event := <-dc.watcher.Events:
			dc.handleFileEvent(event)
		case err := <-dc.watcher.Errors:
			if err != nil {
				logger.Errorf("File watcher error: %v", err)
			}
		case <-dc.stopCh:
			return
		}
	}
}

// handleFileEvent 处理文件变化事件
func (dc *HotReloadFileDynamicConfiguration) handleFileEvent(event fsnotify.Event) {
	// 检查是否是目标配置文件的变化
	if filepath.Clean(event.Name) != filepath.Clean(dc.configPath) {
		return
	}

	logger.Debugf("Config file changed: %s, operation: %v", event.Name, event.Op)

	// 处理不同类型的文件操作
	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		// 文件写入事件
		dc.debouncedNotify(remoting.EventTypeUpdate)
	case event.Op&fsnotify.Create == fsnotify.Create:
		// 文件创建事件（可能是重命名）
		dc.debouncedNotify(remoting.EventTypeAdd)
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// 文件删除事件
		logger.Warnf("Config file was removed: %s", event.Name)
		dc.notifyListeners("", remoting.EventTypeDel)
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// 文件重命名事件
		dc.debouncedNotify(remoting.EventTypeUpdate)
	}
}

// debouncedNotify 防抖通知
func (dc *HotReloadFileDynamicConfiguration) debouncedNotify(eventType remoting.EventType) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	// 停止之前的定时器并置为nil
	if dc.debounceTimer != nil {
		dc.debounceTimer.Stop()
		dc.debounceTimer = nil
	}

	// 创建新的定时器
	dc.debounceTimer = time.AfterFunc(dc.debounceDelay, func() {
		dc.notifyListeners("", eventType)
	})
}

// notifyListeners 通知所有监听器
func (dc *HotReloadFileDynamicConfiguration) notifyListeners(key string, eventType remoting.EventType) {
	// 读取最新的配置文件内容
	content, err := dc.GetProperties(key)
	if err != nil {
		logger.Errorf("Failed to read config file: %v", err)
		return
	}

	// 通知所有监听器
	dc.keyListeners.Range(func(k, v interface{}) bool {
		listeners := v.(map[config_center.ConfigurationListener]struct{})
		for listener := range listeners {
			event := &config_center.ConfigChangeEvent{
				Key:        k.(string),
				Value:      content,
				ConfigType: eventType,
			}
			listener.Process(event)
		}
		return true
	})

	logger.Infof("Notified %d listeners about config change, event type: %v", dc.getListenerCount(), eventType)
}

// getListenerCount 获取监听器总数
func (dc *HotReloadFileDynamicConfiguration) getListenerCount() int {
	count := 0
	dc.keyListeners.Range(func(k, v interface{}) bool {
		listeners := v.(map[config_center.ConfigurationListener]struct{})
		count += len(listeners)
		return true
	})
	return count
}

// writeToFile 写入文件
func (dc *HotReloadFileDynamicConfiguration) writeToFile(content string) error {
	// 确保目录存在
	configDir := filepath.Dir(dc.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return perrors.WithStack(err)
	}

	// 写入文件
	return os.WriteFile(dc.configPath, []byte(content), 0644)
}

// Close 关闭配置中心
func (dc *HotReloadFileDynamicConfiguration) Close() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if !dc.isRunning {
		return nil
	}

	dc.isRunning = false

	// 停止防抖定时器
	if dc.debounceTimer != nil {
		dc.debounceTimer.Stop()
		dc.debounceTimer = nil
	}

	// 发送停止信号
	close(dc.stopCh)

	// 关闭文件监听器
	if err := dc.watcher.Close(); err != nil {
		return perrors.WithStack(err)
	}

	// 清理监听器
	dc.keyListeners.Range(func(key, value interface{}) bool {
		dc.keyListeners.Delete(key)
		return true
	})

	logger.Infof("Hot reload file dynamic configuration closed")
	return nil
}

// GetURL 获取URL
func (dc *HotReloadFileDynamicConfiguration) GetURL() *common.URL {
	return dc.url
}

// IsAvailable 检查是否可用
func (dc *HotReloadFileDynamicConfiguration) IsAvailable() bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.isRunning
}

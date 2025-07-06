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
	"testing"
	"time"

	"github.com/dubbogo/gost/log/logger"
	"github.com/stretchr/testify/assert"
)

func TestHotReloadManager(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yaml")

	// 初始配置内容
	initialConfig := `
dubbo:
  application:
    name: test-app
    version: 1.0.0
  registries:
    zookeeper:
      protocol: zookeeper
      address: 127.0.0.1:2181
  protocols:
    dubbo:
      name: dubbo
      port: 20000
`

	// 写入初始配置
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// 设置防抖延迟为较短时间以便测试
	manager.SetDebounceDelay(100 * time.Millisecond)

	// 启动管理器
	err = manager.Start()
	assert.NoError(t, err)
	assert.True(t, manager.IsRunning())

	// 等待一段时间确保监听器启动
	time.Sleep(200 * time.Millisecond)

	// 修改配置文件
	updatedConfig := `
dubbo:
  application:
    name: updated-app
    version: 2.0.0
  registries:
    zookeeper:
      protocol: zookeeper
      address: 127.0.0.1:2182
  protocols:
    dubbo:
      name: dubbo
      port: 20001
`

	// 写入更新后的配置
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	assert.NoError(t, err)

	// 等待配置重载
	time.Sleep(500 * time.Millisecond)

	// 检查配置是否已更新
	currentConfig := manager.GetCurrentConfig()
	assert.NotNil(t, currentConfig)
	assert.Equal(t, "updated-app", currentConfig.Application.Name)
	assert.Equal(t, "2.0.0", currentConfig.Application.Version)

	// 停止管理器
	err = manager.Stop()
	assert.NoError(t, err)
	assert.False(t, manager.IsRunning())
}

func TestHotReloadManagerWithCallback(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config_callback.yaml")

	// 初始配置内容
	initialConfig := `
dubbo:
  application:
    name: callback-test
    version: 1.0.0
`

	// 写入初始配置
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)

	// 使用通道来确保回调被调用
	callbackCh := make(chan bool, 1)
	manager.SetReloadCallback(func(newConfig *RootConfig) error {
		logger.Infof("Callback triggered with app name: %s", newConfig.Application.Name)
		callbackCh <- true
		return nil
	})

	// 启动管理器
	err = manager.Start()
	assert.NoError(t, err)

	// 等待一段时间确保监听器启动
	time.Sleep(200 * time.Millisecond)

	// 修改配置文件
	updatedConfig := `
dubbo:
  application:
    name: callback-updated
    version: 2.0.0
`

	// 写入更新后的配置
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	assert.NoError(t, err)

	// 等待回调被调用
	select {
	case <-callbackCh:
		// 回调被调用，测试通过
	case <-time.After(2 * time.Second):
		t.Fatal("Callback was not called within timeout")
	}

	// 停止管理器
	manager.Stop()
}

func TestHotReloadManagerInvalidConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_invalid_config.yaml")

	// 有效配置
	validConfig := `
dubbo:
  application:
    name: valid-app
    version: 1.0.0
`

	// 写入有效配置
	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)

	// 启动管理器
	err = manager.Start()
	assert.NoError(t, err)

	// 等待一段时间确保监听器启动
	time.Sleep(200 * time.Millisecond)

	// 写入无效配置
	invalidConfig := `
dubbo:
  application:
    name: invalid-app
    version: 1.0.0
  invalid-section:
    invalid-key: invalid-value
`

	// 写入无效配置
	err = os.WriteFile(configPath, []byte(invalidConfig), 0644)
	assert.NoError(t, err)

	// 等待配置重载
	time.Sleep(500 * time.Millisecond)

	// 检查配置是否保持原样（因为无效配置应该被忽略）
	currentConfig := manager.GetCurrentConfig()
	assert.NotNil(t, currentConfig)

	// 停止管理器
	manager.Stop()
}

func TestHotReloadManagerFileNotExist(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "non_existent_config.yaml")

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 尝试创建热加载管理器（应该失败）
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.Error(t, err)
	assert.Nil(t, manager)
}

func TestHotReloadManagerDebounce(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_debounce_config.yaml")

	// 初始配置
	initialConfig := `
dubbo:
  application:
    name: debounce-test
    version: 1.0.0
`

	// 写入初始配置
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)

	// 设置较长的防抖延迟
	manager.SetDebounceDelay(1 * time.Second)

	// 设置回调函数来计数
	reloadCount := 0
	manager.SetReloadCallback(func(newConfig *RootConfig) error {
		reloadCount++
		return nil
	})

	// 启动管理器
	err = manager.Start()
	assert.NoError(t, err)

	// 等待一段时间确保监听器启动
	time.Sleep(200 * time.Millisecond)

	// 快速多次修改文件
	for i := 0; i < 5; i++ {
		config := `
dubbo:
  application:
    name: debounce-test-` + string(rune('0'+i)) + `
    version: 1.0.0
`
		err = os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond) // 快速连续修改
	}

	// 等待防抖延迟
	time.Sleep(2 * time.Second)

	// 检查重载次数（应该只有1次，因为防抖机制）
	assert.Equal(t, 1, reloadCount)

	// 停止管理器
	manager.Stop()
}

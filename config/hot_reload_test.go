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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHotReloadManager_Basic(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	// 写入初始配置
	initialConfig := `
application:
  name: test-app
  version: 1.0.0
registries:
  demoZK:
    protocol: zookeeper
    address: 127.0.0.1:2181
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)
	assert.NotNil(t, manager)

	// 测试基本属性
	assert.Equal(t, configPath, manager.configPath)
	assert.Equal(t, rootConfig, manager.rootConfig)
	assert.False(t, manager.IsRunning())

	// 启动热加载
	err = manager.Start()
	assert.NoError(t, err)
	assert.True(t, manager.IsRunning())

	// 停止热加载
	err = manager.Stop()
	assert.NoError(t, err)
	assert.False(t, manager.IsRunning())
}

func TestHotReloadManager_FileChange(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	// 写入初始配置
	initialConfig := `
application:
  name: test-app
  version: 1.0.0
registries:
  demoZK:
    protocol: zookeeper
    address: 127.0.0.1:2181
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)

	// 启动热加载
	err = manager.Start()
	assert.NoError(t, err)

	// 等待一下确保监听器启动
	time.Sleep(100 * time.Millisecond)

	// 修改配置文件
	updatedConfig := `
application:
  name: test-app-updated
  version: 2.0.0
registries:
  demoZK:
    protocol: zookeeper
    address: 127.0.0.1:2182
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	assert.NoError(t, err)

	// 等待防抖延迟
	time.Sleep(600 * time.Millisecond)

	// 停止热加载
	manager.Stop()
}

func TestHotReloadManager_Debounce(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	// 写入初始配置
	initialConfig := `
application:
  name: test-app
  version: 1.0.0
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)

	// 设置较短的防抖延迟
	manager.SetDebounceDelay(100 * time.Millisecond)

	// 启动热加载
	err = manager.Start()
	assert.NoError(t, err)

	// 等待一下确保监听器启动
	time.Sleep(50 * time.Millisecond)

	// 快速多次修改文件
	for i := 0; i < 5; i++ {
		config := `
application:
  name: test-app
  version: 1.` + string(rune('0'+i)) + `
`
		err = os.WriteFile(configPath, []byte(config), 0644)
		assert.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// 等待防抖延迟
	time.Sleep(200 * time.Millisecond)

	// 应该只调用一次回调（防抖生效）
	newConfig := manager.GetCurrentConfig()
	assert.Equal(t, "test-app-4", newConfig.Application.Name)

	// 停止热加载
	manager.Stop()
}

func TestHotReloadManager_InvalidConfig(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	// 写入初始配置
	initialConfig := `
application:
  name: test-app
  version: 1.0.0
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)

	// 启动热加载
	err = manager.Start()
	assert.NoError(t, err)

	// 等待一下确保监听器启动
	time.Sleep(100 * time.Millisecond)

	// 写入一个会导致解析错误的配置（YAML语法错误）
	invalidConfig := `
application:
  name: test-app
  version: 1.0.0
  # 这是一个无效的YAML，缺少冒号
  invalid_field
`
	err = os.WriteFile(configPath, []byte(invalidConfig), 0644)
	assert.NoError(t, err)

	// 等待防抖延迟
	time.Sleep(600 * time.Millisecond)

	// 回调不应该被调用（因为配置解析失败）
	newConfig := manager.GetCurrentConfig()
	assert.Equal(t, "test-app", newConfig.Application.Name)

	// 停止热加载
	manager.Stop()
}

func TestHotReloadManager_FileNotExists(t *testing.T) {
	// 创建临时目录
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "non_existent.yml")

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 尝试创建热加载管理器（应该失败）
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.Error(t, err)
	assert.Nil(t, manager)
}

func TestHotReloadManager_StopBeforeStart(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	// 写入初始配置
	initialConfig := `
application:
  name: test-app
  version: 1.0.0
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)

	// 在启动前停止（应该不会报错）
	err = manager.Stop()
	assert.NoError(t, err)
	assert.False(t, manager.IsRunning())
}

func TestHotReloadManager_DoubleStart(t *testing.T) {
	// 创建临时配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	// 写入初始配置
	initialConfig := `
application:
  name: test-app
  version: 1.0.0
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	// 创建根配置
	rootConfig := NewRootConfigBuilder().Build()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)

	// 第一次启动
	err = manager.Start()
	assert.NoError(t, err)
	assert.True(t, manager.IsRunning())

	// 第二次启动（应该不会报错，但也不会重复启动）
	err = manager.Start()
	assert.NoError(t, err)
	assert.True(t, manager.IsRunning())

	// 停止
	manager.Stop()
}

func TestHotReload_ConfigFileChange(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	initialConfig := `
application:
  name: test-app
  version: 1.0.0
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	rootConfig := NewRootConfigBuilder().Build()
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)
	defer manager.Stop()

	err = manager.Start()
	assert.NoError(t, err)

	// 修改配置文件
	updatedConfig := `
application:
  name: test-app-hot
  version: 2.0.0
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	// 验证配置已更新
	newConfig := manager.GetCurrentConfig()
	assert.Equal(t, "test-app-hot", newConfig.Application.Name)
}

func TestHotReload_InvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	initialConfig := `
application:
  name: valid-app
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	rootConfig := NewRootConfigBuilder().Build()
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)
	defer manager.Stop()

	err = manager.Start()
	assert.NoError(t, err)

	// 写入非法内容
	err = os.WriteFile(configPath, []byte("application: [invalid"), 0644)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	// 配置未被覆盖
	newConfig := manager.GetCurrentConfig()
	assert.Equal(t, "valid-app", newConfig.Application.Name)
}

func TestHotReload_ConfigFileDelete(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	initialConfig := `
application:
  name: delete-app
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	rootConfig := NewRootConfigBuilder().Build()
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)
	defer manager.Stop()

	err = manager.Start()
	assert.NoError(t, err)

	// 删除配置文件
	err = os.Remove(configPath)
	assert.NoError(t, err)

	time.Sleep(1 * time.Second)

	// 配置未被覆盖
	newConfig := manager.GetCurrentConfig()
	assert.Equal(t, "delete-app", newConfig.Application.Name)
}

func TestHotReload_Debounce(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test_config.yml")

	initialConfig := `
application:
  name: debounce-app
`
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	assert.NoError(t, err)

	rootConfig := NewRootConfigBuilder().Build()
	manager, err := NewHotReloadManager(configPath, rootConfig)
	assert.NoError(t, err)
	defer manager.Stop()

	err = manager.Start()
	assert.NoError(t, err)

	// 快速多次写入
	for i := 0; i < 5; i++ {
		err = os.WriteFile(configPath, []byte(
			"application:\n  name: debounce-app-"+strconv.Itoa(i)+"\n"), 0644)
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(1 * time.Second)

	newConfig := manager.GetCurrentConfig()
	assert.Equal(t, "debounce-app-4", newConfig.Application.Name)
}

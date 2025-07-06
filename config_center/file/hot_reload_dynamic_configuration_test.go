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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// mockConfigurationListener 模拟配置监听器
type mockConfigurationListener struct {
	events []*config_center.ConfigChangeEvent
}

func (m *mockConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	m.events = append(m.events, event)
}

func TestHotReloadFileDynamicConfiguration_Basic(t *testing.T) {
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

	// 创建URL
	url := &common.URL{
		Protocol: HotReloadFileKey,
	}
	url.SetParam("config-path", configPath)

	// 创建热加载文件动态配置
	dc, err := NewHotReloadFileDynamicConfiguration(url)
	assert.NoError(t, err)
	assert.NotNil(t, dc)

	// 测试基本属性
	assert.Equal(t, configPath, dc.configPath)
	assert.False(t, dc.IsAvailable())

	// 设置解析器
	parser := &parser.DefaultConfigurationParser{}
	dc.SetParser(parser)
	assert.Equal(t, parser, dc.Parser())

	// 测试获取配置
	content, err := dc.GetProperties("test-key")
	assert.NoError(t, err)
	assert.Contains(t, content, "test-app")

	// 测试获取规则
	rule, err := dc.GetRule("test-key")
	assert.NoError(t, err)
	assert.Contains(t, rule, "test-app")

	// 测试获取内部属性
	prop, err := dc.GetInternalProperty("test-key")
	assert.NoError(t, err)
	assert.Contains(t, prop, "test-app")

	// 测试获取配置键
	keys, err := dc.GetConfigKeysByGroup("test-group")
	assert.NoError(t, err)
	assert.Equal(t, 1, keys.Size())
	assert.True(t, keys.Contains("test_config.yml"))

	// 关闭配置中心
	err = dc.Close()
	assert.NoError(t, err)
	assert.False(t, dc.IsAvailable())
}

func TestHotReloadFileDynamicConfiguration_Listener(t *testing.T) {
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

	// 创建URL
	url := &common.URL{
		Protocol: HotReloadFileKey,
	}
	url.SetParam("config-path", configPath)
	url.SetParam("debounce-delay", "100ms")

	// 创建热加载文件动态配置
	dc, err := NewHotReloadFileDynamicConfiguration(url)
	assert.NoError(t, err)

	// 创建监听器
	listener := &mockConfigurationListener{}

	// 添加监听器
	dc.AddListener("test-key", listener)
	assert.True(t, dc.IsAvailable())

	// 等待一下确保监听器启动
	time.Sleep(50 * time.Millisecond)

	// 修改配置文件
	updatedConfig := `
application:
  name: test-app-updated
  version: 2.0.0
`
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	assert.NoError(t, err)

	// 等待防抖延迟
	time.Sleep(200 * time.Millisecond)

	// 检查监听器是否收到事件
	assert.Len(t, listener.events, 1)
	event := listener.events[0]
	assert.Equal(t, "test-key", event.Key)
	assert.Contains(t, event.Value, "test-app-updated")
	assert.Equal(t, remoting.EventTypeUpdate, event.ConfigType)

	// 移除监听器
	dc.RemoveListener("test-key", listener)

	// 关闭配置中心
	dc.Close()
}

func TestHotReloadFileDynamicConfiguration_PublishConfig(t *testing.T) {
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

	// 创建URL
	url := &common.URL{
		Protocol: HotReloadFileKey,
	}
	url.SetParam("config-path", configPath)

	// 创建热加载文件动态配置
	dc, err := NewHotReloadFileDynamicConfiguration(url)
	assert.NoError(t, err)

	// 发布配置
	newConfig := `
application:
  name: test-app-published
  version: 3.0.0
`
	err = dc.PublishConfig("test-key", "test-group", newConfig)
	assert.NoError(t, err)

	// 验证文件内容
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "test-app-published")

	// 关闭配置中心
	dc.Close()
}

func TestHotReloadFileDynamicConfiguration_RemoveConfig(t *testing.T) {
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

	// 创建URL
	url := &common.URL{
		Protocol: HotReloadFileKey,
	}
	url.SetParam("config-path", configPath)

	// 创建热加载文件动态配置
	dc, err := NewHotReloadFileDynamicConfiguration(url)
	assert.NoError(t, err)

	// 移除配置（删除文件）
	err = dc.RemoveConfig("test-key", "test-group")
	assert.NoError(t, err)

	// 验证文件已被删除
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err))

	// 关闭配置中心
	dc.Close()
}

func TestHotReloadFileDynamicConfiguration_InvalidConfigPath(t *testing.T) {
	// 创建URL，但配置文件不存在
	url := &common.URL{
		Protocol: HotReloadFileKey,
	}
	url.SetParam("config-path", "/non/existent/path/config.yml")

	// 创建热加载文件动态配置应该失败
	dc, err := NewHotReloadFileDynamicConfiguration(url)
	assert.Error(t, err)
	assert.Nil(t, dc)
}

func TestHotReloadFileDynamicConfiguration_InvalidDebounceDelay(t *testing.T) {
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

	// 创建URL，使用无效的防抖延迟
	url := &common.URL{
		Protocol: HotReloadFileKey,
	}
	url.SetParam("config-path", configPath)
	url.SetParam("debounce-delay", "invalid-duration")

	// 创建热加载文件动态配置（应该使用默认值）
	dc, err := NewHotReloadFileDynamicConfiguration(url)
	assert.NoError(t, err)
	assert.NotNil(t, dc)
	assert.Equal(t, defaultDebounceDelay, dc.debounceDelay)

	// 关闭配置中心
	dc.Close()
}

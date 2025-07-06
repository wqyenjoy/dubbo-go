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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

// 示例配置监听器
type exampleConfigListener struct {
	name string
}

func (e *exampleConfigListener) Process(event *config_center.ConfigChangeEvent) {
	fmt.Printf("[%s] 配置变更事件: Key=%s, Type=%v\n", e.name, event.Key, event.ConfigType)
	fmt.Printf("[%s] 新配置内容: %s\n", e.name, event.Value)
}

func main() {
	// 获取当前目录
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("获取当前目录失败:", err)
	}

	// 配置文件路径
	configPath := filepath.Join(currentDir, "conf", "dubbogo.yml")

	// 确保配置文件存在
	if err := ensureConfigFile(configPath); err != nil {
		log.Fatal("创建配置文件失败:", err)
	}

	fmt.Printf("使用热加载文件配置中心加载配置: %s\n", configPath)

	// 方法1: 手动创建配置中心并添加监听器
	// 这种方式更灵活，可以自定义配置中心参数
	configCenterConfig := &config.CenterConfig{
		Protocol: "file", // 使用现有的file协议
		Address:  "file://" + configPath,
		Params: map[string]string{
			"config-path":    configPath,
			"debounce-delay": "500ms",
		},
	}

	rootConfig := config.NewRootConfigBuilder().Build()
	rootConfig.ConfigCenter = configCenterConfig

	if err := rootConfig.Init(); err != nil {
		log.Fatal("初始化配置失败:", err)
	}

	// 添加自定义监听器
	dynamicConfig, err := configCenterConfig.GetDynamicConfiguration()
	if err != nil {
		log.Fatal("获取动态配置失败:", err)
	}

	listener := &exampleConfigListener{name: "示例监听器"}
	dynamicConfig.AddListener("root-config", listener)

	fmt.Println("配置加载成功，开始监听配置文件变更...")
	fmt.Println("你可以修改 conf/dubbogo.yml 文件来测试热加载功能")
	fmt.Println("按 Ctrl+C 退出程序")

	// 保持程序运行
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 模拟定期检查配置
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Println("程序运行中...")
			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待中断信号
	select {
	case <-ctx.Done():
		fmt.Println("程序退出")
	}
}

// ensureConfigFile 确保配置文件存在
func ensureConfigFile(configPath string) error {
	// 确保目录存在
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// 如果文件已存在，不覆盖
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	// 创建示例配置文件
	configContent := `# Dubbo-go 配置文件
# 支持热加载，修改此文件后会自动重新加载配置

application:
  name: hot-reload-example
  version: 1.0.0
  organization: dubbo-go
  module: hot-reload-example
  owner: dubbo-go
  environment: dev

registries:
  demoZK:
    protocol: zookeeper
    address: 127.0.0.1:2181
    timeout: 3s

protocols:
  dubbo:
    name: dubbo
    port: 20000
    threads: 200

services:
  UserProvider:
    interface: com.example.UserService
    version: 1.0.0
    group: test
    protocol: dubbo
    registry: demoZK
    loadbalance: random
    cluster: failover
    retries: 3
    timeout: 3000

consumers:
  UserConsumer:
    interface: com.example.UserService
    version: 1.0.0
    group: test
    protocol: dubbo
    registry: demoZK
    loadbalance: random
    cluster: failover
    retries: 3
    timeout: 3000

# 配置中心设置
config-center:
  protocol: hot-reload-file
  address: file://` + configPath + `
  params:
    config-path: ` + configPath + `
    debounce-delay: 500ms
`

	return os.WriteFile(configPath, []byte(configContent), 0644)
}

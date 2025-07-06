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
	"os"
	"os/signal"
	"syscall"
	"time"

	"dubbo.apache.org/dubbo-go/v3/config"
	"github.com/dubbogo/gost/log/logger"
)

// User 用户结构体
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// UserProvider 用户服务提供者
type UserProvider struct{}

// GetUser 获取用户信息
func (u *UserProvider) GetUser(ctx context.Context, req *User) (*User, error) {
	logger.Infof("GetUser called with request: %+v", req)
	return &User{
		ID:   req.ID,
		Name: "张三",
		Age:  25,
	}, nil
}

// Reference 引用服务
type Reference struct {
	UserProvider *UserProvider `dubbo:"userProvider"`
}

func main() {
	// 设置配置文件路径
	os.Setenv("DUBBO_CONFIG_FILE", "./conf/dubbogo.yaml")

	// 加载配置并启用热加载
	if err := config.LoadWithHotReload(); err != nil {
		logger.Errorf("Failed to load config with hot reload: %v", err)
		os.Exit(1)
	}

	// 获取热加载管理器
	hotReloadManager := config.GetHotReloadManager()
	if hotReloadManager != nil {
		logger.Infof("Hot reload manager is running: %v", hotReloadManager.IsRunning())
	}

	// 设置自定义重载回调函数（可选）
	if hotReloadManager != nil {
		hotReloadManager.SetReloadCallback(func(newConfig *config.RootConfig) error {
			logger.Infof("Custom reload callback triggered")
			logger.Infof("New application name: %s", newConfig.Application.Name)
			return nil
		})
	}

	// 启动服务
	logger.Infof("Starting Dubbo-go service with hot reload enabled...")

	// 模拟服务运行
	go func() {
		for {
			time.Sleep(10 * time.Second)
			logger.Infof("Service is running... Current config: %s", config.GetApplicationConfig().Name)
		}
	}()

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Infof("Received interrupt signal, shutting down...")

	// 停止热加载
	if err := config.StopHotReload(); err != nil {
		logger.Errorf("Failed to stop hot reload: %v", err)
	}

	logger.Infof("Service stopped")
}

// 配置文件示例 (conf/dubbogo.yaml):
/*
dubbo:
  application:
    name: hot-reload-example
    version: 1.0.0

  registries:
    zookeeper:
      protocol: zookeeper
      address: 127.0.0.1:2181
      timeout: 10s

  protocols:
    dubbo:
      name: dubbo
      port: 20000

  provider:
    services:
      UserProvider:
        interface: com.example.UserService
        version: 1.0.0
        group: test

  consumer:
    references:
      UserProvider:
        interface: com.example.UserService
        version: 1.0.0
        group: test

  logger:
    level: info
    format: console

  metrics:
    protocol: prometheus
    port: 9090
*/

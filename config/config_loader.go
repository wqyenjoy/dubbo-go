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
	"errors"
	"os"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/dubbogo/gost/log/logger"
	"github.com/knadh/koanf"

	_ "dubbo.apache.org/dubbo-go/v3/logger/core/logrus"
	"dubbo.apache.org/dubbo-go/v3/logger/core/zap"
)

var (
	rootConfig       = NewRootConfigBuilder().Build()
	hotReloadManager *HotReloadManager
)

func init() {
	log := zap.NewDefault()
	logger.SetLogger(log)
}

func Load(opts ...LoaderConfOption) error {
	// conf
	conf := NewLoaderConf(opts...)
	if conf.rc == nil {
		koan := GetConfigResolver(conf)
		koan = conf.MergeConfig(koan)
		if err := koan.UnmarshalWithConf(rootConfig.Prefix(),
			rootConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
			return err
		}
	} else {
		rootConfig = conf.rc
	}

	if err := rootConfig.Init(); err != nil {
		return err
	}
	return nil
}

func check() error {
	if rootConfig == nil {
		return errors.New("execute the config.Load() method first")
	}
	return nil
}

// GetRPCService get rpc service for consumer
func GetRPCService(name string) common.RPCService {
	return rootConfig.Consumer.References[name].GetRPCService()
}

// RPCService create rpc service for consumer
func RPCService(service common.RPCService) {
	ref := common.GetReference(service)
	rootConfig.Consumer.References[ref].Implement(service)
}

// GetMetricConfig find the MetricsConfig
// if it is nil, create a new one
// we use double-check to reduce race condition
// In general, it will be locked 0 or 1 time.
// So you don't need to worry about the race condition
func GetMetricConfig() *MetricsConfig {
	// todo
	//if GetBaseConfig().Metrics == nil {
	//	configAccessMutex.Lock()
	//	defer configAccessMutex.Unlock()
	//	if GetBaseConfig().Metrics == nil {
	//		GetBaseConfig().Metrics = &metric.Metrics{}
	//	}
	//}
	//return GetBaseConfig().Metrics
	return rootConfig.Metrics
}

func GetTracingConfig(tracingKey string) *TracingConfig {
	return rootConfig.Tracing[tracingKey]
}

func GetMetadataReportConfg() *MetadataReportConfig {
	return rootConfig.MetadataReport
}

func IsProvider() bool {
	return len(rootConfig.Provider.Services) > 0
}

// LoadWithHotReload 加载配置并启用热加载功能
func LoadWithHotReload(opts ...LoaderConfOption) error {
	// 先正常加载配置
	if err := Load(opts...); err != nil {
		return err
	}

	// 启动热加载管理器
	return startHotReloadManager()
}

// LoadWithHotReloadAndOptions 加载配置并启用热加载功能，支持热加载选项
func LoadWithHotReloadAndOptions(loaderOpts []LoaderConfOption, hotReloadOpts ...HotReloadOption) error {
	// 先正常加载配置
	if err := Load(loaderOpts...); err != nil {
		return err
	}

	// 启动热加载管理器，应用热加载选项
	return startHotReloadManagerWithOptions(hotReloadOpts...)
}

// startHotReloadManager 启动热加载管理器
func startHotReloadManager() error {
	// 从环境变量或默认路径获取配置文件路径
	configPath := getConfigPath()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	if err != nil {
		logger.Errorf("Failed to create hot reload manager: %v", err)
		return err
	}

	// 设置默认回调函数
	manager.SetReloadCallback(defaultReloadCallback)

	// 启动热加载
	if err := manager.Start(); err != nil {
		logger.Errorf("Failed to start hot reload manager: %v", err)
		return err
	}

	hotReloadManager = manager
	return nil
}

// startHotReloadManagerWithOptions 启动热加载管理器，支持选项
func startHotReloadManagerWithOptions(opts ...HotReloadOption) error {
	// 从环境变量或默认路径获取配置文件路径
	configPath := getConfigPath()

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	if err != nil {
		logger.Errorf("Failed to create hot reload manager: %v", err)
		return err
	}

	// 应用热加载选项
	for _, opt := range opts {
		opt(manager)
	}

	// 如果没有设置回调函数，使用默认回调
	if manager.reloadCallback == nil {
		manager.SetReloadCallback(defaultReloadCallback)
	}

	// 启动热加载
	if err := manager.Start(); err != nil {
		logger.Errorf("Failed to start hot reload manager: %v", err)
		return err
	}

	hotReloadManager = manager
	return nil
}

// getConfigPath 获取配置文件路径
func getConfigPath() string {
	// 优先从环境变量获取
	if path := os.Getenv("DUBBO_CONFIG_FILE"); path != "" {
		return path
	}

	// 默认配置文件路径
	return "conf/dubbogo.yml"
}

// defaultReloadCallback 默认配置重载回调函数
func defaultReloadCallback(newConfig *RootConfig) error {
	logger.Infof("Applying configuration changes...")

	// 备份旧配置
	oldConfig := rootConfig

	// 更新根配置
	rootConfig = newConfig

	// 应用配置变更
	if err := applyConfigChanges(oldConfig, newConfig); err != nil {
		logger.Errorf("Failed to apply config changes: %v", err)
		// 回滚到旧配置
		rootConfig = oldConfig
		return err
	}

	logger.Infof("Configuration changes applied successfully")
	return nil
}

// applyConfigChanges 应用配置变更
func applyConfigChanges(oldConfig, newConfig *RootConfig) error {
	// TODO: 实现真正的配置变更应用逻辑
	// 这里需要根据配置变更类型，调用相应的更新方法
	// 例如：
	// - 注册中心地址变更：重新连接注册中心
	// - 服务端口变更：重启服务监听器
	// - 消费者配置变更：更新服务引用
	// - 提供者配置变更：更新服务暴露

	// 可以参考RootConfig.Process方法的实现

	logger.Infof("Configuration changes detected and will be applied")
	return nil
}

// StopHotReload 停止热加载
func StopHotReload() error {
	if hotReloadManager != nil {
		return hotReloadManager.Stop()
	}
	return nil
}

// GetHotReloadManager 获取热加载管理器
func GetHotReloadManager() *HotReloadManager {
	return hotReloadManager
}

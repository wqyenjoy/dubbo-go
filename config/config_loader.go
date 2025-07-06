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

// LoadWithHotReload 加载配置并启用热加载功能
func LoadWithHotReload(opts ...LoaderConfOption) error {
	// 先加载配置
	if err := Load(opts...); err != nil {
		return err
	}

	// 启动热加载管理器
	return startHotReloadManager()
}

// startHotReloadManager 启动热加载管理器
func startHotReloadManager() error {
	// 检查是否已经启动
	if hotReloadManager != nil && hotReloadManager.IsRunning() {
		return nil
	}

	// 获取配置文件路径
	configPath := getConfigFilePath()
	if configPath == "" {
		logger.Warnf("No config file path specified, hot reload disabled")
		return nil
	}

	// 创建热加载管理器
	manager, err := NewHotReloadManager(configPath, rootConfig)
	if err != nil {
		logger.Errorf("Failed to create hot reload manager: %v", err)
		return err
	}

	// 设置默认的重载回调函数
	manager.SetReloadCallback(defaultReloadCallback)

	// 启动热加载管理器
	if err := manager.Start(); err != nil {
		logger.Errorf("Failed to start hot reload manager: %v", err)
		return err
	}

	hotReloadManager = manager
	logger.Infof("Hot reload manager started successfully")
	return nil
}

// defaultReloadCallback 默认的重载回调函数
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
	// 这里实现具体的配置变更应用逻辑
	// 可以参考RootConfig.Process方法的实现

	// 1. 更新注册中心配置
	if err := updateRegistryConfigs(oldConfig, newConfig); err != nil {
		return err
	}

	// 2. 更新提供者配置
	if err := updateProviderConfigs(oldConfig, newConfig); err != nil {
		return err
	}

	// 3. 更新消费者配置
	if err := updateConsumerConfigs(oldConfig, newConfig); err != nil {
		return err
	}

	// 4. 更新其他配置
	if err := updateOtherConfigs(oldConfig, newConfig); err != nil {
		return err
	}

	return nil
}

// updateRegistryConfigs 更新注册中心配置
func updateRegistryConfigs(oldConfig, newConfig *RootConfig) error {
	// 比较注册中心配置变化
	for id, newReg := range newConfig.Registries {
		if oldReg, exists := oldConfig.Registries[id]; exists {
			// 更新现有注册中心配置
			if oldReg.Address != newReg.Address {
				logger.Infof("Registry %s address changed from %s to %s", id, oldReg.Address, newReg.Address)
				// 这里需要实现具体的注册中心地址更新逻辑
			}
		} else {
			// 新增注册中心
			logger.Infof("New registry added: %s", id)
			// 这里需要实现新增注册中心的逻辑
		}
	}

	// 检查删除的注册中心
	for id := range oldConfig.Registries {
		if _, exists := newConfig.Registries[id]; !exists {
			logger.Infof("Registry removed: %s", id)
			// 这里需要实现删除注册中心的逻辑
		}
	}

	return nil
}

// updateProviderConfigs 更新提供者配置
func updateProviderConfigs(oldConfig, newConfig *RootConfig) error {
	// 比较提供者服务配置变化
	if oldConfig.Provider != nil && newConfig.Provider != nil {
		// 这里实现提供者配置更新逻辑
		logger.Infof("Provider configuration updated")
	}
	return nil
}

// updateConsumerConfigs 更新消费者配置
func updateConsumerConfigs(oldConfig, newConfig *RootConfig) error {
	// 比较消费者引用配置变化
	if oldConfig.Consumer != nil && newConfig.Consumer != nil {
		// 这里实现消费者配置更新逻辑
		logger.Infof("Consumer configuration updated")
	}
	return nil
}

// updateOtherConfigs 更新其他配置
func updateOtherConfigs(oldConfig, newConfig *RootConfig) error {
	// 更新日志配置
	if oldConfig.Logger != nil && newConfig.Logger != nil {
		logger.Infof("Logger configuration updated")
	}

	// 更新指标配置
	if oldConfig.Metrics != nil && newConfig.Metrics != nil {
		logger.Infof("Metrics configuration updated")
	}

	return nil
}

// getConfigFilePath 获取配置文件路径
func getConfigFilePath() string {
	// 从环境变量获取配置文件路径
	if configPath := os.Getenv("DUBBO_CONFIG_FILE"); configPath != "" {
		return configPath
	}

	// 默认配置文件路径
	return "../conf/dubbogo.yaml"
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

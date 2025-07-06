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
		currentRootConfig := GetAtomicRootConfig()
		if err := koan.UnmarshalWithConf(currentRootConfig.Prefix(),
			currentRootConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
			return err
		}
		SetAtomicRootConfig(currentRootConfig)
	} else {
		SetAtomicRootConfig(conf.rc)
	}

	currentRootConfig := GetAtomicRootConfig()
	if err := currentRootConfig.Init(); err != nil {
		return err
	}
	return nil
}

func check() error {
	if GetAtomicRootConfig() == nil {
		return errors.New("execute the config.Load() method first")
	}
	return nil
}

// GetRPCService get rpc service for consumer
func GetRPCService(name string) common.RPCService {
	currentRootConfig := GetAtomicRootConfig()
	return currentRootConfig.Consumer.References[name].GetRPCService()
}

// RPCService create rpc service for consumer
func RPCService(service common.RPCService) {
	ref := common.GetReference(service)
	currentRootConfig := GetAtomicRootConfig()
	currentRootConfig.Consumer.References[ref].Implement(service)
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
	currentRootConfig := GetAtomicRootConfig()
	return currentRootConfig.Metrics
}

func GetTracingConfig(tracingKey string) *TracingConfig {
	currentRootConfig := GetAtomicRootConfig()
	return currentRootConfig.Tracing[tracingKey]
}

func GetMetadataReportConfg() *MetadataReportConfig {
	currentRootConfig := GetAtomicRootConfig()
	return currentRootConfig.MetadataReport
}

func IsProvider() bool {
	currentRootConfig := GetAtomicRootConfig()
	return len(currentRootConfig.Provider.Services) > 0
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
	currentRootConfig := GetAtomicRootConfig()
	manager, err := NewHotReloadManager(configPath, currentRootConfig)
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
	currentRootConfig := GetAtomicRootConfig()
	manager, err := NewHotReloadManager(configPath, currentRootConfig)
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
	oldConfig := GetAtomicRootConfig()

	// 更新根配置
	SetAtomicRootConfig(newConfig)

	// 应用配置变更
	if err := applyConfigChanges(oldConfig, newConfig); err != nil {
		logger.Errorf("Failed to apply config changes: %v", err)
		// 回滚到旧配置
		SetAtomicRootConfig(oldConfig)
		return err
	}

	logger.Infof("Configuration changes applied successfully")
	return nil
}

// applyConfigChanges 应用配置变更
func applyConfigChanges(oldConfig, newConfig *RootConfig) error {
	logger.Infof("Applying configuration changes...")

	// 1. 应用注册中心配置变更
	if err := applyRegistryChanges(oldConfig, newConfig); err != nil {
		logger.Errorf("Failed to apply registry changes: %v", err)
		return err
	}

	// 2. 应用协议配置变更
	if err := applyProtocolChanges(oldConfig, newConfig); err != nil {
		logger.Errorf("Failed to apply protocol changes: %v", err)
		return err
	}

	// 3. 应用提供者配置变更
	if err := applyProviderChanges(oldConfig, newConfig); err != nil {
		logger.Errorf("Failed to apply provider changes: %v", err)
		return err
	}

	// 4. 应用消费者配置变更
	if err := applyConsumerChanges(oldConfig, newConfig); err != nil {
		logger.Errorf("Failed to apply consumer changes: %v", err)
		return err
	}

	// 5. 应用应用配置变更
	if err := applyApplicationChanges(oldConfig, newConfig); err != nil {
		logger.Errorf("Failed to apply application changes: %v", err)
		return err
	}

	logger.Infof("Configuration changes applied successfully")
	return nil
}

// applyRegistryChanges 应用注册中心配置变更
func applyRegistryChanges(oldConfig, newConfig *RootConfig) error {
	// 检查注册中心配置是否有变更
	for name, newRegistry := range newConfig.Registries {
		oldRegistry, exists := oldConfig.Registries[name]
		if !exists {
			// 新增注册中心
			logger.Infof("New registry added: %s", name)
			if err := newRegistry.Init(); err != nil {
				return err
			}
			continue
		}

		// 检查地址是否变更
		if oldRegistry.Address != newRegistry.Address {
			logger.Infof("Registry address changed for %s: %s -> %s", name, oldRegistry.Address, newRegistry.Address)
			// TODO: 实现注册中心重新连接逻辑
			// 这里需要调用注册中心的重连方法
		}
	}

	// 检查是否有删除的注册中心
	for name := range oldConfig.Registries {
		if _, exists := newConfig.Registries[name]; !exists {
			logger.Infof("Registry removed: %s", name)
			// TODO: 实现注册中心销毁逻辑
		}
	}

	return nil
}

// applyProtocolChanges 应用协议配置变更
func applyProtocolChanges(oldConfig, newConfig *RootConfig) error {
	// 检查协议配置是否有变更
	for name, newProtocol := range newConfig.Protocols {
		oldProtocol, exists := oldConfig.Protocols[name]
		if !exists {
			// 新增协议
			logger.Infof("New protocol added: %s", name)
			if err := newProtocol.Init(); err != nil {
				return err
			}
			continue
		}

		// 检查端口是否变更
		if oldProtocol.Port != newProtocol.Port {
			logger.Infof("Protocol port changed for %s: %d -> %d", name, oldProtocol.Port, newProtocol.Port)
			// TODO: 实现协议重启逻辑
			// 这里需要调用协议的重启方法
		}
	}

	return nil
}

// applyProviderChanges 应用提供者配置变更
func applyProviderChanges(oldConfig, newConfig *RootConfig) error {
	// 检查服务配置是否有变更
	for name, newService := range newConfig.Provider.Services {
		oldService, exists := oldConfig.Provider.Services[name]
		if !exists {
			// 新增服务
			logger.Infof("New provider service added: %s", name)
			continue
		}

		// 检查服务配置变更
		if oldService.Version != newService.Version || oldService.Group != newService.Group {
			logger.Infof("Provider service config changed for %s", name)
			// TODO: 实现服务重新暴露逻辑
		}
	}

	return nil
}

// applyConsumerChanges 应用消费者配置变更
func applyConsumerChanges(oldConfig, newConfig *RootConfig) error {
	// 检查引用配置是否有变更
	for name, newReference := range newConfig.Consumer.References {
		oldReference, exists := oldConfig.Consumer.References[name]
		if !exists {
			// 新增引用
			logger.Infof("New consumer reference added: %s", name)
			continue
		}

		// 检查引用配置变更
		if oldReference.Version != newReference.Version || oldReference.Group != newReference.Group {
			logger.Infof("Consumer reference config changed for %s", name)
			// TODO: 实现引用重新加载逻辑
		}
	}

	return nil
}

// applyApplicationChanges 应用应用配置变更
func applyApplicationChanges(oldConfig, newConfig *RootConfig) error {
	// 检查应用名称是否变更
	if oldConfig.Application.Name != newConfig.Application.Name {
		logger.Infof("Application name changed: %s -> %s", oldConfig.Application.Name, newConfig.Application.Name)
		// 应用名称变更通常需要重启应用，这里只记录日志
	}

	// 检查应用版本是否变更
	if oldConfig.Application.Version != newConfig.Application.Version {
		logger.Infof("Application version changed: %s -> %s", oldConfig.Application.Version, newConfig.Application.Version)
	}

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

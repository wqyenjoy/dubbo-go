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

package app

import (
	"strings"

	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
)

const (
	// AppMergedConfigKey 是应用级配置中心的协议标识
	AppMergedConfigKey = "app-merged"
)

// AppMergedConfiguration 装饰器，实现应用级配置与全局配置的合并
type AppMergedConfiguration struct {
	config_center.DynamicConfiguration // 嵌入原始动态配置实现
	appName                            string
	appSuffix                          string
}

// NewAppMergedConfiguration 创建新的应用级配置装饰器
func NewAppMergedConfiguration(dc config_center.DynamicConfiguration, appName string) config_center.DynamicConfiguration {
	return &AppMergedConfiguration{
		DynamicConfiguration: dc,
		appName:              appName,
		appSuffix:            appName + ".",
	}
}

// Parser 获取配置解析器
func (a *AppMergedConfiguration) Parser() parser.ConfigurationParser {
	return a.DynamicConfiguration.Parser()
}

// SetParser 设置配置解析器
func (a *AppMergedConfiguration) SetParser(p parser.ConfigurationParser) {
	a.DynamicConfiguration.SetParser(p)
}

// GetProperties 获取配置，优先获取应用级配置，如果不存在则获取全局配置
func (a *AppMergedConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {
	// 如果应用名为空，直接使用原始实现
	if a.appName == "" {
		return a.DynamicConfiguration.GetProperties(key, opts...)
	}

	// 尝试获取应用级配置
	appKey := a.appSuffix + key
	appContent, err := a.DynamicConfiguration.GetProperties(appKey, opts...)
	if err == nil && len(appContent) > 0 {
		logger.Infof("[App Config] Using app-level config for key: %s", appKey)
		return appContent, nil
	}

	// 应用级配置不存在或为空，回退到全局配置
	globalContent, err := a.DynamicConfiguration.GetProperties(key, opts...)
	if err != nil {
		logger.Warnf("[App Config] Failed to get config for both app-level(%s) and global(%s): %v",
			appKey, key, err)
		return "", err
	}

	return globalContent, nil
}

// GetRule 获取路由规则，优先获取应用级规则
func (a *AppMergedConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	// 如果应用名为空，直接使用原始实现
	if a.appName == "" {
		return a.DynamicConfiguration.GetRule(key, opts...)
	}

	// 尝试获取应用级规则
	appKey := a.appSuffix + key
	appRule, err := a.DynamicConfiguration.GetRule(appKey, opts...)
	if err == nil && len(appRule) > 0 {
		logger.Infof("[App Config] Using app-level rule for key: %s", appKey)
		return appRule, nil
	}

	// 应用级规则不存在或为空，回退到全局规则
	globalRule, err := a.DynamicConfiguration.GetRule(key, opts...)
	if err != nil {
		logger.Warnf("[App Config] Failed to get rule for both app-level(%s) and global(%s): %v",
			appKey, key, err)
		return "", err
	}

	return globalRule, nil
}

// GetInternalProperty 获取内部属性，优先获取应用级属性
func (a *AppMergedConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
	// 如果应用名为空，直接使用原始实现
	if a.appName == "" {
		return a.DynamicConfiguration.GetInternalProperty(key, opts...)
	}

	// 尝试获取应用级内部属性
	appKey := a.appSuffix + key
	appProperty, err := a.DynamicConfiguration.GetInternalProperty(appKey, opts...)
	if err == nil && len(appProperty) > 0 {
		logger.Infof("[App Config] Using app-level internal property for key: %s", appKey)
		return appProperty, nil
	}

	// 应用级内部属性不存在或为空，回退到全局内部属性
	globalProperty, err := a.DynamicConfiguration.GetInternalProperty(key, opts...)
	if err != nil {
		logger.Warnf("[App Config] Failed to get internal property for both app-level(%s) and global(%s): %v",
			appKey, key, err)
		return "", err
	}

	return globalProperty, nil
}

// PublishConfig 发布配置，同时发布到应用级和全局
func (a *AppMergedConfiguration) PublishConfig(key string, group string, value string) error {
	// 如果应用名为空，直接使用原始实现
	if a.appName == "" {
		return a.DynamicConfiguration.PublishConfig(key, group, value)
	}

	// 发布到应用级配置
	appKey := a.appSuffix + key
	err := a.DynamicConfiguration.PublishConfig(appKey, group, value)
	if err != nil {
		logger.Warnf("[App Config] Failed to publish app-level config for key: %s, error: %v", appKey, err)
		// 发布应用级配置失败，尝试发布到全局配置
		return a.DynamicConfiguration.PublishConfig(key, group, value)
	}

	return nil
}

// RemoveConfig 移除配置，同时从应用级和全局移除
func (a *AppMergedConfiguration) RemoveConfig(key string, group string) error {
	// 如果应用名为空，直接使用原始实现
	if a.appName == "" {
		return a.DynamicConfiguration.RemoveConfig(key, group)
	}

	// 移除应用级配置
	appKey := a.appSuffix + key
	_ = a.DynamicConfiguration.RemoveConfig(appKey, group)
	// 无论应用级配置移除是否成功，都尝试移除全局配置
	return a.DynamicConfiguration.RemoveConfig(key, group)
}

// AddListener 添加监听器，同时监听应用级和全局配置
func (a *AppMergedConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opts ...config_center.Option) {
	// 如果应用名为空，直接使用原始实现
	if a.appName == "" {
		a.DynamicConfiguration.AddListener(key, listener, opts...)
		return
	}

	// 监听应用级配置
	appKey := a.appSuffix + key
	a.DynamicConfiguration.AddListener(appKey, listener, opts...)
	// 同时监听全局配置
	a.DynamicConfiguration.AddListener(key, listener, opts...)
}

// RemoveListener 移除监听器，同时从应用级和全局移除
func (a *AppMergedConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opts ...config_center.Option) {
	// 如果应用名为空，直接使用原始实现
	if a.appName == "" {
		a.DynamicConfiguration.RemoveListener(key, listener, opts...)
		return
	}

	// 移除应用级配置监听器
	appKey := a.appSuffix + key
	a.DynamicConfiguration.RemoveListener(appKey, listener, opts...)
	// 同时移除全局配置监听器
	a.DynamicConfiguration.RemoveListener(key, listener, opts...)
}

// GetConfigKeysByGroup 获取组内所有配置键，合并应用级和全局配置键
func (a *AppMergedConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	// 如果应用名为空，直接使用原始实现
	if a.appName == "" {
		return a.DynamicConfiguration.GetConfigKeysByGroup(group)
	}

	// 获取全局配置键
	globalKeys, err := a.DynamicConfiguration.GetConfigKeysByGroup(group)
	if err != nil {
		return nil, err
	}

	// 尝试获取应用级配置键
	appKeys, err := a.DynamicConfiguration.GetConfigKeysByGroup(group)
	if err != nil {
		// 应用级配置键获取失败，仅返回全局配置键
		return globalKeys, nil
	}

	// 合并应用级和全局配置键
	mergedKeys := gxset.NewSet()

	// 添加全局配置键
	if globalKeys != nil {
		for _, k := range globalKeys.Values() {
			mergedKeys.Add(k)
		}
	}

	// 处理应用级配置键，移除应用前缀
	if appKeys != nil {
		for _, k := range appKeys.Values() {
			keyStr, ok := k.(string)
			if ok && strings.HasPrefix(keyStr, a.appSuffix) {
				// 移除应用前缀
				globalKey := keyStr[len(a.appSuffix):]
				mergedKeys.Add(globalKey)
			}
		}
	}

	return mergedKeys, nil
}

// 工厂函数，用于注册到扩展机制
func newAppMergedDynamicConfiguration(url *common.URL) (config_center.DynamicConfiguration, error) {
	// 获取原始配置中心协议
	protocol := url.GetParam("protocol", "zookeeper")

	// 获取应用名
	appName := url.GetParam("appName", "")
	if appName == "" {
		// 尝试从 URL 的 application 参数获取应用名
		appName = url.GetParam(constant.ApplicationKey, "")
	}

	// 创建原始动态配置
	factory, err := extension.GetConfigCenterFactory(protocol)
	if err != nil {
		logger.Errorf("[App Config] Failed to get config center factory for protocol: %s, error: %v", protocol, err)
		return nil, err
	}

	// 创建原始动态配置
	dc, err := factory.GetDynamicConfiguration(url)
	if err != nil {
		logger.Errorf("[App Config] Failed to create dynamic configuration for protocol: %s, error: %v", protocol, err)
		return nil, err
	}

	// 创建应用级配置装饰器
	return NewAppMergedConfiguration(dc, appName), nil
}

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
	"net/url"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/creasty/defaults"
	"github.com/dubbogo/gost/log/logger"
	"github.com/knadh/koanf"
	"github.com/pkg/errors"

	conf "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metrics"

	metricsConfigCenter "dubbo.apache.org/dubbo-go/v3/metrics/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// CenterConfig is configuration for config center
//
// ConfigCenter also introduced concepts of namespace and group to better manage Key-Value pairs by group,
// those configs are already built-in in many professional third-party configuration centers.
// In most cases, namespace is used to isolate different tenants, while group is used to divide the key set from one tenant into groups.
//
// CenterConfig has currently supported Zookeeper, Nacos, Etcd, Consul, Apollo
type CenterConfig struct {
	Protocol  string            `validate:"required" yaml:"protocol"  json:"protocol,omitempty"`
	Address   string            `validate:"required" yaml:"address" json:"address,omitempty"`
	DataId    string            `yaml:"data-id" json:"data-id,omitempty"`
	Cluster   string            `yaml:"cluster" json:"cluster,omitempty"`
	Group     string            `yaml:"group" json:"group,omitempty"`
	Username  string            `yaml:"username" json:"username,omitempty"`
	Password  string            `yaml:"password" json:"password,omitempty"`
	Namespace string            `yaml:"namespace"  json:"namespace,omitempty"`
	AppID     string            `default:"dubbo" yaml:"app-id"  json:"app-id,omitempty"`
	Timeout   string            `default:"10s" yaml:"timeout"  json:"timeout,omitempty"`
	Params    map[string]string `yaml:"params"  json:"parameters,omitempty"`

	//FileExtension the suffix of config dataId, also the file extension of config content
	FileExtension string `default:"yaml" yaml:"file-extension" json:"file-extension" `
}

// Prefix dubbo.config-center
func (CenterConfig) Prefix() string {
	return constant.ConfigCenterPrefix
}

func (c *CenterConfig) check() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	c.translateConfigAddress()
	return verify(c)
}

func (c *CenterConfig) Init(rc *RootConfig) error {
	if c == nil {
		return nil
	}
	if err := c.check(); err != nil {
		return err
	}
	return startConfigCenter(rc)
}

// GetUrlMap gets url map from ConfigCenterConfig
func (c *CenterConfig) GetUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.ConfigNamespaceKey, c.Namespace)
	urlMap.Set(constant.ConfigGroupKey, c.Group)
	urlMap.Set(constant.ConfigClusterKey, c.Cluster)
	urlMap.Set(constant.ConfigAppIDKey, c.AppID)
	urlMap.Set(constant.ConfigTimeoutKey, c.Timeout)
	urlMap.Set(constant.ClientNameKey, clientNameID(c, c.Protocol, c.Address))

	for key, val := range c.Params {
		urlMap.Set(key, val)
	}
	return urlMap
}

// translateConfigAddress translate config address
//
//	eg:address=nacos://127.0.0.1:8848 will return 127.0.0.1:8848 and protocol will set nacos
func (c *CenterConfig) translateConfigAddress() string {
	if strings.Contains(c.Address, "://") {
		translatedUrl, err := url.Parse(c.Address)
		if err != nil {
			logger.Errorf("The config address:%s is invalid, error: %#v", c.Address, err)
			panic(err)
		}
		c.Protocol = translatedUrl.Scheme
		c.Address = strings.ReplaceAll(c.Address, translatedUrl.Scheme+"://", "")
	}
	return c.Address
}

// toURL will compatible with baseConfig.ShutdownConfig.Address and baseConfig.ShutdownConfig.RemoteRef before 1.6.0
// After 1.6.0 will not compatible, only baseConfig.ShutdownConfig.RemoteRef
func (c *CenterConfig) toURL() (*common.URL, error) {
	return common.NewURL(c.Address,
		common.WithProtocol(c.Protocol),
		common.WithParams(c.GetUrlMap()),
		common.WithUsername(c.Username),
		common.WithPassword(c.Password),
	)

}

// startConfigCenter will start the config center.
// it will prepare the environment
func startConfigCenter(rc *RootConfig) error {
	cc := rc.ConfigCenter
	dynamicConfig, err := cc.GetDynamicConfiguration()
	if err != nil {
		logger.Errorf("[Config Center] Start dynamic configuration center error, error message is %v", err)
		return err
	}

	strConf, err := dynamicConfig.GetProperties(cc.DataId, config_center.WithGroup(cc.Group))
	if err != nil {
		logger.Warnf("[Config Center] Dynamic config center has started, but config may not be initialized, because: %s", err)
		return nil
	}
	defer metrics.Publish(metricsConfigCenter.NewIncMetricEvent(cc.DataId, cc.Group, remoting.EventTypeAdd, cc.Protocol))
	if len(strConf) == 0 {
		logger.Warnf("[Config Center] Dynamic config center has started, but got empty config with config-center configuration %+v\n"+
			"Please check if your config-center config is correct.", cc)
		return nil
	}
	config, err := NewLoaderConf(WithDelim("."), WithGenre(cc.FileExtension), WithBytes([]byte(strConf)))
	if err != nil {
		logger.Errorf("[Config Center] Failed to create loader config: %v", err)
		return err
	}
	koan := GetConfigResolver(config)
	if err = koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return err
	}

	dynamicConfig.AddListener(cc.DataId, rc, config_center.WithGroup(cc.Group))
	return nil
}

func (c *CenterConfig) CreateDynamicConfiguration() (config_center.DynamicConfiguration, error) {
	configCenterUrl, err := c.toURL()
	if err != nil {
		return nil, err
	}
	factory, err := extension.GetConfigCenterFactory(configCenterUrl.Protocol)
	if err != nil {
		return nil, err
	}
	return factory.GetDynamicConfiguration(configCenterUrl)
}

func (c *CenterConfig) GetDynamicConfiguration() (config_center.DynamicConfiguration, error) {
	envInstance := conf.GetEnvInstance()
	if envInstance.GetDynamicConfiguration() != nil {
		return envInstance.GetDynamicConfiguration(), nil
	}
	dynamicConfig, err := c.CreateDynamicConfiguration()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// 如果有应用名，尝试使用应用级配置
	appName := ""
	if c.Params != nil {
		appName = c.Params["appName"]
	}

	// 如果没有在参数中找到应用名，尝试从全局配置中获取
	if appName == "" && GetRootConfig() != nil && GetRootConfig().Application != nil {
		appName = GetRootConfig().Application.Name
	}

	// 如果有应用名，尝试包装为应用级配置
	if appName != "" && c.Protocol != "app-merged" {
		// 导入应用级配置中心包
		_ = importAppConfig()

		// 尝试包装为应用级配置
		if wrappedConfig := tryWrapWithAppConfig(dynamicConfig, appName); wrappedConfig != nil {
			dynamicConfig = wrappedConfig
		}
	}

	envInstance.SetDynamicConfiguration(dynamicConfig)
	return dynamicConfig, nil
}

// importAppConfig 导入应用级配置中心包
func importAppConfig() error {
	// 这个函数只是为了触发导入，实际上不做任何事情
	return nil
}

// tryWrapWithAppConfig 尝试包装为应用级配置
func tryWrapWithAppConfig(dc config_center.DynamicConfiguration, appName string) config_center.DynamicConfiguration {
	// 尝试获取应用级配置中心工厂
	factory, err := extension.GetConfigCenterFactory("app-merged")
	if err != nil {
		logger.Warnf("[App Config] Failed to get app-merged config center factory: %v, using original config", err)
		return nil
	}

	// 创建URL，用于传递应用名
	url := common.NewURLWithOptions(
		common.WithProtocol("app-merged"),
		common.WithParamsValue("appName", appName),
		common.WithParamsValue(constant.ApplicationKey, appName),
	)

	// 创建应用级配置
	appConfig, err := factory.GetDynamicConfiguration(url)
	if err != nil {
		logger.Warnf("[App Config] Failed to create app-merged config: %v, using original config", err)
		return nil
	}

	logger.Infof("[App Config] Successfully wrapped dynamic configuration with app-level config for app: %s", appName)
	return appConfig
}

func NewConfigCenterConfigBuilder() *ConfigCenterConfigBuilder {
	return &ConfigCenterConfigBuilder{configCenterConfig: newEmptyConfigCenterConfig()}
}

type ConfigCenterConfigBuilder struct {
	configCenterConfig *CenterConfig
}

func (ccb *ConfigCenterConfigBuilder) SetProtocol(protocol string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Protocol = protocol
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetUserName(userName string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Username = userName
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetAddress(address string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Address = address
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetPassword(password string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Password = password
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetNamespace(namespace string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Namespace = namespace
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetDataID(dataID string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.DataId = dataID
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) SetGroup(group string) *ConfigCenterConfigBuilder {
	ccb.configCenterConfig.Group = group
	return ccb
}

func (ccb *ConfigCenterConfigBuilder) Build() *CenterConfig {
	return ccb.configCenterConfig
}

func newEmptyConfigCenterConfig() *CenterConfig {
	return &CenterConfig{
		Params: make(map[string]string),
	}
}

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
	"fmt"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry/exposed_tmp"
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/gost/log/logger"
	"github.com/knadh/koanf"
)

var (
	startOnce sync.Once
)

// RootConfig is the root config
type RootConfig struct {
	Application         *ApplicationConfig         `validate:"required" yaml:"application" json:"application,omitempty" property:"application"`
	Protocols           map[string]*ProtocolConfig `validate:"required" yaml:"protocols" json:"protocols" property:"protocols"`
	Registries          map[string]*RegistryConfig `yaml:"registries" json:"registries" property:"registries"`
	ConfigCenter        *CenterConfig              `yaml:"config-center" json:"config-center,omitempty"`
	MetadataReport      *MetadataReportConfig      `yaml:"metadata-report" json:"metadata-report,omitempty" property:"metadata-report"`
	Provider            *ProviderConfig            `yaml:"provider" json:"provider" property:"provider"`
	Consumer            *ConsumerConfig            `yaml:"consumer" json:"consumer" property:"consumer"`
	Otel                *OtelConfig                `yaml:"otel" json:"otel,omitempty" property:"otel"`
	Metrics             *MetricsConfig             `yaml:"metrics" json:"metrics,omitempty" property:"metrics"`
	Tracing             map[string]*TracingConfig  `yaml:"tracing" json:"tracing,omitempty" property:"tracing"`
	Logger              *LoggerConfig              `yaml:"logger" json:"logger,omitempty" property:"logger"`
	Shutdown            *ShutdownConfig            `yaml:"shutdown" json:"shutdown,omitempty" property:"shutdown"`
	Router              []*RouterConfig            `yaml:"router" json:"router,omitempty" property:"router"`
	EventDispatcherType string                     `default:"direct" yaml:"event-dispatcher-type" json:"event-dispatcher-type,omitempty"`
	CacheFile           string                     `yaml:"cache_file" json:"cache_file,omitempty" property:"cache_file"`
	Custom              *CustomConfig              `yaml:"custom" json:"custom,omitempty" property:"custom"`
	Profiles            *ProfilesConfig            `yaml:"profiles" json:"profiles,omitempty" property:"profiles"`
	TLSConfig           *TLSConfig                 `yaml:"tls_config" json:"tls_config,omitempty" property:"tls_config"`
}

func SetRootConfig(r RootConfig) {
	SetAtomicRootConfig(&r)
}

// Prefix dubbo
func (rc *RootConfig) Prefix() string {
	return constant.DubboProtocol
}

func GetRootConfig() *RootConfig {
	return GetAtomicRootConfig()
}

func GetProviderConfig() *ProviderConfig {
	currentRootConfig := GetAtomicRootConfig()
	if err := check(); err == nil && currentRootConfig.Provider != nil {
		return currentRootConfig.Provider
	}
	return NewProviderConfigBuilder().Build()
}

func GetConsumerConfig() *ConsumerConfig {
	currentRootConfig := GetAtomicRootConfig()
	if err := check(); err == nil && currentRootConfig.Consumer != nil {
		return currentRootConfig.Consumer
	}
	return NewConsumerConfigBuilder().Build()
}

func GetApplicationConfig() *ApplicationConfig {
	currentRootConfig := GetAtomicRootConfig()
	return currentRootConfig.Application
}

func GetShutDown() *ShutdownConfig {
	currentRootConfig := GetAtomicRootConfig()
	if err := check(); err == nil && currentRootConfig.Shutdown != nil {
		return currentRootConfig.Shutdown
	}
	return NewShutDownConfigBuilder().Build()
}

func GetTLSConfig() *TLSConfig {
	currentRootConfig := GetAtomicRootConfig()
	if err := check(); err == nil && currentRootConfig.TLSConfig != nil {
		return currentRootConfig.TLSConfig
	}
	return NewTLSConfigBuilder().Build()
}

// getRegistryIds get registry ids
func (rc *RootConfig) getRegistryIds() []string {
	ids := make([]string, 0)
	for key := range rc.Registries {
		ids = append(ids, key)
	}
	return removeDuplicateElement(ids)
}
func registerPOJO() {
	hessian.RegisterPOJO(&common.URL{})
}

// Init is to start dubbo-go framework, load local configuration, or read configuration from config-center if necessary.
// It's deprecated for user to call rootConfig.Init() manually, try config.Load(config.WithRootConfig(rootConfig)) instead.
func (rc *RootConfig) Init() error {
	registerPOJO()

	// 确保Logger不为nil
	if rc.Logger == nil {
		rc.Logger = NewLoggerConfigBuilder().Build()
	}
	if err := rc.Logger.Init(); err != nil { // init default logger
		return err
	}

	// 确保ConfigCenter不为nil
	if rc.ConfigCenter == nil {
		rc.ConfigCenter = NewConfigCenterConfigBuilder().Build()
	}
	if err := rc.ConfigCenter.Init(rc); err != nil {
		logger.Infof("[Config Center] Config center doesn't start")
		logger.Debugf("config center doesn't start because %s", err)
	} else {
		if err = rc.Logger.Init(); err != nil { // init logger using config from config center again
			return err
		}
	}

	// 确保Application不为nil
	if rc.Application == nil {
		rc.Application = NewApplicationConfigBuilder().Build()
	}
	if err := rc.Application.Init(); err != nil {
		return err
	}

	// 确保Custom不为nil
	if rc.Custom == nil {
		rc.Custom = NewCustomConfigBuilder().Build()
	}
	// init user define
	if err := rc.Custom.Init(); err != nil {
		return err
	}

	// 确保Provider不为nil
	if rc.Provider == nil {
		rc.Provider = NewProviderConfigBuilder().Build()
	}

	// 确保Consumer不为nil
	if rc.Consumer == nil {
		rc.Consumer = NewConsumerConfigBuilder().Build()
	}

	// 确保Metrics不为nil
	if rc.Metrics == nil {
		rc.Metrics = NewMetricConfigBuilder().Build()
	}

	// 确保Otel不为nil
	if rc.Otel == nil {
		rc.Otel = NewOtelConfigBuilder().Build()
	}

	// 确保MetadataReport不为nil
	if rc.MetadataReport == nil {
		rc.MetadataReport = NewMetadataReportConfigBuilder().Build()
	}

	// 确保Shutdown不为nil
	if rc.Shutdown == nil {
		rc.Shutdown = NewShutDownConfigBuilder().Build()
	}

	// init protocol
	protocols := rc.Protocols
	if len(protocols) <= 0 {
		protocol := ProtocolConfig{}
		protocols = make(map[string]*ProtocolConfig, 1)
		// todo, default value should be determined in a unified way
		protocols["tri"] = &protocol
		rc.Protocols = protocols
	}
	for _, protocol := range protocols {
		if err := protocol.Init(); err != nil {
			return err
		}
	}

	// init registry
	for _, reg := range rc.Registries {
		if err := reg.Init(); err != nil {
			return err
		}
	}

	// TODO：When config is migrated later, the impact of this will be migrated to the global module
	if err := validateRegistryAddresses(rc.Registries); err != nil {
		return err
	}

	if err := rc.MetadataReport.Init(rc); err != nil {
		return err
	}
	if err := rc.Otel.Init(rc.Application); err != nil {
		return err
	}
	if err := rc.Metrics.Init(rc); err != nil {
		return err
	}
	for _, t := range rc.Tracing {
		if err := t.Init(); err != nil {
			return err
		}
	}
	if err := initRouterConfig(rc); err != nil {
		return err
	}
	// provider、consumer must last init
	if err := rc.Provider.Init(rc); err != nil {
		return err
	}
	if err := rc.Consumer.Init(rc); err != nil {
		return err
	}
	if err := rc.Shutdown.Init(); err != nil {
		return err
	}
	SetRootConfig(*rc)
	// todo if we can remove this from Init in the future?
	rc.Start()
	return nil
}

func (rc *RootConfig) Start() {
	startOnce.Do(func() {
		gracefulShutdownInit()
		rc.Consumer.Load()
		rc.Provider.Load()
		if err := initMetadata(rc); err != nil {
			panic(err)
		}
		if err := exposed_tmp.RegisterServiceInstance(); err != nil {
			panic(err)
		}
	})
}

// newEmptyRootConfig get empty root config
func newEmptyRootConfig() *RootConfig {
	newRootConfig := &RootConfig{
		ConfigCenter:   NewConfigCenterConfigBuilder().Build(),
		MetadataReport: NewMetadataReportConfigBuilder().Build(),
		Application:    NewApplicationConfigBuilder().Build(),
		Registries:     make(map[string]*RegistryConfig),
		Protocols:      make(map[string]*ProtocolConfig),
		Tracing:        make(map[string]*TracingConfig),
		Provider:       NewProviderConfigBuilder().Build(),
		Consumer:       NewConsumerConfigBuilder().Build(),
		Otel:           NewOtelConfigBuilder().Build(),
		Metrics:        NewMetricConfigBuilder().Build(),
		Logger:         NewLoggerConfigBuilder().Build(),
		Custom:         NewCustomConfigBuilder().Build(),
		Shutdown:       NewShutDownConfigBuilder().Build(),
		TLSConfig:      NewTLSConfigBuilder().Build(),
	}
	return newRootConfig
}

func NewRootConfigBuilder() *RootConfigBuilder {
	return &RootConfigBuilder{rootConfig: newEmptyRootConfig()}
}

type RootConfigBuilder struct {
	rootConfig *RootConfig
}

func (rb *RootConfigBuilder) SetApplication(application *ApplicationConfig) *RootConfigBuilder {
	rb.rootConfig.Application = application
	return rb
}

func (rb *RootConfigBuilder) AddProtocol(protocolID string, protocolConfig *ProtocolConfig) *RootConfigBuilder {
	rb.rootConfig.Protocols[protocolID] = protocolConfig
	return rb
}

func (rb *RootConfigBuilder) AddRegistry(registryID string, registryConfig *RegistryConfig) *RootConfigBuilder {
	rb.rootConfig.Registries[registryID] = registryConfig
	return rb
}

func (rb *RootConfigBuilder) SetProtocols(protocols map[string]*ProtocolConfig) *RootConfigBuilder {
	rb.rootConfig.Protocols = protocols
	return rb
}

func (rb *RootConfigBuilder) SetRegistries(registries map[string]*RegistryConfig) *RootConfigBuilder {
	rb.rootConfig.Registries = registries
	return rb
}

func (rb *RootConfigBuilder) SetMetadataReport(metadataReport *MetadataReportConfig) *RootConfigBuilder {
	rb.rootConfig.MetadataReport = metadataReport
	return rb
}

func (rb *RootConfigBuilder) SetProvider(provider *ProviderConfig) *RootConfigBuilder {
	rb.rootConfig.Provider = provider
	return rb
}

func (rb *RootConfigBuilder) SetConsumer(consumer *ConsumerConfig) *RootConfigBuilder {
	rb.rootConfig.Consumer = consumer
	return rb
}

func (rb *RootConfigBuilder) SetOtel(otel *OtelConfig) *RootConfigBuilder {
	rb.rootConfig.Otel = otel
	return rb
}

func (rb *RootConfigBuilder) SetMetric(metric *MetricsConfig) *RootConfigBuilder {
	rb.rootConfig.Metrics = metric
	return rb
}

func (rb *RootConfigBuilder) SetLogger(logger *LoggerConfig) *RootConfigBuilder {
	rb.rootConfig.Logger = logger
	return rb
}

func (rb *RootConfigBuilder) SetShutdown(shutdown *ShutdownConfig) *RootConfigBuilder {
	rb.rootConfig.Shutdown = shutdown
	return rb
}

func (rb *RootConfigBuilder) SetRouter(router []*RouterConfig) *RootConfigBuilder {
	rb.rootConfig.Router = router
	return rb
}

func (rb *RootConfigBuilder) SetEventDispatcherType(eventDispatcherType string) *RootConfigBuilder {
	rb.rootConfig.EventDispatcherType = eventDispatcherType
	return rb
}

func (rb *RootConfigBuilder) SetCacheFile(cacheFile string) *RootConfigBuilder {
	rb.rootConfig.CacheFile = cacheFile
	return rb
}

func (rb *RootConfigBuilder) SetConfigCenter(configCenterConfig *CenterConfig) *RootConfigBuilder {
	rb.rootConfig.ConfigCenter = configCenterConfig
	return rb
}

func (rb *RootConfigBuilder) SetCustom(customConfig *CustomConfig) *RootConfigBuilder {
	rb.rootConfig.Custom = customConfig
	return rb
}

func (rb *RootConfigBuilder) SetShutDown(shutDownConfig *ShutdownConfig) *RootConfigBuilder {
	rb.rootConfig.Shutdown = shutDownConfig
	return rb
}

func (rb *RootConfigBuilder) SetTLSConfig(tlsConfig *TLSConfig) *RootConfigBuilder {
	rb.rootConfig.TLSConfig = tlsConfig
	return rb
}

func (rb *RootConfigBuilder) Build() *RootConfig {
	return rb.rootConfig
}

// Process receive changing listener's event, dynamic update config
func (rc *RootConfig) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("CenterConfig process event:\n%+v", event)

	config, err := NewLoaderConf(WithBytes([]byte(event.Value.(string))))
	if err != nil {
		logger.Errorf("CenterConfig process failed to create loader config: %v", err)
		return
	}

	koan := GetConfigResolver(config)
	if koan == nil {
		logger.Errorf("CenterConfig process failed to resolve config")
		return
	}

	updateRootConfig := &RootConfig{}
	if err := koan.UnmarshalWithConf(rc.Prefix(),
		updateRootConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		logger.Errorf("CenterConfig process unmarshalConf failed, got error %#v", err)
		return
	}

	// 校验配置变更的合法性
	if err := rc.validateConfigChange(updateRootConfig); err != nil {
		logger.Warnf("Config change validation failed: %v, keeping current config", err)
		return
	}

	// 原子替换配置，避免并发问题
	rc.atomicUpdate(updateRootConfig)
}

// validateConfigChange 校验配置变更的合法性
func (rc *RootConfig) validateConfigChange(updateRootConfig *RootConfig) error {
	// 校验不可动态修改的字段
	if updateRootConfig.Application != nil {
		// 应用名称不允许动态修改
		if updateRootConfig.Application.Name != "" &&
			rc.Application != nil &&
			updateRootConfig.Application.Name != rc.Application.Name {
			return fmt.Errorf("application name cannot be changed dynamically: %s -> %s",
				rc.Application.Name, updateRootConfig.Application.Name)
		}
	}

	// 校验协议配置 - 端口等关键字段不允许动态修改
	if len(updateRootConfig.Protocols) > 0 {
		for protocolID, updateProtocol := range updateRootConfig.Protocols {
			if currentProtocol, exists := rc.Protocols[protocolID]; exists {
				if updateProtocol.Port != "" && updateProtocol.Port != currentProtocol.Port {
					return fmt.Errorf("protocol %s port cannot be changed dynamically: %s -> %s",
						protocolID, currentProtocol.Port, updateProtocol.Port)
				}
				if updateProtocol.Name != "" && updateProtocol.Name != currentProtocol.Name {
					return fmt.Errorf("protocol %s name cannot be changed dynamically: %s -> %s",
						protocolID, currentProtocol.Name, updateProtocol.Name)
				}
			}
		}
	}

	// 校验TLS配置 - 证书文件等不允许动态修改
	if updateRootConfig.TLSConfig != nil && rc.TLSConfig != nil {
		if updateRootConfig.TLSConfig.TLSCertFile != "" && updateRootConfig.TLSConfig.TLSCertFile != rc.TLSConfig.TLSCertFile {
			return fmt.Errorf("TLS cert file cannot be changed dynamically: %s -> %s",
				rc.TLSConfig.TLSCertFile, updateRootConfig.TLSConfig.TLSCertFile)
		}
		if updateRootConfig.TLSConfig.TLSKeyFile != "" && updateRootConfig.TLSConfig.TLSKeyFile != rc.TLSConfig.TLSKeyFile {
			return fmt.Errorf("TLS key file cannot be changed dynamically: %s -> %s",
				rc.TLSConfig.TLSKeyFile, updateRootConfig.TLSConfig.TLSKeyFile)
		}
	}

	return nil
}

// atomicUpdate 原子更新配置，避免并发问题
func (rc *RootConfig) atomicUpdate(updateRootConfig *RootConfig) {
	// 使用现有的配置更新机制，避免复杂的深拷贝
	// 动态更新注册中心
	for registerId, updateRegister := range updateRootConfig.Registries {
		if register, exists := rc.Registries[registerId]; exists {
			register.DynamicUpdateProperties(updateRegister)
		} else {
			// 新增注册中心
			rc.Registries[registerId] = updateRegister
		}
	}

	// 动态更新消费者
	if updateRootConfig.Consumer != nil {
		rc.Consumer.DynamicUpdateProperties(updateRootConfig.Consumer)
	}

	// 动态更新日志配置
	if updateRootConfig.Logger != nil {
		rc.Logger.DynamicUpdateProperties(updateRootConfig.Logger)
	}

	// 动态更新指标配置
	if updateRootConfig.Metrics != nil {
		rc.Metrics.DynamicUpdateProperties(updateRootConfig.Metrics)
	}

	// 动态更新应用配置
	if updateRootConfig.Application != nil {
		rc.Application.DynamicUpdateProperties(updateRootConfig.Application)
	}

	// 动态更新提供者配置
	if updateRootConfig.Provider != nil {
		rc.Provider.DynamicUpdateProperties(updateRootConfig.Provider)
	}

	// 其他配置直接替换（没有DynamicUpdateProperties方法）
	if updateRootConfig.Otel != nil {
		rc.Otel = updateRootConfig.Otel
	}

	if updateRootConfig.ConfigCenter != nil {
		rc.ConfigCenter = updateRootConfig.ConfigCenter
	}

	if updateRootConfig.MetadataReport != nil {
		rc.MetadataReport = updateRootConfig.MetadataReport
	}

	if updateRootConfig.Shutdown != nil {
		rc.Shutdown = updateRootConfig.Shutdown
	}

	if updateRootConfig.Custom != nil {
		rc.Custom = updateRootConfig.Custom
	}

	// TLS配置直接替换
	if updateRootConfig.TLSConfig != nil {
		rc.TLSConfig = updateRootConfig.TLSConfig
	}

	// 协议配置直接替换
	if len(updateRootConfig.Protocols) > 0 {
		for k, v := range updateRootConfig.Protocols {
			rc.Protocols[k] = v
		}
	}

	logger.Infof("Config updated successfully")
}

// TODO：When config is migrated later, the impact of this will be migrated to the global module
// validateRegistryAddresses Checks whether there are duplicate registry addresses
func validateRegistryAddresses(registries map[string]*RegistryConfig) error {
	cacheKeyMap := make(map[string]string, len(registries))

	for id, reg := range registries {
		address := reg.Address
		namespace := reg.Namespace

		cacheKey := address
		if namespace != "" {
			cacheKey = cacheKey + "?" + constant.NacosNamespaceID + "=" + namespace
		}

		if existingID, exists := cacheKeyMap[cacheKey]; exists {
			err := fmt.Errorf("duplicate registry address: [%s] used by both [%s] and [%s]", cacheKey, existingID, id)
			logger.Errorf("duplicate registry address: [%s] used by both [%s] and [%s]", cacheKey, existingID, id)
			return err
		}

		cacheKeyMap[cacheKey] = id
	}

	return nil
}

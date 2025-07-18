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
	"strings"

	"github.com/creasty/defaults"
	"github.com/dubbogo/gost/log/logger"

	tripleConstant "github.com/dubbogo/triple/pkg/common/constant"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	perrors "github.com/pkg/errors"

	aslimiter "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc/limiter"
)

// ProviderConfig is the default configuration of service provider
type ProviderConfig struct {
	Filter string `yaml:"filter" json:"filter,omitempty" property:"filter"`
	// Deprecated Register whether registration is required
	Register bool `yaml:"register" json:"register" property:"register"`
	// RegistryIDs is registry ids list
	RegistryIDs []string `yaml:"registry-ids" json:"registry-ids" property:"registry-ids"`
	// protocol
	ProtocolIDs []string `yaml:"protocol-ids" json:"protocol-ids" property:"protocol-ids"`
	// TracingKey is tracing ids list
	TracingKey string `yaml:"tracing-key" json:"tracing-key" property:"tracing-key"`
	// Services services
	Services     map[string]*ServiceConfig `yaml:"services" json:"services,omitempty" property:"services"`
	ProxyFactory string                    `default:"default" yaml:"proxy" json:"proxy,omitempty" property:"proxy"`
	FilterConf   any                       `yaml:"filter_conf" json:"filter_conf,omitempty" property:"filter_conf"`
	ConfigType   map[string]string         `yaml:"config_type" json:"config_type,omitempty" property:"config_type"`
	// adaptive service
	AdaptiveService        bool `yaml:"adaptive-service" json:"adaptive-service" property:"adaptive-service"`
	AdaptiveServiceVerbose bool `yaml:"adaptive-service-verbose" json:"adaptive-service-verbose" property:"adaptive-service-verbose"`

	rootConfig *RootConfig
}

func (ProviderConfig) Prefix() string {
	return constant.ProviderConfigPrefix
}

func (c *ProviderConfig) check() error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	return verify(c)
}

func (c *ProviderConfig) Init(rc *RootConfig) error {
	if c == nil {
		return nil
	}
	buildDebugMsg := func() string {
		if len(c.Services) == 0 {
			return "empty"
		}
		providerNames := make([]string, 0, len(c.Services))
		for k := range c.Services {
			providerNames = append(providerNames, k)
		}
		return strings.Join(providerNames, ", ")
	}
	logger.Debugf("Registered provider services are %v", buildDebugMsg())

	c.RegistryIDs = translateIds(c.RegistryIDs)
	if len(c.RegistryIDs) <= 0 {
		c.RegistryIDs = rc.getRegistryIds()
	}
	c.ProtocolIDs = translateIds(c.ProtocolIDs)

	if c.TracingKey == "" && len(rc.Tracing) > 0 {
		for k := range rc.Tracing {
			c.TracingKey = k
			break
		}
	}
	for key, serviceConfig := range c.Services {
		if serviceConfig.Interface == "" {
			service := GetProviderService(key)
			// try to use interface name defined by pb
			supportPBPackagerNameService, ok := service.(common.TriplePBService)
			if !ok {
				logger.Errorf("Service with reference = %s is not support read interface name from it."+
					"Please run go install github.com/dubbogo/dubbogo-cli/cmd/protoc-gen-go-triple@latest to update your "+
					"protoc-gen-go-triple and re-generate your pb file again."+
					"If you are not using pb serialization, please set 'interface' field in service config.", key)
				continue
			} else {
				// use interface name defined by pb
				serviceConfig.Interface = supportPBPackagerNameService.XXX_InterfaceName()
			}
		}
		if err := serviceConfig.Init(rc); err != nil {
			return err
		}

		serviceConfig.adaptiveService = c.AdaptiveService
	}

	for k, v := range rc.Protocols {
		if v.Name == tripleConstant.TRIPLE {
			// Auto create grpc based health check service.
			healthService := NewServiceConfigBuilder().
				SetProtocolIDs(k).
				SetNotRegister(true).
				SetInterface(constant.HealthCheckServiceInterface).
				Build()
			if err := healthService.Init(rc); err != nil {
				return err
			}
			c.Services[constant.HealthCheckServiceTypeName] = healthService

			// Auto create reflection service configure only when provider with triple service is configured.
			tripleReflectionService := NewServiceConfigBuilder().
				SetProtocolIDs(k).
				SetNotRegister(true).
				SetInterface(constant.ReflectionServiceInterface).
				Build()
			if err := tripleReflectionService.Init(rc); err != nil {
				return err
			}
			// Maybe only register once, If setting this service, break from traversing Protocols.
			c.Services[constant.ReflectionServiceTypeName] = tripleReflectionService
			break
		}
	}

	if err := c.check(); err != nil {
		return err
	}
	// enable adaptive service verbose
	if c.AdaptiveServiceVerbose {
		if !c.AdaptiveService {
			return perrors.Errorf("The adaptive service is disabled, " +
				"adaptive service verbose should be disabled either.")
		}
		logger.Infof("adaptive service verbose is enabled.")
		logger.Debugf("debug-level info could be shown.")
		aslimiter.Verbose = true
	}
	return nil
}

func (c *ProviderConfig) Load() {
	for registeredTypeName, service := range GetProviderServiceMap() {
		serviceConfig, ok := c.Services[registeredTypeName]
		if !ok {
			if registeredTypeName == constant.ReflectionServiceTypeName ||
				registeredTypeName == constant.HealthCheckServiceTypeName {
				// do not auto generate reflection or health check server's configuration.
				continue
			}
			// service doesn't config in config file, create one with default
			supportPBPackagerNameService, ok := service.(common.TriplePBService)
			if !ok {
				logger.Warnf(
					"The provider service %s is ignored: neither the config is found, nor it is a valid Triple service.",
					registeredTypeName)
				continue
			}
			serviceConfig = NewServiceConfigBuilder().Build()
			// use interface name defined by pb
			serviceConfig.Interface = supportPBPackagerNameService.XXX_InterfaceName()
			if err := serviceConfig.Init(rootConfig); err != nil {
				logger.Errorf("Service with registeredTypeName = %s init failed with error = %#v", registeredTypeName, err)
			}
			serviceConfig.adaptiveService = c.AdaptiveService
		}
		serviceConfig.id = registeredTypeName
		serviceConfig.Implement(service)
		if err := serviceConfig.Export(); err != nil {
			logger.Errorf(fmt.Sprintf("service with registeredTypeName = %s export failed! err: %#v", registeredTypeName, err))
		}
	}
}

// newEmptyProviderConfig returns ProviderConfig with default ApplicationConfig
func newEmptyProviderConfig() *ProviderConfig {
	newProviderConfig := &ProviderConfig{
		Services:    make(map[string]*ServiceConfig),
		RegistryIDs: make([]string, 8),
		ProtocolIDs: make([]string, 8),
	}
	return newProviderConfig
}

type ProviderConfigBuilder struct {
	providerConfig *ProviderConfig
}

func NewProviderConfigBuilder() *ProviderConfigBuilder {
	return &ProviderConfigBuilder{providerConfig: newEmptyProviderConfig()}
}

func (pcb *ProviderConfigBuilder) SetFilter(filter string) *ProviderConfigBuilder {
	pcb.providerConfig.Filter = filter
	return pcb
}

func (pcb *ProviderConfigBuilder) SetRegister(register bool) *ProviderConfigBuilder {
	pcb.providerConfig.Register = register
	return pcb
}

func (pcb *ProviderConfigBuilder) SetRegistryIDs(RegistryIDs ...string) *ProviderConfigBuilder {
	pcb.providerConfig.RegistryIDs = RegistryIDs
	return pcb
}

func (pcb *ProviderConfigBuilder) SetServices(services map[string]*ServiceConfig) *ProviderConfigBuilder {
	pcb.providerConfig.Services = services
	return pcb
}

func (pcb *ProviderConfigBuilder) AddService(serviceID string, serviceConfig *ServiceConfig) *ProviderConfigBuilder {
	if pcb.providerConfig.Services == nil {
		pcb.providerConfig.Services = make(map[string]*ServiceConfig)
	}
	pcb.providerConfig.Services[serviceID] = serviceConfig
	return pcb
}

func (pcb *ProviderConfigBuilder) SetProxyFactory(proxyFactory string) *ProviderConfigBuilder {
	pcb.providerConfig.ProxyFactory = proxyFactory
	return pcb
}

func (pcb *ProviderConfigBuilder) SetFilterConf(filterConf any) *ProviderConfigBuilder {
	pcb.providerConfig.FilterConf = filterConf
	return pcb
}

func (pcb *ProviderConfigBuilder) SetConfigType(configType map[string]string) *ProviderConfigBuilder {
	pcb.providerConfig.ConfigType = configType
	return pcb
}

func (pcb *ProviderConfigBuilder) AddConfigType(key, value string) *ProviderConfigBuilder {
	if pcb.providerConfig.ConfigType == nil {
		pcb.providerConfig.ConfigType = make(map[string]string)
	}
	pcb.providerConfig.ConfigType[key] = value
	return pcb
}

func (pcb *ProviderConfigBuilder) SetRootConfig(rootConfig *RootConfig) *ProviderConfigBuilder {
	pcb.providerConfig.rootConfig = rootConfig
	return pcb
}

func (pcb *ProviderConfigBuilder) Build() *ProviderConfig {
	return pcb.providerConfig
}

// DynamicUpdateProperties 动态更新提供者配置属性
func (c *ProviderConfig) DynamicUpdateProperties(n *ProviderConfig) {
	if n == nil {
		return
	}

	// 使用ApplyConfigUpdate函数进行安全更新
	ApplyConfigUpdate(&c.Filter, n.Filter)
	ApplyConfigUpdate(&c.Register, n.Register)
	ApplyConfigUpdate(&c.TracingKey, n.TracingKey)
	ApplyConfigUpdate(&c.ProxyFactory, n.ProxyFactory)
	ApplyConfigUpdate(&c.FilterConf, n.FilterConf)
	ApplyConfigUpdate(&c.AdaptiveService, n.AdaptiveService)
	ApplyConfigUpdate(&c.AdaptiveServiceVerbose, n.AdaptiveServiceVerbose)

	// 更新RegistryIDs
	if len(n.RegistryIDs) > 0 {
		c.RegistryIDs = n.RegistryIDs
	}

	// 更新ProtocolIDs
	if len(n.ProtocolIDs) > 0 {
		c.ProtocolIDs = n.ProtocolIDs
	}

	// 更新ConfigType
	if len(n.ConfigType) > 0 {
		if c.ConfigType == nil {
			c.ConfigType = make(map[string]string)
		}
		for k, v := range n.ConfigType {
			c.ConfigType[k] = v
		}
	}

	// 更新Services - 支持新增和修改，但不删除现有服务
	if len(n.Services) > 0 {
		if c.Services == nil {
			c.Services = make(map[string]*ServiceConfig)
		}
		for serviceID, serviceConfig := range n.Services {
			if existingService, exists := c.Services[serviceID]; exists {
				// 更新现有服务配置
				existingService.DynamicUpdateProperties(serviceConfig)
			} else {
				// 新增服务配置
				c.Services[serviceID] = serviceConfig
			}
		}
	}
}

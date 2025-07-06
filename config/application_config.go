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
	"strconv"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/creasty/defaults"
	"github.com/pkg/errors"
)

// ApplicationConfig is a configuration for current applicationConfig, whether the applicationConfig is a provider or a consumer
type ApplicationConfig struct {
	Organization string `default:"dubbo-go" yaml:"organization" json:"organization,omitempty" property:"organization"`
	Name         string `yaml:"name" json:"name,omitempty" property:"name"`
	Module       string `default:"sample" yaml:"module" json:"module,omitempty" property:"module"`
	Group        string `yaml:"group" json:"group,omitempty" property:"module"`
	Version      string `yaml:"version" json:"version,omitempty" property:"version"`
	Owner        string `default:"dubbo-go" yaml:"owner" json:"owner,omitempty" property:"owner"`
	Environment  string `yaml:"environment" json:"environment,omitempty" property:"environment"`
	// the metadata type. remote or local
	MetadataType            string `default:"local" yaml:"metadata-type" json:"metadataType,omitempty" property:"metadataType"`
	Tag                     string `yaml:"tag" json:"tag,omitempty" property:"tag"`
	MetadataServicePort     string `yaml:"metadata-service-port" json:"metadata-service-port,omitempty" property:"metadata-service-port"`
	MetadataServiceProtocol string `yaml:"metadata-service-protocol" json:"metadata-service-protocol,omitempty" property:"metadata-service-protocol"`
}

// Prefix dubbo.application
func (ac *ApplicationConfig) Prefix() string {
	return constant.ApplicationConfigPrefix
}

// Init  application config and set default value
func (ac *ApplicationConfig) Init() error {
	if ac == nil {
		return errors.New("application is null")
	}
	if err := ac.check(); err != nil {
		return err
	}
	if ac.Name == "" {
		ac.Name = constant.DefaultDubboApp
	}
	return nil
}

func (ac *ApplicationConfig) check() error {
	if err := defaults.Set(ac); err != nil {
		return err
	}
	return verify(ac)
}

func NewApplicationConfigBuilder() *ApplicationConfigBuilder {
	return &ApplicationConfigBuilder{application: &ApplicationConfig{}}
}

type ApplicationConfigBuilder struct {
	application *ApplicationConfig
}

func (acb *ApplicationConfigBuilder) SetOrganization(organization string) *ApplicationConfigBuilder {
	acb.application.Organization = organization
	return acb
}

func (acb *ApplicationConfigBuilder) SetName(name string) *ApplicationConfigBuilder {
	acb.application.Name = name
	return acb
}

func (acb *ApplicationConfigBuilder) SetModule(module string) *ApplicationConfigBuilder {
	acb.application.Module = module
	return acb
}

func (acb *ApplicationConfigBuilder) SetVersion(version string) *ApplicationConfigBuilder {
	acb.application.Version = version
	return acb
}

func (acb *ApplicationConfigBuilder) SetOwner(owner string) *ApplicationConfigBuilder {
	acb.application.Owner = owner
	return acb
}

func (acb *ApplicationConfigBuilder) SetEnvironment(environment string) *ApplicationConfigBuilder {
	acb.application.Environment = environment
	return acb
}

func (acb *ApplicationConfigBuilder) SetMetadataType(metadataType string) *ApplicationConfigBuilder {
	acb.application.MetadataType = metadataType
	return acb
}

func (acb *ApplicationConfigBuilder) SetMetadataServicePort(port int) *ApplicationConfigBuilder {
	acb.application.MetadataServicePort = strconv.Itoa(port)
	return acb
}

func (acb *ApplicationConfigBuilder) SetMetadataServiceProtocol(protocol string) *ApplicationConfigBuilder {
	acb.application.MetadataServiceProtocol = protocol
	return acb
}

func (acb *ApplicationConfigBuilder) Build() *ApplicationConfig {
	return acb.application
}

// DynamicUpdateProperties 动态更新应用配置属性
func (ac *ApplicationConfig) DynamicUpdateProperties(n *ApplicationConfig) {
	if n == nil {
		return
	}

	// 使用ApplyConfigUpdate函数进行安全更新
	ApplyConfigUpdate(&ac.Organization, n.Organization)
	ApplyConfigUpdate(&ac.Name, n.Name)
	ApplyConfigUpdate(&ac.Module, n.Module)
	ApplyConfigUpdate(&ac.Group, n.Group)
	ApplyConfigUpdate(&ac.Version, n.Version)
	ApplyConfigUpdate(&ac.Owner, n.Owner)
	ApplyConfigUpdate(&ac.Environment, n.Environment)
	ApplyConfigUpdate(&ac.MetadataType, n.MetadataType)
	ApplyConfigUpdate(&ac.Tag, n.Tag)
	ApplyConfigUpdate(&ac.MetadataServicePort, n.MetadataServicePort)
	ApplyConfigUpdate(&ac.MetadataServiceProtocol, n.MetadataServiceProtocol)
}

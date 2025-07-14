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
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/creasty/defaults"
	"github.com/pkg/errors"
)

// ApplicationConfig is a configuration for current application, whether the application is a provider or a consumer
type ApplicationConfig struct {
	mutex sync.RWMutex // Mutex for concurrency protection

	Organization string `default:"dubbo-go" yaml:"organization" json:"organization,omitempty" property:"organization"`
	Name         string `yaml:"name" json:"name,omitempty" property:"name"`
	Module       string `default:"sample" yaml:"module" json:"module,omitempty" property:"module"`
	Group        string `yaml:"group" json:"group,omitempty" property:"group"` // Fixed property tag
	Version      string `yaml:"version" json:"version,omitempty" property:"version"`
	Owner        string `default:"dubbo-go" yaml:"owner" json:"owner,omitempty" property:"owner"`
	Environment  string `yaml:"environment" json:"environment,omitempty" property:"environment"`
	// the metadata type. remote or local
	MetadataType            string `default:"local" yaml:"metadata-type" json:"metadataType,omitempty" property:"metadataType"`
	Tag                     string `yaml:"tag" json:"tag,omitempty" property:"tag"`
	MetadataServicePort     string `yaml:"metadata-service-port" json:"metadata-service-port,omitempty" property:"metadata-service-port"`
	MetadataServiceProtocol string `yaml:"metadata-service-protocol" json:"metadata-service-protocol,omitempty" property:"metadata-service-protocol"`

	// List of configuration change listeners
	changeListeners []ConfigChangeListener
}

// ConfigChangeListener is a function type that will be called when configuration changes
type ConfigChangeListener func(old, new *ApplicationConfig)

// AddChangeListener adds a listener that will be notified when configuration changes
func (ac *ApplicationConfig) AddChangeListener(listener ConfigChangeListener) {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()
	ac.changeListeners = append(ac.changeListeners, listener)
}

// GetSnapshot returns a deep copy of the current configuration
func (ac *ApplicationConfig) GetSnapshot() *ApplicationConfig {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	// Create a new instance and copy all fields
	return &ApplicationConfig{
		Organization:            ac.Organization,
		Name:                    ac.Name,
		Module:                  ac.Module,
		Group:                   ac.Group,
		Version:                 ac.Version,
		Owner:                   ac.Owner,
		Environment:             ac.Environment,
		MetadataType:            ac.MetadataType,
		Tag:                     ac.Tag,
		MetadataServicePort:     ac.MetadataServicePort,
		MetadataServiceProtocol: ac.MetadataServiceProtocol,
	}
}

// Prefix dubbo.application
func (ac *ApplicationConfig) Prefix() string {
	return constant.ApplicationConfigPrefix
}

// Init application config and set default value
func (ac *ApplicationConfig) Init() error {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

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

func (acb *ApplicationConfigBuilder) SetGroup(group string) *ApplicationConfigBuilder {
	acb.application.Group = group
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
	// Create a new instance to avoid sharing the pointer with the builder
	snapshot := &ApplicationConfig{
		Organization:            acb.application.Organization,
		Name:                    acb.application.Name,
		Module:                  acb.application.Module,
		Group:                   acb.application.Group,
		Version:                 acb.application.Version,
		Owner:                   acb.application.Owner,
		Environment:             acb.application.Environment,
		MetadataType:            acb.application.MetadataType,
		Tag:                     acb.application.Tag,
		MetadataServicePort:     acb.application.MetadataServicePort,
		MetadataServiceProtocol: acb.application.MetadataServiceProtocol,
	}
	return snapshot
}

// DynamicUpdateProperties updates application config properties with thread safety
func (ac *ApplicationConfig) DynamicUpdateProperties(n *ApplicationConfig) {
	if n == nil {
		return
	}

	// Create a snapshot of the current config for change listeners
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	oldConfig := ac.GetSnapshot()

	// Update fields with proper locking
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

	// Notify listeners about the change
	for _, listener := range ac.changeListeners {
		go listener(oldConfig, ac.GetSnapshot())
	}
}

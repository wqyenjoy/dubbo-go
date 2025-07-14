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

// Package app provides application level configuration center support
package app

import (
	"strings"
	"sync"

	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"

	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
)

// AppMergedConfiguration is a decorator for DynamicConfiguration that adds application-level configuration support
// It follows the priority order: application-level config > global config
type AppMergedConfiguration struct {
	config    config_center.DynamicConfiguration // Underlying dynamic configuration implementation
	appName   string
	appSuffix string
	mutex     sync.RWMutex // Mutex for concurrency protection

	// List of configuration change listeners
	changeListeners []ConfigChangeListener
}

// ConfigChangeListener is a function type that will be called when configuration changes
type ConfigChangeListener func(key string, oldValue, newValue string)

// AddChangeListener adds a listener that will be notified when configuration changes
func (a *AppMergedConfiguration) AddChangeListener(listener ConfigChangeListener) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.changeListeners = append(a.changeListeners, listener)
}

// NewAppMergedConfiguration creates a new application-level configuration decorator
func NewAppMergedConfiguration(appName string, config config_center.DynamicConfiguration) *AppMergedConfiguration {
	return &AppMergedConfiguration{
		config:    config,
		appName:   appName,
		appSuffix: appName + ".",
		mutex:     sync.RWMutex{},
	}
}

// Parser gets the configuration parser
func (a *AppMergedConfiguration) Parser() parser.ConfigurationParser {
	return a.config.Parser()
}

// SetParser sets the configuration parser
func (a *AppMergedConfiguration) SetParser(p parser.ConfigurationParser) {
	a.config.SetParser(p)
}

// GetProperties gets configuration with thread safety, prioritizing application-level config
func (a *AppMergedConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// If application name is empty, use the original implementation
	if a.appName == "" {
		return a.config.GetProperties(key, opts...)
	}

	// Try to get application-level configuration
	appKey := a.appSuffix + key
	appContent, err := a.config.GetProperties(appKey, opts...)
	if err == nil && len(appContent) > 0 {
		logger.Infof("[App Config] Using app-level config for key: %s", appKey)
		return appContent, nil
	}

	// Fall back to global configuration if application-level config doesn't exist or is empty
	globalContent, err := a.config.GetProperties(key, opts...)
	if err != nil {
		logger.Warnf("[App Config] Failed to get config for both app-level(%s) and global(%s): %v",
			appKey, key, err)
		return "", err
	}

	return globalContent, nil
}

// GetRule gets routing rules with thread safety, prioritizing application-level rules
func (a *AppMergedConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// If application name is empty, use the original implementation
	if a.appName == "" {
		return a.config.GetRule(key, opts...)
	}

	// Try to get application-level rule
	appKey := a.appSuffix + key
	appRule, err := a.config.GetRule(appKey, opts...)
	if err == nil && len(appRule) > 0 {
		logger.Infof("[App Config] Using app-level rule for key: %s", appKey)
		return appRule, nil
	}

	// Fall back to global rule if application-level rule doesn't exist or is empty
	globalRule, err := a.config.GetRule(key, opts...)
	if err != nil {
		logger.Warnf("[App Config] Failed to get rule for both app-level(%s) and global(%s): %v",
			appKey, key, err)
		return "", err
	}

	return globalRule, nil
}

// GetInternalProperty gets internal property with thread safety, prioritizing application-level property
func (a *AppMergedConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// If application name is empty, use the original implementation
	if a.appName == "" {
		return a.config.GetInternalProperty(key, opts...)
	}

	// Try to get application-level internal property
	appKey := a.appSuffix + key
	appProperty, err := a.config.GetInternalProperty(appKey, opts...)
	if err == nil && len(appProperty) > 0 {
		logger.Infof("[App Config] Using app-level internal property for key: %s", appKey)
		return appProperty, nil
	}

	// Fall back to global internal property if application-level property doesn't exist or is empty
	globalProperty, err := a.config.GetInternalProperty(key, opts...)
	if err != nil {
		logger.Warnf("[App Config] Failed to get internal property for both app-level(%s) and global(%s): %v",
			appKey, key, err)
		return "", err
	}

	return globalProperty, nil
}

// PublishConfig publishes configuration with thread safety
func (a *AppMergedConfiguration) PublishConfig(key string, group string, value string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// If application name is empty, use the original implementation
	if a.appName == "" {
		return a.config.PublishConfig(key, group, value)
	}

	// Publish to application-level configuration
	appKey := a.appSuffix + key
	err := a.config.PublishConfig(appKey, group, value)
	if err != nil {
		logger.Warnf("[App Config] Failed to publish app-level config for key: %s, error: %v", appKey, err)
		// Fall back to publishing to global configuration if publishing to application-level fails
		return a.config.PublishConfig(key, group, value)
	}

	// Notify listeners about the change
	for _, listener := range a.changeListeners {
		go listener(key, "", value) // We don't have the old value here
	}

	return nil
}

// RemoveConfig removes configuration with thread safety
func (a *AppMergedConfiguration) RemoveConfig(key string, group string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// If application name is empty, use the original implementation
	if a.appName == "" {
		return a.config.RemoveConfig(key, group)
	}

	// Remove application-level configuration
	appKey := a.appSuffix + key
	_ = a.config.RemoveConfig(appKey, group)
	// Always try to remove global configuration regardless of whether removing application-level config succeeds
	return a.config.RemoveConfig(key, group)
}

// AddListener adds a listener with thread safety
func (a *AppMergedConfiguration) AddListener(key string, listener config_center.ConfigurationListener, opts ...config_center.Option) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// If application name is empty, use the original implementation
	if a.appName == "" {
		a.config.AddListener(key, listener, opts...)
		return
	}

	// Listen to application-level configuration
	appKey := a.appSuffix + key
	a.config.AddListener(appKey, listener, opts...)
	// Also listen to global configuration
	a.config.AddListener(key, listener, opts...)
}

// RemoveListener removes a listener with thread safety
func (a *AppMergedConfiguration) RemoveListener(key string, listener config_center.ConfigurationListener, opts ...config_center.Option) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// If application name is empty, use the original implementation
	if a.appName == "" {
		a.config.RemoveListener(key, listener, opts...)
		return
	}

	// Remove listener from application-level configuration
	appKey := a.appSuffix + key
	a.config.RemoveListener(appKey, listener, opts...)
	// Also remove listener from global configuration
	a.config.RemoveListener(key, listener, opts...)
}

// GetConfigKeysByGroup gets all configuration keys in a group with thread safety
func (a *AppMergedConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// If application name is empty, use the original implementation
	if a.appName == "" {
		return a.config.GetConfigKeysByGroup(group)
	}

	// Get global configuration keys
	globalKeys, err := a.config.GetConfigKeysByGroup(group)
	if err != nil {
		return nil, err
	}

	// Try to get application-level configuration keys
	appKeys, err := a.config.GetConfigKeysByGroup(group)
	if err != nil {
		// If getting application-level keys fails, just return global keys
		return globalKeys, nil
	}

	// Merge application-level and global configuration keys
	mergedKeys := gxset.NewSet()

	// Add global configuration keys
	if globalKeys != nil {
		for _, k := range globalKeys.Values() {
			mergedKeys.Add(k)
		}
	}

	// Process application-level keys, removing application prefix
	if appKeys != nil {
		for _, k := range appKeys.Values() {
			keyStr, ok := k.(string)
			if ok && strings.HasPrefix(keyStr, a.appSuffix) {
				// Remove application prefix
				globalKey := keyStr[len(a.appSuffix):]
				mergedKeys.Add(globalKey)
			}
		}
	}

	return mergedKeys, nil
}

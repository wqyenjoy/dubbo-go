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

// This is an example showing how to use the application-level configuration center.
// It is not meant to be run directly, but rather to serve as a reference.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/app"
	"github.com/dubbogo/gost/log/logger"
)

// Example 1: Using the application-level configuration center with code
func exampleWithCode() {
	// Create a URL for the configuration center
	url := common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:2181"),
		common.WithParamsValue(constant.ApplicationKey, "my-app"),
	)

	// Get the configuration center factory
	factory, err := extension.GetConfigCenterFactory("zookeeper")
	if err != nil {
		logger.Errorf("Failed to get config center factory: %v", err)
		return
	}

	// Create the dynamic configuration
	dynamicConfig, err := factory.GetDynamicConfiguration(url)
	if err != nil {
		logger.Errorf("Failed to create dynamic configuration: %v", err)
		return
	}

	// Wrap with application-level support
	appConfig := app.WrapWithAppConfig(dynamicConfig, "my-app")

	// Use the application-level configuration
	// This will first check for "my-app.dubbo.registry.address", then "dubbo.registry.address"
	registryAddress, err := appConfig.GetProperties("dubbo.registry.address")
	if err != nil {
		logger.Errorf("Failed to get registry address: %v", err)
		return
	}

	logger.Infof("Registry address: %s", registryAddress)

	// Publish a configuration value (will publish to "my-app.dubbo.provider.timeout")
	err = appConfig.PublishConfig("dubbo.provider.timeout", "dubbo", "5000")
	if err != nil {
		logger.Errorf("Failed to publish config: %v", err)
		return
	}

	logger.Info("Published configuration successfully")
}

// Example 2: Using the application-level configuration center with YAML
func exampleWithYAML() {
	// Create a configuration center config
	configCenterConfig := config.NewConfigCenterConfigBuilder().
		SetProtocol("app-merged").
		SetAddress("zookeeper://127.0.0.1:2181").
		SetGroup("dubbo").
		SetNamespace("public").
		Build()

	// Add application name parameter
	configCenterConfig.Params["appName"] = "my-app"
	configCenterConfig.Params["protocol"] = "zookeeper"

	// Create a root config
	rootConfig := config.NewRootConfigBuilder().
		SetConfigCenter(configCenterConfig).
		Build()

	// Initialize the configuration
	if err := rootConfig.Init(); err != nil {
		logger.Errorf("Failed to initialize root config: %v", err)
		return
	}

	logger.Info("Configuration initialized successfully")

	// Wait for configuration changes
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)
	<-c
}

// Example 3: Using a configuration listener
func exampleWithListener() {
	// Create a URL for the configuration center
	url := common.NewURLWithOptions(
		common.WithProtocol("zookeeper"),
		common.WithLocation("127.0.0.1:2181"),
		common.WithParamsValue(constant.ApplicationKey, "my-app"),
	)

	// Get the configuration center factory
	factory, err := extension.GetConfigCenterFactory("zookeeper")
	if err != nil {
		logger.Errorf("Failed to get config center factory: %v", err)
		return
	}

	// Create the dynamic configuration
	dynamicConfig, err := factory.GetDynamicConfiguration(url)
	if err != nil {
		logger.Errorf("Failed to create dynamic configuration: %v", err)
		return
	}

	// Wrap with application-level support
	appConfig := app.WrapWithAppConfig(dynamicConfig, "my-app")

	// Create a configuration listener
	listener := &MyConfigurationListener{}

	// Add the listener for a specific key
	appConfig.AddListener("dubbo.provider.timeout", listener)

	// Wait for configuration changes
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)
	<-c

	// Remove the listener when done
	appConfig.RemoveListener("dubbo.provider.timeout", listener)
}

// MyConfigurationListener implements the ConfigurationListener interface
type MyConfigurationListener struct{}

// Process is called when the configuration changes
func (l *MyConfigurationListener) Process(event *config_center.ConfigChangeEvent) {
	logger.Infof("Configuration changed: key=%s, value=%s", event.Key, event.Value)

	// Update your application's configuration here
	if event.Key == "dubbo.provider.timeout" {
		timeout := event.Value
		fmt.Printf("Provider timeout updated to: %s\n", timeout)
	}
}

func main() {
	// This is just a placeholder. In a real application, you would use one of the example functions.
	logger.Info("This is an example file and is not meant to be run directly.")
}

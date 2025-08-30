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
	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/dubbogo/gost/log/logger"
	"github.com/pkg/errors"

	_ "dubbo.apache.org/dubbo-go/v3/logger/core/logrus"
	"dubbo.apache.org/dubbo-go/v3/logger/core/zap"
)

var (
	rootConfig = NewRootConfigBuilder().Build()
)

func init() {
	log := zap.NewDefault()
	logger.SetLogger(log)
}

func Load(opts ...LoaderConfOption) (*RootConfig, error) {
	conf, err := NewLoaderConf(opts...)
	if err != nil {
		return nil, err
	}

	koan := GetConfigResolver(conf)
	if koan == nil {
		return nil, errors.New("failed to resolve config")
	}

	conf.MergeConfig(koan)

	rc := &RootConfig{}
	if err := koan.Unmarshal("", rc); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config")
	}

	// 设置全局配置，保持向后兼容
	SetAtomicRootConfig(rc)

	// 初始化配置
	if err := rc.Init(); err != nil {
		return nil, err
	}

	return rc, nil
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







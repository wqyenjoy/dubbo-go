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
	"github.com/dubbogo/gost/log/logger"

	"github.com/knadh/koanf"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
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

func Load(opts ...LoaderConfOption) error {
	conf := NewLoaderConf(opts...)

	koan := GetConfigResolver(conf)
	if koan == nil {
		return errors.New("failed to resolve config")
	}

	conf.MergeConfig(koan)

	// 使用带有默认子配置的RootConfig，避免nil字段在Init时引发panic
	rc := NewRootConfigBuilder().Build()
	// 使用yaml标签并限定到根前缀进行反序列化
	if err := koan.UnmarshalWithConf(rc.Prefix(), rc, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return errors.Wrap(err, "failed to unmarshal config")
	}

	// 兜底：确保Protocols map存在且元素非nil，避免后续Init时或测试访问字段发生NPE
	if rc.Protocols == nil {
		rc.Protocols = make(map[string]*ProtocolConfig)
	}
	// 若存在协议分支但某个条目仍为nil，则对子树进行单独反序列化填充（使用yaml标签）
	if raw := koan.Get(constant.DubboProtocol + ".protocols"); raw != nil {
		if mm, ok := raw.(map[string]any); ok {
			for name := range mm {
				if rc.Protocols[name] == nil {
					pc := &ProtocolConfig{}
					_ = koan.Cut(constant.DubboProtocol+".protocols."+name).UnmarshalWithConf("", pc, koanf.UnmarshalConf{Tag: "yaml"})
					rc.Protocols[name] = pc
				}
			}
		}
	}

	// 设置全局配置，保持向后兼容
	SetAtomicRootConfig(rc)

	// 初始化配置
	if err := rc.Init(); err != nil {
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

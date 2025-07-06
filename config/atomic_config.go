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
	"sync/atomic"
)

var (
	// atomicRootConfig 使用atomic.Value保护rootConfig
	atomicRootConfig atomic.Value
)

// init 初始化atomicRootConfig
func init() {
	atomicRootConfig.Store(NewRootConfigBuilder().Build())
}

// GetAtomicRootConfig 线程安全地获取根配置
func GetAtomicRootConfig() *RootConfig {
	if rc := atomicRootConfig.Load(); rc != nil {
		return rc.(*RootConfig)
	}
	return nil
}

// SetAtomicRootConfig 线程安全地设置根配置
func SetAtomicRootConfig(rc *RootConfig) {
	atomicRootConfig.Store(rc)
}

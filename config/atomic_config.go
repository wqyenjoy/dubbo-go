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
	// atomicRootConfig uses atomic.Value to protect rootConfig
	atomicRootConfig atomic.Value
)

// init initializes atomicRootConfig
func init() {
	atomicRootConfig.Store(NewRootConfigBuilder().Build())
}

// GetAtomicRootConfig safely gets the root configuration in a thread-safe manner
func GetAtomicRootConfig() *RootConfig {
	if rc := atomicRootConfig.Load(); rc != nil {
		return rc.(*RootConfig)
	}
	return nil
}

// SetAtomicRootConfig safely sets the root configuration in a thread-safe manner
func SetAtomicRootConfig(rc *RootConfig) {
	atomicRootConfig.Store(rc)
	// Synchronously update package-level variable for compatibility with code and tests that directly use rootConfig
	rootConfig = rc
}

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
	"sync"
	"testing"
	"time"
)

func TestAtomicRootConfig_ThreadSafety(t *testing.T) {
	// 并发读写测试
	var wg sync.WaitGroup
	concurrentCount := 100
	readCount := 1000
	writeCount := 100

	// 启动多个读取协程
	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < readCount; j++ {
				config := GetAtomicRootConfig()
				if config == nil {
					t.Errorf("GetAtomicRootConfig returned nil")
					return
				}
				// 读取一些配置值
				_ = config.Application.Name
				_ = config.Application.Version
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// 启动多个写入协程
	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < writeCount; j++ {
				newConfig := NewRootConfigBuilder().
					SetApplication(NewApplicationConfigBuilder().
						SetName("test-app-" + string(rune(id))).
						SetVersion("1.0." + string(rune(j))).
						Build()).
					Build()
				SetAtomicRootConfig(newConfig)
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	// 验证最终状态
	finalConfig := GetAtomicRootConfig()
	if finalConfig == nil {
		t.Fatal("Final config is nil")
	}
}

func TestAtomicRootConfig_Consistency(t *testing.T) {
	// 测试配置一致性
	initialConfig := GetAtomicRootConfig()
	if initialConfig == nil {
		t.Fatal("Initial config is nil")
	}

	// 创建新配置
	newConfig := NewRootConfigBuilder().
		SetApplication(NewApplicationConfigBuilder().
			SetName("test-app").
			SetVersion("2.0.0").
			Build()).
		Build()

	// 设置新配置
	SetAtomicRootConfig(newConfig)

	// 验证配置已更新
	updatedConfig := GetAtomicRootConfig()
	if updatedConfig == nil {
		t.Fatal("Updated config is nil")
	}

	if updatedConfig.Application.Name != "test-app" {
		t.Errorf("Expected app name 'test-app', got '%s'", updatedConfig.Application.Name)
	}

	if updatedConfig.Application.Version != "2.0.0" {
		t.Errorf("Expected app version '2.0.0', got '%s'", updatedConfig.Application.Version)
	}
}

func TestAtomicRootConfig_RaceCondition(t *testing.T) {
	// 使用go test -race来检测竞态条件
	var wg sync.WaitGroup
	iterations := 1000

	// 并发读写测试
	for i := 0; i < iterations; i++ {
		wg.Add(2)

		// 读取协程
		go func() {
			defer wg.Done()
			config := GetAtomicRootConfig()
			if config != nil {
				_ = config.Application.Name
			}
		}()

		// 写入协程
		go func() {
			defer wg.Done()
			newConfig := NewRootConfigBuilder().
				SetApplication(NewApplicationConfigBuilder().
					SetName("race-test").
					Build()).
				Build()
			SetAtomicRootConfig(newConfig)
		}()
	}

	wg.Wait()
}

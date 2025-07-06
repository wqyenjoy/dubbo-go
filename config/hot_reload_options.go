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
	"time"
)

// HotReloadOptions 热加载配置选项
type HotReloadOptions struct {
	Enabled       bool                    // 是否启用热加载
	ConfigPath    string                  // 配置文件路径
	DebounceDelay time.Duration           // 防抖延迟时间
	AutoStart     bool                    // 是否自动启动
	Callback      func(*RootConfig) error // 自定义重载回调函数
}

// HotReloadOption 热加载选项函数类型
type HotReloadOption func(*HotReloadOptions)

// WithHotReloadEnabled 启用热加载
func WithHotReloadEnabled(enabled bool) HotReloadOption {
	return func(opts *HotReloadOptions) {
		opts.Enabled = enabled
	}
}

// WithHotReloadConfigPath 设置配置文件路径
func WithHotReloadConfigPath(path string) HotReloadOption {
	return func(opts *HotReloadOptions) {
		opts.ConfigPath = path
	}
}

// WithHotReloadDebounceDelay 设置防抖延迟时间
func WithHotReloadDebounceDelay(delay time.Duration) HotReloadOption {
	return func(opts *HotReloadOptions) {
		opts.DebounceDelay = delay
	}
}

// WithHotReloadAutoStart 设置是否自动启动
func WithHotReloadAutoStart(autoStart bool) HotReloadOption {
	return func(opts *HotReloadOptions) {
		opts.AutoStart = autoStart
	}
}

// WithHotReloadCallback 设置自定义重载回调函数
func WithHotReloadCallback(callback func(*RootConfig) error) HotReloadOption {
	return func(opts *HotReloadOptions) {
		opts.Callback = callback
	}
}

// NewHotReloadOptions 创建新的热加载选项
func NewHotReloadOptions(opts ...HotReloadOption) *HotReloadOptions {
	options := &HotReloadOptions{
		Enabled:       false,                  // 默认不启用
		ConfigPath:    "",                     // 默认使用全局配置路径
		DebounceDelay: 500 * time.Millisecond, // 默认500ms防抖
		AutoStart:     false,                  // 默认不自动启动
		Callback:      nil,                    // 默认无自定义回调
	}

	for _, opt := range opts {
		opt(options)
	}

	return options
}

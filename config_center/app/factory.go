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

package app

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
)

func init() {
	// 注册应用级配置中心工厂
	extension.SetConfigCenterFactory(AppMergedConfigKey, createAppMergedFactory)
}

// 创建工厂函数
func createAppMergedFactory() config_center.DynamicConfigurationFactory {
	return &appMergedDynamicConfigurationFactory{}
}

// 应用级配置中心工厂
type appMergedDynamicConfigurationFactory struct{}

// GetDynamicConfiguration 获取应用级动态配置
func (f *appMergedDynamicConfigurationFactory) GetDynamicConfiguration(url *common.URL) (config_center.DynamicConfiguration, error) {
	return newAppMergedDynamicConfiguration(url)
}

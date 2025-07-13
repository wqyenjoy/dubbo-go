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
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"github.com/dubbogo/gost/log/logger"
)

// WrapWithAppConfig 将普通动态配置包装为应用级配置
func WrapWithAppConfig(dc config_center.DynamicConfiguration, appName string) config_center.DynamicConfiguration {
	// 如果应用名为空，直接返回原始配置
	if appName == "" {
		return dc
	}

	// 创建URL，用于传递应用名
	url := common.NewURLWithOptions(
		common.WithProtocol(AppMergedConfigKey),
		common.WithParamsValue("appName", appName),
		common.WithParamsValue(constant.ApplicationKey, appName),
	)

	// 创建应用级配置
	appConfig, err := newAppMergedDynamicConfiguration(url)
	if err != nil {
		logger.Warnf("[App Config] Failed to create app-merged config: %v, using original config", err)
		return dc
	}

	logger.Infof("[App Config] Successfully wrapped dynamic configuration with app-level config for app: %s", appName)
	return appConfig
}

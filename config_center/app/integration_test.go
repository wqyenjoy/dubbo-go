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
	"testing"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/stretchr/testify/assert"
)

// TestIntegration_AppMergedConfiguration 测试应用级配置中心的集成功能
func TestIntegration_AppMergedConfiguration(t *testing.T) {
	// 跳过集成测试，除非明确指定要运行
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 使用mock配置中心代替真实的配置中心
	mock := NewMockDynamicConfiguration()
	mock.data["timeout"] = "1000"
	mock.data["test-app.timeout"] = "2000"

	// 创建应用级配置
	appConfig := NewAppMergedConfiguration("test-app", mock)

	// 验证优先获取应用级配置
	value, err := appConfig.GetProperties("timeout")
	assert.NoError(t, err)
	assert.Equal(t, "2000", value)

	// 使用包装函数
	wrapped := WrapWithAppConfig(mock, "test-app")
	value, err = wrapped.GetProperties("timeout")
	assert.NoError(t, err)
	assert.Equal(t, "2000", value)

	// 测试空应用名
	wrapped = WrapWithAppConfig(mock, "")
	value, err = wrapped.GetProperties("timeout")
	assert.NoError(t, err)
	assert.Equal(t, "1000", value)

	// 测试URL创建
	url := common.NewURLWithOptions(
		common.WithProtocol(AppMergedConfigKey),
		common.WithLocation("memory://localhost"),
		common.WithParamsValue(constant.ApplicationKey, "test-app"),
	)

	// 验证URL参数
	assert.Equal(t, "test-app", url.GetParam(constant.ApplicationKey, ""))
	assert.Equal(t, AppMergedConfigKey, url.Protocol)
}

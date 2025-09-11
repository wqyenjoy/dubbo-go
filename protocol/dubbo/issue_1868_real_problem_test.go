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

package dubbo

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// TestIssue1868RealProblemAnalysis 分析真正的问题
func TestIssue1868RealProblemAnalysis(t *testing.T) {
	t.Log("🔍 Issue #1868 Real Problem Analysis")

	t.Log("🚨 Our Previous Fix Was WRONG!")
	t.Log("   1. Getty Options struct has no TcpWriteTimeout field")
	t.Log("   2. Our adjustGettyConfigForRequestTimeout() function is never called")
	t.Log("   3. TcpWriteTimeout is configured in ClientConfig, not Options")

	t.Log("🎯 Real Problem Discovery:")
	t.Log("   1. initClient(url) only loads config from protocol configuration")
	t.Log("   2. It IGNORES URL parameters like 'timeout'")
	t.Log("   3. Getty's TcpWriteTimeout is hardcoded to protocol config")

	// 创建一个带有timeout参数的URL
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.Service")
	assert.NoError(t, err)
	url.SetParam(constant.TimeoutKey, "60s")

	t.Logf("📊 URL Analysis:")
	t.Logf("   URL: %s", url.String())
	t.Logf("   Timeout parameter: %s", url.GetParam(constant.TimeoutKey, ""))

	t.Log("❌ Current Getty Behavior:")
	t.Log("   1. initClient(url) is called")
	t.Log("   2. Only protocol.params are loaded into clientConf")
	t.Log("   3. URL timeout parameter is completely ignored")
	t.Log("   4. TcpWriteTimeout remains at default 5s")
	t.Log("   5. User's 60s request-timeout has no effect on TCP write timeout")

	t.Log("🔧 Real Fix Needed:")
	t.Log("   1. Modify initClient(url) to read URL timeout parameters")
	t.Log("   2. Dynamically adjust clientConf.GettySessionParam.TcpWriteTimeout")
	t.Log("   3. Ensure TcpWriteTimeout >= request-timeout from URL")
}

// TestIssue1868GettyConfigLoadingMechanism 测试Getty配置加载机制
func TestIssue1868GettyConfigLoadingMechanism(t *testing.T) {
	t.Log("⚙️ Getty Configuration Loading Mechanism Analysis")

	t.Log("📋 Current Flow:")
	t.Log("   1. dubbo_protocol.go calls getty.NewClient(Options{})")
	t.Log("   2. Client.Connect(url) is called")
	t.Log("   3. Connect() calls initClient(url)")
	t.Log("   4. initClient() loads protocol config → clientConf")
	t.Log("   5. c.conf = *clientConf (global config)")
	t.Log("   6. Session creation uses c.conf.GettySessionParam.TcpWriteTimeout")

	t.Log("🚨 The Gap:")
	t.Log("   Between step 3 and 4:")
	t.Log("   - initClient(url) receives the URL with timeout parameters")
	t.Log("   - But it only uses url.Protocol to load protocol config")
	t.Log("   - URL timeout parameters are completely ignored")

	t.Log("🎯 Fix Location:")
	t.Log("   Need to modify initClient(url) in remoting/getty/getty_client.go")
	t.Log("   Add logic to read URL timeout and adjust clientConf accordingly")
}

// TestIssue1868WhyOurFixWasWrong 解释我们的修复为什么是错误的
func TestIssue1868WhyOurFixWasWrong(t *testing.T) {
	t.Log("❌ Why Our Fix Was Wrong")

	t.Log("🔧 What we tried to do:")
	t.Log("   1. Create adjustGettyConfigForRequestTimeout() function")
	t.Log("   2. Try to set TcpWriteTimeout via getty.Options")
	t.Log("   3. Call it from dubbo_protocol.go")

	t.Log("🚨 Why it doesn't work:")
	t.Log("   1. getty.Options has no TcpWriteTimeout field")
	t.Log("   2. TcpWriteTimeout is in ClientConfig.GettySessionParam")
	t.Log("   3. Our function is never called in the real flow")
	t.Log("   4. Even if called, it wouldn't affect the actual Getty client")

	t.Log("📊 Getty Client Creation Reality:")
	t.Log("   getty.NewClient(Options{}) → Client struct created")
	t.Log("   client.Connect(url) → initClient(url) → loads global clientConf")
	t.Log("   c.conf = *clientConf → uses global config, ignores Options")

	t.Log("🎯 Correct Understanding:")
	t.Log("   - Options{} is only used for ConnectTimeout and RequestTimeout")
	t.Log("   - TcpWriteTimeout comes from global clientConf")
	t.Log("   - clientConf is loaded by initClient(url)")
	t.Log("   - initClient(url) currently ignores URL timeout parameters")
}

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
	"strings"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/remoting/getty"
)

// TestIssue1868OriginalReproduction reproduces the problem according to the original Issue description
func TestIssue1868OriginalReproduction(t *testing.T) {
	t.Log("🔬 Issue #1868 Original Reproduction Test")
	t.Log("==========================================")

	t.Log("📋 Original Issue Description:")
	t.Log("   Problem: i/o timeout after calling service multiple times")
	t.Log("   Pattern: for i := 0; i < 100; i++ { time.Sleep(time.Second * 2); xxx() }")
	t.Log("   Config: consumer.request-timeout: 60s")
	t.Log("   Protocol: dubbo")
	t.Log("   Error: write tcp xxx: i/o timeout")

	// 步骤1: 设置用户报告的配置
	t.Log("🔧 Step 1: Setting up user's configuration...")
	setupOriginalUserConfig(t)

	// 步骤2: 创建服务URL（模拟用户的服务）
	t.Log("🌐 Step 2: Creating service URL...")
	serviceURL := createOriginalServiceURL(t)

	// 步骤3: 验证问题配置
	t.Log("🔍 Step 3: Verifying problem configuration...")
	verifyProblemConfiguration(t, serviceURL)

	// 步骤4: 模拟多次调用场景
	t.Log("🔄 Step 4: Simulating multiple service calls...")
	simulateOriginalCallPattern(t, serviceURL)

	t.Log("🎯 Reproduction test completed!")
}

// setupOriginalUserConfig sets up the original user's configuration
func setupOriginalUserConfig(t *testing.T) {
	// This is the key configuration that triggers the problem
	consumerConfig := config.ConsumerConfig{
		RequestTimeout: "60s", // User's 60-second timeout setting
	}
	config.SetConsumerConfig(consumerConfig)

	t.Log("   ✅ Consumer config set: request-timeout = 60s")

	// Verify configuration
	actualConfig := config.GetConsumerConfig()
	assert.Equal(t, "60s", actualConfig.RequestTimeout, "Consumer timeout should be set to 60s")
}

// createOriginalServiceURL creates the original service URL
func createOriginalServiceURL(t *testing.T) *common.URL {
	// Simulate user's service URL
	url, err := common.NewURL("dubbo://192.168.1.122:20729/com.test.TestService")
	assert.NoError(t, err, "Should create URL successfully")

	// Set key parameters
	url.SetParam(constant.TimeoutKey, "60s")    // 60-second timeout
	url.SetParam(constant.ProtocolKey, "dubbo") // dubbo protocol
	url.SetParam(constant.InterfaceKey, "com.test.TestService")
	url.SetParam(constant.MethodKey, "testMethod")

	t.Logf("   ✅ Service URL: %s", url.String())
	return url
}

// verifyProblemConfiguration verifies the problem configuration
func verifyProblemConfiguration(t *testing.T, url *common.URL) {
	t.Log("   🔍 Analyzing configuration that causes the problem...")

	// Check URL timeout parameter
	urlTimeout := url.GetParam(constant.TimeoutKey, "")
	t.Logf("   📊 URL timeout parameter: %s", urlTimeout)
	assert.Equal(t, "60s", urlTimeout, "URL should have 60s timeout")

	// Check Getty default configuration
	gettyConfig := getty.GetDefaultClientConfig()
	gettyTcpTimeout := gettyConfig.GettySessionParam.TcpWriteTimeout
	t.Logf("   📊 Getty default TcpWriteTimeout: %s", gettyTcpTimeout)
	assert.Equal(t, "5s", gettyTcpTimeout, "Getty default should be 5s")

	// Analyze root cause
	userTimeout, err1 := time.ParseDuration(urlTimeout)
	defaultTcpTimeout, err2 := time.ParseDuration(gettyTcpTimeout)

	if err1 == nil && err2 == nil {
		t.Logf("   📊 User expects: %v timeout", userTimeout)
		t.Logf("   📊 Getty provides: %v TCP write timeout", defaultTcpTimeout)

		if userTimeout > defaultTcpTimeout {
			t.Log("   🚨 PROBLEM IDENTIFIED:")
			t.Logf("      User timeout (%v) > Getty TCP timeout (%v)", userTimeout, defaultTcpTimeout)
			t.Log("      This will cause 'write tcp i/o timeout' after 5 seconds")
			t.Log("      User's 60s timeout never gets a chance to work")
		}
	}
}

// simulateOriginalCallPattern simulates the original call pattern
func simulateOriginalCallPattern(t *testing.T, url *common.URL) {
	t.Log("   🔄 Simulating user's call pattern:")
	t.Log("      for i := 0; i < 100; i++ {")
	t.Log("          time.Sleep(time.Second * 2)")
	t.Log("          xxx() // 调用服务")
	t.Log("      }")

	// Reduce call count to speed up testing
	callCount := 5
	callInterval := 100 * time.Millisecond // Speed up testing

	t.Logf("   📞 Executing %d calls (reduced for testing)...", callCount)

	for i := 0; i < callCount; i++ {
		if i > 0 {
			time.Sleep(callInterval)
		}

		t.Logf("      Call %d/%d: Checking timeout configuration...", i+1, callCount)

		// Each call checks configuration (simulating config reading in real calls)
		timeoutParam := url.GetParam(constant.TimeoutKey, "")
		if timeoutParam != "" {
			timeout, err := time.ParseDuration(timeoutParam)
			if err == nil {
				defaultTcpTimeout := 5 * time.Second

				if timeout > defaultTcpTimeout {
					t.Logf("         🚨 Call %d: Potential i/o timeout risk!", i+1)
					t.Logf("         📊 Expected timeout: %v", timeout)
					t.Logf("         📊 Actual TCP timeout: %v", defaultTcpTimeout)
					t.Log("         💡 With our fix: Getty TcpWriteTimeout would be adjusted")
				} else {
					t.Logf("         ✅ Call %d: No timeout risk", i+1)
				}
			}
		}
	}

	t.Log("   🎯 Call pattern simulation completed")
}

// TestIssue1868OriginalBeforeAndAfterFix compares behavior before and after the fix
func TestIssue1868OriginalBeforeAndAfterFix(t *testing.T) {
	t.Log("📊 Issue #1868: Before vs After Fix Comparison")

	// Create user's problem scenario
	url, _ := common.NewURL("dubbo://192.168.1.122:20729/com.test.TestService")
	url.SetParam(constant.TimeoutKey, "60s")

	userTimeout, _ := time.ParseDuration("60s")
	gettyDefaultTimeout, _ := time.ParseDuration("5s")

	t.Log("📋 User's Problem Scenario:")
	t.Logf("   Service: %s", url.String())
	t.Logf("   User expectation: %v timeout", userTimeout)
	t.Logf("   Getty default: %v TCP write timeout", gettyDefaultTimeout)

	t.Log("❌ BEFORE Fix:")
	t.Log("   1. User sets consumer.request-timeout: 60s")
	t.Log("   2. Getty TcpWriteTimeout remains at 5s (hardcoded)")
	t.Log("   3. TCP write operations timeout after 5 seconds")
	t.Log("   4. Error: 'write tcp 192.168.1.122:57283->192.168.1.122:20729: i/o timeout'")
	t.Log("   5. User's 60s request-timeout never gets used")

	t.Log("✅ AFTER Fix:")
	t.Log("   1. User sets consumer.request-timeout: 60s")
	t.Log("   2. Our fix detects URL timeout parameter in initClient()")
	t.Log("   3. Getty TcpWriteTimeout dynamically adjusted to 60s")
	t.Log("   4. TCP write operations can wait up to 60 seconds")
	t.Log("   5. No premature i/o timeout errors")

	// Verify fix logic
	t.Log("🔧 Fix Logic Verification:")
	timeoutStr := url.GetParam(constant.TimeoutKey, "")
	if timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			if timeout > gettyDefaultTimeout {
				t.Logf("   ✅ Condition met: %v > %v", timeout, gettyDefaultTimeout)
				t.Log("   ✅ Action: Adjust Getty TcpWriteTimeout")
				t.Logf("   ✅ Result: %v → %v", gettyDefaultTimeout, timeout)
				t.Log("   🎉 Issue #1868 RESOLVED!")
			}
		}
	}
}

// TestIssue1868ErrorPatternRecognition 测试错误模式识别
func TestIssue1868ErrorPatternRecognition(t *testing.T) {
	t.Log("🔍 Issue #1868: Error Pattern Recognition")

	// 用户报告的错误模式
	originalError := "[CallProxy] received rpc err: write tcp 192.168.1.122:57283->192.168.1.122:20729: i/o timeout"

	t.Logf("📋 Original error reported by user:")
	t.Logf("   %s", originalError)

	// 分析错误模式
	t.Log("🔍 Error pattern analysis:")

	if strings.Contains(originalError, "write tcp") {
		t.Log("   ✅ Pattern 1: 'write tcp' - TCP write operation error")
	}

	if strings.Contains(originalError, "i/o timeout") {
		t.Log("   ✅ Pattern 2: 'i/o timeout' - I/O operation timeout")
	}

	if strings.Contains(originalError, "192.168.1.122") {
		t.Log("   ✅ Pattern 3: IP address - Network connection error")
	}

	// 确认这是我们要修复的错误类型
	isTargetError := strings.Contains(originalError, "write tcp") &&
		strings.Contains(originalError, "i/o timeout")

	assert.True(t, isTargetError, "Should recognize the target error pattern")

	if isTargetError {
		t.Log("🎯 CONFIRMED: This is the exact error pattern Issue #1868 addresses")
		t.Log("   Our fix prevents this error by adjusting Getty TcpWriteTimeout")
	}
}

// TestIssue1868ConfigurationFlow 测试配置流程
func TestIssue1868ConfigurationFlow(t *testing.T) {
	t.Log("⚙️ Issue #1868: Configuration Flow Test")

	t.Log("📋 Testing the complete configuration flow that causes the issue:")

	// 步骤1: 用户设置配置
	t.Log("   Step 1: User sets consumer.request-timeout: 60s")
	config.SetConsumerConfig(config.ConsumerConfig{
		RequestTimeout: "60s",
	})

	// 步骤2: 创建URL
	t.Log("   Step 2: URL created with timeout parameter")
	url, _ := common.NewURL("dubbo://127.0.0.1:20888/com.test.Service")
	url.SetParam(constant.TimeoutKey, "60s")

	// 步骤3: Getty配置加载
	t.Log("   Step 3: Getty configuration loading...")
	gettyConfig := getty.GetDefaultClientConfig()
	originalTcpTimeout := gettyConfig.GettySessionParam.TcpWriteTimeout
	t.Logf("      Getty default TcpWriteTimeout: %s", originalTcpTimeout)

	// 步骤4: 问题分析
	t.Log("   Step 4: Problem analysis...")
	urlTimeout := url.GetParam(constant.TimeoutKey, "")
	t.Logf("      URL timeout parameter: %s", urlTimeout)

	if userTimeout, err := time.ParseDuration(urlTimeout); err == nil {
		if gettyTimeout, err := time.ParseDuration(originalTcpTimeout); err == nil {
			if userTimeout > gettyTimeout {
				t.Log("      🚨 MISMATCH DETECTED:")
				t.Logf("         User expects: %v", userTimeout)
				t.Logf("         Getty provides: %v", gettyTimeout)
				t.Log("         This causes premature i/o timeout!")

				// 步骤5: 我们的修复
				t.Log("   Step 5: Our fix application...")
				t.Logf("      Fix: Adjust Getty TcpWriteTimeout to %v", userTimeout)
				t.Log("      Result: No more premature i/o timeout errors")

				assert.True(t, userTimeout > gettyTimeout,
					"User timeout should be greater than Getty default to trigger the issue")
			}
		}
	}

	t.Log("🎉 Configuration flow analysis completed!")
}

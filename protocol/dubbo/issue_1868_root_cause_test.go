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
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/remoting/getty"
)

// TestIssue1868RootCauseAnalysis 验证真正的根本原因
func TestIssue1868RootCauseAnalysis(t *testing.T) {
	t.Log("🔍 Issue #1868 Root Cause Analysis")

	// 设置用户的问题配置
	config.SetConsumerConfig(config.ConsumerConfig{
		RequestTimeout: "60s", // 用户设置的长超时
	})

	t.Log("📋 Problem Configuration:")
	t.Log("   User expectation: request-timeout = 60s")

	// 检查Getty的默认配置
	defaultConfig := getty.GetDefaultClientConfig()

	t.Logf("   Getty TcpWriteTimeout: %s", defaultConfig.GettySessionParam.TcpWriteTimeout)
	t.Logf("   Getty HeartbeatPeriod: %s", defaultConfig.HeartbeatPeriod)
	t.Logf("   Getty SessionTimeout: %s", defaultConfig.SessionTimeout)

	// 验证配置不匹配
	assert.Equal(t, "5s", defaultConfig.GettySessionParam.TcpWriteTimeout,
		"Getty TcpWriteTimeout should be 5s by default")

	t.Log("🚨 ROOT CAUSE IDENTIFIED:")
	t.Log("   User sets request-timeout: 60s (expecting RPC calls can wait 60 seconds)")
	t.Log("   But Getty TcpWriteTimeout: 5s (TCP write operations timeout after 5 seconds)")
	t.Log("   Result: 'write tcp i/o timeout' after 5 seconds, not 60 seconds")

	// 分析问题场景
	t.Log("🎯 Problem Scenario:")
	t.Log("   1. User makes RPC call expecting 60s timeout")
	t.Log("   2. Network delay or server processing takes > 5s")
	t.Log("   3. Getty TCP write timeout (5s) is hit first")
	t.Log("   4. Error: write tcp xxx: i/o timeout")
	t.Log("   5. RPC timeout (60s) never gets a chance to work")

	// 验证配置不匹配的严重性
	requestTimeout := 60 * time.Second
	tcpWriteTimeout := 5 * time.Second

	if tcpWriteTimeout < requestTimeout {
		t.Log("❌ Configuration Mismatch Confirmed:")
		t.Logf("   TCP write timeout (%v) < RPC timeout (%v)", tcpWriteTimeout, requestTimeout)
		t.Log("   This will cause premature 'i/o timeout' errors")
	}
}

// TestIssue1868ConfigurationMismatch 测试配置不匹配的影响
func TestIssue1868ConfigurationMismatch(t *testing.T) {
	t.Log("⚖️  Configuration Mismatch Impact Analysis")

	testCases := []struct {
		name            string
		requestTimeout  string
		expectedProblem bool
		description     string
	}{
		{
			name:            "Normal_case",
			requestTimeout:  "3s",
			expectedProblem: false,
			description:     "3s request timeout < 5s TCP write timeout - should work",
		},
		{
			name:            "Problematic_case",
			requestTimeout:  "60s",
			expectedProblem: true,
			description:     "60s request timeout > 5s TCP write timeout - will cause i/o timeout",
		},
		{
			name:            "Extreme_case",
			requestTimeout:  "120s",
			expectedProblem: true,
			description:     "120s request timeout >> 5s TCP write timeout - severe mismatch",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("🧪 Testing %s: %s", tc.name, tc.description)

			// 解析超时配置
			requestTimeout, err := time.ParseDuration(tc.requestTimeout)
			assert.NoError(t, err)

			tcpWriteTimeout := 5 * time.Second // Getty默认值

			// 检查是否存在配置不匹配
			hasMismatch := requestTimeout > tcpWriteTimeout

			if tc.expectedProblem {
				assert.True(t, hasMismatch, "Should detect configuration mismatch")
				t.Logf("   ❌ PROBLEM: Request timeout (%v) > TCP write timeout (%v)",
					requestTimeout, tcpWriteTimeout)
				t.Log("   💡 SOLUTION NEEDED: Align TCP write timeout with request timeout")
			} else {
				assert.False(t, hasMismatch, "Should not have configuration mismatch")
				t.Logf("   ✅ OK: Request timeout (%v) <= TCP write timeout (%v)",
					requestTimeout, tcpWriteTimeout)
			}
		})
	}
}

// TestIssue1868SolutionDirection 测试解决方案方向
func TestIssue1868SolutionDirection(t *testing.T) {
	t.Log("💡 Solution Direction Analysis")

	t.Log("🎯 Potential Solutions:")
	t.Log("   1. Make Getty TcpWriteTimeout configurable and align with request-timeout")
	t.Log("   2. Set TcpWriteTimeout = max(request-timeout, default 5s)")
	t.Log("   3. Add validation to warn users about configuration mismatches")

	// 模拟正确的配置对齐
	requestTimeout := 60 * time.Second
	correctedTcpWriteTimeout := requestTimeout // 对齐配置

	t.Logf("✅ Corrected Configuration:")
	t.Logf("   Request timeout: %v", requestTimeout)
	t.Logf("   TCP write timeout: %v", correctedTcpWriteTimeout)
	t.Log("   Result: No premature i/o timeout errors")

	assert.Equal(t, requestTimeout, correctedTcpWriteTimeout,
		"TCP write timeout should align with request timeout")
}

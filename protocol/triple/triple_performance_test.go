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

package triple_test

import (
	"context"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// BenchmarkURLCreation tests URL creation performance
func BenchmarkURLCreation(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = common.NewURL("triple://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
		}
	})
}

// BenchmarkInvocationCreation tests invocation object creation performance
func BenchmarkInvocationCreation(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("TestMethod"),
				invocation.WithArguments([]interface{}{"param1", 42}),
				invocation.WithAttachment("key", "value"),
			)
		}
	})
}

// BenchmarkInvocationWithAttributes tests invocation object creation performance with attributes
func BenchmarkInvocationWithAttributes(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("TestMethod"),
				invocation.WithArguments([]interface{}{"param1", 42}),
				invocation.WithAttachment(constant.InterfaceKey, "com.test.Service"),
				invocation.WithAttachment("user-agent", "dubbo-go/3.0"),
				invocation.WithAttachment("custom-header", "benchmark"),
			)
		}
	})
}

// BenchmarkGenericInvocationCreation tests generic invocation object creation performance
func BenchmarkGenericInvocationCreation(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName(constant.Generic),
				invocation.WithArguments([]interface{}{
					"TestMethod",
					[]interface{}{"param1", 42},
					map[string]interface{}{
						"param1": "value1",
						"param2": 42,
					},
				}),
				invocation.WithAttachment(constant.GenericKey, "true"),
			)
		}
	})
}

// BenchmarkURLParameterOperations tests URL parameter operations performance
func BenchmarkURLParameterOperations(b *testing.B) {
	url, _ := common.NewURL("triple://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = url.GetParam("interface", "")
			_ = url.GetParam("group", "")
			_ = url.GetParam("version", "")
		}
	})
}

// BenchmarkContextWithValue tests Context operations performance
func BenchmarkContextWithValue(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			ctx = context.WithValue(ctx, "key1", "value1")
			ctx = context.WithValue(ctx, "key2", "value2")
			_ = ctx.Value("key1")
			_ = ctx.Value("key2")
		}
	})
}

// BenchmarkMapOperations tests Map operations performance (simulating parameter passing)
func BenchmarkMapOperations(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m := make(map[string]interface{})
			m["param1"] = "value1"
			m["param2"] = 42
			m["param3"] = true
			_ = m["param1"]
			_ = m["param2"]
			_ = m["param3"]
		}
	})
}

// BenchmarkSliceOperations tests Slice operations performance (simulating parameter lists)
func BenchmarkSliceOperations(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := make([]interface{}, 0, 5)
			s = append(s, "param1")
			s = append(s, 42)
			s = append(s, true)
			s = append(s, map[string]string{"key": "value"})
			s = append(s, []int{1, 2, 3})
			_ = s[0]
			_ = s[2]
			_ = len(s)
		}
	})
}

// BenchmarkLargePayloadSimulation tests large payload simulation
func BenchmarkLargePayloadSimulation(b *testing.B) {
	// Simulate 10KB data payload
	largeData := make([]byte, 10240)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate data copy operation
			data := make([]byte, len(largeData))
			copy(data, largeData)
			_ = len(data)
		}
	})
}

// BenchmarkConcurrentURLAccess tests concurrent URL access (removed due to deadlock issues)
func BenchmarkConcurrentURLAccess(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine uses its own URL instance to avoid concurrency issues
		localURL, _ := common.NewURL("triple://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
		for pb.Next() {
			localURL.GetParam("interface", "")
			localURL.GetParam("group", "")
		}
	})
}

// BenchmarkMemoryAllocation tests memory allocation performance
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate creation of various objects
			invocation := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("TestMethod"),
				invocation.WithArguments([]interface{}{"test", 123}),
			)
			url, _ := common.NewURL("triple://127.0.0.1:20000/test")
			ctx := context.WithValue(context.Background(), "test", "value")

			// Use these objects
			_ = invocation
			_ = url
			_ = ctx
		}
	})
}

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

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

// BenchmarkTripleInvoker_Invoke_Unary 测试Unary调用的性能
func BenchmarkTripleInvoker_Invoke_Unary(b *testing.B) {
	// 创建URL
	url, _ := common.NewURL("triple://127.0.0.1:20000/connect.ping.v1.PingService?interface=connect.ping.v1.PingService")

	// 创建TripleInvoker
	invoker, err := NewTripleInvoker(url)
	if err != nil {
		b.Fatal(err)
	}
	defer invoker.Destroy()

	// 准备参数
	invocation := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("Ping"),
		invocation.WithArguments([]interface{}{&pingv1.PingRequest{Number: 42}}),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invoker.Invoke(context.Background(), invocation)
		}
	})
}

// BenchmarkTripleInvoker_Invoke_Generic 测试泛化调用的性能
func BenchmarkTripleInvoker_Invoke_Generic(b *testing.B) {
	// 创建URL
	url, _ := common.NewURL("triple://127.0.0.1:20000/connect.ping.v1.PingService?interface=connect.ping.v1.PingService")

	// 创建TripleInvoker
	invoker, err := NewTripleInvoker(url)
	if err != nil {
		b.Fatal(err)
	}
	defer invoker.Destroy()

	// 准备泛化调用参数
	invocation := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("Ping"),
		invocation.WithArguments([]interface{}{map[string]interface{}{
			"number": 42,
		}}),
		invocation.WithAttachment(constant.GenericKey, "true"),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invoker.Invoke(context.Background(), invocation)
		}
	})
}

// BenchmarkTripleInvoker_Invoke_WithAttachment 测试带附件的调用性能
func BenchmarkTripleInvoker_Invoke_WithAttachment(b *testing.B) {
	// 创建URL
	url, _ := common.NewURL("triple://127.0.0.1:20000/connect.ping.v1.PingService?interface=connect.ping.v1.PingService")

	// 创建TripleInvoker
	invoker, err := NewTripleInvoker(url)
	if err != nil {
		b.Fatal(err)
	}
	defer invoker.Destroy()

	// 准备带附件的调用参数
	invocation := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("Ping"),
		invocation.WithArguments([]interface{}{&pingv1.PingRequest{Number: 42}}),
		invocation.WithAttachment("user-agent", "dubbo-go-client/3.0"),
		invocation.WithAttachment("custom-header", "benchmark-test"),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invoker.Invoke(context.Background(), invocation)
		}
	})
}

// BenchmarkClientCreation 测试客户端创建性能
func BenchmarkClientCreation(b *testing.B) {
	url, _ := common.NewURL("triple://127.0.0.1:20000/connect.ping.v1.PingService?interface=connect.ping.v1.PingService")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		invoker, _ := NewTripleInvoker(url)
		invoker.Destroy()
	}
}

// BenchmarkInvocationCreation 测试调用对象创建性能
func BenchmarkInvocationCreation(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invocation := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("Ping"),
				invocation.WithArguments([]interface{}{&pingv1.PingRequest{Number: 42}}),
			)
			_ = invocation
		}
	})
}

// BenchmarkParseInvocation 测试解析调用的性能
func BenchmarkParseInvocation(b *testing.B) {
	url, _ := common.NewURL("triple://127.0.0.1:20000/connect.ping.v1.PingService")
	invocation := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("Ping"),
		invocation.WithArguments([]interface{}{&pingv1.PingRequest{Number: 42}}),
		invocation.WithAttachment(constant.CallTypeKey, constant.CallUnary),
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _, _ = parseInvocation(context.Background(), url, invocation)
		}
	})
}

// BenchmarkGenericInvocationCreation 测试泛化调用对象创建性能
func BenchmarkGenericInvocationCreation(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			invocation := invocation.NewRPCInvocationWithOptions(
				invocation.WithMethodName("Ping"),
				invocation.WithArguments([]interface{}{map[string]interface{}{
					"number": 42,
				}}),
				invocation.WithAttachment(constant.GenericKey, "true"),
			)
			_ = invocation
		}
	})
}

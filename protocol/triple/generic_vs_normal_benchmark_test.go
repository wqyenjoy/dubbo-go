/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the \"License\"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an \"AS IS\" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package triple_test

import (
	"context"
	"encoding/json"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// BenchmarkComparison compares the performance of normal calls vs generic calls
func BenchmarkComparison(b *testing.B) {
	b.Run("Normal_Call", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				invocation.NewRPCInvocationWithOptions(
					invocation.WithMethodName("GetUser"),
					invocation.WithArguments([]interface{}{"user123", 25}),
					invocation.WithAttachment("traceId", "abc123"),
				)
			}
		})
	})

	b.Run("Generic_Call", func(b *testing.B) {
		// Prepare generic call parameters
		genericArgs := map[string]interface{}{
			"method": "GetUser",
			"types":  []string{"java.lang.String", "int"},
			"args":   []interface{}{"user123", 25},
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				invocation.NewRPCInvocationWithOptions(
					invocation.WithMethodName(constant.Generic),
					invocation.WithArguments([]interface{}{genericArgs}),
					invocation.WithAttachment("traceId", "abc123"),
					invocation.WithAttachment(constant.GenericKey, "true"),
				)
			}
		})
	})

	b.Run("Generic_Call_With_JSON", func(b *testing.B) {
		// Simulate generic call with JSON serialization
		genericArgs := map[string]interface{}{
			"method": "GetUser",
			"types":  []string{"java.lang.String", "int"},
			"args":   []interface{}{"user123", 25},
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Serialize parameters (typically required for generic calls)
				jsonData, _ := json.Marshal(genericArgs)

				invocation.NewRPCInvocationWithOptions(
					invocation.WithMethodName(constant.Generic),
					invocation.WithArguments([]interface{}{string(jsonData)}),
					invocation.WithAttachment("traceId", "abc123"),
					invocation.WithAttachment(constant.GenericKey, "true"),
				)
			}
		})
	})
}

// BenchmarkGenericOverhead analyzes the additional overhead of generic calls
func BenchmarkGenericOverhead(b *testing.B) {
	b.Run("Parameter_Marshalling", func(b *testing.B) {
		// Test the overhead of parameter serialization
		args := []interface{}{"user123", 25, map[string]interface{}{
			"name": "John",
			"age":  30,
			"tags": []string{"vip", "active"},
		}}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			json.Marshal(args)
		}
	})

	b.Run("Generic_Parameter_Building", func(b *testing.B) {
		// Test the overhead of building generic call parameters
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			genericArgs := map[string]interface{}{
				"method": "GetUser",
				"types":  []string{"java.lang.String", "int", "java.util.Map"},
				"args": []interface{}{
					"user123",
					25,
					map[string]interface{}{
						"name": "John",
						"age":  30,
						"tags": []string{"vip", "active"},
					},
				},
			}
			_ = genericArgs
		}
	})

	b.Run("Context_Propagation", func(b *testing.B) {
		// Test the overhead of context propagation
		ctx := context.Background()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			localCtx := ctx
			for pb.Next() {
				localCtx = context.WithValue(localCtx, "generic_call", "true")
				localCtx = context.WithValue(localCtx, "method", "GetUser")
				localCtx = context.WithValue(localCtx, "trace_id", "abc123")
				_ = localCtx.Value("generic_call")
			}
		})
	})
}

// BenchmarkComplexGenericCall tests complex generic call scenarios
func BenchmarkComplexGenericCall(b *testing.B) {
	b.Run("Complex_Object_Generic", func(b *testing.B) {
		// Complex generic call with nested objects
		complexArgs := map[string]interface{}{
			"method": "CreateOrder",
			"types":  []string{"com.example.Order"},
			"args": []interface{}{
				map[string]interface{}{
					"id":     "order123",
					"userId": "user456",
					"amount": 99.99,
					"items":  []interface{}{"item1", "item2", "item3"},
					"metadata": map[string]interface{}{
						"source":    "mobile_app",
						"version":   "2.1.0",
						"timestamp": 1640995200,
						"features": map[string]interface{}{
							"express": true,
							"insured": false,
						},
					},
				},
			},
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				invocation.NewRPCInvocationWithOptions(
					invocation.WithMethodName(constant.Generic),
					invocation.WithArguments([]interface{}{complexArgs}),
					invocation.WithAttachment("traceId", "complex123"),
					invocation.WithAttachment(constant.GenericKey, "true"),
				)
			}
		})
	})
}

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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// 协议转换器使用演示
// 此示例展示了如何使用协议转换器进行不同场景的转换

func main() {
	fmt.Println("🌉 Triple协议转换器使用演示")
	fmt.Println("========================================")

	// 等待确保转换器服务已启动
	fmt.Println("⏳ 检查协议转换器服务状态...")
	if !checkConverterService() {
		fmt.Println("❌ 协议转换器服务未运行")
		fmt.Println("请先启动转换器:")
		fmt.Println("  cd examples/triple_generic_demo/protocol_converter")
		fmt.Println("  go run converter.go")
		return
	}
	fmt.Println("✅ 协议转换器服务运行正常")

	// 演示场景1: 序列化格式转换
	demonstrateSerializationConversion()

	// 演示场景2: 协议转换 (RPC to HTTP)
	demonstrateProtocolConversion()

	// 演示场景3: 批量转换请求
	demonstrateBatchConversion()

	// 演示场景4: 实际业务场景模拟
	demonstrateBusinessScenarios()

	// 演示场景5: 性能测试
	demonstratePerformanceTest()

	fmt.Println("\n🎉 协议转换器演示完成!")
	fmt.Println("详细文档请参考: protocol_converter/README.md")
}

// 检查协议转换器服务是否运行
func checkConverterService() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	
	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	return resp.StatusCode == http.StatusOK
}

// 发送转换请求的通用函数
func sendConvertRequest(request map[string]interface{}) (*ConvertResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	
	resp, err := client.Post("http://localhost:8080/convert", 
		"application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	var convertResp ConvertResponse
	if err := json.NewDecoder(resp.Body).Decode(&convertResp); err != nil {
		return nil, fmt.Errorf("decode response failed: %v", err)
	}

	return &convertResp, nil
}

// 演示场景1: 序列化格式转换
func demonstrateSerializationConversion() {
	fmt.Println("\n📊 演示场景1: 序列化格式转换")
	fmt.Println("----------------------------------------")

	scenarios := []struct {
		name   string
		source string
		target string
		method string
		types  []string
		args   []interface{}
	}{
		{
			name:   "Hessian2 → JSON",
			source: "hessian2",
			target: "json",
			method: "Hello",
			types:  []string{"java.lang.String"},
			args:   []interface{}{"格式转换测试"},
		},
		{
			name:   "JSON → Hessian2",
			source: "json",
			target: "hessian2",
			method: "Add",
			types:  []string{"int", "int"},
			args:   []interface{}{42, 58},
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\n🔄 %s:\n", scenario.name)
		
		request := map[string]interface{}{
			"source_protocol":      "triple",
			"target_protocol":      "triple",
			"source_serialization": scenario.source,
			"target_serialization": scenario.target,
			"method_name":          scenario.method,
			"param_types":          scenario.types,
			"args":                 scenario.args,
		}

		resp, err := sendConvertRequest(request)
		if err != nil {
			fmt.Printf("  ❌ 请求失败: %v\n", err)
			continue
		}

		if resp.Success {
			fmt.Printf("  ✅ 转换成功, 耗时: %s\n", resp.Duration)
			fmt.Printf("  📄 结果: %v\n", resp.Result)
		} else {
			fmt.Printf("  ❌ 转换失败: %s\n", resp.Error)
		}
	}
}

// 演示场景2: 协议转换
func demonstrateProtocolConversion() {
	fmt.Println("\n🔄 演示场景2: 协议转换")
	fmt.Println("----------------------------------------")

	scenarios := []struct {
		name           string
		sourceProtocol string
		targetProtocol string
		method         string
		types          []string
		args           []interface{}
		description    string
	}{
		{
			name:           "Triple → HTTP",
			sourceProtocol: "triple",
			targetProtocol: "http",
			method:         "GetUserInfo",
			types:          []string{"java.lang.String"},
			args:           []interface{}{"protocol_user"},
			description:    "将RPC调用转换为HTTP REST调用",
		},
		{
			name:           "Triple → gRPC",
			sourceProtocol: "triple",
			targetProtocol: "grpc",
			method:         "Hello",
			types:          []string{"java.lang.String"},
			args:           []interface{}{"gRPC兼容性测试"},
			description:    "Triple协议与gRPC的兼容性演示",
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\n🌉 %s:\n", scenario.name)
		fmt.Printf("   %s\n", scenario.description)
		
		request := map[string]interface{}{
			"source_protocol":      scenario.sourceProtocol,
			"target_protocol":      scenario.targetProtocol,
			"source_serialization": "hessian2",
			"target_serialization": "json",
			"method_name":          scenario.method,
			"param_types":          scenario.types,
			"args":                 scenario.args,
		}

		resp, err := sendConvertRequest(request)
		if err != nil {
			fmt.Printf("  ❌ 请求失败: %v\n", err)
			continue
		}

		if resp.Success {
			fmt.Printf("  ✅ 协议转换成功, 耗时: %s\n", resp.Duration)
			fmt.Printf("  📄 结果类型: %T\n", resp.Result)
			if m, ok := resp.Result.(map[string]interface{}); ok {
				for k, v := range m {
					fmt.Printf("      %s: %v\n", k, v)
				}
			} else {
				fmt.Printf("  📄 结果内容: %v\n", resp.Result)
			}
		} else {
			fmt.Printf("  ❌ 协议转换失败: %s\n", resp.Error)
		}
	}
}

// 演示场景3: 批量转换请求
func demonstrateBatchConversion() {
	fmt.Println("\n📦 演示场景3: 批量转换请求")
	fmt.Println("----------------------------------------")

	fmt.Println("同时发送多个不同类型的转换请求...")
	
	requests := []map[string]interface{}{
		{
			"source_protocol":      "triple",
			"target_protocol":      "triple",
			"source_serialization": "hessian2",
			"target_serialization": "json",
			"method_name":          "Hello",
			"param_types":          []string{"java.lang.String"},
			"args":                 []interface{}{"批量测试1"},
		},
		{
			"source_protocol":      "triple",
			"target_protocol":      "triple",
			"source_serialization": "json",
			"target_serialization": "hessian2",
			"method_name":          "Add",
			"param_types":          []string{"int", "int"},
			"args":                 []interface{}{10, 20},
		},
		{
			"source_protocol":      "triple",
			"target_protocol":      "http",
			"source_serialization": "hessian2",
			"target_serialization": "json",
			"method_name":          "GetUserInfo",
			"param_types":          []string{"java.lang.String"},
			"args":                 []interface{}{"批量用户"},
		},
	}

	start := time.Now()
	results := make(chan string, len(requests))

	// 并发发送请求
	for i, request := range requests {
		go func(index int, req map[string]interface{}) {
			resp, err := sendConvertRequest(req)
			if err != nil {
				results <- fmt.Sprintf("请求%d: 失败 - %v", index+1, err)
			} else if resp.Success {
				results <- fmt.Sprintf("请求%d: 成功 - 耗时%s", index+1, resp.Duration)
			} else {
				results <- fmt.Sprintf("请求%d: 失败 - %s", index+1, resp.Error)
			}
		}(i, request)
	}

	// 收集结果
	for i := 0; i < len(requests); i++ {
		result := <-results
		fmt.Printf("  %s\n", result)
	}

	totalTime := time.Since(start)
	fmt.Printf("\n📊 批量请求完成统计:\n")
	fmt.Printf("  总请求数: %d\n", len(requests))
	fmt.Printf("  总耗时: %v\n", totalTime)
	fmt.Printf("  平均耗时: %v\n", totalTime/time.Duration(len(requests)))
}

// 演示场景4: 实际业务场景模拟
func demonstrateBusinessScenarios() {
	fmt.Println("\n🏢 演示场景4: 实际业务场景模拟")
	fmt.Println("----------------------------------------")

	scenarios := []struct {
		title       string
		description string
		request     map[string]interface{}
	}{
		{
			title:       "API网关场景",
			description: "移动App通过HTTP调用后端Triple服务",
			request: map[string]interface{}{
				"source_protocol":      "triple",
				"target_protocol":      "http",
				"source_serialization": "hessian2",
				"target_serialization": "json",
				"method_name":          "GetUserInfo",
				"param_types":          []string{"java.lang.String"},
				"args":                 []interface{}{"mobile_user_12345"},
			},
		},
		{
			title:       "微服务集成场景",
			description: "不同序列化格式的服务间调用",
			request: map[string]interface{}{
				"source_protocol":      "triple",
				"target_protocol":      "triple",
				"source_serialization": "json",
				"target_serialization": "hessian2",
				"method_name":          "ComplexOperation",
				"param_types":          []string{"java.util.Map"},
				"args": []interface{}{
					map[string]interface{}{
						"service":  "order-service",
						"operation": "process_order",
						"data": map[string]interface{}{
							"order_id": "ORD-2024-001",
							"items": []interface{}{
								map[string]interface{}{"id": "item1", "qty": 2},
								map[string]interface{}{"id": "item2", "qty": 1},
							},
							"total_amount": 299.99,
						},
					},
				},
			},
		},
		{
			title:       "协议升级场景",
			description: "从传统协议迁移到gRPC兼容协议",
			request: map[string]interface{}{
				"source_protocol":      "triple",
				"target_protocol":      "grpc",
				"source_serialization": "hessian2",
				"target_serialization": "json",
				"method_name":          "ProcessList",
				"param_types":          []string{"java.util.List"},
				"args": []interface{}{
					[]interface{}{
						"legacy_system_data_1",
						"legacy_system_data_2",
						"legacy_system_data_3",
					},
				},
			},
		},
	}

	for _, scenario := range scenarios {
		fmt.Printf("\n🎯 %s:\n", scenario.title)
		fmt.Printf("   场景: %s\n", scenario.description)
		
		start := time.Now()
		resp, err := sendConvertRequest(scenario.request)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("  ❌ 场景执行失败: %v\n", err)
			continue
		}

		if resp.Success {
			fmt.Printf("  ✅ 场景执行成功\n")
			fmt.Printf("  ⏱️  总耗时: %v (转换器内部: %s)\n", elapsed, resp.Duration)
			fmt.Printf("  📊 转换路径: %s(%s) → %s(%s)\n",
				resp.SourceProtocol, resp.SourceSerialization,
				resp.TargetProtocol, resp.TargetSerialization)
			
			// 只显示结果的摘要，避免输出过长
			if resultMap, ok := resp.Result.(map[string]interface{}); ok {
				fmt.Printf("  📄 结果摘要: %d个字段\n", len(resultMap))
			} else {
				fmt.Printf("  📄 结果类型: %T\n", resp.Result)
			}
		} else {
			fmt.Printf("  ❌ 场景执行失败: %s\n", resp.Error)
		}
	}
}

// 演示场景5: 性能测试
func demonstratePerformanceTest() {
	fmt.Println("\n⚡ 演示场景5: 性能测试")
	fmt.Println("----------------------------------------")

	testRequest := map[string]interface{}{
		"source_protocol":      "triple",
		"target_protocol":      "triple",
		"source_serialization": "hessian2",
		"target_serialization": "json",
		"method_name":          "Hello",
		"param_types":          []string{"java.lang.String"},
		"args":                 []interface{}{"性能测试"},
	}

	// 热身阶段
	fmt.Println("🔥 执行热身请求...")
	for i := 0; i < 5; i++ {
		sendConvertRequest(testRequest)
	}

	// 性能测试
	fmt.Println("\n📊 开始性能测试 (50次请求)...")
	
	const numRequests = 50
	results := make([]time.Duration, 0, numRequests)
	successCount := 0
	
	overallStart := time.Now()
	
	for i := 0; i < numRequests; i++ {
		start := time.Now()
		resp, err := sendConvertRequest(testRequest)
		requestTime := time.Since(start)
		
		results = append(results, requestTime)
		
		if err == nil && resp.Success {
			successCount++
		}
		
		if i%10 == 9 { // 每10个请求显示一次进度
			fmt.Printf("  完成 %d/%d 请求...\n", i+1, numRequests)
		}
	}
	
	overallTime := time.Since(overallStart)

	// 计算统计数据
	var totalTime time.Duration
	minTime := results[0]
	maxTime := results[0]
	
	for _, t := range results {
		totalTime += t
		if t < minTime {
			minTime = t
		}
		if t > maxTime {
			maxTime = t
		}
	}
	
	avgTime := totalTime / time.Duration(len(results))
	throughput := float64(numRequests) / overallTime.Seconds()

	fmt.Printf("\n📈 性能测试结果:\n")
	fmt.Printf("  总请求数: %d\n", numRequests)
	fmt.Printf("  成功请求: %d (成功率: %.1f%%)\n", successCount, 
		float64(successCount)/float64(numRequests)*100)
	fmt.Printf("  总耗时: %v\n", overallTime)
	fmt.Printf("  平均延迟: %v\n", avgTime)
	fmt.Printf("  最小延迟: %v\n", minTime)
	fmt.Printf("  最大延迟: %v\n", maxTime)
	fmt.Printf("  吞吐量: %.2f 请求/秒\n", throughput)

	// 性能评级
	if avgTime < 50*time.Millisecond {
		fmt.Printf("  🏆 性能评级: 优秀\n")
	} else if avgTime < 100*time.Millisecond {
		fmt.Printf("  👍 性能评级: 良好\n")
	} else if avgTime < 200*time.Millisecond {
		fmt.Printf("  👌 性能评级: 一般\n")
	} else {
		fmt.Printf("  ⚠️  性能评级: 需要优化\n")
	}
}

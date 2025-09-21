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
	"log"
	"net/http"
	"time"
)

// Protocol Converter Usage Demo
// This example demonstrates how to use the protocol converter for different conversion scenarios

func main() {
	log.Println(" Triple Protocol Converter Usage Demo")
	log.Println("")

	// Wait to ensure the converter service has started
	log.Println(" Checking Protocol Converter service status...")
	if !checkConverterService() {
		log.Println(" Protocol Converter service is not running")
		log.Println("Please start the converter first:")
		log.Println("  cd examples/triple_generic_demo/protocol_converter")
		log.Println("  go run converter.go")
		return
	}
	log.Println(" Protocol Converter service is running normally")

	// Demo Scenario 1: Serialization format conversion
	demonstrateSerializationConversion()

	// Demo Scenario 2: Protocol conversion (RPC to HTTP)
	demonstrateProtocolConversion()

	// Demo Scenario 3: Batch conversion requests
	demonstrateBatchConversion()

	// Demo Scenario 4: Real business scenario simulation
	demonstrateBusinessScenarios()

	// Demo Scenario 5: Performance testing
	demonstratePerformanceTest()

	log.Println("\n Protocol Converter demo completed!")
	log.Println("For detailed documentation, please refer to: protocol_converter/README.md")
}

// checkConverterService checks if the protocol converter service is running
func checkConverterService() bool {
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get("http://localhost:8080/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// sendConvertRequest is a generic function to send conversion requests
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

// demonstrateSerializationConversion demonstrates serialization format conversion
func demonstrateSerializationConversion() {
	log.Println("\n Demo Scenario 1: Serialization Format Conversion")
	log.Println("")

	scenarios := []struct {
		name   string
		source string
		target string
		method string
		types  []string
		args   []interface{}
	}{
		{
			name:   "Hessian2 -> JSON",
			source: "hessian2",
			target: "json",
			method: "Hello",
			types:  []string{"java.lang.String"},
			args:   []interface{}{"Format Conversion Test"},
		},
		{
			name:   "JSON -> Hessian2",
			source: "json",
			target: "hessian2",
			method: "Add",
			types:  []string{"int", "int"},
			args:   []interface{}{42, 58},
		},
	}

	for _, scenario := range scenarios {
		log.Printf("\n %s:\n", scenario.name)

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
			log.Printf("   Request failed: %v\n", err)
			continue
		}

		if resp.Success {
			log.Printf("   Conversion successful, duration: %s\n", resp.Duration)
			log.Printf("  📄 Result: %v\n", resp.Result)
		} else {
			log.Printf("   Conversion failed: %s\n", resp.Error)
		}
	}
}

// demonstrateProtocolConversion demonstrates protocol conversion
func demonstrateProtocolConversion() {
	log.Println("\n Demo Scenario 2: Protocol Conversion")
	log.Println("")

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
			name:           "Triple -> HTTP",
			sourceProtocol: "triple",
			targetProtocol: "http",
			method:         "GetUserInfo",
			types:          []string{"java.lang.String"},
			args:           []interface{}{"protocol_user"},
			description:    "Convert RPC calls to HTTP REST calls",
		},
		{
			name:           "Triple -> gRPC",
			sourceProtocol: "triple",
			targetProtocol: "grpc",
			method:         "Hello",
			types:          []string{"java.lang.String"},
			args:           []interface{}{"gRPC Compatibility Test"},
			description:    "Demonstration of Triple protocol compatibility with gRPC",
		},
	}

	for _, scenario := range scenarios {
		log.Printf("\n %s:\n", scenario.name)
		log.Printf("   %s\n", scenario.description)

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
			log.Printf("   Request failed: %v\n", err)
			continue
		}

		if resp.Success {
			log.Printf("   Protocol conversion successful, duration: %s\n", resp.Duration)
			log.Printf("  📄 Result type: %T\n", resp.Result)
			if m, ok := resp.Result.(map[string]interface{}); ok {
				for k, v := range m {
					log.Printf("      %s: %v\n", k, v)
				}
			} else {
				log.Printf("  📄 Result content: %v\n", resp.Result)
			}
		} else {
			log.Printf("   Protocol conversion failed: %s\n", resp.Error)
		}
	}
}

// demonstrateBatchConversion demonstrates batch conversion requests
func demonstrateBatchConversion() {
	log.Println("\n📦 Demo Scenario 3: Batch Conversion Requests")
	log.Println("")

	log.Println("Sending multiple different types of conversion requests simultaneously...")

	requests := []map[string]interface{}{
		{
			"source_protocol":      "triple",
			"target_protocol":      "triple",
			"source_serialization": "hessian2",
			"target_serialization": "json",
			"method_name":          "Hello",
			"param_types":          []string{"java.lang.String"},
			"args":                 []interface{}{"Batch Test 1"},
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
			"args":                 []interface{}{"Batch User"},
		},
	}

	start := time.Now()
	results := make(chan string, len(requests))

	// Send requests concurrently
	for i, request := range requests {
		go func(index int, req map[string]interface{}) {
			resp, err := sendConvertRequest(req)
			if err != nil {
				results <- fmt.Sprintf("Request %d: Failed - %v", index+1, err)
			} else if resp.Success {
				results <- fmt.Sprintf("Request %d: Success - Duration %s", index+1, resp.Duration)
			} else {
				results <- fmt.Sprintf("Request %d: Failed - %s", index+1, resp.Error)
			}
		}(i, request)
	}

	// Collect results
	for i := 0; i < len(requests); i++ {
		result := <-results
		log.Printf("  %s\n", result)
	}

	totalTime := time.Since(start)
	log.Printf("\n Batch Request Completion Statistics:\n")
	log.Printf("  Total requests: %d\n", len(requests))
	log.Printf("  Total time: %v\n", totalTime)
	log.Printf("  Average time: %v\n", totalTime/time.Duration(len(requests)))
}

// demonstrateBusinessScenarios demonstrates real business scenario simulation
func demonstrateBusinessScenarios() {
	log.Println("\n🏢 Demo Scenario 4: Real Business Scenario Simulation")
	log.Println("")

	scenarios := []struct {
		title       string
		description string
		request     map[string]interface{}
	}{
		{
			title:       "API Gateway Scenario",
			description: "Mobile App calls backend Triple service via HTTP",
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
			title:       "Microservice Integration Scenario",
			description: "Inter-service calls with different serialization formats",
			request: map[string]interface{}{
				"source_protocol":      "triple",
				"target_protocol":      "triple",
				"source_serialization": "json",
				"target_serialization": "hessian2",
				"method_name":          "ComplexOperation",
				"param_types":          []string{"java.util.Map"},
				"args": []interface{}{
					map[string]interface{}{
						"service":   "order-service",
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
			title:       "Protocol Upgrade Scenario",
			description: "Migration from legacy protocols to gRPC-compatible protocols",
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
		log.Printf("\n🎯 %s:\n", scenario.title)
		log.Printf("   Scenario: %s\n", scenario.description)

		start := time.Now()
		resp, err := sendConvertRequest(scenario.request)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("   Scenario execution failed: %v\n", err)
			continue
		}

		if resp.Success {
			log.Printf("   Scenario execution successful\n")
			log.Printf("    Total time: %v (Converter internal: %s)\n", elapsed, resp.Duration)
			log.Printf("   Conversion path: %s(%s) -> %s(%s)\n",
				resp.SourceProtocol, resp.SourceSerialization,
				resp.TargetProtocol, resp.TargetSerialization)

			// Only show result summary to avoid too long output
			if resultMap, ok := resp.Result.(map[string]interface{}); ok {
				log.Printf("  📄 Result summary: %d fields\n", len(resultMap))
			} else {
				log.Printf("  📄 Result type: %T\n", resp.Result)
			}
		} else {
			log.Printf("   Scenario execution failed: %s\n", resp.Error)
		}
	}
}

// demonstratePerformanceTest demonstrates performance testing
func demonstratePerformanceTest() {
	log.Println("\n⚡ Demo Scenario 5: Performance Testing")
	log.Println("")

	testRequest := map[string]interface{}{
		"source_protocol":      "triple",
		"target_protocol":      "triple",
		"source_serialization": "hessian2",
		"target_serialization": "json",
		"method_name":          "Hello",
		"param_types":          []string{"java.lang.String"},
		"args":                 []interface{}{"Performance Test"},
	}

	// Warm-up phase
	log.Println("🔥 Executing warm-up requests...")
	for i := 0; i < 5; i++ {
		sendConvertRequest(testRequest)
	}

	// Performance test
	log.Println("\n Starting performance test (50 requests)...")

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

		if i%10 == 9 { // Show progress every 10 requests
			log.Printf("  Completed %d/%d requests...\n", i+1, numRequests)
		}
	}

	overallTime := time.Since(overallStart)

	// Calculate statistics
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

	log.Printf("\n📈 Performance Test Results:\n")
	log.Printf("  Total requests: %d\n", numRequests)
	log.Printf("  Successful requests: %d (Success rate: %.1f%%)\n", successCount,
		float64(successCount)/float64(numRequests)*100)
	log.Printf("  Total time: %v\n", overallTime)
	log.Printf("  Average latency: %v\n", avgTime)
	log.Printf("  Minimum latency: %v\n", minTime)
	log.Printf("  Maximum latency: %v\n", maxTime)
	log.Printf("  Throughput: %.2f requests/sec\n", throughput)

	// Performance rating
	if avgTime < 50*time.Millisecond {
		log.Printf("  🏆 Performance rating: Excellent\n")
	} else if avgTime < 100*time.Millisecond {
		log.Printf("  👍 Performance rating: Good\n")
	} else if avgTime < 200*time.Millisecond {
		log.Printf("  👌 Performance rating: Fair\n")
	} else {
		log.Printf("  ⚠️  Performance rating: Needs optimization\n")
	}
}

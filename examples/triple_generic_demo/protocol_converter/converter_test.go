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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// Mock service for testing
type MockDemoService struct{}

func (MockDemoService) Hello(ctx interface{}, name string) (string, error) {
	return "hello, " + name, nil
}

func (MockDemoService) Add(ctx interface{}, a, b int32) (int32, error) {
	return a + b, nil
}

func (MockDemoService) Reference() string {
	return "com.example.DemoService"
}

// Test setup helper
func setupTestEnvironment(t *testing.T) (*server.Server, *ProtocolConverter) {
	const (
		ip   = "127.0.0.1"
		port = 50071 // Use different port for testing
	)

	// Create test server
	srv, err := server.NewServer(
		server.WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithIp(ip),
			protocol.WithPort(port),
		),
		server.WithServerSerialization(constant.Hessian2Serialization),
		server.SetServerApplication(&global.ApplicationConfig{
			Name:                    "test-converter-app",
			MetadataServiceProtocol: "file",
		}),
		server.WithServerNotRegister(),
	)
	if err != nil {
		t.Fatalf("create server failed: %v", err)
	}

	// Register mock service
	if err := srv.RegisterService(&MockDemoService{},
		server.WithSerialization(constant.Hessian2Serialization)); err != nil {
		t.Fatalf("register service failed: %v", err)
	}

	// Start server in background
	go func() { _ = srv.Serve() }()
	time.Sleep(1 * time.Second) // Wait for server to start

	// Create protocol converter
	providerURL := "tri://127.0.0.1:50071/com.example.DemoService"
	converter, err := NewProtocolConverter(providerURL, "com.example.DemoService")
	if err != nil {
		t.Fatalf("create converter failed: %v", err)
	}

	return srv, converter
}

func TestProtocolConverter_SerializationConversion(t *testing.T) {
	srv, converter := setupTestEnvironment(t)
	defer srv.Stop()

	tests := []struct {
		name                string
		sourceSerialization SerializationType
		targetSerialization SerializationType
		methodName          string
		paramTypes          []string
		args                []interface{}
		expectSuccess       bool
	}{
		{
			name:                "Hessian2 to JSON",
			sourceSerialization: SerializationHessian2,
			targetSerialization: SerializationJSON,
			methodName:          "Hello",
			paramTypes:          []string{"java.lang.String"},
			args:                []interface{}{"test"},
			expectSuccess:       true,
		},
		{
			name:                "JSON to Hessian2",
			sourceSerialization: SerializationJSON,
			targetSerialization: SerializationHessian2,
			methodName:          "Add",
			paramTypes:          []string{"int", "int"},
			args:                []interface{}{int32(10), int32(20)},
			expectSuccess:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ConvertRequest{
				SourceProtocol:      ProtocolTriple,
				TargetProtocol:      ProtocolTriple,
				SourceSerialization: tt.sourceSerialization,
				TargetSerialization: tt.targetSerialization,
				MethodName:          tt.methodName,
				ParamTypes:          tt.paramTypes,
				Args:                tt.args,
			}

			resp := converter.Convert(req)

			if tt.expectSuccess {
				if !resp.Success {
					t.Errorf("Expected success but got error: %s", resp.Error)
				}
				if resp.Result == nil {
					t.Errorf("Expected result but got nil")
				}
			} else {
				if resp.Success {
					t.Errorf("Expected failure but got success")
				}
			}

			t.Logf("Duration: %s", resp.Duration)
		})
	}
}

func TestProtocolConverter_ProtocolConversion(t *testing.T) {
	srv, converter := setupTestEnvironment(t)
	defer srv.Stop()

	tests := []struct {
		name           string
		sourceProtocol ProtocolType
		targetProtocol ProtocolType
		methodName     string
		paramTypes     []string
		args           []interface{}
		expectSuccess  bool
	}{
		{
			name:           "Triple to HTTP",
			sourceProtocol: ProtocolTriple,
			targetProtocol: ProtocolHTTP,
			methodName:     "Hello",
			paramTypes:     []string{"java.lang.String"},
			args:           []interface{}{"http_test"},
			expectSuccess:  true, // HTTP mock returns success
		},
		{
			name:           "Triple to gRPC",
			sourceProtocol: ProtocolTriple,
			targetProtocol: ProtocolGRPC,
			methodName:     "Add",
			paramTypes:     []string{"int", "int"},
			args:           []interface{}{int32(5), int32(15)},
			expectSuccess:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ConvertRequest{
				SourceProtocol:      tt.sourceProtocol,
				TargetProtocol:      tt.targetProtocol,
				SourceSerialization: SerializationHessian2,
				TargetSerialization: SerializationJSON,
				MethodName:          tt.methodName,
				ParamTypes:          tt.paramTypes,
				Args:                tt.args,
			}

			resp := converter.Convert(req)

			if tt.expectSuccess {
				if !resp.Success {
					t.Errorf("Expected success but got error: %s", resp.Error)
				}
			} else {
				if resp.Success {
					t.Errorf("Expected failure but got success")
				}
			}

			t.Logf("Protocol conversion %s->%s: %v, Duration: %s",
				tt.sourceProtocol, tt.targetProtocol, resp.Success, resp.Duration)
		})
	}
}

func TestProtocolConverter_ErrorHandling(t *testing.T) {
	srv, converter := setupTestEnvironment(t)
	defer srv.Stop()

	tests := []struct {
		name       string
		methodName string
		paramTypes []string
		args       []interface{}
	}{
		{
			name:       "Non-existent method",
			methodName: "NonExistentMethod",
			paramTypes: []string{},
			args:       []interface{}{},
		},
		{
			name:       "Wrong parameter types",
			methodName: "Add",
			paramTypes: []string{"java.lang.String", "java.lang.String"},
			args:       []interface{}{"not", "numbers"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ConvertRequest{
				SourceProtocol:      ProtocolTriple,
				TargetProtocol:      ProtocolTriple,
				SourceSerialization: SerializationHessian2,
				TargetSerialization: SerializationJSON,
				MethodName:          tt.methodName,
				ParamTypes:          tt.paramTypes,
				Args:                tt.args,
			}

			resp := converter.Convert(req)

			// Should fail
			if resp.Success {
				t.Errorf("Expected failure for %s but got success", tt.name)
			}

			if resp.Error == "" {
				t.Errorf("Expected error message for %s but got empty string", tt.name)
			}

			t.Logf("Error handling test %s: %s", tt.name, resp.Error)
		})
	}
}

func TestProtocolGateway_HTTPEndpoints(t *testing.T) {
	srv, converter := setupTestEnvironment(t)
	defer srv.Stop()

	gateway := NewProtocolGateway(converter, 8081) // Use different port

	tests := []struct {
		name           string
		method         string
		endpoint       string
		body           string
		expectedStatus int
	}{
		{
			name:           "Health check",
			method:         "GET",
			endpoint:       "/health",
			body:           "",
			expectedStatus: http.StatusOK,
		},
		{
			name:     "Valid conversion request",
			method:   "POST",
			endpoint: "/convert",
			body: `{
				"source_protocol": "triple",
				"target_protocol": "triple",
				"source_serialization": "hessian2", 
				"target_serialization": "json",
				"method_name": "Hello",
				"param_types": ["java.lang.String"],
				"args": ["test"]
			}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid JSON request",
			method:         "POST",
			endpoint:       "/convert",
			body:           `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Wrong method for convert",
			method:         "GET",
			endpoint:       "/convert",
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error

			if tt.body != "" {
				req, err = http.NewRequest(tt.method, tt.endpoint, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, tt.endpoint, nil)
			}

			if err != nil {
				t.Fatalf("create request failed: %v", err)
			}

			rr := httptest.NewRecorder()

			// Route request to appropriate handler
			if tt.endpoint == "/health" {
				gateway.handleHealth(rr, req)
			} else if tt.endpoint == "/convert" {
				gateway.handleConvert(rr, req)
			}

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d but got %d", tt.expectedStatus, rr.Code)
			}

			t.Logf("HTTP test %s: status=%d, body=%s", tt.name, rr.Code, rr.Body.String())
		})
	}
}

func TestProtocolConverter_Performance(t *testing.T) {
	srv, converter := setupTestEnvironment(t)
	defer srv.Stop()

	const numRequests = 100

	req := &ConvertRequest{
		SourceProtocol:      ProtocolTriple,
		TargetProtocol:      ProtocolTriple,
		SourceSerialization: SerializationHessian2,
		TargetSerialization: SerializationJSON,
		MethodName:          "Hello",
		ParamTypes:          []string{"java.lang.String"},
		Args:                []interface{}{"performance_test"},
	}

	start := time.Now()
	successCount := 0

	for i := 0; i < numRequests; i++ {
		resp := converter.Convert(req)
		if resp.Success {
			successCount++
		}
	}

	totalDuration := time.Since(start)
	averageLatency := totalDuration / time.Duration(numRequests)

	t.Logf("Performance test results:")
	t.Logf("  Total requests: %d", numRequests)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Success rate: %.2f%%", float64(successCount)/float64(numRequests)*100)
	t.Logf("  Total duration: %v", totalDuration)
	t.Logf("  Average latency: %v", averageLatency)
	t.Logf("  Throughput: %.2f req/sec", float64(numRequests)/totalDuration.Seconds())

	// Performance assertions
	if averageLatency > 10*time.Millisecond {
		t.Errorf("Average latency too high: %v", averageLatency)
	}

	if float64(successCount)/float64(numRequests) < 0.95 {
		t.Errorf("Success rate too low: %.2f%%", float64(successCount)/float64(numRequests)*100)
	}
}

func TestProtocolConverter_ConcurrentCalls(t *testing.T) {
	srv, converter := setupTestEnvironment(t)
	defer srv.Stop()

	const (
		numGoroutines     = 10
		callsPerGoroutine = 10
	)

	results := make(chan bool, numGoroutines*callsPerGoroutine)

	req := &ConvertRequest{
		SourceProtocol:      ProtocolTriple,
		TargetProtocol:      ProtocolTriple,
		SourceSerialization: SerializationHessian2,
		TargetSerialization: SerializationJSON,
		MethodName:          "Add",
		ParamTypes:          []string{"int", "int"},
		Args:                []interface{}{int32(1), int32(1)},
	}

	start := time.Now()

	// Start concurrent goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < callsPerGoroutine; j++ {
				// Modify args to make each call unique
				localReq := *req
				localReq.Args = []interface{}{int32(goroutineID), int32(j)}

				resp := converter.Convert(&localReq)
				results <- resp.Success
			}
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines*callsPerGoroutine; i++ {
		if <-results {
			successCount++
		}
	}

	duration := time.Since(start)
	totalCalls := numGoroutines * callsPerGoroutine

	t.Logf("Concurrency test results:")
	t.Logf("  Concurrent goroutines: %d", numGoroutines)
	t.Logf("  Calls per goroutine: %d", callsPerGoroutine)
	t.Logf("  Total calls: %d", totalCalls)
	t.Logf("  Successful calls: %d", successCount)
	t.Logf("  Success rate: %.2f%%", float64(successCount)/float64(totalCalls)*100)
	t.Logf("  Total duration: %v", duration)
	t.Logf("  Throughput: %.2f calls/sec", float64(totalCalls)/duration.Seconds())

	// Concurrency assertions
	if float64(successCount)/float64(totalCalls) < 0.9 {
		t.Errorf("Success rate under concurrent load too low: %.2f%%",
			float64(successCount)/float64(totalCalls)*100)
	}
}

// Benchmark tests
func BenchmarkProtocolConverter_TripleToTriple(b *testing.B) {
	// Setup (not timed)
	b.StopTimer()
	srv, converter := setupTestEnvironment(&testing.T{})
	defer srv.Stop()

	req := &ConvertRequest{
		SourceProtocol:      ProtocolTriple,
		TargetProtocol:      ProtocolTriple,
		SourceSerialization: SerializationHessian2,
		TargetSerialization: SerializationJSON,
		MethodName:          "Hello",
		ParamTypes:          []string{"java.lang.String"},
		Args:                []interface{}{"benchmark"},
	}
	b.StartTimer()

	// Benchmark (timed)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp := converter.Convert(req)
			if !resp.Success {
				b.Errorf("Conversion failed: %s", resp.Error)
			}
		}
	})
}

func BenchmarkProtocolConverter_SerializationConversion(b *testing.B) {
	b.StopTimer()
	srv, converter := setupTestEnvironment(&testing.T{})
	defer srv.Stop()

	req := &ConvertRequest{
		SourceProtocol:      ProtocolTriple,
		TargetProtocol:      ProtocolTriple,
		SourceSerialization: SerializationJSON,
		TargetSerialization: SerializationHessian2,
		MethodName:          "Add",
		ParamTypes:          []string{"int", "int"},
		Args:                []interface{}{int32(100), int32(200)},
	}
	b.StartTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			converter.Convert(req)
		}
	})
}

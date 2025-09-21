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
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// DemoService represents the service interface
type DemoService struct{}

// Hello is a simple greeting method
func (DemoService) Hello(ctx context.Context, name string) (string, error) {
	return fmt.Sprintf("Hello, %s! (from Triple Generic Provider)", name), nil
}

// Add is a simple math method
func (DemoService) Add(ctx context.Context, a, b int32) (int32, error) {
	result := a + b
	fmt.Printf("Add(%d, %d) = %d\n", a, b, result)
	return result, nil
}

// GetUserInfo returns user information
func (DemoService) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	userInfo := map[string]interface{}{
		"id":       userID,
		"name":     fmt.Sprintf("User_%s", userID),
		"email":    fmt.Sprintf("%s@example.com", userID),
		"created":  time.Now().Format("2006-01-02 15:04:05"),
		"active":   true,
		"metadata": map[string]string{"region": "us-west", "tier": "premium"},
	}
	fmt.Printf("GetUserInfo(%s) called\n", userID)
	return userInfo, nil
}

// ProcessList processes a list of items
func (DemoService) ProcessList(ctx context.Context, items []string) ([]string, error) {
	processed := make([]string, len(items))
	for i, item := range items {
		processed[i] = fmt.Sprintf("processed_%s", item)
	}
	fmt.Printf("ProcessList() processed %d items\n", len(items))
	return processed, nil
}

// ComplexOperation demonstrates complex parameter handling
func (DemoService) ComplexOperation(ctx context.Context, request map[string]interface{}) (map[string]interface{}, error) {
	response := map[string]interface{}{
		"status":    "success",
		"timestamp": time.Now().Unix(),
		"processed": true,
		"input":     request,
		"output": map[string]interface{}{
			"id":   "12345",
			"data": "complex operation result",
		},
	}
	fmt.Printf("ComplexOperation() called with %+v\n", request)
	return response, nil
}

// Reference returns the service reference
func (DemoService) Reference() string {
	return "com.example.DemoService"
}

// HealthService for health checks
type HealthService struct{}

func (HealthService) Check(ctx context.Context) (string, error) {
	return "OK", nil
}

func (HealthService) Reference() string {
	return "com.example.HealthService"
}

// MetricsCollector for collecting metrics
type MetricsCollector struct {
	mu       sync.RWMutex
	requests map[string]int64
	errors   map[string]int64
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requests: make(map[string]int64),
		errors:   make(map[string]int64),
	}
}

func (mc *MetricsCollector) RecordRequest(method string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.requests[method]++
}

func (mc *MetricsCollector) RecordError(method string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.errors[method]++
}

func (mc *MetricsCollector) GetStats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["requests"] = make(map[string]int64)
	stats["errors"] = make(map[string]int64)

	for k, v := range mc.requests {
		stats["requests"].(map[string]int64)[k] = v
	}
	for k, v := range mc.errors {
		stats["errors"].(map[string]int64)[k] = v
	}

	return stats
}

var metricsCollector = NewMetricsCollector()

func main() {
	const (
		ip   = "127.0.0.1"
		port = 50051
	)

	fmt.Println("=== Triple Generic Call Provider Demo ===")
	fmt.Printf("Starting server on %s:%d\n", ip, port)

	// Create server with Triple protocol
	srv, err := server.NewServer(
		server.WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithIp(ip),
			protocol.WithPort(port),
		),
		server.WithServerSerialization(constant.Hessian2Serialization),
		server.SetServerApplication(&global.ApplicationConfig{
			Name:                    "triple-generic-provider",
			Version:                 "1.0.0",
			MetadataServiceProtocol: "file", // Use file protocol to support generic calls
		}),
		server.WithServerNotRegister(), // Skip registry registration, direct connection mode
	)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Register main demo service
	if err := srv.RegisterService(&DemoService{},
		server.WithSerialization(constant.Hessian2Serialization)); err != nil {
		log.Fatalf("Failed to register DemoService: %v", err)
	}
	fmt.Println("✓ Registered DemoService with methods:")
	fmt.Println("  - Hello(name string) string")
	fmt.Println("  - Add(a, b int32) int32")
	fmt.Println("  - GetUserInfo(userID string) map[string]interface{}")
	fmt.Println("  - ProcessList(items []string) []string")
	fmt.Println("  - ComplexOperation(request map[string]interface{}) map[string]interface{}")

	// Register health service
	if err := srv.RegisterService(&HealthService{}); err != nil {
		log.Fatalf("Failed to register HealthService: %v", err)
	}
	fmt.Println("✓ Registered HealthService")

	// Print generic call information
	fmt.Println("\n=== Generic Call Information ===")
	fmt.Println("Generic Method: $invoke")
	fmt.Println("Parameters: [methodName, paramTypes, args]")
	fmt.Println("Example: $invoke(\"Hello\", [\"java.lang.String\"], [\"world\"])")
	fmt.Println("Supported serializations: hessian2, json")

	// Start metrics reporting goroutine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := metricsCollector.GetStats()
			fmt.Printf("\n=== Metrics (every 30s) ===\n")
			fmt.Printf("Requests: %+v\n", stats["requests"])
			fmt.Printf("Errors: %+v\n", stats["errors"])
		}
	}()

	// Start server in goroutine
	go func() {
		fmt.Println("\n🚀 Server starting...")
		if err := srv.Serve(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(1 * time.Second)
	fmt.Println("✅ Server is ready to accept generic calls!")
	fmt.Printf("✅ Listening on %s:%d\n", ip, port)

	// Print usage examples
	fmt.Println("\n=== Usage Examples ===")
	fmt.Println("You can now make generic calls using any Dubbo client:")
	fmt.Printf("1. Direct connection: tri://%s:%d/com.example.DemoService\n", ip, port)
	fmt.Println("2. Call method: $invoke")
	fmt.Println("3. Example calls:")
	fmt.Println(`   Hello: $invoke("Hello", ["java.lang.String"], ["world"])`)
	fmt.Println(`   Add: $invoke("Add", ["int", "int"], [10, 20])`)
	fmt.Println(`   GetUserInfo: $invoke("GetUserInfo", ["java.lang.String"], ["user123"])`)

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")

	// Print final statistics
	stats := metricsCollector.GetStats()
	fmt.Printf("Final statistics:\n")
	fmt.Printf("  Requests: %+v\n", stats["requests"])
	fmt.Printf("  Errors: %+v\n", stats["errors"])

	fmt.Println("✅ Server shutdown complete")
}

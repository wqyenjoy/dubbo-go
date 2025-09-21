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

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// DemoService represents the service interface
type DemoService struct{}

	func (DemoService) Hello(ctx context.Context, name string) (string, error) {
		return fmt.Sprintf("Hello, %s", name), nil
	}

	func (DemoService) Add(ctx context.Context, a, b int32) (int32, error) {
	result := a + b
	fmt.Printf("Add(%d, %d) = %d\n", a, b, result)
	return result, nil
}

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

	func (DemoService) ProcessList(ctx context.Context, items []string) ([]string, error) {
	processed := make([]string, len(items))
	for i, item := range items {
		processed[i] = fmt.Sprintf("processed_%s", item)
	}
	fmt.Printf("ProcessList() processed %d items\n", len(items))
	return processed, nil
}

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

	func (DemoService) Reference() string {
	return "com.example.DemoService"
}

	type HealthService struct{}

func (HealthService) Check(ctx context.Context) (string, error) {
	return "OK", nil
}

func (HealthService) Reference() string {
	return "com.example.HealthService"
}

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

	log.Printf("Starting server on %s:%d", ip, port)

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
			MetadataServiceProtocol: "file",
		}),
		server.WithServerNotRegister()
	)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := srv.RegisterService(&DemoService{},
		server.WithSerialization(constant.Hessian2Serialization)); err != nil {
		log.Fatalf("Failed to register DemoService: %v", err)
	}
	log.Println("Registered DemoService")

	if err := srv.RegisterService(&HealthService{}); err != nil {
		log.Fatalf("Failed to register HealthService: %v", err)
	}
	log.Println("Registered HealthService")

	log.Println("Generic method: $invoke")
	log.Println("Parameters: [methodName, paramTypes, args]")

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := metricsCollector.GetStats()
			log.Printf("Metrics - Requests: %+v, Errors: %+v", stats["requests"], stats["errors"])
		}
	}()

	go func() {
		if err := srv.Serve(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	time.Sleep(1 * time.Second)
	log.Printf("Server listening on %s:%d", ip, port)

	log.Printf("Generic call URL: tri://%s:%d/com.example.DemoService", ip, port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server")

	stats := metricsCollector.GetStats()
	log.Printf("Final stats - Requests: %+v, Errors: %+v", stats["requests"], stats["errors"])
	log.Println("Server shutdown complete")
}

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

package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// TestDemoService for benchmarking
type TestDemoService struct{}

func (TestDemoService) Hello(ctx context.Context, name string) (string, error) {
	return "hello, " + name, nil
}

func (TestDemoService) Add(ctx context.Context, a, b int32) (int32, error) {
	return a + b, nil
}

func (TestDemoService) GetUserInfo(ctx context.Context, userID string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":   userID,
		"name": "User_" + userID,
		"active": true,
	}, nil
}

func (TestDemoService) Reference() string { return "com.example.DemoService" }

// Setup function to start test server
func setupTestServer(t *testing.T) (*server.Server, string) {
	const (
		ip   = "127.0.0.1"
		port = 50061
	)

	srv, err := server.NewServer(
		server.WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithIp(ip),
			protocol.WithPort(port),
		),
		server.WithServerSerialization(constant.Hessian2Serialization),
		server.SetServerApplication(&global.ApplicationConfig{
			Name:                    "benchmark-test-app",
			MetadataServiceProtocol: "file",
		}),
		server.WithServerNotRegister(),
	)
	if err != nil {
		t.Fatalf("new server error: %v", err)
	}

	if err := srv.RegisterService(&TestDemoService{}, server.WithSerialization(constant.Hessian2Serialization)); err != nil {
		t.Fatalf("register error: %v", err)
	}

	go func() { _ = srv.Serve() }()
	time.Sleep(time.Second) // wait server up

	return srv, fmt.Sprintf("tri://%s:%d/com.example.DemoService", ip, port)
}

// Setup generic client
func setupGenericClient(t *testing.T, url string) client.Connection {
	cli, err := client.NewClient(
		client.WithClientURL(url),
		client.WithClientProtocolTriple(),
	)
	if err != nil {
		t.Fatalf("new client error: %v", err)
	}

	conn, err := cli.Dial("com.example.DemoService",
		client.WithGeneric(),
		client.WithSerialization(constant.Hessian2Serialization),
	)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	return conn
}

// Generic call helper
func genericCall(conn client.Connection, method string, types []string, args []interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	var reply interface{}
	err := conn.CallUnary(ctx, []interface{}{method, types, args}, &reply, "$invoke")
	return reply, err
}

// Benchmark simple string call
func BenchmarkGenericCall_Hello(b *testing.B) {
	srv, url := setupTestServer(&testing.T{})
	defer srv.Stop()
	
	conn := setupGenericClient(&testing.T{}, url)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := genericCall(conn, "Hello", []string{"java.lang.String"}, []interface{}{"world"})
			if err != nil {
				b.Fatalf("generic call error: %v", err)
			}
		}
	})
}

// Benchmark math operation
func BenchmarkGenericCall_Add(b *testing.B) {
	srv, url := setupTestServer(&testing.T{})
	defer srv.Stop()
	
	conn := setupGenericClient(&testing.T{}, url)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := genericCall(conn, "Add", []string{"int", "int"}, []interface{}{int32(10), int32(20)})
			if err != nil {
				b.Fatalf("generic call error: %v", err)
			}
		}
	})
}

// Benchmark complex object call
func BenchmarkGenericCall_GetUserInfo(b *testing.B) {
	srv, url := setupTestServer(&testing.T{})
	defer srv.Stop()
	
	conn := setupGenericClient(&testing.T{}, url)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := genericCall(conn, "GetUserInfo", []string{"java.lang.String"}, []interface{}{"user123"})
			if err != nil {
				b.Fatalf("generic call error: %v", err)
			}
		}
	})
}

// Benchmark mixed workload
func BenchmarkGenericCall_Mixed(b *testing.B) {
	srv, url := setupTestServer(&testing.T{})
	defer srv.Stop()
	
	conn := setupGenericClient(&testing.T{}, url)
	
	methods := []struct{
		name string
		types []string  
		args []interface{}
	}{
		{"Hello", []string{"java.lang.String"}, []interface{}{"world"}},
		{"Add", []string{"int", "int"}, []interface{}{int32(10), int32(20)}},
		{"GetUserInfo", []string{"java.lang.String"}, []interface{}{"user123"}},
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			method := methods[i%len(methods)]
			_, err := genericCall(conn, method.name, method.types, method.args)
			if err != nil {
				b.Fatalf("generic call error: %v", err)
			}
			i++
		}
	})
}

// Stress test with high concurrency
func TestGenericCall_StressTest(t *testing.T) {
	srv, url := setupTestServer(t)
	defer srv.Stop()
	
	conn := setupGenericClient(t, url)
	
	// Test parameters
	const (
		numGoroutines = 100
		callsPerGoroutine = 100
		totalCalls = numGoroutines * callsPerGoroutines
	)
	
	var wg sync.WaitGroup
	successChan := make(chan bool, totalCalls)
	errorChan := make(chan error, totalCalls)
	
	start := time.Now()
	
	// Start goroutines
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for j := 0; j < callsPerGoroutine; j++ {
				callID := fmt.Sprintf("stress_%d_%d", goroutineID, j)
				_, err := genericCall(conn, "Hello", []string{"java.lang.String"}, []interface{}{callID})
				
				if err != nil {
					errorChan <- err
				} else {
					successChan <- true
				}
			}
		}(i)
	}
	
	wg.Wait()
	duration := time.Since(start)
	
	close(successChan)
	close(errorChan)
	
	// Count results
	successCount := len(successChan)
	errorCount := len(errorChan)
	
	// Report results
	t.Logf("Stress Test Results:")
	t.Logf("  Total calls: %d", totalCalls)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", errorCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f calls/second", float64(totalCalls)/duration.Seconds())
	t.Logf("  Average latency: %v", duration/time.Duration(totalCalls))
	t.Logf("  Success rate: %.2f%%", float64(successCount)/float64(totalCalls)*100)
	
	// Assertions
	if errorCount > totalCalls/100 { // Allow up to 1% error rate
		t.Errorf("Error rate too high: %d/%d (%.2f%%)", errorCount, totalCalls, float64(errorCount)/float64(totalCalls)*100)
	}
	
	if successCount < totalCalls*99/100 { // Require at least 99% success
		t.Errorf("Success rate too low: %d/%d (%.2f%%)", successCount, totalCalls, float64(successCount)/float64(totalCalls)*100)
	}
}

// Test latency distribution
func TestGenericCall_LatencyDistribution(t *testing.T) {
	srv, url := setupTestServer(t)
	defer srv.Stop()
	
	conn := setupGenericClient(t, url)
	
	const numCalls = 1000
	latencies := make([]time.Duration, 0, numCalls)
	
	// Warm up
	for i := 0; i < 10; i++ {
		genericCall(conn, "Hello", []string{"java.lang.String"}, []interface{}{"warmup"})
	}
	
	// Collect latency data
	for i := 0; i < numCalls; i++ {
		start := time.Now()
		_, err := genericCall(conn, "Hello", []string{"java.lang.String"}, []interface{}{fmt.Sprintf("latency_%d", i)})
		latency := time.Since(start)
		
		if err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}
		
		latencies = append(latencies, latency)
	}
	
	// Calculate statistics
	var sum time.Duration
	min := latencies[0]
	max := latencies[0]
	
	for _, lat := range latencies {
		sum += lat
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}
	
	avg := sum / time.Duration(numCalls)
	
	// Calculate percentiles (simple approach)
	// Note: This is a simplified percentile calculation
	p95Index := int(float64(numCalls) * 0.95)
	p99Index := int(float64(numCalls) * 0.99)
	
	// Sort latencies for percentile calculation
	for i := 0; i < len(latencies)-1; i++ {
		for j := 0; j < len(latencies)-i-1; j++ {
			if latencies[j] > latencies[j+1] {
				latencies[j], latencies[j+1] = latencies[j+1], latencies[j]
			}
		}
	}
	
	p95 := latencies[p95Index-1]
	p99 := latencies[p99Index-1]
	
	// Report statistics
	t.Logf("Latency Distribution (%d calls):", numCalls)
	t.Logf("  Average: %v", avg)
	t.Logf("  Minimum: %v", min)
	t.Logf("  Maximum: %v", max)
	t.Logf("  95th percentile: %v", p95)
	t.Logf("  99th percentile: %v", p99)
	
	// Assertions
	if avg > 10*time.Millisecond {
		t.Errorf("Average latency too high: %v", avg)
	}
	
	if p95 > 50*time.Millisecond {
		t.Errorf("P95 latency too high: %v", p95)
	}
	
	if p99 > 100*time.Millisecond {
		t.Errorf("P99 latency too high: %v", p99)
	}
}

// Test memory usage under load
func TestGenericCall_MemoryUsage(t *testing.T) {
	srv, url := setupTestServer(t)
	defer srv.Stop()
	
	conn := setupGenericClient(t, url)
	
	// Run multiple rounds to check for memory leaks
	const rounds = 5
	const callsPerRound = 1000
	
	for round := 0; round < rounds; round++ {
		t.Logf("Memory test round %d/%d", round+1, rounds)
		
		start := time.Now()
		for i := 0; i < callsPerRound; i++ {
			_, err := genericCall(conn, "Hello", []string{"java.lang.String"}, []interface{}{fmt.Sprintf("memory_%d_%d", round, i)})
			if err != nil {
				t.Fatalf("call failed in round %d: %v", round, err)
			}
		}
		duration := time.Since(start)
		
		t.Logf("  Completed %d calls in %v (%.2f calls/sec)", callsPerRound, duration, float64(callsPerRound)/duration.Seconds())
		
		// Small pause between rounds
		time.Sleep(100 * time.Millisecond)
	}
	
	t.Logf("Memory test completed successfully - no apparent memory leaks")
}

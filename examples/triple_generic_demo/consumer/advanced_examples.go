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
	"sync"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// AsyncGenericCaller provides async generic call functionality
type AsyncGenericCaller struct {
	conn client.Connection
}

// AsyncCallResult represents the result of an async call
type AsyncCallResult struct {
	MethodName string
	Result     interface{}
	Error      error
	Duration   time.Duration
	StartTime  time.Time
	EndTime    time.Time
}

// BatchCallRequest represents a batch call request
type BatchCallRequest struct {
	ID         string
	MethodName string
	ParamTypes []string
	Args       []interface{}
}

// BatchCallResponse represents a batch call response
type BatchCallResponse struct {
	ID       string
	Result   interface{}
	Error    error
	Duration time.Duration
}

// NewAsyncGenericCaller creates a new async generic caller
func NewAsyncGenericCaller() (*AsyncGenericCaller, error) {
	cli, err := client.NewClient(
		client.WithClientURL("tri://127.0.0.1:50051/com.example.DemoService"),
		client.WithClientProtocolTriple(),
	)
	if err != nil {
		return nil, fmt.Errorf("create client failed: %v", err)
	}

	conn, err := cli.Dial("com.example.DemoService",
		client.WithGeneric(),
		client.WithSerialization(constant.Hessian2Serialization),
	)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %v", err)
	}

	return &AsyncGenericCaller{conn: conn}, nil
}

// CallAsync makes an asynchronous generic call
func (agc *AsyncGenericCaller) CallAsync(methodName string, paramTypes []string, args []interface{}) <-chan AsyncCallResult {
	resultChan := make(chan AsyncCallResult, 1)
	
	go func() {
		defer close(resultChan)
		
		startTime := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var reply interface{}
		err := agc.conn.CallUnary(ctx, []interface{}{methodName, paramTypes, args}, &reply, "$invoke")
		
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		resultChan <- AsyncCallResult{
			MethodName: methodName,
			Result:     reply,
			Error:      err,
			Duration:   duration,
			StartTime:  startTime,
			EndTime:    endTime,
		}
	}()
	
	return resultChan
}

// CallBatch executes multiple generic calls concurrently
func (agc *AsyncGenericCaller) CallBatch(requests []BatchCallRequest) []BatchCallResponse {
	responsesChan := make(chan BatchCallResponse, len(requests))
	var wg sync.WaitGroup

	// Execute all calls concurrently
	for _, req := range requests {
		wg.Add(1)
		go func(request BatchCallRequest) {
			defer wg.Done()
			
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			var reply interface{}
			err := agc.conn.CallUnary(ctx, []interface{}{request.MethodName, request.ParamTypes, request.Args}, &reply, "$invoke")
			
			duration := time.Since(startTime)

			responsesChan <- BatchCallResponse{
				ID:       request.ID,
				Result:   reply,
				Error:    err,
				Duration: duration,
			}
		}(req)
	}

	// Wait for all calls to complete
	go func() {
		wg.Wait()
		close(responsesChan)
	}()

	// Collect results
	var responses []BatchCallResponse
	for response := range responsesChan {
		responses = append(responses, response)
	}

	return responses
}

// CallWithTimeout makes a call with custom timeout
func (agc *AsyncGenericCaller) CallWithTimeout(methodName string, paramTypes []string, args []interface{}, timeout time.Duration) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var reply interface{}
	err := agc.conn.CallUnary(ctx, []interface{}{methodName, paramTypes, args}, &reply, "$invoke")
	return reply, err
}

// CallWithRetry makes a call with retry mechanism
func (agc *AsyncGenericCaller) CallWithRetry(methodName string, paramTypes []string, args []interface{}, maxRetries int, retryDelay time.Duration) (interface{}, error) {
	var lastErr error
	
	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			fmt.Printf("Retrying call %s (attempt %d/%d)\n", methodName, i+1, maxRetries+1)
			time.Sleep(retryDelay)
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		var reply interface{}
		err := agc.conn.CallUnary(ctx, []interface{}{methodName, paramTypes, args}, &reply, "$invoke")
		cancel()
		
		if err == nil {
			return reply, nil
		}
		
		lastErr = err
	}
	
	return nil, fmt.Errorf("call failed after %d retries: %v", maxRetries+1, lastErr)
}

// AdvancedExamples demonstrates advanced generic call patterns
func AdvancedExamples() {
	fmt.Println("\n=== Advanced Generic Call Examples ===")
	
	caller, err := NewAsyncGenericCaller()
	if err != nil {
		log.Fatalf("Failed to create async caller: %v", err)
	}

	// Example 1: Async Calls
	fmt.Println("\n--- Example 1: Async Calls ---")
	fmt.Println("Starting 3 async calls...")
	
	asyncCall1 := caller.CallAsync("Hello", []string{"java.lang.String"}, []interface{}{"Async1"})
	asyncCall2 := caller.CallAsync("Add", []string{"int", "int"}, []interface{}{int32(10), int32(20)})
	asyncCall3 := caller.CallAsync("GetUserInfo", []string{"java.lang.String"}, []interface{}{"async_user"})

	// Wait for results
	result1 := <-asyncCall1
	result2 := <-asyncCall2
	result3 := <-asyncCall3

	fmt.Printf("Async Call 1: %s (took %v)\n", result1.Result, result1.Duration)
	fmt.Printf("Async Call 2: %v (took %v)\n", result2.Result, result2.Duration)
	fmt.Printf("Async Call 3: %v (took %v)\n", result3.Result, result3.Duration)

	// Example 2: Batch Calls
	fmt.Println("\n--- Example 2: Batch Calls ---")
	
	batchRequests := []BatchCallRequest{
		{ID: "batch1", MethodName: "Hello", ParamTypes: []string{"java.lang.String"}, Args: []interface{}{"Batch1"}},
		{ID: "batch2", MethodName: "Hello", ParamTypes: []string{"java.lang.String"}, Args: []interface{}{"Batch2"}},
		{ID: "batch3", MethodName: "Add", ParamTypes: []string{"int", "int"}, Args: []interface{}{int32(5), int32(15)}},
		{ID: "batch4", MethodName: "Add", ParamTypes: []string{"int", "int"}, Args: []interface{}{int32(100), int32(200)}},
		{ID: "batch5", MethodName: "GetUserInfo", ParamTypes: []string{"java.lang.String"}, Args: []interface{}{"batch_user"}},
	}

	start := time.Now()
	batchResults := caller.CallBatch(batchRequests)
	batchDuration := time.Since(start)

	fmt.Printf("Batch execution completed in %v\n", batchDuration)
	fmt.Printf("Processed %d calls concurrently:\n", len(batchResults))
	
	for _, result := range batchResults {
		if result.Error != nil {
			fmt.Printf("  %s: ERROR - %v (took %v)\n", result.ID, result.Error, result.Duration)
		} else {
			fmt.Printf("  %s: %v (took %v)\n", result.ID, result.Result, result.Duration)
		}
	}

	// Example 3: Timeout Control
	fmt.Println("\n--- Example 3: Timeout Control ---")
	
	// Short timeout
	shortResult, err := caller.CallWithTimeout(
		"Hello", 
		[]string{"java.lang.String"}, 
		[]interface{}{"ShortTimeout"}, 
		100*time.Millisecond, // Very short timeout
	)
	if err != nil {
		fmt.Printf("Short timeout call failed as expected: %v\n", err)
	} else {
		fmt.Printf("Short timeout call succeeded: %v\n", shortResult)
	}
	
	// Normal timeout
	normalResult, err := caller.CallWithTimeout(
		"Hello", 
		[]string{"java.lang.String"}, 
		[]interface{}{"NormalTimeout"}, 
		3*time.Second,
	)
	if err != nil {
		fmt.Printf("Normal timeout call failed: %v\n", err)
	} else {
		fmt.Printf("Normal timeout call succeeded: %v\n", normalResult)
	}

	// Example 4: Retry Mechanism
	fmt.Println("\n--- Example 4: Retry Mechanism ---")
	
	// Retry with valid method
	retryResult, err := caller.CallWithRetry(
		"Hello",
		[]string{"java.lang.String"},
		[]interface{}{"RetryTest"},
		2, // max 2 retries
		500*time.Millisecond, // 500ms delay between retries
	)
	if err != nil {
		fmt.Printf("Retry call failed: %v\n", err)
	} else {
		fmt.Printf("Retry call succeeded: %v\n", retryResult)
	}

	// Retry with invalid method (will fail and retry)
	fmt.Println("\nTesting retry with invalid method:")
	_, err = caller.CallWithRetry(
		"InvalidMethod",
		nil,
		nil,
		2, // max 2 retries
		200*time.Millisecond,
	)
	if err != nil {
		fmt.Printf("Invalid method retry failed as expected: %v\n", err)
	}

	// Example 5: Concurrent Performance Test
	fmt.Println("\n--- Example 5: Concurrent Performance Test ---")
	
	concurrentCount := 50
	start = time.Now()
	
	var wg sync.WaitGroup
	successChan := make(chan bool, concurrentCount)
	errorChan := make(chan error, concurrentCount)
	
	for i := 0; i < concurrentCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			result, err := caller.CallWithTimeout(
				"Hello",
				[]string{"java.lang.String"},
				[]interface{}{fmt.Sprintf("Concurrent_%d", index)},
				2*time.Second,
			)
			
			if err != nil {
				errorChan <- err
			} else {
				successChan <- true
				if index%10 == 0 { // Log every 10th call
					fmt.Printf("  Call %d completed: %v\n", index, result)
				}
			}
		}(i)
	}
	
	wg.Wait()
	close(successChan)
	close(errorChan)
	
	successCount := len(successChan)
	errorCount := len(errorChan)
	totalDuration := time.Since(start)
	
	fmt.Printf("\nConcurrent Performance Results:\n")
	fmt.Printf("  Total calls: %d\n", concurrentCount)
	fmt.Printf("  Successful: %d\n", successCount)
	fmt.Printf("  Failed: %d\n", errorCount)
	fmt.Printf("  Total duration: %v\n", totalDuration)
	fmt.Printf("  Average per call: %v\n", totalDuration/time.Duration(concurrentCount))
	fmt.Printf("  Throughput: %.2f calls/second\n", float64(concurrentCount)/totalDuration.Seconds())

	fmt.Println("\n=== Advanced Examples Completed ===")
}

// Run this as a separate program
func main() {
	// Wait for provider to be ready
	fmt.Println("Waiting for provider to be ready...")
	time.Sleep(1 * time.Second)
	
	AdvancedExamples()
}

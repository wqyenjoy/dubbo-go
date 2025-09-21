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
	"reflect"
	"time"

	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	providerURL = "tri://127.0.0.1:50051/com.example.DemoService"
	timeout     = 3 * time.Second
)

// GenericCaller wraps the generic call functionality
type GenericCaller struct {
	conn client.Connection
}

// NewGenericCaller creates a new generic caller
func NewGenericCaller() (*GenericCaller, error) {
	// Create client
	cli, err := client.NewClient(
		client.WithClientURL(providerURL),
		client.WithClientProtocolTriple(),
	)
	if err != nil {
		return nil, fmt.Errorf("create client failed: %v", err)
	}

	// Create connection with generic support
	conn, err := cli.Dial("com.example.DemoService",
		client.WithGeneric(),
		client.WithSerialization(constant.Hessian2Serialization),
	)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %v", err)
	}

	return &GenericCaller{conn: conn}, nil
}

// Call makes a generic call
func (gc *GenericCaller) Call(methodName string, paramTypes []string, args []interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var reply interface{}
	err := gc.conn.CallUnary(ctx, []interface{}{methodName, paramTypes, args}, &reply, "$invoke")
	if err != nil {
		return nil, fmt.Errorf("call %s failed: %v", methodName, err)
	}

	return reply, nil
}

// CallWithStringResult calls and expects string result
func (gc *GenericCaller) CallWithStringResult(methodName string, paramTypes []string, args []interface{}) (string, error) {
	result, err := gc.Call(methodName, paramTypes, args)
	if err != nil {
		return "", err
	}

	if str, ok := result.(string); ok {
		return str, nil
	}
	return fmt.Sprintf("%v", result), nil
}

// CallWithMapResult calls and expects map result
func (gc *GenericCaller) CallWithMapResult(methodName string, paramTypes []string, args []interface{}) (map[string]interface{}, error) {
	result, err := gc.Call(methodName, paramTypes, args)
	if err != nil {
		return nil, err
	}

	if m, ok := result.(map[string]interface{}); ok {
		return m, nil
	}
	return map[string]interface{}{"result": result}, nil
}

func main() {
	fmt.Println("=== Triple Generic Call Consumer Demo ===")
	fmt.Printf("Connecting to provider: %s\n", providerURL)

	// Create generic caller
	caller, err := NewGenericCaller()
	if err != nil {
		log.Fatalf("Failed to create generic caller: %v", err)
	}
	fmt.Println("✓ Generic caller created successfully")

	// Wait a moment for connection to be established
	time.Sleep(100 * time.Millisecond)

	// Example 1: Simple Hello call
	fmt.Println("\n=== Example 1: Simple Hello Call ===")
	result, err := caller.CallWithStringResult(
		"Hello",
		[]string{"java.lang.String"},
		[]interface{}{"World"},
	)
	if err != nil {
		fmt.Printf("❌ Hello call failed: %v\n", err)
	} else {
		fmt.Printf("✅ Hello result: %s\n", result)
	}

	// Example 2: Math Add operation
	fmt.Println("\n=== Example 2: Math Add Operation ===")
	addResult, err := caller.Call(
		"Add",
		[]string{"int", "int"},
		[]interface{}{int32(15), int32(27)},
	)
	if err != nil {
		fmt.Printf("❌ Add call failed: %v\n", err)
	} else {
		fmt.Printf("✅ Add(15, 27) = %v (type: %T)\n", addResult, addResult)
	}

	// Example 3: Get User Info (returns complex object)
	fmt.Println("\n=== Example 3: Get User Info ===")
	userInfo, err := caller.CallWithMapResult(
		"GetUserInfo",
		[]string{"java.lang.String"},
		[]interface{}{"user123"},
	)
	if err != nil {
		fmt.Printf("❌ GetUserInfo call failed: %v\n", err)
	} else {
		fmt.Printf("✅ User Info:\n")
		for k, v := range userInfo {
			fmt.Printf("   %s: %v\n", k, v)
		}
	}

	// Example 4: Process List
	fmt.Println("\n=== Example 4: Process List ===")
	listResult, err := caller.Call(
		"ProcessList",
		[]string{"java.util.List"},
		[]interface{}{[]interface{}{"item1", "item2", "item3"}},
	)
	if err != nil {
		fmt.Printf("❌ ProcessList call failed: %v\n", err)
	} else {
		fmt.Printf("✅ ProcessList result: %v\n", listResult)
	}

	// Example 5: Complex Operation
	fmt.Println("\n=== Example 5: Complex Operation ===")
	complexRequest := map[string]interface{}{
		"action": "process",
		"data": map[string]interface{}{
			"items": []interface{}{"a", "b", "c"},
			"config": map[string]interface{}{
				"timeout": 30,
				"retries": 3,
			},
		},
		"metadata": map[string]interface{}{
			"user":      "test_user",
			"timestamp": time.Now().Unix(),
		},
	}

	complexResult, err := caller.CallWithMapResult(
		"ComplexOperation",
		[]string{"java.util.Map"},
		[]interface{}{complexRequest},
	)
	if err != nil {
		fmt.Printf("❌ ComplexOperation call failed: %v\n", err)
	} else {
		fmt.Printf("✅ ComplexOperation result:\n")
		printMap(complexResult, "   ")
	}

	// Example 6: Error handling - call non-existent method
	fmt.Println("\n=== Example 6: Error Handling ===")
	_, err = caller.Call("NonExistentMethod", nil, nil)
	if err != nil {
		fmt.Printf("✅ Expected error for non-existent method: %v\n", err)
	} else {
		fmt.Println("❌ Should have received error for non-existent method")
	}

	// Example 7: Call with empty types (go-go tolerant mode)
	fmt.Println("\n=== Example 7: Empty Types (Tolerant Mode) ===")
	tolerantResult, err := caller.CallWithStringResult(
		"Hello",
		nil, // Empty types
		[]interface{}{"Tolerant"},
	)
	if err != nil {
		fmt.Printf("❌ Tolerant call failed: %v\n", err)
	} else {
		fmt.Printf("✅ Tolerant result: %s\n", tolerantResult)
	}

	// Example 8: Performance test - multiple calls
	fmt.Println("\n=== Example 8: Performance Test ===")
	start := time.Now()
	successCount := 0
	failureCount := 0

	for i := 0; i < 10; i++ {
		_, err := caller.Call(
			"Hello",
			[]string{"java.lang.String"},
			[]interface{}{fmt.Sprintf("test_%d", i)},
		)
		if err != nil {
			failureCount++
		} else {
			successCount++
		}
	}

	duration := time.Since(start)
	fmt.Printf("✅ Performance test completed:\n")
	fmt.Printf("   Total calls: 10\n")
	fmt.Printf("   Successful: %d\n", successCount)
	fmt.Printf("   Failed: %d\n", failureCount)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Average per call: %v\n", duration/10)

	// Example 9: Type reflection demonstration
	fmt.Println("\n=== Example 9: Type Reflection Demo ===")
	reflectResult, err := caller.Call("Add", []string{"int", "int"}, []interface{}{int32(5), int32(10)})
	if err != nil {
		fmt.Printf("❌ Reflection demo failed: %v\n", err)
	} else {
		fmt.Printf("✅ Result type analysis:\n")
		fmt.Printf("   Value: %v\n", reflectResult)
		fmt.Printf("   Type: %T\n", reflectResult)
		fmt.Printf("   Kind: %v\n", reflect.TypeOf(reflectResult).Kind())
		if reflect.TypeOf(reflectResult).Kind() == reflect.Ptr {
			fmt.Printf("   Elem Type: %T\n", reflect.ValueOf(reflectResult).Elem().Interface())
		}
	}

	fmt.Println("\n=== Demo Completed ===")
	fmt.Println("Generic calls executed successfully!")
	fmt.Println("You can now modify this example to test your own services.")
}

// Helper function to pretty print nested maps
func printMap(m map[string]interface{}, indent string) {
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			fmt.Printf("%s%s:\n", indent, k)
			printMap(val, indent+"   ")
		case []interface{}:
			fmt.Printf("%s%s: %v\n", indent, k, val)
		default:
			fmt.Printf("%s%s: %v\n", indent, k, val)
		}
	}
}

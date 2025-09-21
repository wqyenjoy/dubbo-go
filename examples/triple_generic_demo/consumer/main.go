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

	type GenericCaller struct {
	conn client.Connection
}

	func NewGenericCaller() (*GenericCaller, error) {
		cli, err := client.NewClient(
		client.WithClientURL(providerURL),
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

	return &GenericCaller{conn: conn}, nil
}

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
	log.Printf("Connecting to %s", providerURL)

	caller, err := NewGenericCaller()
	if err != nil {
		log.Fatalf("Failed to create generic caller: %v", err)
	}
	log.Println("Generic caller created")

	time.Sleep(100 * time.Millisecond)

	log.Println("Example 1: Simple Hello call")
	result, err := caller.CallWithStringResult(
		"Hello",
		[]string{"java.lang.String"},
		[]interface{}{"World"},
	)
	if err != nil {
		log.Printf("Hello call failed: %v", err)
	} else {
		log.Printf("Hello result: %s", result)
	}

	log.Println("Example 2: Math Add operation")
	addResult, err := caller.Call(
		"Add",
		[]string{"int", "int"},
		[]interface{}{int32(15), int32(27)},
	)
	if err != nil {
		log.Printf("Add call failed: %v", err)
	} else {
		log.Printf("Add(15, 27) = %v (type: %T)", addResult, addResult)
	}

	log.Println("Example 3: Get User Info")
	userInfo, err := caller.CallWithMapResult(
		"GetUserInfo",
		[]string{"java.lang.String"},
		[]interface{}{"user123"},
	)
	if err != nil {
		log.Printf("GetUserInfo call failed: %v", err)
	} else {
		log.Printf("User Info:")
		for k, v := range userInfo {
			log.Printf("  %s: %v", k, v)
		}
	}

	log.Println("Example 4: Process List")
	listResult, err := caller.Call(
		"ProcessList",
		[]string{"java.util.List"},
		[]interface{}{[]interface{}{"item1", "item2", "item3"}},
	)
	if err != nil {
		log.Printf("ProcessList call failed: %v", err)
	} else {
		log.Printf("ProcessList result: %v", listResult)
	}

	log.Println("Example 5: Complex Operation")
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
		log.Printf("ComplexOperation call failed: %v", err)
	} else {
		log.Printf("ComplexOperation result:")
		printMap(complexResult, "  ")
	}

	log.Println("Example 6: Error handling")
	_, err = caller.Call("NonExistentMethod", nil, nil)
	if err != nil {
		log.Printf("Expected error for non-existent method: %v", err)
	} else {
		log.Println("Should have received error for non-existent method")
	}

	log.Println("Example 7: Empty types (tolerant mode)")
	tolerantResult, err := caller.CallWithStringResult(
		"Hello",
		nil,
		[]interface{}{"Tolerant"},
	)
	if err != nil {
		log.Printf("Tolerant call failed: %v", err)
	} else {
		log.Printf("Tolerant result: %s", tolerantResult)
	}

	log.Println("Example 8: Performance test")
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
	log.Printf("Performance test: %d successful, %d failed, duration: %v, avg: %v", successCount, failureCount, duration, duration/10)

	log.Println("Example 9: Type reflection")
	reflectResult, err := caller.Call("Add", []string{"int", "int"}, []interface{}{int32(5), int32(10)})
	if err != nil {
		log.Printf("Reflection demo failed: %v", err)
	} else {
		log.Printf("Value: %v, Type: %T, Kind: %v", reflectResult, reflectResult, reflect.TypeOf(reflectResult).Kind())
		if reflect.TypeOf(reflectResult).Kind() == reflect.Ptr {
			log.Printf("Elem Type: %T", reflect.ValueOf(reflectResult).Elem().Interface())
		}
	}

	log.Println("Demo completed")
}

	func printMap(m map[string]interface{}, indent string) {
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			log.Printf("%s%s:", indent, k)
			printMap(val, indent+"  ")
		case []interface{}:
			log.Printf("%s%s: %v", indent, k, val)
		default:
			log.Printf("%s%s: %v", indent, k, val)
		}
	}
}

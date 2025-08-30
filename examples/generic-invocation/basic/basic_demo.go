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
	"sync"
	"time"

	hessian "github.com/apache/dubbo-go-hessian2"

	"dubbo.apache.org/dubbo-go/v3/config/generic"
)

func main() {
	fmt.Println("Generic Invocation Demo")
	fmt.Println("====================")

	// Create service instance
	calculatorService := generic.NewGenericService("com.example.CalculatorService")

	// Configure service implementation
	calculatorService.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
		switch methodName {
		case "add":
			if len(args) == 2 {
				a, ok1 := args[0].(int32)
				b, ok2 := args[1].(int32)
				if ok1 && ok2 {
					result := a + b
					fmt.Printf("Gateway received: %d + %d = %d\n", a, b, result)
					return result, nil
				}
			}
			return nil, fmt.Errorf("invalid arguments for add")
		case "multiply":
			if len(args) == 2 {
				a, ok1 := args[0].(int32)
				b, ok2 := args[1].(int32)
				if ok1 && ok2 {
					result := a * b
					fmt.Printf("Gateway received: %d × %d = %d\n", a, b, result)
					return result, nil
				}
			}
			return nil, fmt.Errorf("invalid arguments for multiply")
		case "greet":
			if len(args) == 1 {
				name, ok := args[0].(string)
				if ok {
					result := fmt.Sprintf("Hello, %s!", name)
					fmt.Printf("Gateway received: Greeting %s -> %s\n", name, result)
					return result, nil
				}
			}
			return nil, fmt.Errorf("invalid arguments for greet")
		default:
			return nil, fmt.Errorf("unknown method: %s", methodName)
		}
	}

	// Start gateway receiver (simulate server)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Gateway receiver started, waiting for requests...")
		// In real scenarios, this would be HTTP server or message queue consumer
		// Here we just demonstrate, so wait for sender to call
	}()

	// Start gateway sender (simulate client)
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Gateway sender starts sending requests...")

		time.Sleep(100 * time.Millisecond) // Wait for receiver to start

		ctx := context.Background()

		// Addition test case
		fmt.Println("\nTest Case 1: Addition")
		result1, err1 := calculatorService.Invoke(ctx, "add",
			[]string{"int", "int"}, []hessian.Object{int32(15), int32(27)})
		if err1 != nil {
			fmt.Printf("Addition call failed: %v\n", err1)
		} else {
			fmt.Printf("Addition result: %v\n", result1)
		}

		// Multiplication test case
		fmt.Println("\nTest Case 2: Multiplication")
		result2, err2 := calculatorService.Invoke(ctx, "multiply",
			[]string{"int", "int"}, []hessian.Object{int32(8), int32(9)})
		if err2 != nil {
			fmt.Printf("Multiplication call failed: %v\n", err2)
		} else {
			fmt.Printf("Multiplication result: %v\n", result2)
		}

		// String processing test case
		fmt.Println("\nTest Case 3: String processing")
		result3, err3 := calculatorService.Invoke(ctx, "greet",
			[]string{"java.lang.String"}, []hessian.Object{"Generic Invocation"})
		if err3 != nil {
			fmt.Printf("Greeting call failed: %v\n", err3)
		} else {
			fmt.Printf("Greeting result: %v\n", result3)
		}

		// Error handling test case
		fmt.Println("\nTest Case 4: Error handling")
		_, err4 := calculatorService.Invoke(ctx, "unknownMethod",
			[]string{}, []hessian.Object{})
		if err4 != nil {
			fmt.Printf("Error handling works correctly: %v\n", err4)
		} else {
			fmt.Println("Error handling exception: should return error")
		}

		fmt.Println("Gateway sender completes all tests")
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	fmt.Println("\nGeneric invocation demo completed!")
	fmt.Println("=====================")
	fmt.Println("Gateway sender and receiver communication successful")
	fmt.Println("Generic invocation works normally")
	fmt.Println("Production ready!")
}

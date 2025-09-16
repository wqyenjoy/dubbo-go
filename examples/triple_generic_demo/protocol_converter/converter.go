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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	hessian "github.com/apache/dubbo-go-hessian2"
)

// ProtocolConverter handles conversion between different protocols
type ProtocolConverter struct {
	// Triple client connections with different serializations
	tripleHessianConn client.Connection
	tripleJSONConn    client.Connection
	
	// HTTP client for REST API conversion
	httpClient *http.Client
	
	// Configuration
	providerURL string
	serviceName string
}

// SerializationType represents different serialization formats
type SerializationType string

const (
	SerializationHessian2 SerializationType = "hessian2"
	SerializationJSON     SerializationType = "json"
	SerializationProtoBuf SerializationType = "protobuf"
)

// ProtocolType represents different protocol types
type ProtocolType string

const (
	ProtocolTriple ProtocolType = "triple"
	ProtocolDubbo  ProtocolType = "dubbo"
	ProtocolHTTP   ProtocolType = "http"
	ProtocolGRPC   ProtocolType = "grpc"
)

// ConvertRequest represents a protocol conversion request
type ConvertRequest struct {
	SourceProtocol      ProtocolType      `json:"source_protocol"`
	TargetProtocol      ProtocolType      `json:"target_protocol"`
	SourceSerialization SerializationType `json:"source_serialization"`
	TargetSerialization SerializationType `json:"target_serialization"`
	MethodName          string            `json:"method_name"`
	ParamTypes          []string          `json:"param_types"`
	Args                []interface{}     `json:"args"`
}

// ConvertResponse represents a protocol conversion response
type ConvertResponse struct {
	Success             bool              `json:"success"`
	Result              interface{}       `json:"result"`
	Error               string            `json:"error,omitempty"`
	SourceProtocol      ProtocolType      `json:"source_protocol"`
	TargetProtocol      ProtocolType      `json:"target_protocol"`
	SourceSerialization SerializationType `json:"source_serialization"`
	TargetSerialization SerializationType `json:"target_serialization"`
	Duration            string            `json:"duration"`
}

// NewProtocolConverter creates a new protocol converter
func NewProtocolConverter(providerURL, serviceName string) (*ProtocolConverter, error) {
	converter := &ProtocolConverter{
		providerURL: providerURL,
		serviceName: serviceName,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	if err := converter.initializeConnections(); err != nil {
		return nil, fmt.Errorf("failed to initialize connections: %v", err)
	}

	return converter, nil
}

// initializeConnections sets up connections for different protocols and serializations
func (pc *ProtocolConverter) initializeConnections() error {
	// Create client
	cli, err := client.NewClient(
		client.WithClientURL(pc.providerURL),
		client.WithClientProtocolTriple(),
	)
	if err != nil {
		return fmt.Errorf("create client failed: %v", err)
	}

	// Triple with Hessian2 serialization
	pc.tripleHessianConn, err = cli.Dial(pc.serviceName,
		client.WithGeneric(),
		client.WithSerialization(constant.Hessian2Serialization),
	)
	if err != nil {
		return fmt.Errorf("dial hessian2 failed: %v", err)
	}

	// Triple with JSON serialization
	pc.tripleJSONConn, err = cli.Dial(pc.serviceName,
		client.WithGeneric(),
		client.WithSerialization(constant.JSONSerialization),
	)
	if err != nil {
		return fmt.Errorf("dial json failed: %v", err)
	}

	return nil
}

// Convert performs protocol and serialization conversion
func (pc *ProtocolConverter) Convert(req *ConvertRequest) *ConvertResponse {
	start := time.Now()
	
	response := &ConvertResponse{
		SourceProtocol:      req.SourceProtocol,
		TargetProtocol:      req.TargetProtocol,
		SourceSerialization: req.SourceSerialization,
		TargetSerialization: req.TargetSerialization,
	}

	// Step 1: Convert input data to internal format
	internalArgs, err := pc.convertToInternalFormat(req.Args, req.SourceSerialization)
	if err != nil {
		response.Error = fmt.Sprintf("input conversion failed: %v", err)
		response.Duration = time.Since(start).String()
		return response
	}

	// Step 2: Make the call using target protocol
	result, err := pc.makeCall(req.TargetProtocol, req.TargetSerialization, req.MethodName, req.ParamTypes, internalArgs)
	if err != nil {
		response.Error = fmt.Sprintf("call failed: %v", err)
		response.Duration = time.Since(start).String()
		return response
	}

	// Step 3: Convert result to target format
	convertedResult, err := pc.convertFromInternalFormat(result, req.TargetSerialization)
	if err != nil {
		response.Error = fmt.Sprintf("output conversion failed: %v", err)
		response.Duration = time.Since(start).String()
		return response
	}

	response.Success = true
	response.Result = convertedResult
	response.Duration = time.Since(start).String()
	return response
}

// convertToInternalFormat converts input data from source serialization to internal format
func (pc *ProtocolConverter) convertToInternalFormat(args []interface{}, sourceSerialization SerializationType) ([]interface{}, error) {
	switch sourceSerialization {
	case SerializationHessian2:
		return pc.convertFromHessian2(args)
	case SerializationJSON:
		return pc.convertFromJSON(args)
	case SerializationProtoBuf:
		return pc.convertFromProtoBuf(args)
	default:
		return args, nil // Assume already in internal format
	}
}

// convertFromInternalFormat converts result from internal format to target serialization
func (pc *ProtocolConverter) convertFromInternalFormat(result interface{}, targetSerialization SerializationType) (interface{}, error) {
	switch targetSerialization {
	case SerializationHessian2:
		return pc.convertToHessian2(result)
	case SerializationJSON:
		return pc.convertToJSON(result)
	case SerializationProtoBuf:
		return pc.convertToProtoBuf(result)
	default:
		return result, nil // Return as-is
	}
}

// makeCall executes the actual RPC call using the specified protocol
func (pc *ProtocolConverter) makeCall(protocol ProtocolType, serialization SerializationType, methodName string, paramTypes []string, args []interface{}) (interface{}, error) {
	switch protocol {
	case ProtocolTriple:
		return pc.makeTripleCall(serialization, methodName, paramTypes, args)
	case ProtocolHTTP:
		return pc.makeHTTPCall(methodName, args)
	case ProtocolGRPC:
		// Triple is compatible with gRPC, so use Triple connection
		return pc.makeTripleCall(serialization, methodName, paramTypes, args)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// makeTripleCall executes a Triple protocol call
func (pc *ProtocolConverter) makeTripleCall(serialization SerializationType, methodName string, paramTypes []string, args []interface{}) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var conn client.Connection
	switch serialization {
	case SerializationHessian2:
		conn = pc.tripleHessianConn
	case SerializationJSON:
		conn = pc.tripleJSONConn
	default:
		conn = pc.tripleHessianConn // Default to Hessian2
	}

	var reply interface{}
	err := conn.CallUnary(ctx, []interface{}{methodName, paramTypes, args}, &reply, "$invoke")
	return reply, err
}

// makeHTTPCall executes an HTTP REST call (conversion from RPC to HTTP)
func (pc *ProtocolConverter) makeHTTPCall(methodName string, args []interface{}) (interface{}, error) {
	// Convert RPC call to HTTP REST call
	httpURL := pc.convertRPCToHTTPURL(methodName, args)
	
	req, err := http.NewRequest("GET", httpURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request failed: %v", err)
	}

	resp, err := pc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	// For demo purposes, return a mock result
	return map[string]interface{}{
		"status": "success",
		"method": methodName,
		"data":   fmt.Sprintf("HTTP result for %s", methodName),
	}, nil
}

// convertRPCToHTTPURL converts RPC call parameters to HTTP URL
func (pc *ProtocolConverter) convertRPCToHTTPURL(methodName string, args []interface{}) string {
	baseURL := "http://127.0.0.1:8080/api"
	
	// Convert method name to REST endpoint
	endpoint := strings.ToLower(methodName)
	
	// Add parameters as query params
	params := ""
	if len(args) > 0 {
		if str, ok := args[0].(string); ok {
			params = fmt.Sprintf("?param=%s", str)
		}
	}
	
	return fmt.Sprintf("%s/%s%s", baseURL, endpoint, params)
}

// Serialization conversion methods

func (pc *ProtocolConverter) convertFromHessian2(args []interface{}) ([]interface{}, error) {
	// Convert Hessian2 objects to Go native types
	converted := make([]interface{}, len(args))
	for i, arg := range args {
		if hessianObj, ok := arg.(*hessian.Object); ok {
			// Convert Hessian2 object to map
			converted[i] = hessianObj
		} else {
			converted[i] = arg
		}
	}
	return converted, nil
}

func (pc *ProtocolConverter) convertToHessian2(result interface{}) (interface{}, error) {
	// Convert result to Hessian2 compatible format
	return result, nil // Simplified for demo
}

func (pc *ProtocolConverter) convertFromJSON(args []interface{}) ([]interface{}, error) {
	// Convert JSON strings to Go objects
	converted := make([]interface{}, len(args))
	for i, arg := range args {
		if jsonStr, ok := arg.(string); ok {
			var obj interface{}
			if err := json.Unmarshal([]byte(jsonStr), &obj); err == nil {
				converted[i] = obj
			} else {
				converted[i] = arg
			}
		} else {
			converted[i] = arg
		}
	}
	return converted, nil
}

func (pc *ProtocolConverter) convertToJSON(result interface{}) (interface{}, error) {
	// Convert result to JSON format
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return string(jsonBytes), nil
}

func (pc *ProtocolConverter) convertFromProtoBuf(args []interface{}) ([]interface{}, error) {
	// Convert Protocol Buffer messages to Go objects
	// This would require specific protobuf message types
	return args, nil // Simplified for demo
}

func (pc *ProtocolConverter) convertToProtoBuf(result interface{}) (interface{}, error) {
	// Convert result to Protocol Buffer format
	// This would require specific protobuf message types
	return result, nil // Simplified for demo
}

// Protocol Gateway - HTTP server for protocol conversion
type ProtocolGateway struct {
	converter *ProtocolConverter
	server    *http.Server
}

// NewProtocolGateway creates a new protocol gateway
func NewProtocolGateway(converter *ProtocolConverter, port int) *ProtocolGateway {
	gateway := &ProtocolGateway{
		converter: converter,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/convert", gateway.handleConvert)
	mux.HandleFunc("/health", gateway.handleHealth)

	gateway.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return gateway
}

// handleConvert handles protocol conversion requests
func (pg *ProtocolGateway) handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	response := pg.converter.Convert(&req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (pg *ProtocolGateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Start starts the protocol gateway server
func (pg *ProtocolGateway) Start() error {
	fmt.Printf("🌉 Protocol Gateway starting on %s\n", pg.server.Addr)
	return pg.server.ListenAndServe()
}

// Stop stops the protocol gateway server
func (pg *ProtocolGateway) Stop() error {
	return pg.server.Shutdown(context.Background())
}

func main() {
	const (
		providerURL = "tri://127.0.0.1:50051/com.example.DemoService"
		serviceName = "com.example.DemoService"
		gatewayPort = 8080
	)

	fmt.Println("=== Protocol Converter Demo ===")

	// Create protocol converter
	converter, err := NewProtocolConverter(providerURL, serviceName)
	if err != nil {
		log.Fatalf("Failed to create converter: %v", err)
	}
	fmt.Println("✅ Protocol converter initialized")

	// Demo: Different protocol conversion scenarios
	fmt.Println("\n=== Protocol Conversion Examples ===")

	// Example 1: Triple Hessian2 to Triple JSON
	req1 := &ConvertRequest{
		SourceProtocol:      ProtocolTriple,
		TargetProtocol:      ProtocolTriple,
		SourceSerialization: SerializationHessian2,
		TargetSerialization: SerializationJSON,
		MethodName:          "Hello",
		ParamTypes:          []string{"java.lang.String"},
		Args:                []interface{}{"World"},
	}

	resp1 := converter.Convert(req1)
	fmt.Printf("✅ Example 1 - Triple Hessian2 → Triple JSON:\n")
	fmt.Printf("   Success: %v, Duration: %s\n", resp1.Success, resp1.Duration)
	if resp1.Success {
		fmt.Printf("   Result: %v\n", resp1.Result)
	} else {
		fmt.Printf("   Error: %s\n", resp1.Error)
	}

	// Example 2: Triple to HTTP conversion
	req2 := &ConvertRequest{
		SourceProtocol:      ProtocolTriple,
		TargetProtocol:      ProtocolHTTP,
		SourceSerialization: SerializationHessian2,
		TargetSerialization: SerializationJSON,
		MethodName:          "GetUserInfo",
		ParamTypes:          []string{"java.lang.String"},
		Args:                []interface{}{"user123"},
	}

	resp2 := converter.Convert(req2)
	fmt.Printf("\n✅ Example 2 - Triple → HTTP REST:\n")
	fmt.Printf("   Success: %v, Duration: %s\n", resp2.Success, resp2.Duration)
	if resp2.Success {
		fmt.Printf("   Result: %v\n", resp2.Result)
	} else {
		fmt.Printf("   Error: %s\n", resp2.Error)
	}

	// Example 3: Different serialization conversion
	req3 := &ConvertRequest{
		SourceProtocol:      ProtocolTriple,
		TargetProtocol:      ProtocolTriple,
		SourceSerialization: SerializationJSON,
		TargetSerialization: SerializationHessian2,
		MethodName:          "Add",
		ParamTypes:          []string{"int", "int"},
		Args:                []interface{}{int32(15), int32(25)},
	}

	resp3 := converter.Convert(req3)
	fmt.Printf("\n✅ Example 3 - Serialization Conversion (JSON → Hessian2):\n")
	fmt.Printf("   Success: %v, Duration: %s\n", resp3.Success, resp3.Duration)
	if resp3.Success {
		fmt.Printf("   Result: %v\n", resp3.Result)
	} else {
		fmt.Printf("   Error: %s\n", resp3.Error)
	}

	// Start Protocol Gateway
	fmt.Printf("\n=== Starting Protocol Gateway ===\n")
	gateway := NewProtocolGateway(converter, gatewayPort)

	fmt.Println("🌐 Protocol Gateway provides HTTP API for protocol conversion:")
	fmt.Printf("   POST http://localhost:%d/convert - Convert protocols\n", gatewayPort)
	fmt.Printf("   GET  http://localhost:%d/health  - Health check\n", gatewayPort)
	fmt.Println("\n📝 Example curl command:")
	fmt.Printf(`curl -X POST http://localhost:%d/convert \
  -H "Content-Type: application/json" \
  -d '{
    "source_protocol": "triple",
    "target_protocol": "triple", 
    "source_serialization": "hessian2",
    "target_serialization": "json",
    "method_name": "Hello",
    "param_types": ["java.lang.String"],
    "args": ["World"]
  }'`, gatewayPort)

	fmt.Println("\n\n🚀 Gateway starting... (Ctrl+C to stop)")
	if err := gateway.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Gateway failed: %v", err)
	}
}

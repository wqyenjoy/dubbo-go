# Triple Generic Call API Documentation

This document provides detailed API documentation for the Triple Protocol Generic Call functionality in Dubbo-Go.

## Core Interfaces

### GenericService

The `GenericService` interface enables generic invocation without predefined service interfaces.

```go
type GenericService struct {
    Invoke func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) `dubbo:"$invoke"`
    referenceStr string
}
```

#### Methods

- `NewGenericService(referenceStr string) *GenericService`: Creates a new generic service instance
- `Reference() string`: Returns the service reference string

### Client API

#### Creating a Generic Client

```go
// Create client
cli, err := client.NewClient(
    client.WithClientURL("tri://127.0.0.1:50051/com.example.DemoService"),
    client.WithClientProtocolTriple(),
)

// Create connection with generic support
conn, err := cli.Dial("com.example.DemoService",
    client.WithGeneric(),
    client.WithSerialization(constant.Hessian2Serialization),
)
```

#### Making Generic Calls

```go
var reply interface{}
err := conn.CallUnary(ctx, []interface{}{methodName, paramTypes, args}, &reply, "$invoke")
```

**Parameters:**
- `ctx`: Context for timeout and cancellation
- `methodName`: Target method name (string)
- `paramTypes`: Parameter type descriptions ([]string)
- `args`: Method arguments ([]interface{})
- `reply`: Pointer to store the result
- `"$invoke"`: Generic method identifier

### Server API

#### Service Registration

```go
srv, err := server.NewServer(
    server.WithServerProtocol(
        protocol.WithTriple(),
        protocol.WithIp("127.0.0.1"),
        protocol.WithPort(50051),
    ),
    server.WithServerSerialization(constant.Hessian2Serialization),
    server.SetServerApplication(&global.ApplicationConfig{
        Name:                    "generic-provider",
        MetadataServiceProtocol: "file", // Required for generic calls
    }),
)

// Register service
err = srv.RegisterService(&DemoService{}, 
    server.WithSerialization(constant.Hessian2Serialization))
```

## Type Mapping

### Go to Java Type Mapping

| Go Type | Java Type | Parameter String | Example |
|---------|-----------|------------------|---------|
| `string` | `java.lang.String` | `"java.lang.String"` | `"hello"` |
| `int32` | `int` | `"int"` | `123` |
| `int64` | `long` | `"long"` | `123456789L` |
| `float64` | `double` | `"double"` | `3.14` |
| `bool` | `boolean` | `"boolean"` | `true` |
| `[]interface{}` | `java.util.List` | `"java.util.List"` | `["a", "b", "c"]` |
| `map[string]interface{}` | `java.util.Map` | `"java.util.Map"` | `{"key": "value"}` |

### Complex Type Examples

#### List Processing
```go
args := []interface{}{
    []interface{}{"item1", "item2", "item3"},
}
paramTypes := []string{"java.util.List"}
```

#### Map Processing
```go
args := []interface{}{
    map[string]interface{}{
        "name": "John",
        "age": 30,
        "active": true,
    },
}
paramTypes := []string{"java.util.Map"}
```

#### Nested Complex Types
```go
args := []interface{}{
    map[string]interface{}{
        "user": map[string]interface{}{
            "id": "123",
            "profile": map[string]interface{}{
                "name": "John",
                "preferences": []interface{}{"pref1", "pref2"},
            },
        },
        "metadata": map[string]interface{}{
            "timestamp": time.Now().Unix(),
            "source": "api",
        },
    },
}
paramTypes := []string{"java.util.Map"}
```

## Configuration Options

### Server Configuration

#### Basic Configuration
```go
srv, err := server.NewServer(
    server.WithServerProtocol(
        protocol.WithTriple(),
        protocol.WithIp("0.0.0.0"),           // Listen address
        protocol.WithPort(50051),              // Listen port
    ),
    server.WithServerSerialization(constant.Hessian2Serialization),
    server.SetServerApplication(&global.ApplicationConfig{
        Name:                    "my-provider",
        Version:                 "1.0.0",
        MetadataServiceProtocol: "file",       // Required for generic
    }),
    server.WithServerNotRegister(),           // Skip registry registration
)
```

#### Registry Configuration
```go
server.SetServerRegistries(&global.RegistryConfig{
    Protocol: "nacos",
    Address:  "127.0.0.1:8848",
    Group:    "dubbo",
    Namespace: "public",
})
```

### Client Configuration

#### Basic Configuration
```go
cli, err := client.NewClient(
    client.WithClientURL("tri://127.0.0.1:50051/com.example.DemoService"),
    client.WithClientProtocolTriple(),
    client.WithClientSerialization(constant.Hessian2Serialization),
)
```

#### Connection Options
```go
conn, err := cli.Dial("com.example.DemoService",
    client.WithGeneric(),                     // Enable generic calls
    client.WithSerialization(constant.Hessian2Serialization),
    client.WithConnectionTimeout(5*time.Second),
    client.WithRequestTimeout(3*time.Second),
)
```

## Error Handling

### Common Error Types

#### Connection Errors
```go
if err != nil {
    if strings.Contains(err.Error(), "connection refused") {
        // Provider not available
        return handleProviderUnavailable()
    }
    if strings.Contains(err.Error(), "timeout") {
        // Network timeout
        return handleTimeout()
    }
}
```

#### Method Call Errors
```go
if err != nil {
    if strings.Contains(err.Error(), "method not found") {
        // Invalid method name
        return handleMethodNotFound()
    }
    if strings.Contains(err.Error(), "argument type mismatch") {
        // Type conversion error
        return handleTypeMismatch()
    }
}
```

### Error Response Handling
```go
result, err := genericCall(conn, "Hello", types, args)
if err != nil {
    // Check for specific error patterns
    switch {
    case strings.Contains(err.Error(), "timeout"):
        log.Printf("Call timeout: %v", err)
        return nil, fmt.Errorf("service timeout")
    case strings.Contains(err.Error(), "not found"):
        log.Printf("Method not found: %v", err)
        return nil, fmt.Errorf("invalid method")
    default:
        log.Printf("Generic call failed: %v", err)
        return nil, err
    }
}
```

## Performance Optimization

### Connection Pooling
```go
// Reuse connections for better performance
var conn client.Connection
var connOnce sync.Once

func getConnection() client.Connection {
    connOnce.Do(func() {
        cli, _ := client.NewClient(...)
        conn, _ = cli.Dial(...)
    })
    return conn
}
```

### Batch Operations
```go
// Process multiple calls concurrently
type BatchRequest struct {
    Method string
    Types  []string
    Args   []interface{}
}

func processBatch(conn client.Connection, requests []BatchRequest) []interface{} {
    results := make([]interface{}, len(requests))
    var wg sync.WaitGroup
    
    for i, req := range requests {
        wg.Add(1)
        go func(index int, request BatchRequest) {
            defer wg.Done()
            result, _ := genericCall(conn, request.Method, request.Types, request.Args)
            results[index] = result
        }(i, req)
    }
    
    wg.Wait()
    return results
}
```

### Timeout Management
```go
// Use appropriate timeouts for different operations
func callWithTimeout(conn client.Connection, timeout time.Duration) (interface{}, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    var reply interface{}
    err := conn.CallUnary(ctx, []interface{}{method, types, args}, &reply, "$invoke")
    return reply, err
}
```

## Advanced Features

### Async Calls
```go
func callAsync(conn client.Connection) <-chan AsyncResult {
    resultChan := make(chan AsyncResult, 1)
    
    go func() {
        defer close(resultChan)
        result, err := genericCall(conn, method, types, args)
        resultChan <- AsyncResult{Result: result, Error: err}
    }()
    
    return resultChan
}
```

### Retry Mechanism
```go
func callWithRetry(conn client.Connection, maxRetries int) (interface{}, error) {
    var lastErr error
    
    for i := 0; i <= maxRetries; i++ {
        result, err := genericCall(conn, method, types, args)
        if err == nil {
            return result, nil
        }
        lastErr = err
        
        if i < maxRetries {
            time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
        }
    }
    
    return nil, fmt.Errorf("call failed after %d retries: %v", maxRetries+1, lastErr)
}
```

### Circuit Breaker Pattern
```go
type CircuitBreaker struct {
    failures    int
    lastFailure time.Time
    timeout     time.Duration
}

func (cb *CircuitBreaker) Call(fn func() (interface{}, error)) (interface{}, error) {
    if cb.failures >= 3 && time.Since(cb.lastFailure) < cb.timeout {
        return nil, fmt.Errorf("circuit breaker open")
    }
    
    result, err := fn()
    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        return nil, err
    }
    
    cb.failures = 0
    return result, nil
}
```

## Best Practices

### 1. Connection Management
- Reuse connections when possible
- Implement proper connection pooling for high-throughput scenarios
- Set appropriate timeouts for different operations

### 2. Error Handling
- Always check for errors and handle them appropriately
- Implement retry logic for transient failures
- Use circuit breakers for resilience

### 3. Type Safety
- Use consistent type mapping between Go and Java
- Validate parameter types before making calls
- Handle type conversion errors gracefully

### 4. Performance
- Use concurrent calls for independent operations
- Implement proper monitoring and metrics
- Optimize serialization format based on requirements

### 5. Security
- Validate input parameters to prevent injection attacks
- Implement proper authentication and authorization
- Use secure transport (TLS) in production environments

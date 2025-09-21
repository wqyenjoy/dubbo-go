# API Documentation

## Core Interfaces

### GenericService

```go
type GenericService struct {
    Invoke func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) `dubbo:"$invoke"`
    referenceStr string
}
```

### Client API

#### Creating a Generic Client

```go
cli, err := client.NewClient(
    client.WithClientURL("tri://127.0.0.1:50051/com.example.DemoService"),
    client.WithClientProtocolTriple(),
)

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
- `methodName`: Target method name
- `paramTypes`: Parameter type descriptions
- `args`: Method arguments
- `reply`: Pointer to store the result
- `"$invoke"`: Generic method identifier

### Server API

```go
srv, err := server.NewServer(
    server.WithServerProtocol(
        protocol.WithTriple(),
        protocol.WithIp("127.0.0.1"),
        protocol.WithPort(50051),
    ),
    server.WithServerSerialization(constant.Hessian2Serialization),
    server.SetServerApplication(&global.ApplicationConfig{
        Name: "generic-provider",
        MetadataServiceProtocol: "file", // Required for generic calls
    }),
)

err = srv.RegisterService(&DemoService{}, 
    server.WithSerialization(constant.Hessian2Serialization))
```

## Type Mapping

| Go Type | Java Type | Parameter String | Example |
|---------|-----------|------------------|---------|
| string | java.lang.String | "java.lang.String" | "hello" |
| int32 | int | "int" | 123 |
| int64 | long | "long" | 123456789L |
| float64 | double | "double" | 3.14 |
| bool | boolean | "boolean" | true |
| []interface{} | java.util.List | "java.util.List" | ["a", "b", "c"] |
| map[string]interface{} | java.util.Map | "java.util.Map" | {"key": "value"} |

## Configuration Options

### Server Configuration

```go
srv, err := server.NewServer(
    server.WithServerProtocol(
        protocol.WithTriple(),
        protocol.WithIp("0.0.0.0"),
        protocol.WithPort(50051),
    ),
    server.WithServerSerialization(constant.Hessian2Serialization),
    server.SetServerApplication(&global.ApplicationConfig{
        Name: "my-provider",
        Version: "1.0.0",
        MetadataServiceProtocol: "file",
    }),
    server.WithServerNotRegister(),
)
```

### Client Configuration

```go
conn, err := cli.Dial("com.example.DemoService",
    client.WithGeneric(),
    client.WithSerialization(constant.Hessian2Serialization),
    client.WithConnectionTimeout(5*time.Second),
    client.WithRequestTimeout(3*time.Second),
)
```

## Error Handling

### Common Error Types

```go
if err != nil {
    if strings.Contains(err.Error(), "connection refused") {
        // Provider not available
    }
    if strings.Contains(err.Error(), "timeout") {
        // Network timeout
    }
    if strings.Contains(err.Error(), "method not found") {
        // Invalid method name
    }
}
```

## Performance Optimization

### Connection Pooling

```go
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

### Timeout Management

```go
func callWithTimeout(conn client.Connection, timeout time.Duration) (interface{}, error) {
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    var reply interface{}
    err := conn.CallUnary(ctx, []interface{}{method, types, args}, &reply, "$invoke")
    return reply, err
}
```
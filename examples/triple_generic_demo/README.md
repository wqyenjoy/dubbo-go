# Triple Protocol Generic Call Demo

This demo demonstrates Triple Protocol Generic Call functionality in Dubbo-Go, enabling dynamic service invocation without predefined interfaces.

## Overview

Generic calls allow runtime service invocation without:
- Service interface definitions
- Generated stub code
- Compile-time dependencies

Use cases include API gateways, service mesh, dynamic service composition, and testing tools.

## Features

- Complete Triple Protocol Support
- Multiple data types (primitives, collections, maps, complex objects)
- Multiple serialization formats (Hessian2, JSON)
- Protocol conversion (Triple, HTTP, gRPC)
- Async calls, batch operations, retry mechanisms
- Comprehensive error handling
- Performance optimization
- Production-ready monitoring

## Project Structure

```
examples/triple_generic_demo/
├── provider/main.go                    # Service provider
├── consumer/main.go                    # Basic consumer examples
├── consumer/advanced_examples.go       # Advanced usage patterns
├── benchmarks/                         # Performance tests
├── protocol_converter/                 # Protocol conversion tools
└── docs/                              # Documentation
```

## Quick Start

### 1. Start Provider

```bash
cd examples/triple_generic_demo/provider
go run main.go
```

### 2. Run Consumer

```bash
cd examples/triple_generic_demo/consumer
go run main.go
```

## Basic Usage

### Creating a Generic Client

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

### Making Generic Calls

```go
var reply interface{}
err := conn.CallUnary(ctx, []interface{}{methodName, paramTypes, args}, &reply, "$invoke")
```

## Examples

### Simple String Method

```go
result, err := caller.Call("Hello", 
    []string{"java.lang.String"}, 
    []interface{}{"World"})
```

### Math Operations

```go
result, err := caller.Call("Add",
    []string{"int", "int"},
    []interface{}{int32(15), int32(27)})
```

### Complex Object Return

```go
result, err := caller.Call("GetUserInfo",
    []string{"java.lang.String"},
    []interface{}{"user123"})
```

## Type Mapping

| Go Type | Java Type | Parameter String |
|---------|-----------|------------------|
| string | java.lang.String | "java.lang.String" |
| int32 | int | "int" |
| int64 | long | "long" |
| bool | boolean | "boolean" |
| []interface{} | java.util.List | "java.util.List" |
| map[string]interface{} | java.util.Map | "java.util.Map" |

## Configuration

### Server

```go
srv, err := server.NewServer(
    server.WithServerProtocol(
        protocol.WithTriple(),
        protocol.WithIp("127.0.0.1"),
        protocol.WithPort(50051),
    ),
    server.WithServerSerialization(constant.Hessian2Serialization),
    server.SetServerApplication(&global.ApplicationConfig{
        MetadataServiceProtocol: "file", // Required for generic calls
    }),
)
```

### Client

```go
conn, err := cli.Dial("com.example.DemoService",
    client.WithGeneric(),
    client.WithSerialization(constant.Hessian2Serialization),
)
```

## Testing

```bash
# Run benchmarks
cd benchmarks
go test -bench=. -v

# Test protocol converter
cd protocol_converter
./test_converter.sh
```

## Documentation

- [API Documentation](docs/API.md)
- [Troubleshooting Guide](docs/TROUBLESHOOTING.md)
- [Protocol Converter](protocol_converter/README.md)

## License

Apache License 2.0
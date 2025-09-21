# Protocol Converter

Multi-protocol converter for Triple Protocol Generic Call functionality.

## Overview

The protocol converter enables seamless conversion between different protocols and serialization formats:

- **Protocols**: Triple, HTTP, gRPC
- **Serialization**: Hessian2, JSON, Protocol Buffers

## Usage

### Starting the Converter

```bash
go run converter.go
```

### Making Conversion Requests

```bash
curl -X POST http://localhost:8080/convert \
  -H "Content-Type: application/json" \
  -d '{
    "source_protocol": "triple",
    "target_protocol": "http",
    "source_serialization": "hessian2",
    "target_serialization": "json",
    "method_name": "Hello",
    "param_types": ["java.lang.String"],
    "args": ["World"]
  }'
```

## API Endpoints

- `POST /convert` - Convert between protocols
- `GET /health` - Health check

## Configuration

The converter supports various protocol combinations:

### Protocol Conversion

| Source | Target | Description |
|--------|--------|-------------|
| Triple | HTTP | RPC to REST conversion |
| Triple | gRPC | Direct compatibility |
| HTTP | Triple | REST to RPC conversion |

### Serialization Conversion

| Source | Target | Use Case |
|--------|--------|----------|
| Hessian2 | JSON | Binary to text format |
| JSON | Hessian2 | Text to binary format |
| ProtoBuf | JSON | Cross-language compatibility |

## Testing

```bash
# Run automated tests
./test_converter.sh

# Run unit tests
go test -v

# Run usage demo
go run usage_demo.go
```

## Examples

See `usage_demo.go` for comprehensive examples of different conversion scenarios.
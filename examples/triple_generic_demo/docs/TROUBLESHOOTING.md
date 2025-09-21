# Troubleshooting Guide

## Connection Issues

### "connection refused"

**Causes:**
- Provider server not running
- Wrong IP address or port
- Firewall blocking connection

**Solutions:**
```bash
# Check if provider is running
netstat -an | grep :50051

# Test connectivity
telnet 127.0.0.1 50051
```

**Code fix:**
```go
// Bind to all interfaces
server.WithServerProtocol(
    protocol.WithTriple(),
    protocol.WithIp("0.0.0.0"),  // Instead of "127.0.0.1"
    protocol.WithPort(50051),
)
```

### "connection timeout"

**Solutions:**
```go
// Increase timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Or set global timeout
client.WithConnectionTimeout(10*time.Second)
```

## Method Call Issues

### "method not found"

**Causes:**
- Method name spelling error
- Method not registered
- Case sensitivity mismatch

**Solutions:**
```go
// Check exact method name
func (s *MyService) HelloWorld(ctx context.Context, name string) (string, error) {
    return "Hello " + name, nil
}

// Correct call (case sensitive)
result, err := conn.CallUnary(ctx, 
    []interface{}{"HelloWorld", []string{"java.lang.String"}, []interface{}{"World"}}, 
    &reply, "$invoke")
```

### "argument type mismatch"

**Solutions:**
```go
// Wrong
result, err := conn.CallUnary(ctx, 
    []interface{}{"Add", []string{"java.lang.String"}, []interface{}{10, 20}}, // Wrong types
    &reply, "$invoke")

// Correct
result, err := conn.CallUnary(ctx, 
    []interface{}{"Add", []string{"int", "int"}, []interface{}{int32(10), int32(20)}}, 
    &reply, "$invoke")
```

## Serialization Issues

### "hessian2 serialization failed"

**Solutions:**
```go
// Use basic types for complex data
complexData := map[string]interface{}{
    "id":   "12345",
    "name": "John Doe",
    "data": []interface{}{"a", "b", "c"},
    // Avoid: circular references, channels, functions
}

// Alternative: Use JSON serialization
client.WithSerialization(constant.JSONSerialization)
```

## Configuration Issues

### "MetadataService not found"

**Solution:**
```go
// Must set MetadataServiceProtocol
server.SetServerApplication(&global.ApplicationConfig{
    Name: "my-provider",
    MetadataServiceProtocol: "file", // Required for generic calls
})
```

## Performance Issues

### Slow calls

**Diagnosis:**
```go
start := time.Now()
result, err := genericCall(conn, method, types, args)
duration := time.Since(start)
log.Printf("Call took: %v", duration)
```

**Solutions:**
1. Reuse connections
2. Use efficient serialization (Hessian2)
3. Implement connection pooling

## Debugging

### Enable Debug Logging

```go
import "github.com/dubbogo/gost/log/logger"
logger.SetLoggerLevel(logger.DEBUG)
```

### Network Analysis

```bash
# Capture traffic
tcpdump -i any -s 0 -w dubbo_traffic.pcap port 50051

# Test with simple cases
result, err := conn.CallUnary(ctx, 
    []interface{}{"Hello", []string{"java.lang.String"}, []interface{}{"test"}}, 
    &reply, "$invoke")
```

## Environment Issues

### macOS

```bash
# Check localhost resolution
nslookup localhost

# Disable firewall temporarily
sudo pfctl -d
```

### Docker

```yaml
# Ensure port mapping
ports:
  - "50051:50051"
```

## Getting Help

When reporting issues, provide:
1. Complete error message
2. Client and server configuration
3. Network setup details
4. Go version and OS
5. Debug logs (sanitized)

## Useful Commands

```bash
# Get Go version
go version

# Check network
netstat -an | grep :50051

# Test connectivity
curl -v telnet://127.0.0.1:50051
```
# Troubleshooting Guide for Triple Generic Calls

This guide helps you diagnose and resolve common issues when using Triple Protocol Generic Calls in Dubbo-Go.

## Common Issues and Solutions

### 1. Connection Issues

#### Problem: "connection refused"
```
Error: dial failed: connection refused
```

**Causes:**
- Provider server is not running
- Wrong IP address or port
- Firewall blocking the connection
- Provider bound to different interface (localhost vs 0.0.0.0)

**Solutions:**
```bash
# Check if provider is running
netstat -an | grep :50051

# Test connectivity
telnet 127.0.0.1 50051

# Check provider logs for binding address
```

**Code fix:**
```go
// Provider: Bind to all interfaces
server.WithServerProtocol(
    protocol.WithTriple(),
    protocol.WithIp("0.0.0.0"),  // Instead of "127.0.0.1"
    protocol.WithPort(50051),
)
```

#### Problem: "connection timeout"
```
Error: context deadline exceeded
```

**Causes:**
- Network latency too high
- Provider under heavy load
- Timeout configuration too low

**Solutions:**
```go
// Client: Increase timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Or set global timeout
client.WithConnectionTimeout(10*time.Second)
```

### 2. Method Call Issues

#### Problem: "method not found"
```
Error: method 'MyMethod' not found on service
```

**Causes:**
- Method name spelling error
- Method not registered with the service
- Case sensitivity mismatch

**Solutions:**
```go
// Check exact method name from provider
func (s *MyService) HelloWorld(ctx context.Context, name string) (string, error) {
    return "Hello " + name, nil
}

// Correct call (case sensitive)
result, err := conn.CallUnary(ctx, 
    []interface{}{"HelloWorld", []string{"java.lang.String"}, []interface{}{"World"}}, 
    &reply, "$invoke")
```

**Debug tip:**
```go
// Enable debug logging to see registered methods
import "github.com/dubbogo/gost/log"
log.SetLoggerLevel(log.DEBUG)
```

#### Problem: "argument type mismatch"
```
Error: cannot convert argument type
```

**Causes:**
- Incorrect parameter type specification
- Go type doesn't match Java type expectation
- Wrong number of parameters

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

**Type mapping reference:**
```go
// Go -> Java type mappings
map[string]string{
    "string":                "java.lang.String",
    "int32":                 "int", 
    "int64":                 "long",
    "float64":               "double",
    "bool":                  "boolean",
    "[]interface{}":         "java.util.List",
    "map[string]interface{}": "java.util.Map",
}
```

### 3. Serialization Issues

#### Problem: "hessian2 serialization failed"
```
Error: hessian2: cannot encode type
```

**Causes:**
- Unsupported data type
- Circular references in objects
- Complex nested structures

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

#### Problem: "deserialization error on response"
```
Error: cannot decode response type
```

**Causes:**
- Response type doesn't match expected format
- Provider returns unsupported type

**Solutions:**
```go
// Use interface{} to receive any type
var reply interface{}
err := conn.CallUnary(ctx, args, &reply, "$invoke")

// Then type assert as needed
if result, ok := reply.(string); ok {
    fmt.Printf("String result: %s\n", result)
} else if result, ok := reply.(map[string]interface{}); ok {
    fmt.Printf("Map result: %v\n", result)
}
```

### 4. Configuration Issues

#### Problem: "MetadataService not found"
```
Error: service metadata not available
```

**Causes:**
- Missing MetadataServiceProtocol configuration
- Registry configuration issue

**Solution:**
```go
// Provider: Must set MetadataServiceProtocol
server.SetServerApplication(&global.ApplicationConfig{
    Name:                    "my-provider",
    MetadataServiceProtocol: "file", // Required for generic calls
})
```

#### Problem: "$invoke handler not registered"
```
Error: $invoke method handler not found
```

**Causes:**
- Generic call support not properly enabled
- Server configuration missing

**Solutions:**
```go
// Ensure server supports generic calls
srv, err := server.NewServer(
    server.WithServerProtocol(protocol.WithTriple()),
    server.WithServerSerialization(constant.Hessian2Serialization),
    // MetadataServiceProtocol is crucial
    server.SetServerApplication(&global.ApplicationConfig{
        MetadataServiceProtocol: "file",
    }),
)
```

### 5. Performance Issues

#### Problem: "calls are very slow"

**Diagnosis:**
```go
// Add timing to identify bottlenecks
start := time.Now()
result, err := genericCall(conn, method, types, args)
duration := time.Since(start)
log.Printf("Call took: %v", duration)
```

**Common causes and solutions:**

1. **Connection overhead**
```go
// Reuse connections
var globalConn client.Connection
var connOnce sync.Once

func getConnection() client.Connection {
    connOnce.Do(func() {
        cli, _ := client.NewClient(...)
        globalConn, _ = cli.Dial(...)
    })
    return globalConn
}
```

2. **Serialization overhead**
```go
// Use more efficient serialization
client.WithSerialization(constant.Hessian2Serialization) // Generally faster than JSON
```

3. **Network latency**
```go
// Use connection pooling for high throughput
// Implement client-side connection pooling
type ConnectionPool struct {
    connections []client.Connection
    current     int
    mu          sync.Mutex
}
```

#### Problem: "memory usage growing"

**Diagnosis:**
```go
// Monitor memory usage
import "runtime"

var m runtime.MemStats
runtime.ReadMemStats(&m)
log.Printf("Memory usage: %d KB", m.Alloc/1024)
```

**Solutions:**
```go
// Ensure proper resource cleanup
defer func() {
    if conn != nil {
        // Clean up connections
    }
}()

// Avoid storing large response objects
// Process and discard data promptly
```

### 6. Network and Firewall Issues

#### Problem: "requests hanging/timing out"

**Diagnosis:**
```bash
# Check network connectivity
ping 127.0.0.1
telnet 127.0.0.1 50051

# Check for packet loss
traceroute 127.0.0.1

# Monitor network traffic
tcpdump -i lo0 port 50051
```

**Solutions:**
- Configure appropriate timeouts
- Check firewall rules
- Verify network configuration

### 7. Debugging Techniques

#### Enable Debug Logging
```go
import "github.com/dubbogo/gost/log/logger"

// Set debug level
logger.SetLoggerLevel(logger.DEBUG)

// Add custom logging
log.Printf("Making generic call: method=%s, types=%v, args=%v", method, types, args)
```

#### Use Network Analysis Tools
```bash
# Capture network traffic
tcpdump -i any -s 0 -w dubbo_traffic.pcap port 50051

# Analyze with Wireshark
wireshark dubbo_traffic.pcap
```

#### Test with Simple Cases
```go
// Start with simplest possible call
result, err := conn.CallUnary(ctx, 
    []interface{}{"Hello", []string{"java.lang.String"}, []interface{}{"test"}}, 
    &reply, "$invoke")

if err != nil {
    log.Printf("Even simple call failed: %v", err)
    // This indicates fundamental configuration issue
}
```

### 8. Environment-Specific Issues

#### macOS Issues
```bash
# Check if localhost resolves correctly
nslookup localhost

# macOS might have stricter firewall
sudo pfctl -d  # Disable firewall temporarily for testing
```

#### Linux Issues
```bash
# Check iptables
sudo iptables -L

# Check SELinux (if applicable)
getenforce
```

#### Docker/Container Issues
```yaml
# Ensure proper port mapping in docker-compose.yml
ports:
  - "50051:50051"

# Check container networking
docker network ls
docker network inspect bridge
```

### 9. Production Debugging

#### Health Check Endpoint
```go
// Add health check to your service
func (s *HealthService) Check(ctx context.Context) (string, error) {
    return "OK", nil
}

// Test with
result, err := conn.CallUnary(ctx, 
    []interface{}{"Check", []string{}, []interface{}{}}, 
    &reply, "$invoke")
```

#### Metrics and Monitoring
```go
// Add metrics collection
var (
    callCount    = prometheus.NewCounterVec(...)
    callDuration = prometheus.NewHistogramVec(...)
    errorCount   = prometheus.NewCounterVec(...)
)

// Instrument calls
func instrumentedCall(conn client.Connection, method string, types []string, args []interface{}) (interface{}, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        callDuration.WithLabelValues(method).Observe(duration.Seconds())
        callCount.WithLabelValues(method).Inc()
    }()
    
    result, err := genericCall(conn, method, types, args)
    if err != nil {
        errorCount.WithLabelValues(method, err.Error()).Inc()
    }
    
    return result, err
}
```

## Getting Help

### Log Collection
When reporting issues, please provide:

1. **Complete error message**
2. **Client configuration**
3. **Provider configuration** 
4. **Network setup**
5. **Go version and OS**
6. **Debug logs** (with sensitive data removed)

### Useful Commands
```bash
# Get Go version
go version

# Check network configuration  
ifconfig
netstat -an | grep :50051

# Test basic connectivity
curl -v telnet://127.0.0.1:50051
```

### Community Resources
- [Dubbo-Go GitHub Issues](https://github.com/apache/dubbo-go/issues)
- [Dubbo-Go Documentation](https://dubbo.apache.org/zh/docs3-v2/golang-sdk/)
- [Dubbo Community Slack](https://dubbo.apache.org/community/)

Remember: When in doubt, start with the simplest possible configuration and gradually add complexity while testing each step.

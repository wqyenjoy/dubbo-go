# 协议转换器 (Protocol Converter)

这个模块展示了如何在Triple泛化调用中实现多协议转换，支持不同协议和序列化格式之间的无缝转换。

## 🎯 功能特性

### 📡 支持的协议
- **Triple Protocol**: Dubbo-Go 3.0的核心协议
- **gRPC Protocol**: 与gRPC兼容的调用
- **HTTP/REST**: 将RPC调用转换为HTTP REST API
- **Dubbo Protocol**: 传统Dubbo协议 (通过适配器)

### 🔄 支持的序列化格式
- **Hessian2**: 高效的二进制序列化
- **JSON**: 易于调试的文本格式
- **Protocol Buffers**: Google的结构化数据序列化
- **自定义格式**: 可扩展的序列化支持

## 🏗️ 架构设计

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Source App    │───▶│ Protocol Gateway │───▶│   Target App    │
│  (Any Protocol) │    │   (Converter)    │    │  (Any Protocol) │
└─────────────────┘    └──────────────────┘    └─────────────────┘
        │                        │                        │
        │                        │                        │
   ┌────▼────┐              ┌────▼────┐              ┌────▼────┐
   │Hessian2 │              │Internal │              │  JSON   │
   │ Format  │              │ Format  │              │ Format  │
   └─────────┘              └─────────┘              └─────────┘
```

## 🚀 快速开始

### 1. 启动测试环境

```bash
# Terminal 1: 启动Provider
cd examples/triple_generic_demo/provider
go run main.go

# Terminal 2: 启动Protocol Converter
cd examples/triple_generic_demo/protocol_converter
go run converter.go
```

### 2. 使用协议转换API

```bash
# 使用curl测试协议转换
curl -X POST http://localhost:8080/convert \
  -H "Content-Type: application/json" \
  -d '{
    "source_protocol": "triple",
    "target_protocol": "triple",
    "source_serialization": "hessian2",
    "target_serialization": "json",
    "method_name": "Hello",
    "param_types": ["java.lang.String"],
    "args": ["World"]
  }'
```

## 📊 转换场景示例

### 场景 1: 序列化格式转换
```json
{
  "source_protocol": "triple",
  "target_protocol": "triple",
  "source_serialization": "hessian2",
  "target_serialization": "json",
  "method_name": "Add",
  "param_types": ["int", "int"],
  "args": [15, 25]
}
```

### 场景 2: 协议转换 (RPC → HTTP)
```json
{
  "source_protocol": "triple",
  "target_protocol": "http",
  "source_serialization": "hessian2",
  "target_serialization": "json",
  "method_name": "GetUserInfo",
  "param_types": ["java.lang.String"],
  "args": ["user123"]
}
```

### 场景 3: 跨协议调用 (Triple → gRPC)
```json
{
  "source_protocol": "triple",
  "target_protocol": "grpc",
  "source_serialization": "json",
  "target_serialization": "protobuf",
  "method_name": "ProcessList",
  "param_types": ["java.util.List"],
  "args": [["item1", "item2", "item3"]]
}
```

## 🔧 核心转换机制

### 1. 输入数据转换
```go
// 从源序列化格式转换为内部格式
internalArgs, err := pc.convertToInternalFormat(req.Args, req.SourceSerialization)

switch sourceSerialization {
case SerializationHessian2:
    return pc.convertFromHessian2(args)
case SerializationJSON:
    return pc.convertFromJSON(args)
case SerializationProtoBuf:
    return pc.convertFromProtoBuf(args)
}
```

### 2. 协议调用转换
```go
// 使用目标协议执行调用
result, err := pc.makeCall(req.TargetProtocol, req.TargetSerialization, 
    req.MethodName, req.ParamTypes, internalArgs)

switch protocol {
case ProtocolTriple:
    return pc.makeTripleCall(serialization, methodName, paramTypes, args)
case ProtocolHTTP:
    return pc.makeHTTPCall(methodName, args)
case ProtocolGRPC:
    return pc.makeTripleCall(serialization, methodName, paramTypes, args)
}
```

### 3. 输出数据转换
```go
// 从内部格式转换为目标序列化格式
convertedResult, err := pc.convertFromInternalFormat(result, req.TargetSerialization)

switch targetSerialization {
case SerializationHessian2:
    return pc.convertToHessian2(result)
case SerializationJSON:
    return pc.convertToJSON(result)
case SerializationProtoBuf:
    return pc.convertToProtoBuf(result)
}
```

## 🌐 HTTP协议网关

协议转换器提供HTTP网关功能，可以将任何RPC调用转换为HTTP API调用：

### RPC → HTTP 转换规则

| RPC调用 | HTTP端点 | 说明 |
|--------|----------|------|
| `Hello(name)` | `GET /api/hello?param=name` | 简单方法转换 |
| `GetUserInfo(id)` | `GET /api/getuserinfo?param=id` | 查询类方法 |
| `Add(a, b)` | `GET /api/add?a=10&b=20` | 多参数方法 |

### HTTP响应格式
```json
{
  "status": "success",
  "method": "Hello", 
  "data": "HTTP result for Hello"
}
```

## 🛠️ 扩展开发

### 添加新协议支持
```go
// 在makeCall方法中添加新协议
case ProtocolMyCustom:
    return pc.makeCustomCall(serialization, methodName, paramTypes, args)

// 实现自定义协议调用
func (pc *ProtocolConverter) makeCustomCall(serialization SerializationType, 
    methodName string, paramTypes []string, args []interface{}) (interface{}, error) {
    // 实现自定义协议调用逻辑
    return result, nil
}
```

### 添加新序列化支持
```go
// 添加新序列化类型
const SerializationMyFormat SerializationType = "myformat"

// 实现转换方法
func (pc *ProtocolConverter) convertFromMyFormat(args []interface{}) ([]interface{}, error) {
    // 实现从自定义格式的转换
    return converted, nil
}

func (pc *ProtocolConverter) convertToMyFormat(result interface{}) (interface{}, error) {
    // 实现到自定义格式的转换  
    return converted, nil
}
```

## 📈 性能优化

### 1. 连接复用
- 为不同序列化格式创建专用连接
- 使用连接池管理多个连接
- 实现连接健康检查和自动恢复

### 2. 缓存机制
```go
// 转换结果缓存
type ConversionCache struct {
    cache map[string]interface{}
    mutex sync.RWMutex
}

func (cc *ConversionCache) Get(key string) (interface{}, bool) {
    cc.mutex.RLock()
    defer cc.mutex.RUnlock()
    value, exists := cc.cache[key]
    return value, exists
}
```

### 3. 异步转换
```go
// 异步转换支持
func (pc *ProtocolConverter) ConvertAsync(req *ConvertRequest) <-chan *ConvertResponse {
    responseChan := make(chan *ConvertResponse, 1)
    
    go func() {
        defer close(responseChan)
        response := pc.Convert(req)
        responseChan <- response
    }()
    
    return responseChan
}
```

## 🔍 监控和调试

### 转换指标收集
```go
// 转换性能指标
type ConversionMetrics struct {
    TotalConversions    int64
    SuccessfulConversions int64
    FailedConversions   int64
    AverageLatency      time.Duration
    ProtocolBreakdown   map[ProtocolType]int64
}
```

### 调试日志
```go
log.Printf("Converting: %s(%s) → %s(%s), method: %s", 
    req.SourceProtocol, req.SourceSerialization,
    req.TargetProtocol, req.TargetSerialization,
    req.MethodName)
```

## 🎯 使用场景

### 1. 微服务网关
- 统一不同协议的微服务访问
- 提供HTTP API网关功能
- 支持协议版本升级和迁移

### 2. 协议迁移
- 从Dubbo协议迁移到Triple协议
- 渐进式序列化格式升级
- 向后兼容性保证

### 3. 测试工具
- 跨协议集成测试
- 性能对比测试
- 协议兼容性验证

### 4. 数据转换
- 不同序列化格式间的数据转换
- 协议特定数据处理
- 格式标准化

## 🤝 最佳实践

1. **连接管理**: 合理管理连接生命周期，避免连接泄漏
2. **错误处理**: 实现完善的错误处理和重试机制  
3. **性能监控**: 添加详细的性能指标和监控
4. **安全考虑**: 验证输入参数，防止注入攻击
5. **文档维护**: 保持转换规则和API文档的更新

---

通过这个协议转换器，您可以轻松实现不同协议和序列化格式之间的转换，为复杂的微服务架构提供灵活的互操作性支持！🚀

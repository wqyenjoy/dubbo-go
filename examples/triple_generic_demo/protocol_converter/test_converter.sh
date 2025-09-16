#!/bin/bash

# Protocol Converter 测试脚本
# 这个脚本演示了不同协议转换场景的测试

echo "🧪 Protocol Converter 测试脚本"
echo "=================================="

# 检查curl是否可用
if ! command -v curl &> /dev/null; then
    echo "❌ curl 未安装，请先安装 curl"
    exit 1
fi

# 检查jq是否可用 (用于格式化JSON输出)
if command -v jq &> /dev/null; then
    JQ_AVAILABLE=true
    echo "✅ 发现 jq，将格式化JSON输出"
else
    JQ_AVAILABLE=false
    echo "⚠️  未发现 jq，JSON输出将不格式化"
fi

GATEWAY_URL="http://localhost:8080"

# 检查Protocol Gateway是否运行
echo ""
echo "🔍 检查 Protocol Gateway 状态..."
if curl -s "${GATEWAY_URL}/health" > /dev/null 2>&1; then
    echo "✅ Protocol Gateway 运行正常"
else
    echo "❌ Protocol Gateway 未运行"
    echo "请先启动 Protocol Gateway:"
    echo "   cd examples/triple_generic_demo/protocol_converter"
    echo "   go run converter.go"
    exit 1
fi

# 函数：发送转换请求
send_conversion_request() {
    local test_name="$1"
    local json_payload="$2"
    
    echo ""
    echo "📤 测试: $test_name"
    echo "----------------------------------------"
    echo "请求负载:"
    if [ "$JQ_AVAILABLE" = true ]; then
        echo "$json_payload" | jq .
    else
        echo "$json_payload"
    fi
    
    echo ""
    echo "响应结果:"
    if [ "$JQ_AVAILABLE" = true ]; then
        curl -s -X POST "${GATEWAY_URL}/convert" \
            -H "Content-Type: application/json" \
            -d "$json_payload" | jq .
    else
        curl -s -X POST "${GATEWAY_URL}/convert" \
            -H "Content-Type: application/json" \
            -d "$json_payload"
    fi
    
    echo ""
    echo "----------------------------------------"
}

# 测试1: Triple Hessian2 → Triple JSON (序列化转换)
TEST1_JSON='{
  "source_protocol": "triple",
  "target_protocol": "triple", 
  "source_serialization": "hessian2",
  "target_serialization": "json",
  "method_name": "Hello",
  "param_types": ["java.lang.String"],
  "args": ["Converter Test"]
}'

send_conversion_request "序列化格式转换 (Hessian2 → JSON)" "$TEST1_JSON"

# 测试2: Triple → HTTP (协议转换)
TEST2_JSON='{
  "source_protocol": "triple",
  "target_protocol": "http",
  "source_serialization": "hessian2", 
  "target_serialization": "json",
  "method_name": "GetUserInfo",
  "param_types": ["java.lang.String"],
  "args": ["converter_user"]
}'

send_conversion_request "协议转换 (Triple → HTTP)" "$TEST2_JSON"

# 测试3: 数学运算转换
TEST3_JSON='{
  "source_protocol": "triple",
  "target_protocol": "triple",
  "source_serialization": "json",
  "target_serialization": "hessian2", 
  "method_name": "Add",
  "param_types": ["int", "int"],
  "args": [100, 200]
}'

send_conversion_request "数学运算转换 (JSON → Hessian2)" "$TEST3_JSON"

# 测试4: 复杂对象转换
TEST4_JSON='{
  "source_protocol": "triple", 
  "target_protocol": "triple",
  "source_serialization": "hessian2",
  "target_serialization": "json",
  "method_name": "ComplexOperation",
  "param_types": ["java.util.Map"],
  "args": [{
    "operation": "convert_test",
    "data": {
      "items": ["test1", "test2", "test3"],
      "config": {
        "format": "protocol_conversion",
        "version": "1.0"
      }
    }
  }]
}'

send_conversion_request "复杂对象转换" "$TEST4_JSON"

# 测试5: gRPC兼容性测试
TEST5_JSON='{
  "source_protocol": "triple",
  "target_protocol": "grpc", 
  "source_serialization": "hessian2",
  "target_serialization": "json",
  "method_name": "Hello",
  "param_types": ["java.lang.String"],
  "args": ["gRPC Compatible Test"]
}'

send_conversion_request "gRPC兼容性测试" "$TEST5_JSON"

# 测试6: 错误处理测试 (无效方法)
TEST6_JSON='{
  "source_protocol": "triple",
  "target_protocol": "triple",
  "source_serialization": "hessian2", 
  "target_serialization": "json",
  "method_name": "NonExistentMethod",
  "param_types": [],
  "args": []
}'

send_conversion_request "错误处理测试 (无效方法)" "$TEST6_JSON"

# 性能测试
echo ""
echo "⚡ 性能测试"
echo "=================================="
echo "执行10次转换请求，测量平均响应时间..."

PERFORMANCE_JSON='{
  "source_protocol": "triple",
  "target_protocol": "triple",
  "source_serialization": "hessian2",
  "target_serialization": "json", 
  "method_name": "Hello",
  "param_types": ["java.lang.String"],
  "args": ["Performance Test"]
}'

total_time=0
successful_requests=0

for i in {1..10}; do
    start_time=$(date +%s%N)
    
    response=$(curl -s -X POST "${GATEWAY_URL}/convert" \
        -H "Content-Type: application/json" \
        -d "$PERFORMANCE_JSON")
    
    end_time=$(date +%s%N)
    request_time=$((($end_time - $start_time) / 1000000)) # 转换为毫秒
    
    # 检查请求是否成功
    if echo "$response" | grep -q '"success":true'; then
        successful_requests=$((successful_requests + 1))
        total_time=$((total_time + request_time))
        echo "  请求 $i: ${request_time}ms ✅"
    else
        echo "  请求 $i: 失败 ❌"
    fi
done

if [ $successful_requests -gt 0 ]; then
    average_time=$((total_time / successful_requests))
    echo ""
    echo "📊 性能测试结果:"
    echo "   成功请求: $successful_requests/10"
    echo "   平均响应时间: ${average_time}ms" 
    echo "   总耗时: ${total_time}ms"
    
    if [ $average_time -lt 50 ]; then
        echo "   性能评级: 优秀 🏆"
    elif [ $average_time -lt 100 ]; then
        echo "   性能评级: 良好 👍"
    elif [ $average_time -lt 200 ]; then
        echo "   性能评级: 一般 👌"
    else
        echo "   性能评级: 需要优化 ⚠️"
    fi
else
    echo "❌ 所有性能测试请求都失败了"
fi

# 并发测试
echo ""
echo "🚀 并发测试"
echo "=================================="
echo "同时发送5个并发转换请求..."

CONCURRENT_JSON='{
  "source_protocol": "triple",
  "target_protocol": "triple",
  "source_serialization": "hessian2",
  "target_serialization": "json",
  "method_name": "Add", 
  "param_types": ["int", "int"],
  "args": [50, 75]
}'

concurrent_start=$(date +%s%N)

# 并发执行5个请求
for i in {1..5}; do
    (
        response=$(curl -s -X POST "${GATEWAY_URL}/convert" \
            -H "Content-Type: application/json" \
            -d "$CONCURRENT_JSON")
        
        if echo "$response" | grep -q '"success":true'; then
            echo "  并发请求 $i: 成功 ✅"
        else
            echo "  并发请求 $i: 失败 ❌"
        fi
    ) &
done

wait # 等待所有并发请求完成

concurrent_end=$(date +%s%N)
concurrent_total_time=$((($concurrent_end - $concurrent_start) / 1000000))

echo ""
echo "📊 并发测试结果:"
echo "   总耗时: ${concurrent_total_time}ms"
echo "   平均每个请求: $((concurrent_total_time / 5))ms"

# 测试总结
echo ""
echo "🎉 协议转换器测试完成!"
echo "=================================="
echo "✅ 完成了以下测试:"
echo "   • 序列化格式转换 (Hessian2 ↔ JSON)"
echo "   • 协议转换 (Triple → HTTP)"
echo "   • gRPC兼容性验证"
echo "   • 复杂对象处理"
echo "   • 错误处理机制"
echo "   • 性能基准测试"
echo "   • 并发处理测试"
echo ""
echo "🔧 如需自定义测试，请修改此脚本或直接使用curl命令"
echo "📚 详细文档请参考: protocol_converter/README.md"

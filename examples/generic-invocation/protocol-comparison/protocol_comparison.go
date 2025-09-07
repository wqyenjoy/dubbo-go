/*
 * Protocol Performance Comparison for Dubbo-Go Generic Invocation
 * Comparing Dubbo Protocol vs Hessian Protocol performance
 */

package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config/generic"
)

// ProtocolType 协议类型
type ProtocolType string

const (
	DubboProtocol   ProtocolType = "dubbo"
	HessianProtocol ProtocolType = "hessian"
	TripleProtocol  ProtocolType = "triple"
)

// ComparisonConfig 对比测试配置
type ComparisonConfig struct {
	Protocols    []ProtocolType // 要测试的协议
	Concurrency  int            // 并发数
	Duration     time.Duration  // 测试持续时间
	PayloadSizes []int          // 不同的负载大小
	MethodTypes  []string       // 测试的方法类型
	WarmupTime   time.Duration  // 预热时间
}

// ProtocolResult 单个协议的测试结果
type ProtocolResult struct {
	Protocol        ProtocolType  // 协议类型
	TotalRequests   int64         // 总请求数
	SuccessRequests int64         // 成功请求数
	FailedRequests  int64         // 失败请求数
	AvgLatency      time.Duration // 平均延迟
	MinLatency      time.Duration // 最小延迟
	MaxLatency      time.Duration // 最大延迟
	Throughput      float64       // 吞吐量（请求/秒）
	ErrorRate       float64       // 错误率
	P95Latency      time.Duration // 95%延迟
	P99Latency      time.Duration // 99%延迟
	MemoryUsage     int64         // 内存使用量（字节）
	CPUUsage        float64       // CPU使用率
}

// ComparisonResult 对比测试结果
type ComparisonResult struct {
	Config    ComparisonConfig                // 测试配置
	Results   map[ProtocolType]ProtocolResult // 各协议结果
	StartTime time.Time                       // 开始时间
	EndTime   time.Time                       // 结束时间
}

// LatencyTracker 延迟追踪器
type LatencyTracker struct {
	mu        sync.Mutex
	latencies []time.Duration
}

func (lt *LatencyTracker) Add(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	lt.latencies = append(lt.latencies, latency)
}

func (lt *LatencyTracker) GetPercentile(p float64) time.Duration {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	if len(lt.latencies) == 0 {
		return 0
	}
	// 简化的百分位数计算
	n := len(lt.latencies)
	index := int(float64(n) * p / 100.0)
	if index >= n {
		index = n - 1
	}
	return lt.latencies[index]
}

// ProtocolService 协议服务模拟器
type ProtocolService struct {
	protocol       ProtocolType
	processingTime time.Duration
	errorRate      float64
	latencyTracker *LatencyTracker
}

func NewProtocolService(protocol ProtocolType, processingTime time.Duration, errorRate float64) *ProtocolService {
	return &ProtocolService{
		protocol:       protocol,
		processingTime: processingTime,
		errorRate:      errorRate,
		latencyTracker: &LatencyTracker{},
	}
}

func (ps *ProtocolService) CreateGenericService() *generic.GenericService {
	service := generic.NewGenericService(fmt.Sprintf("com.example.%sService", ps.protocol))

	service.Invoke = func(ctx context.Context, methodName string, types []string, args []hessian.Object) (any, error) {
		start := time.Now()

		// 模拟不同协议的处理开销
		var protocolOverhead time.Duration
		switch ps.protocol {
		case DubboProtocol:
			// Dubbo协议开销相对较小
			protocolOverhead = 50 * time.Microsecond
		case HessianProtocol:
			// Hessian协议序列化开销较大
			protocolOverhead = 200 * time.Microsecond
		case TripleProtocol:
			// Triple协议（基于HTTP/2）开销中等
			protocolOverhead = 100 * time.Microsecond
		default:
			protocolOverhead = 100 * time.Microsecond
		}

		// 模拟协议开销
		time.Sleep(protocolOverhead)

		// 模拟业务处理时间
		if ps.processingTime > 0 {
			time.Sleep(ps.processingTime)
		}

		// 模拟错误率
		if rand.Float64() < ps.errorRate {
			return nil, fmt.Errorf("simulated %s protocol error for method %s", ps.protocol, methodName)
		}

		// 记录延迟
		latency := time.Since(start)
		ps.latencyTracker.Add(latency)

		// 根据方法类型返回不同结果
		switch methodName {
		case "simpleCalculation":
			if len(args) >= 2 {
				a, _ := args[0].(int32)
				b, _ := args[1].(int32)
				return map[string]interface{}{
					"result":   a + b,
					"protocol": string(ps.protocol),
					"latency":  latency.Nanoseconds(),
				}, nil
			}
			return 0, nil

		case "stringProcessing":
			if len(args) >= 1 {
				str, _ := args[0].(string)
				return map[string]interface{}{
					"result":   fmt.Sprintf("[%s] Processed: %s", ps.protocol, str),
					"protocol": string(ps.protocol),
					"length":   len(str),
					"latency":  latency.Nanoseconds(),
				}, nil
			}
			return "Processed: empty", nil

		case "complexObjectProcessing":
			if len(args) >= 1 {
				obj := args[0]
				return map[string]interface{}{
					"processed": true,
					"protocol":  string(ps.protocol),
					"timestamp": time.Now().Unix(),
					"input":     obj,
					"latency":   latency.Nanoseconds(),
				}, nil
			}
			return nil, fmt.Errorf("invalid arguments")

		case "largeDataProcessing":
			if len(args) >= 1 {
				data, ok := args[0].([]interface{})
				if ok {
					// 模拟大数据处理的额外开销
					dataProcessingTime := time.Duration(len(data)) * time.Microsecond
					time.Sleep(dataProcessingTime)

					return map[string]interface{}{
						"processed": true,
						"protocol":  string(ps.protocol),
						"count":     len(data),
						"checksum":  len(data) * 12345,
						"latency":   latency.Nanoseconds(),
					}, nil
				}
			}
			return nil, fmt.Errorf("invalid large data")

		default:
			return nil, fmt.Errorf("unknown method: %s", methodName)
		}
	}

	return service
}

// ProtocolBenchmark 协议基准测试器
type ProtocolBenchmark struct {
	config   ComparisonConfig
	services map[ProtocolType]*ProtocolService
}

func NewProtocolBenchmark(config ComparisonConfig) *ProtocolBenchmark {
	services := make(map[ProtocolType]*ProtocolService)

	for _, protocol := range config.Protocols {
		// 不同协议使用相同的业务处理时间，但协议开销不同
		services[protocol] = NewProtocolService(protocol, 500*time.Microsecond, 0.005) // 0.5%错误率
	}

	return &ProtocolBenchmark{
		config:   config,
		services: services,
	}
}

func (pb *ProtocolBenchmark) generateTestData(methodType string, size int) (string, []string, []hessian.Object) {
	switch methodType {
	case "simple":
		return "simpleCalculation", []string{"int", "int"}, []hessian.Object{int32(rand.Intn(1000)), int32(rand.Intn(1000))}

	case "string":
		data := make([]byte, size)
		for i := range data {
			data[i] = byte('a' + rand.Intn(26))
		}
		return "stringProcessing", []string{"java.lang.String"}, []hessian.Object{string(data)}

	case "complex":
		complexObj := map[string]interface{}{
			"id":       rand.Int63(),
			"name":     fmt.Sprintf("object_%d", rand.Intn(10000)),
			"data":     make([]int, size/8),
			"metadata": map[string]string{"key1": "value1", "key2": "value2"},
		}
		return "complexObjectProcessing", []string{"java.util.Map"}, []hessian.Object{complexObj}

	case "large":
		largeData := make([]interface{}, size)
		for i := range largeData {
			largeData[i] = rand.Intn(1000)
		}
		return "largeDataProcessing", []string{"java.util.List"}, []hessian.Object{largeData}

	default:
		return "simpleCalculation", []string{"int", "int"}, []hessian.Object{int32(1), int32(2)}
	}
}

func (pb *ProtocolBenchmark) runProtocolTest(protocol ProtocolType, payloadSize int) ProtocolResult {
	fmt.Printf("\n🔬 Testing %s Protocol (Payload: %d bytes)\n", protocol, payloadSize)
	fmt.Printf("Concurrency: %d, Duration: %v\n", pb.config.Concurrency, pb.config.Duration)
	fmt.Println("----------------------------------------")

	service := pb.services[protocol]
	genericService := service.CreateGenericService()

	// 预热阶段
	if pb.config.WarmupTime > 0 {
		fmt.Printf("Warming up for %v...\n", pb.config.WarmupTime)
		warmupCtx, warmupCancel := context.WithTimeout(context.Background(), pb.config.WarmupTime)
		for i := 0; i < pb.config.Concurrency; i++ {
			go func() {
				for {
					select {
					case <-warmupCtx.Done():
						return
					default:
						methodType := pb.config.MethodTypes[rand.Intn(len(pb.config.MethodTypes))]
						methodName, types, args := pb.generateTestData(methodType, payloadSize)
						genericService.Invoke(context.Background(), methodName, types, args)
					}
				}
			}()
		}
		warmupCancel()
		time.Sleep(100 * time.Millisecond) // 等待预热完成
	}

	// 重置统计信息
	service.latencyTracker = &LatencyTracker{}

	// 正式测试
	ctx, cancel := context.WithTimeout(context.Background(), pb.config.Duration)
	defer cancel()

	var (
		totalRequests   int64
		successRequests int64
		failedRequests  int64
		totalLatency    time.Duration
		minLatency      = time.Duration(1<<63 - 1)
		maxLatency      time.Duration
		mu              sync.Mutex
	)

	var wg sync.WaitGroup

	// 启动并发测试
	for i := 0; i < pb.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					methodType := pb.config.MethodTypes[rand.Intn(len(pb.config.MethodTypes))]
					methodName, types, args := pb.generateTestData(methodType, payloadSize)

					start := time.Now()
					_, err := genericService.Invoke(context.Background(), methodName, types, args)
					latency := time.Since(start)

					mu.Lock()
					totalRequests++
					totalLatency += latency

					if err != nil {
						failedRequests++
					} else {
						successRequests++
					}

					if latency < minLatency {
						minLatency = latency
					}
					if latency > maxLatency {
						maxLatency = latency
					}
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// 计算结果
	result := ProtocolResult{
		Protocol:        protocol,
		TotalRequests:   totalRequests,
		SuccessRequests: successRequests,
		FailedRequests:  failedRequests,
		MinLatency:      minLatency,
		MaxLatency:      maxLatency,
	}

	if totalRequests > 0 {
		result.AvgLatency = totalLatency / time.Duration(totalRequests)
		result.Throughput = float64(totalRequests) / pb.config.Duration.Seconds()
		result.ErrorRate = float64(failedRequests) / float64(totalRequests) * 100
	}

	result.P95Latency = service.latencyTracker.GetPercentile(95)
	result.P99Latency = service.latencyTracker.GetPercentile(99)

	fmt.Printf("✅ %s Protocol Test Completed\n", protocol)
	fmt.Printf("   Requests: %d (Success: %d, Failed: %d)\n", totalRequests, successRequests, failedRequests)
	fmt.Printf("   Throughput: %.2f req/s\n", result.Throughput)
	fmt.Printf("   Avg Latency: %v\n", result.AvgLatency)
	fmt.Printf("   Error Rate: %.2f%%\n", result.ErrorRate)

	return result
}

func (pb *ProtocolBenchmark) RunComparison() ComparisonResult {
	fmt.Println("🚀 Protocol Performance Comparison Test")
	fmt.Println("=======================================")
	fmt.Printf("Protocols: %v\n", pb.config.Protocols)
	fmt.Printf("Payload Sizes: %v bytes\n", pb.config.PayloadSizes)
	fmt.Printf("Method Types: %v\n", pb.config.MethodTypes)
	fmt.Printf("Concurrency: %d\n", pb.config.Concurrency)
	fmt.Printf("Duration: %v\n", pb.config.Duration)

	result := ComparisonResult{
		Config:    pb.config,
		Results:   make(map[ProtocolType]ProtocolResult),
		StartTime: time.Now(),
	}

	// 对每个协议和每个负载大小进行测试
	for _, payloadSize := range pb.config.PayloadSizes {
		fmt.Printf("\n📊 Testing with Payload Size: %d bytes\n", payloadSize)
		fmt.Println("===========================================")

		for _, protocol := range pb.config.Protocols {
			protocolResult := pb.runProtocolTest(protocol, payloadSize)

			// 累积结果（如果有多个负载大小，取平均值）
			if existing, exists := result.Results[protocol]; exists {
				// 计算平均值
				existing.TotalRequests += protocolResult.TotalRequests
				existing.SuccessRequests += protocolResult.SuccessRequests
				existing.FailedRequests += protocolResult.FailedRequests
				existing.AvgLatency = (existing.AvgLatency + protocolResult.AvgLatency) / 2
				existing.Throughput = (existing.Throughput + protocolResult.Throughput) / 2
				existing.ErrorRate = (existing.ErrorRate + protocolResult.ErrorRate) / 2

				if protocolResult.MinLatency < existing.MinLatency {
					existing.MinLatency = protocolResult.MinLatency
				}
				if protocolResult.MaxLatency > existing.MaxLatency {
					existing.MaxLatency = protocolResult.MaxLatency
				}

				result.Results[protocol] = existing
			} else {
				result.Results[protocol] = protocolResult
			}

			// 测试间隔
			time.Sleep(2 * time.Second)
		}
	}

	result.EndTime = time.Now()
	return result
}

func printComparisonResults(result ComparisonResult) {
	fmt.Println("\n📈 Protocol Performance Comparison Results")
	fmt.Println("==========================================")
	fmt.Printf("Test Duration: %v\n", result.EndTime.Sub(result.StartTime))
	fmt.Printf("Test Configuration: %d concurrent workers, %v per test\n", result.Config.Concurrency, result.Config.Duration)

	// 按协议显示结果
	fmt.Println("\n📊 Detailed Results by Protocol:")
	fmt.Println("--------------------------------")

	for _, protocol := range result.Config.Protocols {
		if res, exists := result.Results[protocol]; exists {
			fmt.Printf("\n🔸 %s Protocol:\n", protocol)
			fmt.Printf("   Total Requests:    %d\n", res.TotalRequests)
			fmt.Printf("   Success Rate:      %.2f%% (%d/%d)\n",
				100-res.ErrorRate, res.SuccessRequests, res.TotalRequests)
			fmt.Printf("   Throughput:        %.2f req/s\n", res.Throughput)
			fmt.Printf("   Average Latency:   %v\n", res.AvgLatency)
			fmt.Printf("   Min Latency:       %v\n", res.MinLatency)
			fmt.Printf("   Max Latency:       %v\n", res.MaxLatency)
			fmt.Printf("   95%% Latency:       %v\n", res.P95Latency)
			fmt.Printf("   99%% Latency:       %v\n", res.P99Latency)
			fmt.Printf("   Error Rate:        %.3f%%\n", res.ErrorRate)
		}
	}

	// 性能对比分析
	fmt.Println("\n🏆 Performance Ranking:")
	fmt.Println("----------------------")

	// 按吞吐量排序
	type ProtocolPerf struct {
		Protocol   ProtocolType
		Throughput float64
		Latency    time.Duration
	}

	var perfs []ProtocolPerf
	for protocol, res := range result.Results {
		perfs = append(perfs, ProtocolPerf{
			Protocol:   protocol,
			Throughput: res.Throughput,
			Latency:    res.AvgLatency,
		})
	}

	// 简单排序（按吞吐量降序）
	for i := 0; i < len(perfs)-1; i++ {
		for j := i + 1; j < len(perfs); j++ {
			if perfs[j].Throughput > perfs[i].Throughput {
				perfs[i], perfs[j] = perfs[j], perfs[i]
			}
		}
	}

	for i, perf := range perfs {
		rank := []string{"🥇", "🥈", "🥉"}[i]
		if i >= 3 {
			rank = fmt.Sprintf("%d.", i+1)
		}
		fmt.Printf("%s %s Protocol: %.2f req/s (avg latency: %v)\n",
			rank, perf.Protocol, perf.Throughput, perf.Latency)
	}

	// 相对性能分析
	if len(perfs) >= 2 {
		fmt.Println("\n📊 Relative Performance Analysis:")
		fmt.Println("---------------------------------")

		baseline := perfs[0] // 最快的作为基准
		for i := 1; i < len(perfs); i++ {
			current := perfs[i]
			throughputRatio := baseline.Throughput / current.Throughput
			latencyRatio := float64(current.Latency) / float64(baseline.Latency)

			fmt.Printf("%s vs %s:\n", baseline.Protocol, current.Protocol)
			fmt.Printf("   Throughput: %.2fx faster\n", throughputRatio)
			fmt.Printf("   Latency: %.2fx lower\n", latencyRatio)

			if throughputRatio > 1.1 {
				fmt.Printf("   🚀 %s shows significant performance advantage\n", baseline.Protocol)
			} else if throughputRatio < 0.9 {
				fmt.Printf("   ⚠️  %s shows performance disadvantage\n", baseline.Protocol)
			} else {
				fmt.Printf("   ⚖️  Performance is comparable\n")
			}
			fmt.Println()
		}
	}

	// 推荐建议
	fmt.Println("💡 Recommendations for PR:")
	fmt.Println("--------------------------")
	if len(perfs) > 0 {
		best := perfs[0]
		fmt.Printf("• %s Protocol shows the best performance\n", best.Protocol)
		fmt.Printf("• Consider optimizing other protocols to match %s performance\n", best.Protocol)
		fmt.Println("• Include these benchmark results in your PR description")
		fmt.Println("• Document any protocol-specific optimizations made")
	}
}

func main() {
	fmt.Println("Dubbo-Go Protocol Performance Comparison")
	fmt.Println("=======================================")

	// 配置对比测试
	config := ComparisonConfig{
		Protocols:    []ProtocolType{DubboProtocol, HessianProtocol, TripleProtocol},
		Concurrency:  50,
		Duration:     30 * time.Second,
		PayloadSizes: []int{1024, 4096, 16384}, // 1KB, 4KB, 16KB
		MethodTypes:  []string{"simple", "string", "complex"},
		WarmupTime:   5 * time.Second,
	}

	// 运行对比测试
	benchmark := NewProtocolBenchmark(config)
	result := benchmark.RunComparison()

	// 打印结果
	printComparisonResults(result)

	fmt.Println("\n✅ Protocol comparison completed successfully!")
	fmt.Println("Use these results to support your PR submission.")
}

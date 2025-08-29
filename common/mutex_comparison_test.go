package common

import (
	"sync"
	"testing"
)

// 模拟同学测试结果中的场景
// 无竞争场景测试
func BenchmarkMutexNoContention(b *testing.B) {
	var mu sync.Mutex
	var counter int

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		counter++
		mu.Unlock()
	}
}

func BenchmarkRWMutexNoContention(b *testing.B) {
	var mu sync.RWMutex
	var counter int

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		counter++
		mu.Unlock()
	}
}

// 有工作负载场景测试
func BenchmarkMutexWithWork(b *testing.B) {
	var mu sync.Mutex
	var counter int

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			// 模拟一些工作负载
			for j := 0; j < 10; j++ {
				counter += j
			}
			mu.Unlock()
		}
	})
}

func BenchmarkRWMutexWithWork(b *testing.B) {
	var mu sync.RWMutex
	var counter int

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			// 模拟一些工作负载
			for j := 0; j < 10; j++ {
				counter += j
			}
			mu.Unlock()
		}
	})
}

// 纯写操作高竞争场景
func BenchmarkMutexHighContention(b *testing.B) {
	var mu sync.Mutex
	var counter int

	b.ResetTimer()
	b.SetParallelism(16) // 高并发
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			mu.Unlock()
		}
	})
}

func BenchmarkRWMutexHighContention(b *testing.B) {
	var mu sync.RWMutex
	var counter int

	b.ResetTimer()
	b.SetParallelism(16) // 高并发
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			mu.Unlock()
		}
	})
}

package common

import (
	"sync"
	"testing"
)

// 测试纯写操作场景下 Mutex vs RWMutex 的性能
// 使用 RWMutex 的版本
type URLWithRWMutex struct {
	params map[string]string
	lock   sync.RWMutex
}

func (u *URLWithRWMutex) SetParam(key, value string) {
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.params == nil {
		u.params = make(map[string]string)
	}
	u.params[key] = value
}

// 使用 Mutex 的版本
type URLWithMutex struct {
	params map[string]string
	lock   sync.Mutex
}

func (u *URLWithMutex) SetParam(key, value string) {
	u.lock.Lock()
	defer u.lock.Unlock()
	if u.params == nil {
		u.params = make(map[string]string)
	}
	u.params[key] = value
}

// 基准测试：纯写操作 - RWMutex
func BenchmarkWriteOnly_RWMutex(b *testing.B) {
	url := &URLWithRWMutex{}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			url.SetParam("key", "value")
			i++
		}
	})
}

// 基准测试：纯写操作 - Mutex
func BenchmarkWriteOnly_Mutex(b *testing.B) {
	url := &URLWithMutex{}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			url.SetParam("key", "value")
			i++
		}
	})
}

// 基准测试：高并发写操作 - RWMutex
func BenchmarkHighConcurrencyWrite_RWMutex(b *testing.B) {
	url := &URLWithRWMutex{}
	b.ResetTimer()
	b.SetParallelism(100) // 设置更高的并发度
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			url.SetParam("key", "value")
			i++
		}
	})
}

// 基准测试：高并发写操作 - Mutex
func BenchmarkHighConcurrencyWrite_Mutex(b *testing.B) {
	url := &URLWithMutex{}
	b.ResetTimer()
	b.SetParallelism(100) // 设置更高的并发度
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			url.SetParam("key", "value")
			i++
		}
	})
}

/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package hessian

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func init() {
	// 设置随机种子，确保测试结果一致性
	rand.Seed(1)
}

func TestMultipleLevelRecursiveDep(t *testing.T) {
	// ensure encode() and decode() are consistent
	data := generateLargeMap(2, 10) // about 1M

	encoder := NewEncoder()
	err := encoder.Encode(data)
	if err != nil {
		panic(err)
	}
	bytes := encoder.Buffer()

	decoder := NewDecoder(bytes)
	obj, err := decoder.Decode()
	if err != nil {
		panic(err)
	}

	s1 := fmt.Sprintf("%v", obj)
	s2 := fmt.Sprintf("%v", data)
	if s1 != s2 {
		t.Error("deserialize mismatched")
	}
}

func TestMultipleLevelRecursiveDep2(t *testing.T) {
	// ensure decode() a large object is fast
	data := generateLargeMap(3, 5) // about 10MB

	now := time.Now()

	encoder := NewEncoder()
	err := encoder.Encode(data)
	if err != nil {
		panic(err)
	}
	bytes := encoder.Buffer()
	fmt.Printf("hessian2 serialize %s %dKB\n", time.Since(now), len(bytes)/1024)

	now = time.Now()
	decoder := NewDecoder(bytes)
	obj, err := decoder.Decode()
	if err != nil {
		panic(err)
	}
	rt := time.Since(now)
	fmt.Printf("hessian2 deserialize %s\n", rt)

	if rt > 1*time.Second {
		t.Error("deserialize too slow")
	}
	s1 := fmt.Sprintf("%v", obj)
	s2 := fmt.Sprintf("%v", data)
	if s1 != s2 {
		t.Error("deserialize mismatched")
	}
}
func BenchmarkMultipleLevelRecursiveDep(b *testing.B) {
	// benchmark for decode()
	data := generateLargeMap(2, 5) // about 300KB

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoder := NewEncoder()
		err := encoder.Encode(data)
		if err != nil {
			panic(err)
		}
		bytes := encoder.Buffer()

		decoder := NewDecoder(bytes)
		_, err = decoder.Decode()
		if err != nil {
			panic(err)
		}
	}
}

// 添加一个新的benchmark测试函数，使用预编码数据以专注测试解码性能
func BenchmarkDecodeOnly(b *testing.B) {
	data := generateLargeMap(2, 5) // about 300KB

	encoder := NewEncoder()
	err := encoder.Encode(data)
	if err != nil {
		panic(err)
	}
	bytes := encoder.Buffer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		decoder := NewDecoder(bytes)
		_, err = decoder.Decode()
		if err != nil {
			panic(err)
		}
	}
}

// 添加一个测试函数验证benchmark循环语法修复
func TestBenchmarkLoopFix(t *testing.T) {
	// 这个测试函数只是为了验证benchmark循环语法的修复
	data := generateLargeMap(1, 2) // 小数据量

	encoder := NewEncoder()
	err := encoder.Encode(data)
	if err != nil {
		t.Fatal(err)
	}
	bytes := encoder.Buffer()

	// 测试正确的循环语法
	for i := 0; i < 3; i++ {
		decoder := NewDecoder(bytes)
		_, err = decoder.Decode()
		if err != nil {
			t.Fatal(err)
		}
	}
}

func generateLargeMap(depth int, size int) map[string]interface{} {
	data := map[string]interface{}{}

	if depth != 0 {
		// generate sub map
		for i := 0; i < size; i++ {
			data[fmt.Sprintf("m%d", i)] = generateLargeMap(depth-1, size)
		}

		// generate sub list
		for i := 0; i < size; i++ {
			var sublist []interface{}
			for j := 0; j < size; j++ {
				sublist = append(sublist, generateLargeMap(depth-1, size))
			}
			data[fmt.Sprintf("l%d", i)] = sublist
		}
	}

	// generate string element
	for i := 0; i < size; i++ {
		data[fmt.Sprintf("s%d", i)] = generateRandomString()
	}
	// generate int element
	for i := 0; i < size; i++ {
		data[fmt.Sprintf("i%d", i)] = rand.Int31()
	}
	// generate float element
	for i := 0; i < size; i++ {
		data[fmt.Sprintf("f%d", i)] = rand.Float32()
	}

	return data
}

func generateRandomString() string {
	return "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"[rand.Int31n(20):]
}

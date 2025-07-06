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

package file

import (
	"os"
	"sync"
	"sync/atomic"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"github.com/dubbogo/gost/log/logger"
	"github.com/fsnotify/fsnotify"
)

// ref 引用计数结构
type ref struct {
	cnt  int32
	list map[config_center.ConfigurationListener]struct{}
	mu   sync.RWMutex
}

// debounceInfo 防抖信息
type debounceInfo struct {
	timer *time.Timer
	mu    sync.Mutex
}

// CacheListener is file watcher
type CacheListener struct {
	watch        *fsnotify.Watcher
	keyListeners sync.Map
	rootPath     string
	debounceMap  sync.Map // 防抖映射
}

// NewCacheListener creates a new CacheListener
func NewCacheListener(rootPath string) *CacheListener {
	cl := &CacheListener{rootPath: rootPath}
	// start watcher
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Errorf("file : listen config fail, error:%v ", err)
	}
	go func() {
		for {
			select {
			case event := <-watch.Events:
				key := event.Name
				logger.Debugf("watcher %s, event %v", cl.rootPath, event)

				// 使用防抖机制处理文件变更
				cl.handleFileEvent(key, event)

			case err := <-watch.Errors:
				// err may be nil, ignore
				if err != nil {
					logger.Warnf("file : listen watch fail:%+v", err)
				}
			}
		}
	}()
	cl.watch = watch

	extension.AddCustomShutdownCallback(func() {
		cl.watch.Close()
	})

	return cl
}

// handleFileEvent 处理文件事件，带防抖机制
func (cl *CacheListener) handleFileEvent(key string, event fsnotify.Event) {
	// 获取或创建防抖信息
	debounceValue, _ := cl.debounceMap.LoadOrStore(key, &debounceInfo{})
	debounce := debounceValue.(*debounceInfo)

	debounce.mu.Lock()
	defer debounce.mu.Unlock()

	// 重置定时器
	if debounce.timer != nil {
		debounce.timer.Stop()
	}

	// 创建新的定时器，50ms后执行
	debounce.timer = time.AfterFunc(50*time.Millisecond, func() {
		cl.processFileEvent(key, event)
	})
}

// processFileEvent 处理文件事件
func (cl *CacheListener) processFileEvent(key string, event fsnotify.Event) {
	if event.Op&fsnotify.Write == fsnotify.Write {
		if l, ok := cl.keyListeners.Load(key); ok {
			ref := l.(*ref)
			ref.mu.RLock()
			listeners := make(map[config_center.ConfigurationListener]struct{}, len(ref.list))
			for listener := range ref.list {
				listeners[listener] = struct{}{}
			}
			ref.mu.RUnlock()
			dataChangeCallback(listeners, key, remoting.EventTypeUpdate)
		}
	}
	if event.Op&fsnotify.Create == fsnotify.Create {
		if l, ok := cl.keyListeners.Load(key); ok {
			ref := l.(*ref)
			ref.mu.RLock()
			listeners := make(map[config_center.ConfigurationListener]struct{}, len(ref.list))
			for listener := range ref.list {
				listeners[listener] = struct{}{}
			}
			ref.mu.RUnlock()
			dataChangeCallback(listeners, key, remoting.EventTypeAdd)
		}
	}
	if event.Op&fsnotify.Remove == fsnotify.Remove {
		if l, ok := cl.keyListeners.Load(key); ok {
			ref := l.(*ref)
			ref.mu.RLock()
			listeners := make(map[config_center.ConfigurationListener]struct{}, len(ref.list))
			for listener := range ref.list {
				listeners[listener] = struct{}{}
			}
			ref.mu.RUnlock()
			removeCallback(listeners, key, remoting.EventTypeDel)
		}
	}
}

func removeCallback(lmap map[config_center.ConfigurationListener]struct{}, key string, event remoting.EventType) {
	if len(lmap) == 0 {
		logger.Warnf("file watch callback but configuration listener is empty, key:%s, event:%v", key, event)
		return
	}
	for l := range lmap {
		callback(l, key, "", event)
	}
}

func dataChangeCallback(lmap map[config_center.ConfigurationListener]struct{}, key string, event remoting.EventType) {
	if len(lmap) == 0 {
		logger.Warnf("file watch callback but configuration listener is empty, key:%s, event:%v", key, event)
		return
	}
	c := getFileContent(key)
	for l := range lmap {
		callback(l, key, c, event)
	}
}

func callback(listener config_center.ConfigurationListener, path, data string, event remoting.EventType) {
	listener.Process(&config_center.ConfigChangeEvent{Key: path, Value: data, ConfigType: event})
}

// Close will remove key listener and close watcher
func (cl *CacheListener) Close() error {
	cl.keyListeners.Range(func(key, value any) bool {
		cl.keyListeners.Delete(key)
		return true
	})
	return cl.watch.Close()
}

// AddListener will add a listener if loaded
// if you watcher a file or directory not exist, will error with no such file or directory
func (cl *CacheListener) AddListener(key string, listener config_center.ConfigurationListener) {
	// 使用引用计数机制
	refValue, loaded := cl.keyListeners.LoadOrStore(key, &ref{
		cnt:  1,
		list: map[config_center.ConfigurationListener]struct{}{listener: {}},
	})

	ref := refValue.(*ref)
	if loaded {
		// 已存在，增加引用计数
		ref.mu.Lock()
		ref.list[listener] = struct{}{}
		atomic.AddInt32(&ref.cnt, 1)
		ref.mu.Unlock()
		return
	}

	// 第一次添加，注册文件监听
	if err := cl.watch.Add(key); err != nil {
		logger.Errorf("watcher add path:%s err:%v", key, err)
	}
}

// RemoveListener will delete a listener if loaded
func (cl *CacheListener) RemoveListener(key string, listener config_center.ConfigurationListener) {
	refValue, loaded := cl.keyListeners.Load(key)
	if !loaded {
		return
	}

	ref := refValue.(*ref)
	ref.mu.Lock()
	delete(ref.list, listener)
	newCnt := atomic.AddInt32(&ref.cnt, -1)
	ref.mu.Unlock()

	// 只有当引用计数为0时才真正移除文件监听
	if newCnt == 0 {
		cl.keyListeners.Delete(key)
		// 清理防抖信息
		cl.debounceMap.Delete(key)
		if err := cl.watch.Remove(key); err != nil {
			logger.Errorf("watcher remove path:%s err:%v", key, err)
		}
	}
}

func getFileContent(path string) string {
	c, err := os.ReadFile(path)
	if err != nil {
		logger.Errorf("read file path:%s err:%v", path, err)
		return ""
	}

	return string(c)
}

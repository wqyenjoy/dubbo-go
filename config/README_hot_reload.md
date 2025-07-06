# Dubbo-go 配置文件热加载功能

## 概述

Dubbo-go 配置文件热加载功能允许在应用运行过程中动态修改配置文件（如 `dubbogo.yml`），无需重启服务即可使配置生效。这大大提高了系统的可维护性和运维效率。

## 功能特性

- **实时文件监听**：使用 `fsnotify` 库实时监听配置文件变化
- **防抖机制**：避免频繁的文件修改导致多次重载
- **配置验证**：重载前验证新配置的有效性
- **差异检测**：智能检测配置变化，只处理有变化的配置项
- **安全回滚**：配置应用失败时自动回滚到旧配置
- **自定义回调**：支持自定义配置重载回调函数
- **线程安全**：使用读写锁保证并发安全

## 使用方法

### 1. 基本使用

```go
package main

import (
    "dubbo.apache.org/dubbo-go/v3/config"
    "os"
)

func main() {
    // 设置配置文件路径（可选，默认使用环境变量或默认路径）
    os.Setenv("DUBBO_CONFIG_FILE", "./conf/dubbogo.yaml")
    
    // 加载配置并启用热加载
    if err := config.LoadWithHotReload(); err != nil {
        panic(err)
    }
    
    // 应用正常运行...
}
```

### 2. 高级配置

```go
package main

import (
    "dubbo.apache.org/dubbo-go/v3/config"
    "time"
)

func main() {
    // 先加载配置
    if err := config.Load(); err != nil {
        panic(err)
    }
    
    // 获取热加载管理器
    manager := config.GetHotReloadManager()
    if manager != nil {
        // 设置自定义防抖延迟
        manager.SetDebounceDelay(1 * time.Second)
        
        // 设置自定义重载回调
        manager.SetReloadCallback(func(newConfig *config.RootConfig) error {
            // 自定义配置应用逻辑
            log.Printf("配置已更新: %s", newConfig.Application.Name)
            return nil
        })
        
        // 启动热加载
        if err := manager.Start(); err != nil {
            panic(err)
        }
    }
}
```

### 3. 手动控制

```go
// 停止热加载
if err := config.StopHotReload(); err != nil {
    log.Printf("停止热加载失败: %v", err)
}

// 检查热加载状态
manager := config.GetHotReloadManager()
if manager != nil && manager.IsRunning() {
    log.Println("热加载正在运行")
}
```

## 配置项说明

### 环境变量

- `DUBBO_CONFIG_FILE`：指定配置文件路径（可选）

### 默认配置

- 默认配置文件路径：`../conf/dubbogo.yaml`
- 默认防抖延迟：500ms
- 默认不自动启动热加载

## 支持的配置变更

热加载功能支持以下配置项的动态更新：

### 应用配置
- 应用名称 (`application.name`)
- 应用版本 (`application.version`)

### 注册中心配置
- 注册中心地址 (`registries.*.address`)
- 注册中心参数 (`registries.*.timeout`, `registries.*.username`, 等)
- 新增/删除注册中心

### 协议配置
- 协议端口 (`protocols.*.port`)
- 协议参数 (`protocols.*.serialization`, 等)

### 提供者配置
- 服务配置 (`provider.services.*`)
- 服务参数 (`provider.services.*.timeout`, `provider.services.*.retries`, 等)

### 消费者配置
- 引用配置 (`consumer.references.*`)
- 引用参数 (`consumer.references.*.timeout`, `consumer.references.*.retries`, 等)

### 其他配置
- 日志配置 (`logger.*`)
- 指标配置 (`metrics.*`)
- 元数据配置 (`metadata-report.*`)

## 安全考虑

### 配置验证
- 重载前验证新配置的格式和有效性
- 无效配置会被忽略，保持原配置不变

### 错误处理
- 配置应用失败时自动回滚
- 详细的错误日志记录

### 并发安全
- 使用读写锁保证配置访问的线程安全
- 避免配置更新过程中的竞态条件

## 性能优化

### 防抖机制
- 避免频繁的文件修改导致多次重载
- 可配置的防抖延迟时间

### 差异检测
- 只处理有变化的配置项
- 减少不必要的配置应用操作

### 资源管理
- 及时释放文件监听器资源
- 优雅关闭热加载管理器

## 示例

### 完整示例

参考 `examples/hot_reload_example/` 目录下的完整示例：

```bash
# 运行示例
cd examples/hot_reload_example
go run main.go

# 在另一个终端修改配置文件
echo "dubbo:
  application:
    name: updated-app
    version: 2.0.0" > conf/dubbogo.yaml
```

### 测试

运行测试用例：

```bash
go test -v ./config -run TestHotReload
```

## 注意事项

1. **文件权限**：确保应用有读取配置文件的权限
2. **文件格式**：支持 YAML 和 JSON 格式的配置文件
3. **编辑器兼容**：某些编辑器保存文件时可能触发多次事件，防抖机制可以处理这种情况
4. **配置依赖**：某些配置项之间可能存在依赖关系，修改时需要注意
5. **服务影响**：某些配置变更可能影响正在运行的服务，建议在低峰期进行配置更新

## 故障排除

### 常见问题

1. **热加载未启动**
   - 检查配置文件路径是否正确
   - 检查文件是否存在且有读取权限

2. **配置变更未生效**
   - 检查配置文件格式是否正确
   - 查看日志中的错误信息
   - 确认防抖延迟是否合适

3. **性能问题**
   - 调整防抖延迟时间
   - 检查是否有频繁的配置文件修改

### 日志调试

启用调试日志查看详细信息：

```go
// 设置日志级别为 debug
config.GetApplicationConfig().Logger.Level = "debug"
```

## 未来计划

- [ ] 支持更多配置文件格式
- [ ] 支持配置变更的版本管理
- [ ] 支持配置变更的回滚功能
- [ ] 支持配置变更的通知机制
- [ ] 支持配置变更的审计日志 
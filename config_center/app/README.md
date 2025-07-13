# Dubbo-go 应用级配置中心

## 概述

应用级配置中心是 Dubbo-go 提供的一个增强功能，允许在全局配置的基础上，为每个应用提供特定的配置覆盖。这使得运维人员可以更精细地控制服务配置，实现应用级别的灰度发布和配置管理。

## 配置优先级

Dubbo-go 的配置优先级从高到低为：

1. 应用级配置中心配置（`<appName>.properties`）
2. 全局配置中心配置（`dubbo.properties`）
3. 本地 YAML 配置文件
4. 环境变量 / 命令行参数

## 使用方法

### 1. YAML 配置方式

在 YAML 配置文件中，可以通过以下方式启用应用级配置中心：

```yaml
dubbo:
  application:
    name: order-service                    # 应用名称
  config-center:
    protocol: app-merged                   # 使用应用级配置中心
    address: nacos://127.0.0.1:8848        # 底层配置中心地址
    namespace: public
    group: dubbo
    params:
      appName: order-service               # 应用名称（可选，默认使用 application.name）
      protocol: nacos                      # 底层配置中心协议（可选，默认为 zookeeper）
```

### 2. 代码方式

```go
package main

import (
    "dubbo.apache.org/dubbo-go/v3/config"
    _ "dubbo.apache.org/dubbo-go/v3/imports"
)

func main() {
    // 创建配置中心配置
    configCenterConfig := config.NewConfigCenterConfigBuilder().
        SetProtocol("app-merged").
        SetAddress("nacos://127.0.0.1:8848").
        SetGroup("dubbo").
        SetNamespace("public").
        Build()
    
    // 添加应用名参数
    configCenterConfig.Params["appName"] = "order-service"
    configCenterConfig.Params["protocol"] = "nacos"
    
    // 使用配置
    rootConfig := config.NewRootConfigBuilder().
        SetConfigCenter(configCenterConfig).
        Build()
    
    // 初始化配置
    if err := rootConfig.Init(); err != nil {
        panic(err)
    }
    
    // 启动应用...
}
```

## 配置文件命名规则

在配置中心（如 Nacos、ZooKeeper 等）中，应用级配置和全局配置的命名规则如下：

- 全局配置：`dubbo.properties`（或 `dubbo.yaml`、`dubbo.json` 等，取决于 `file-extension` 设置）
- 应用级配置：`<appName>.properties`（或 `<appName>.yaml`、`<appName>.json` 等）

例如，对于名为 `order-service` 的应用：

- 全局配置：`dubbo.properties`
- 应用级配置：`order-service.properties`

## 配置示例

### Nacos 配置中心

在 Nacos 控制台中，创建以下配置：

1. 全局配置（Data ID: `dubbo.properties`, Group: `dubbo`）：
```properties
dubbo.provider.timeout=3000
dubbo.provider.retries=2
dubbo.application.qos.port=33333
```

2. 应用级配置（Data ID: `order-service.properties`, Group: `dubbo`）：
```properties
dubbo.provider.timeout=5000
dubbo.application.qos.port=44444
```

当 `order-service` 应用启动时，它将使用：
- `timeout=5000`（来自应用级配置）
- `retries=2`（来自全局配置）
- `qos.port=44444`（来自应用级配置）

而其他应用将使用全局配置中的默认值。

## 注意事项

1. 应用级配置中心是对原有配置中心的增强，不影响原有功能
2. 应用级配置仅覆盖同名配置项，未覆盖的配置项仍使用全局配置
3. 应用级配置变更和全局配置变更都会触发配置更新
4. 配置中心必须支持按 key 获取配置（如 Nacos、ZooKeeper 等） 
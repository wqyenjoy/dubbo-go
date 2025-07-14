# Application-Level Configuration Center

This package provides application-level configuration center support for Dubbo-Go, similar to Dubbo Java's application-level configuration capabilities.

## Overview

The application-level configuration center allows different applications to use different configurations, which is useful for:

- Gradual rollout of configuration changes
- Application-specific configuration management
- A/B testing with different configuration values

## Priority Order

The configuration lookup follows this priority order:

1. Application-level configuration (`<appName>.properties`)
2. Global configuration (`dubbo.properties`)

## Usage

### Basic Usage

To use application-level configuration, simply wrap your existing dynamic configuration:

```go
import (
    "dubbo.apache.org/dubbo-go/v3/config_center/app"
)

// Assuming you have a dynamic configuration instance
appName := "my-app"
appConfig := app.WrapWithAppConfig(dynamicConfig, appName)

// Use appConfig as you would normally use dynamicConfig
```

### Configuration Center URL

You can also specify the application name in the configuration center URL:

```go
url := common.NewURLWithOptions(
    common.WithProtocol("zookeeper"),
    common.WithLocation("127.0.0.1:2181"),
    common.WithParamsValue("appName", "my-app"),
)
```

### Configuration Keys

For application-level configuration, keys are prefixed with the application name:

- Global configuration key: `dubbo.registry.address`
- Application-level configuration key: `my-app.dubbo.registry.address`

The application-level configuration center automatically handles these prefixes for you.

## Implementation Details

The implementation uses the decorator pattern to wrap the original dynamic configuration:

1. `AppMergedConfiguration` - Core decorator implementing the `DynamicConfiguration` interface
2. `appMergedDynamicConfigurationFactory` - Factory for creating application-level configurations
3. `WrapWithAppConfig` - Helper function to automatically upgrade a regular configuration to an application-level one

## Thread Safety

All methods in `AppMergedConfiguration` are thread-safe, using read-write locks to protect concurrent access.

## Change Notification

The implementation supports change notification through the standard `ConfigurationListener` interface.
You can register listeners using the `AddListener` method, and they will be notified of changes to both
application-level and global configurations.

## Example

```go
// Create a configuration center client
configCenter, err := extension.GetConfigCenterFactory("zookeeper").GetDynamicConfiguration(url)
if err != nil {
    // Handle error
}

// Wrap with application-level support
appConfig := app.WrapWithAppConfig(configCenter, "my-app")

// Get a configuration value (will check my-app.key first, then key)
value, err := appConfig.GetProperties("key")

// Publish a configuration value (will publish to my-app.key)
err = appConfig.PublishConfig("key", "group", "value")
``` 
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/file"
)

func main() {
	// 创建配置文件
	configPath := "conf/dubbogo.yaml"
	if err := createConfigFile(configPath); err != nil {
		log.Fatalf("Failed to create config file: %v", err)
	}

	// 创建热加载文件配置中心
	url, _ := common.NewURL("hot-reload-file://localhost",
		common.WithParamsValue("config-path", configPath),
		common.WithParamsValue("debounce-delay", "500ms"))

	dc, err := file.NewHotReloadFileDynamicConfiguration(url)
	if err != nil {
		log.Fatalf("Failed to create hot reload dynamic configuration: %v", err)
	}

	// 添加配置变更监听器
	listener := &configChangeListener{}
	dc.AddListener("dubbo.application.name", listener)

	fmt.Println("Hot reload demo started. Press Ctrl+C to exit.")
	fmt.Println("You can modify conf/dubbogo.yaml to see hot reload in action.")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down...")
	dc.Close()
}

type configChangeListener struct{}

func (l *configChangeListener) Process(event *config_center.ConfigChangeEvent) {
	fmt.Printf("Config changed: key=%s, value=%s\n", event.Key, event.Value)
}

func createConfigFile(path string) error {
	// 确保目录存在
	dir := "conf"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 创建配置文件内容
	content := `application:
  name: "hot-reload-demo"
  version: "1.0.0"
  owner: "demo"
  organization: "demo"

protocols:
  tri:
    name: "tri"
    port: 20000

provider:
  services:
    UserProvider:
      interface: "com.example.UserService"
      version: "1.0.0"
      group: "demo"

consumer:
  references:
    UserService:
      interface: "com.example.UserService"
      version: "1.0.0"
      group: "demo"

registries:
  demoZK:
    protocol: "zookeeper"
    address: "127.0.0.1:2181"
    timeout: "3s"

logger:
  level: "info"
  format: "text"
`

	return os.WriteFile(path, []byte(content), 0644)
}

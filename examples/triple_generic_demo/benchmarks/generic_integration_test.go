package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

// demo service for test (same as provider/main.go)
type TestDemoService struct{}

func (TestDemoService) Hello(ctx context.Context, name string) (string, error) {
	return "hello, " + name, nil
}

func (TestDemoService) Reference() string { return "com.example.DemoService" }

func TestGenericInvoke_Triple_Hessian2(t *testing.T) {
	const (
		ip   = "127.0.0.1"
		port = 50061
		intf = "com.example.DemoService"
	)

	// start provider server
	srv, err := server.NewServer(
		server.WithServerProtocol(
			protocol.WithTriple(),
			protocol.WithIp(ip),
			protocol.WithPort(port),
		),
		server.WithServerSerialization(constant.Hessian2Serialization),
		server.SetServerApplication(&global.ApplicationConfig{
			Name:                    "test-app",
			MetadataServiceProtocol: "file",
		}),
		server.WithServerNotRegister(),
	)
	if err != nil {
		t.Fatalf("new server error: %v", err)
	}
	if err := srv.RegisterService(TestDemoService{}, server.WithSerialization(constant.Hessian2Serialization)); err != nil {
		t.Fatalf("register error: %v", err)
	}
	go func() { _ = srv.Serve() }()
	// wait server up
	time.Sleep(time.Second)

	// build consumer client (generic + hessian2)
	cli, err := client.NewClient(
		client.WithClientURL("tri://"+ip+":50061/"+intf),
		client.WithClientProtocolTriple(),
	)
	if err != nil {
		t.Fatalf("new client error: %v", err)
	}
	conn, err := cli.Dial(intf,
		client.WithGeneric(),
		client.WithSerialization(constant.Hessian2Serialization),
	)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	call := func(method string, types []string, argv []any) (string, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		var reply string
		if err := conn.CallUnary(ctx, []any{method, types, argv}, &reply, "$invoke"); err != nil {
			return "", err
		}
		return reply, nil
	}

	// basic
	if got, err := call("Hello", []string{"java.lang.String"}, []any{"world"}); err != nil {
		t.Fatalf("basic $invoke error: %v", err)
	} else if got != "hello, world" {
		t.Fatalf("want 'hello, world', got %v", got)
	}

	// empty types (go-go tolerant)
	if got, err := call("Hello", nil, []any{"world"}); err != nil {
		t.Fatalf("empty types $invoke error: %v", err)
	} else if got != "hello, world" {
		t.Fatalf("want 'hello, world' (empty types), got %v", got)
	}

	// loop 100 with stats
	{
		start := time.Now()
		succ, fail := 0, 0
		var firstErr error
		for i := 0; i < 100; i++ {
			if got, err := call("Hello", []string{"java.lang.String"}, []any{"world"}); err != nil {
				fail++
				if firstErr == nil {
					firstErr = fmt.Errorf("loop i=%d error: %w", i, err)
				}
			} else if got != "hello, world" {
				fail++
				if firstErr == nil {
					firstErr = fmt.Errorf("loop i=%d want 'hello, world', got %v", i, got)
				}
			} else {
				succ++
			}
		}
		dur := time.Since(start)
		if fail > 0 {
			t.Fatalf("loop100: succ=%d fail=%d dur=%s firstErr=%v", succ, fail, dur, firstErr)
		}
		t.Logf("loop100: succ=%d fail=%d dur=%s", succ, fail, dur)
	}

	// concurrent 10 * 50 with stats
	{
		start := time.Now()
		succCh := make(chan struct{}, 10*50)
		errCh := make(chan error, 10*50)
		var wg sync.WaitGroup
		for g := 0; g < 10; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					if got, err := call("Hello", []string{"java.lang.String"}, []any{"world"}); err != nil {
						errCh <- err
						return
					} else if got != "hello, world" {
						errCh <- fmt.Errorf("want 'hello, world', got %v", got)
						return
					} else {
						succCh <- struct{}{}
					}
				}
			}()
		}
		wg.Wait()
		close(succCh)
		close(errCh)
		succ, fail := 0, 0
		var firstErr error
		for range succCh {
			succ++
		}
		for e := range errCh {
			if e != nil {
				fail++
				if firstErr == nil {
					firstErr = e
				}
			}
		}
		dur := time.Since(start)
		if fail > 0 {
			t.Fatalf("concurrent 10x50: succ=%d fail=%d dur=%s firstErr=%v", succ, fail, dur, firstErr)
		}
		t.Logf("concurrent 10x50: succ=%d fail=%d dur=%s", succ, fail, dur)
	}

	// NotExist should error
	if _, err := call("NotExist", nil, nil); err == nil {
		t.Fatalf("want error for NotExist, got nil")
	}
}

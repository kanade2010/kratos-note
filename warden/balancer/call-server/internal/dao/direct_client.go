package dao

import (
	"context"

	"github.com/bilibili/kratos/pkg/net/rpc/warden"

	"google.golang.org/grpc"

	"fmt"
	demoapi "call-server/api"
	"google.golang.org/grpc/balancer/roundrobin"
)

// target server addrs.
const target = "direct://default/10.0.75.2:30001,10.0.75.2:30002" // NOTE: example

// NewClient new member grpc client
func NewClient(cfg *warden.ClientConfig, opts ...grpc.DialOption) (demoapi.DemoClient, error) {
	client := warden.NewClient(cfg, opts...)
	conn, err := client.Dial(context.Background(), target)
	if err != nil {
		return nil, err
	}
	// 注意替换这里：
	// NewDemoClient方法是在"api"目录下代码生成的
	// 对应proto文件内自定义的service名字，请使用正确方法名替换
	return demoapi.NewDemoClient(conn), nil
}

// NewClient new member grpc client
func NewGrpcConn(cfg *warden.ClientConfig, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	fmt.Println("-----tag: NewGrpcConn...")
	opts = append(opts, grpc.WithBalancerName(roundrobin.Name))
	client := warden.NewClient(cfg, opts...)
	
	conn, err := client.Dial(context.Background(), target)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
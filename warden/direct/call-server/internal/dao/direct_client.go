package dao

import (
	"context"

	"github.com/bilibili/kratos/pkg/net/rpc/warden"

	"google.golang.org/grpc"

	demoapi "call-server/api"
)

// target server addrs.
const target = "direct://default/127.0.0.1:9003" // NOTE: example

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

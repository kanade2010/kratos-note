#go微服务框架kratos学习笔记四(kratos warden-quickstart warden-direct方式client调用)

[toc]

---

## warden direct

本文是学习kratos warden第一节，kratos warden的直连方式client调用，我直接用demo项目做示例

### demo-server

先创建一个用作grpc-server
```go
kratos new grpc-server
```

在创建一个调用grpc-server接口的call-server
```go
kratos new call-server
```

现在该目录下有两个服务
```
[I:\VSProject\kratos-note\warden\direct]$ tree
卷 办公 的文件夹 PATH 列表
卷序列号为 00650064 0007:32FD
I:.
├─call-server
│  ├─api
│  ├─cmd
│  ├─configs
│  ├─internal
│  │  ├─dao
│  │  ├─di
│  │  ├─model
│  │  ├─server
│  │  │  ├─grpc
│  │  │  └─http
│  │  └─service
│  └─test
└─grpc-server
    ├─api
    ├─cmd
    ├─configs
    ├─internal
    │  ├─dao
    │  ├─di
    │  ├─model
    │  ├─server
    │  │  ├─grpc
    │  │  └─http
    │  └─service
    └─test

```

我们后面用call-server直连 grpc-server 调用grpc接口。

### grpc.toml
配置里面修改下端口，同时call-server里面添加grpc-server的地址，我这是9003.
```
[Server]
    addr = "0.0.0.0:9004"
    timeout = "1s"
    
[Client]
    addr = "0.0.0.0:9003"
    timeout = "1s"
```


### 服务注册
grpc-server的internal/server/grpc目录打开server.go文件，可以看到以下代码，替换注释内容就可以启动一个gRPC、我们就是demo项目不需要替换。

```go
package grpc

import (
	pb "grpc-server/api"

	"github.com/bilibili/kratos/pkg/conf/paladin"
	"github.com/bilibili/kratos/pkg/net/rpc/warden"
)

// New new a grpc server.
func New(svc pb.DemoServer) (ws *warden.Server, err error) {
	var (
		cfg warden.ServerConfig
		ct paladin.TOML
	)
	if err = paladin.Get("grpc.toml").Unmarshal(&ct); err != nil {
		return
	}
	if err = ct.Get("Server").UnmarshalTOML(&cfg); err != nil {
		return
	}
	ws = warden.NewServer(&cfg)
	// 注意替换这里：
	// RegisterDemoServer方法是在"api"目录下代码生成的
	// 对应proto文件内自定义的service名字，请使用正确方法名替换
	pb.RegisterDemoServer(ws.Server(), svc)
	ws, err = ws.Start()
	return
}

```


>接着直接启动grpc-server

```go
kratos run
I:/VSProject/go/pkg/mod/github.com/bilibili/kratos@v0.3.2-0.20191224125553-6e1180f53a8e/pkg/net/rpc/warden/server.go:329 warden: start grpc listen addr: [::]:9003

```

### 服务发现

client端要调用grpc的接口必须有它生成的protobuf文件

一般是下面两种方式:

>1、拷贝proto文件到自己项目下并且执行代码生成
2、直接import服务端的api package


这里因为demo服务api一模一样，我就不做import了，直接取api里面的pb.go文件

在`internal/dao` 里面直接修改dao文件并添加一个`direct_client.go`  

`direct_client.go`  
其中，target为gRPC用于服务发现的目标，使用标准url资源格式提供给resolver用于服务发现。warden默认使用direct直连方式，直接与server端进行连接。其他服务发现方式下回见。
```go
package dao

import (
	"context"

	"github.com/bilibili/kratos/pkg/net/rpc/warden"

	"google.golang.org/grpc"
)

// target server addrs.
const target = "direct://default/127.0.0.1:9003" // NOTE: example

// NewClient new member grpc client
func NewClient(cfg *warden.ClientConfig, opts ...grpc.DialOption) (DemoClient, error) {
	client := warden.NewClient(cfg, opts...)
	conn, err := client.Dial(context.Background(), target)
	if err != nil {
		return nil, err
	}
	// 注意替换这里：
	// NewDemoClient方法是在"api"目录下代码生成的
	// 对应proto文件内自定义的service名字，请使用正确方法名替换
	return NewDemoClient(conn), nil
}

```

### client direct 调用

dao.go 改成如下。
添加一个dao里面直接demoClient，newDao里面做democlient的初始化、并添加一个SayHello接口。

```go
package dao

import (
	"context"
	"time"

	demoapi "call-server/api"
	"call-server/internal/model"

	"github.com/bilibili/kratos/pkg/cache/memcache"
	"github.com/bilibili/kratos/pkg/cache/redis"
	"github.com/bilibili/kratos/pkg/conf/paladin"
	"github.com/bilibili/kratos/pkg/database/sql"
	"github.com/bilibili/kratos/pkg/net/rpc/warden"
	"github.com/bilibili/kratos/pkg/sync/pipeline/fanout"
	xtime "github.com/bilibili/kratos/pkg/time"
	grpcempty "github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/google/wire"
)

var Provider = wire.NewSet(New, NewDB, NewRedis, NewMC)

//go:generate kratos tool genbts
// Dao dao interface
type Dao interface {
	Close()
	Ping(ctx context.Context) (err error)
	// bts: -nullcache=&model.Article{ID:-1} -check_null_code=$!=nil&&$.ID==-1
	Article(c context.Context, id int64) (*model.Article, error)
	SayHello(c context.Context, req *demoapi.HelloReq) (resp *grpcempty.Empty, err error)
}

// dao dao.
type dao struct {
	db         *sql.DB
	redis      *redis.Redis
	mc         *memcache.Memcache
	demoClient demoapi.DemoClient
	cache      *fanout.Fanout
	demoExpire int32
}

// New new a dao and return.
func New(r *redis.Redis, mc *memcache.Memcache, db *sql.DB) (d Dao, cf func(), err error) {
	return newDao(r, mc, db)
}

func newDao(r *redis.Redis, mc *memcache.Memcache, db *sql.DB) (d *dao, cf func(), err error) {
	var cfg struct {
		DemoExpire xtime.Duration
	}
	if err = paladin.Get("application.toml").UnmarshalTOML(&cfg); err != nil {
		return
	}

	grpccfg := &warden.ClientConfig{}
	paladin.Get("grpc.toml").UnmarshalTOML(grpccfg)
	var grpcClient demoapi.DemoClient
	if grpcClient, err = NewClient(grpccfg); err != nil {
		return
	}

	d = &dao{
		db:         db,
		redis:      r,
		mc:         mc,
		demoClient: grpcClient,
		cache:      fanout.New("cache"),
		demoExpire: int32(time.Duration(cfg.DemoExpire) / time.Second),
	}
	cf = d.Close
	return
}

// Close close the resource.
func (d *dao) Close() {
	d.cache.Close()
}

// Ping ping the resource.
func (d *dao) Ping(ctx context.Context) (err error) {
	return nil
}

// SayHello say hello.
func (d *dao) SayHello(c context.Context, req *demoapi.HelloReq) (resp *grpcempty.Empty, err error) {
	if resp, err = d.demoClient.SayHello(c, req); err != nil {
		err = errors.Wrapf(err, "%v", req.Name)
	}
	return
}

```

`service.go`里面调用dao.SayHello()
```go
// SayHello grpc demo func.
func (s *Service) SayHello(ctx context.Context, req *pb.HelloReq) (reply *empty.Empty, err error) {
	reply = new(empty.Empty)
	s.dao.SayHello(ctx, req)
	fmt.Printf("hello %s", req.Name)
	return
}
```

最后调用call-server的http接口测试：

![](https://img2018.cnblogs.com/blog/1384555/201912/1384555-20191229124630925-135589833.png)


从grpc-server的日志里面可以看到我们的调用是成功的， 本节完(๑′ᴗ‵๑)Ｉ Lᵒᵛᵉᵧₒᵤ❤。

![](https://img2018.cnblogs.com/blog/1384555/201912/1384555-20191229124630924-1611085176.png)


本例子源代码 ：https://github.com/ailumiyana/kratos-note

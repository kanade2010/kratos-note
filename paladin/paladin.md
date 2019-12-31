# go微服务框架kratos学习笔记五(kratos 配置中心 paladin config sdk [断剑重铸之日，骑士归来之时])

[toc]

---


本节看看kratos的配置中心`paladin`(骑士)。
kratos对配置文件进行了梳理，配置管理模块化，如redis有redis的单独配置文件、bm有bm的单独配置文件，及为方便易用。

paladin 本质是一个config SDK客户端，包括了remote、file、mock几个抽象功能，方便使用本地文件或者远程配置中心，并且集成了对象自动reload功能。

现在看看paladin的几种配置方式 :

## 静态配置

照常 new 一个demo项目.

```go
kratos new paladin
```

随便找个配置，看目录结构都知道http.toml在configs下，可以直接用名字get到，应该是kratos工具做了封装。

`http.toml`
```
[Server]
    addr = "0.0.0.0:8000"
    timeout = "1s"
```

```go
// New new a bm server.
func New(s pb.DemoServer) (engine *bm.Engine, err error) {
	var (
		cfg bm.ServerConfig
		ct paladin.TOML
	)
	if err = paladin.Get("http.toml").Unmarshal(&ct); err != nil {
		return
	}
	if err = ct.Get("Server").UnmarshalTOML(&cfg); err != nil {
		return
	}
```

Get() 取到的是个Value结构,利用了encoding包(encoding包定义了供其它包使用的可以将数据在字节水平和文本表示之间转换的接口)做抽象接口。

```go
// Value is config value, maybe a json/toml/ini/string file.
type Value struct {
	val   interface{}
	slice interface{}
	raw   string
}

// Unmarshal is the interface implemented by an object that can unmarshal a textual representation of itself.
func (v *Value) Unmarshal(un encoding.TextUnmarshaler) error {
	text, err := v.Raw()
	if err != nil {
		return err
	}
	return un.UnmarshalText([]byte(text))
}

// UnmarshalTOML unmarhsal toml to struct.
func (v *Value) UnmarshalTOML(dst interface{}) error {
	text, err := v.Raw()
	if err != nil {
		return err
	}
	return toml.Unmarshal([]byte(text), dst)
}

```

直接kratos run的话，默认是读取的configs下的本地文件。 kratos/tool/run.go 里面是可以找到的.
```go
package main

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/urfave/cli"
)

func runAction(c *cli.Context) error {
	base, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dir := buildDir(base, "cmd", 5)
	conf := path.Join(filepath.Dir(dir), "configs")
	args := append([]string{"run", "main.go", "-conf", conf}, c.Args()...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	return nil
}
```

## flag注入

如果我们进行了build
```go
I:\VSProject\kratos-note\paladin\paladin>cd cmd

I:\VSProject\kratos-note\paladin\paladin\cmd>kratos build
directory: I:\VSProject\kratos-note\paladin\paladin/cmd
kratos: 0.3.1
build success.

I:\VSProject\kratos-note\paladin\paladin\cmd>cmd.exe
INFO 12/30-22:25:07.054 I:/VSProject/kratos-note/paladin/paladin/cmd/main.go:19 paladin start
panic: lack of remote config center args

goroutine 1 [running]:
github.com/bilibili/kratos/pkg/conf/paladin.Init(0x0, 0x0, 0x0, 0x0, 0x0)
        I:/VSProject/go/pkg/mod/github.com/bilibili/kratos@v0.3.2-0.20191224125553-6e1180f53a8e/pkg/conf/paladin/default.go:32 +0x25f
main.main()
        I:/VSProject/kratos-note/paladin/paladin/cmd/main.go:20 +0x103

I:\VSProject\kratos-note\paladin\paladin\cmd>
```

会发现直接运行时跑不起来的，因为这时候找不到配置文件，因为这时候我们没有调用kratos run，paladin找不到配置目录。

实际paladin里面会有一个confPath变量，主函数做paladin.init()的时候会做flag注入。也方便了开发环境开发人员自行做配置修改。
```go
package paladin

import (
	"context"
	"errors"
	"flag"
)

var (
	// DefaultClient default client.
	DefaultClient Client
	confPath      string
)

func init() {
	flag.StringVar(&confPath, "conf", "", "default config path")
}

// Init init config client.
// If confPath is set, it inits file client by default
// Otherwise we could pass args to init remote client
// args[0]: driver name, string type
func Init(args ...interface{}) (err error) {
	if confPath != "" {
		DefaultClient, err = NewFile(confPath)
	} else {
		var (
			driver Driver
		)
		
   ......
```

```go
I:\VSProject\kratos-note\paladin\paladin\cmd>cmd.exe -conf=I:\VSProject\kratos-note\paladin\paladin\configs
INFO 12/30-22:41:43.717 I:/VSProject/kratos-note/paladin/paladin/cmd/main.go:19 paladin start
2019/12/30 22:41:43 start watch filepath: I:\VSProject\kratos-note\paladin\paladin\configs
INFO 12/30-22:41:43.781 I:/VSProject/go/pkg/mod/github.com/bilibili/kratos@v0.3.2-0.20191224125553-6e1180f53a8e/pkg/net/http/blademaster/server.go:98 blademaster: start http listen addr: 0.0.0.0:8000
[warden] config is Deprecated, argument will be ignored. please use -grpc flag or GRPC env to configure warden server.
INFO 12/30-22:41:43.790 I:/VSProject/go/pkg/mod/github.com/bilibili/kratos@v0.3.2-0.20191224125553-6e1180f53a8e/pkg/net/rpc/warden/server.go:329 warden: start grpc listen addr: [::]:9000
```


## 在线热加载配置

在线读取、变更的配置信息，比如某个业务开关，实现配置reload实时更新。


```go
// Map is config map, key(filename) -> value(file).
type Map struct {
	values atomic.Value
}
```

paladin.Map 通过 atomic.Value 自动热加载

```go
# service.go
type Service struct {
	ac *paladin.Map
}

func New() *Service {
	// paladin.Map 通过atomic.Value支持自动热加载
	var ac = new(paladin.TOML)
	if err := paladin.Watch("application.toml", ac); err != nil {
		panic(err)
	}
	s := &Service{
		ac: ac,
	}
	return s
}

func (s *Service) Test() {
	sw, err := s.ac.Get("switch").Bool()
	if err != nil {
		// TODO
	}
	
	// or use default value
	sw := paladin.Bool(s.ac.Get("switch"), false)
}
```

## 远程配置中心

通过环境变量注入，例如：APP_ID/DEPLOY_ENV/ZONE/HOSTNAME，然后通过paladin实现远程配置中心SDK进行配合使用。

目前只可以看到这个步骤是在Init()的时候做的，paladin本质是个客户端包，在不知道服务端实现的情况下暂时没找到样例，有机会遇见再补上。

```go
// Init init config client.
// If confPath is set, it inits file client by default
// Otherwise we could pass args to init remote client
// args[0]: driver name, string type
func Init(args ...interface{}) (err error) {
	if confPath != "" {
		DefaultClient, err = NewFile(confPath)
	} else {
		var (
			driver Driver
		)
		argsLackErr := errors.New("lack of remote config center args")
		if len(args) == 0 {
			panic(argsLackErr.Error())
		}
		argsInvalidErr := errors.New("invalid remote config center args")
		driverName, ok := args[0].(string)
		if !ok {
			panic(argsInvalidErr.Error())
		}
		driver, err = GetDriver(driverName)
		if err != nil {
			return
		}
		DefaultClient, err = driver.New()
	}
	if err != nil {
		return
	}
	return
}
```

体感paladin使用舒适度还是挺不错的、

> 断剑重铸之日，骑士归来之时

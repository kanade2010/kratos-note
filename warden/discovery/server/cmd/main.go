package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"
	"context"

	"server/internal/di"
	"github.com/bilibili/kratos/pkg/conf/env"
	"github.com/bilibili/kratos/pkg/conf/paladin"
	"github.com/bilibili/kratos/pkg/log"
	"github.com/bilibili/kratos/pkg/naming"
	"github.com/bilibili/kratos/pkg/naming/discovery"
)

func main() {
	flag.Parse()
	log.Init(nil) // debug flag: log.dir={path}
	defer log.Close()
	log.Info("abc start")
	paladin.Init()
	_, closeFunc, err := di.InitApp()
	if err != nil {
		panic(err)
	}

	ip := "192.168.0.77"
	port := "9000"
	hn, _ := os.Hostname()
	dis := discovery.New(nil)
	ins := &naming.Instance{
		Zone:     env.Zone,
		Env:      env.DeployEnv,
		AppID:    "demo.service",
		Hostname: hn,
		Addrs: []string{
			"grpc://" + ip + ":" + port,
		},
	}

	cancel, err := dis.Register(context.Background(), ins)
	if err != nil {
		panic(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-c
		log.Info("get a signal %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			closeFunc()
			log.Info("abc exit")
			time.Sleep(time.Second)
			cancel()
			return
		case syscall.SIGHUP:
		default:
			cancel()
			return
		}
	}
}

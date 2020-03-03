package main

import (
	"time"
	"fmt"
	"sync/atomic"
	//"net/http"
	"context"
	"github.com/ailumiyana/latency"

	"github.com/bilibili/kratos/pkg/net/http/blademaster"
	"github.com/bilibili/kratos/pkg/net/netutil/breaker"
	xtime "github.com/bilibili/kratos/pkg/time"
	"github.com/bilibili/kratos/pkg/log"

)


func main(){

	log.Init(&log.Config{
		Stdout: true,
		V: 5,
	})

	k10delay := latency.New("k10", "cost")

	var num int32 = 0

	fmt.Println("Start")

	conf := &blademaster.ClientConfig{
		Dial: xtime.Duration(time.Millisecond * 10),
		Timeout: xtime.Duration(time.Millisecond * 10),
		Breaker:&breaker.Config{
			Window:  xtime.Duration(1 * time.Second),
			Bucket:  1,
			Request: 30,
			K:1.5,
	  },
	}

	cli := blademaster.NewClient(conf)

	k10delay.Start()
	for atomic.AddInt32(&num, 1) < 300 {
		time.Sleep(10*time.Millisecond)
		go func() {
			//http.Get("http://127.0.0.1:8089/get")
			fmt.Println(cli.Get(context.Background(), "http://127.0.0.1:8089/get", "", nil, nil))
		}()
	}

	fmt.Println(k10delay.End())
	
	fmt.Println("end:", atomic.AddInt32(&num, 1))

	time.Sleep(time.Hour)
}
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
	//grpcempty "github.com/golang/protobuf/ptypes/empty"
	//"github.com/pkg/errors"

	"github.com/google/wire"
	"github.com/bilibili/kratos/pkg/container/pool"
	"io"
	"reflect"
	"google.golang.org/grpc"

)

var Provider = wire.NewSet(New, NewDB, NewRedis, NewMC)

//go:generate kratos tool genbts
// Dao dao interface
type Dao interface {
	Close()
	Ping(ctx context.Context) (err error)
	// bts: -nullcache=&model.Article{ID:-1} -check_null_code=$!=nil&&$.ID==-1
	Article(c context.Context, id int64) (*model.Article, error)
	//SayHello(c context.Context, req *demoapi.HelloReq) (resp *grpcempty.Empty, err error)

	//get an demo grpcConn/grpcClient/ from rpc pool
	GrpcConnPut(ctx context.Context, cc *grpc.ClientConn) (err error)
	GrpcConn(ctx context.Context) (gcc *grpc.ClientConn, err error)
	GrpcClient(ctx context.Context) (cli demoapi.DemoClient, err error)
}

// dao dao.
type dao struct {
	db         *sql.DB
	redis      *redis.Redis
	mc         *memcache.Memcache
	cache      *fanout.Fanout
	demoExpire int32
	rpcPool    pool.Pool 
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

	// new pool
	pool_config := &pool.Config{
		Active:      0,
		Idle:        0,
		IdleTimeout: xtime.Duration(0 * time.Second),
		WaitTimeout: xtime.Duration(30 * time.Millisecond),
	}

	rpcPool := pool.NewSlice(pool_config)
	rpcPool.New = func(ctx context.Context) (cli io.Closer, err error) {
		wcfg := &warden.ClientConfig{}
		paladin.Get("grpc.toml").UnmarshalTOML(wcfg)
		if cli, err = NewGrpcConn(wcfg); err != nil {
			return
		}

		return
	}

	d = &dao{
		db:         db,
		redis:      r,
		mc:         mc,
		cache:      fanout.New("cache"),
		demoExpire: int32(time.Duration(cfg.DemoExpire) / time.Second),
		rpcPool:    rpcPool,
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
/*func (d *dao) SayHello(ctx context.Context, req *demoapi.HelloReq) (resp *grpcempty.Empty, err error) {
	if resp, err = d.demoClient.SayHello(ctx, req); err != nil {
		err = errors.Wrapf(err, "%v", req.Name)
	}
	return
}*/

func (d *dao) GrpcClient(ctx context.Context) (cli demoapi.DemoClient, err error) {
	var cc io.Closer
	if cc, err = d.rpcPool.Get(ctx); err != nil {
		return
	}

	cli = demoapi.NewDemoClient(reflect.ValueOf(cc).Interface().(*grpc.ClientConn))
	return
}

func (d *dao) GrpcConnPut(ctx context.Context, cc *grpc.ClientConn) (err error) {
	err = d.rpcPool.Put(ctx, cc, false)
	return
}

func (d *dao) GrpcConn(ctx context.Context) (gcc *grpc.ClientConn, err error) {
	var cc io.Closer
	if cc, err = d.rpcPool.Get(ctx); err != nil {
		return
	}

	gcc = reflect.ValueOf(cc).Interface().(*grpc.ClientConn)
	return
}
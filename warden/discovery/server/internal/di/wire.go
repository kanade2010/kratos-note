// +build wireinject
// The build tag makes sure the stub is not built in the final build.

package di

import (
	pb "server/api"
	"server/internal/dao"
	"server/internal/server/grpc"
	"server/internal/server/http"
	"server/internal/service"

	"github.com/google/wire"
)

var daoProvider = wire.NewSet(dao.New, dao.NewDB, dao.NewRedis, dao.NewMC)
var serviceProvider = wire.NewSet(service.New, wire.Bind(new(pb.DemoServer), new(*service.Service)))

func InitApp() (*App, func(), error) {
	panic(wire.Build(daoProvider, serviceProvider, http.New, grpc.New, NewApp))
}

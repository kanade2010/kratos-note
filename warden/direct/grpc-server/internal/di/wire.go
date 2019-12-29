// +build wireinject
// The build tag makes sure the stub is not built in the final build.

package di

import (
	"grpc-server/internal/dao"
	"grpc-server/internal/service"
	"grpc-server/internal/server/grpc"
	"grpc-server/internal/server/http"

	"github.com/google/wire"
)

//go:generate kratos t wire
func InitApp() (*App, func(), error) {
	panic(wire.Build(dao.Provider, service.Provider, http.New, grpc.New, NewApp))
}

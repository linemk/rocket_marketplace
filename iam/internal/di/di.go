package di

import (
	"github.com/jackc/pgx/v5/pgxpool"

	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"github.com/linemk/rocket-shop/iam/internal/api"
	"github.com/linemk/rocket-shop/iam/internal/config"
	sessionrepo "github.com/linemk/rocket-shop/iam/internal/repository/session"
	userrepo "github.com/linemk/rocket-shop/iam/internal/repository/user"
	authservice "github.com/linemk/rocket-shop/iam/internal/service/auth"
	userservice "github.com/linemk/rocket-shop/iam/internal/service/user"
	"github.com/linemk/rocket-shop/platform/pkg/cache"
	authv1 "github.com/linemk/rocket-shop/shared/pkg/proto/auth/v1"
	userv1 "github.com/linemk/rocket-shop/shared/pkg/proto/user/v1"
)

type Container struct {
	UserRepo        userrepo.Repository
	SessionRepo     sessionrepo.Repository
	UserService     userservice.Service
	AuthService     authservice.Service
	UserHandler     userv1.UserServiceServer
	AuthHandler     authv1.AuthServiceServer
	ExtAuthzHandler authv3.AuthorizationServer
}

func New(db *pgxpool.Pool, cacheClient cache.Client) *Container {
	userRepository := userrepo.NewRepository(db)
	sessionRepository := sessionrepo.NewRepository(cacheClient)

	userSvc := userservice.NewService(userRepository)
	authSvc := authservice.NewService(userRepository, sessionRepository, config.AppConfig().Session)

	return &Container{
		UserRepo:        userRepository,
		SessionRepo:     sessionRepository,
		UserService:     userSvc,
		AuthService:     authSvc,
		UserHandler:     api.NewUserV1Handler(userSvc),
		AuthHandler:     api.NewAuthV1Handler(authSvc),
		ExtAuthzHandler: api.NewExtAuthzV1Handler(authSvc),
	}
}

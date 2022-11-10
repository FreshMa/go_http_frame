package service

import (
	"myserver/internal/ctx"
	"myserver/internal/server"
	"net/http"
)

type UserService interface {
	List(c *ctx.Context)
	SignUp(c *ctx.Context)
}

func RegisterUserService(svr server.Server, user UserService) {
	svr.Route(http.MethodGet, "/user/list", user.List)
	svr.Route(http.MethodGet, "/user/*", user.List)
	svr.Route(http.MethodPost, "/user/signup", user.SignUp)
}

type MQService interface {
	Push(c *ctx.Context)
	Consume(c *ctx.Context)
}

func RegisterMQService(svr server.Server, mq MQService) {
	svr.Route(http.MethodPost, "/mq/push", mq.Push)
}

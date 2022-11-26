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
	CreateExchange(c *ctx.Context)
	DeclareAndBindQueue(c *ctx.Context)
	Push(c *ctx.Context)
	Consume(c *ctx.Context)
}

func RegisterMQService(svr server.Server, mq MQService) {
	svr.Route(http.MethodPost, "/mq/push", mq.Push)
	svr.Route(http.MethodPost, "/mq/exchange/create", mq.CreateExchange)
	svr.Route(http.MethodPost, "/mq/queue/declare_bind", mq.DeclareAndBindQueue)
}

type KafkaService interface {
	Publish(c *ctx.Context)
}

func RegisterKafkaService(svr server.Server, kaf KafkaService) {
	svr.Route(http.MethodPost, "/mq/kafka/publist", kaf.Publish)
}

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"time"

	"myserver/internal/config"
	"myserver/internal/ctx"
	"myserver/internal/middleware"
	"myserver/internal/mq"
	"myserver/internal/server"
	"myserver/internal/service"
)

var (
	configPath = flag.String("config", "config/config.yml", "config file path")
)

func main() {
	flag.Parse()

	//svr := NewServer(Metric(), NotifyShutdown())
	conf, err := config.NewConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to read file:%s, err:%v\n", *configPath, err)
	}
	g := server.NewGracefulShutdown()
	svr := server.NewServer(g.RejectRequestMiddleware(), middleware.Metric())

	// 启动rabbitmq
	mqCli, err := conf.GetCliConfigByName("rabbitmq")
	if err != nil {
		log.Fatalf("get mq config failed, err:%v\n", err)
	}
	rabbitMQ, err := mq.NewRabbitMQ(mqCli.Addr)
	if err != nil {
		log.Fatalf("failed to init rabbitmq, err:%v\n", err)
	}

	// 注册路由
	// TODO 后续会把repo当做userSvc的依赖也注入进去
	userSvc := service.DefaultUserService()
	mqSvc := service.NewMQService(rabbitMQ)

	service.RegisterUserService(svr, userSvc)
	service.RegisterMQService(svr, mqSvc)

	// 启用优雅关闭
	go WaitForShutdown(g.WaitServerShutdown(svr),
		g.RejectRequestAndWaiting,
		rabbitMQ.GracefulClose,
	)

	svr.Start(conf.Servers[0].Listen)
}

func WaitForShutdown(hooks ...ctx.Hook) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	sig := <-ch
	log.Printf("recv signal %s, application will exit\n", sig)
	time.AfterFunc(time.Minute, func() {
		log.Printf("shutdown gracefully error, exit\n")
		os.Exit(1)
	})
	//time.Sleep(5 * time.Second)

	for _, h := range hooks {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := h(ctx)
		if err != nil {
			log.Printf("failed to run hook:%v\n", err)
		}
		cancel()
	}
	os.Exit(0)
}

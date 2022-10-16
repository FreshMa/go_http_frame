package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// 优雅关闭
// 1. 停止接收新请求：需要一个开关，开关打开的时候需要停止接收新情求，使用一个middleware (done)
// 2. 处理完当前的剩余请求：维持请求计数，这个也得用一个middleware吧
// 3. 关闭当前的svr (done)
// 4. 释放资源
// 5. 超时强制关闭 (done)

func WaitForShutdown(hooks ...Hook) {
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

func main() {
	//svr := NewServer(Metric(), NotifyShutdown())
	g := NewGracefulShutdown()
	svr := NewServer(g.RejectRequestMiddleware(), Metric())

	svr.Route(http.MethodGet, "/user/list", List)
	svr.Route(http.MethodGet, "/user/*", List)
	svr.Route(http.MethodPost, "/user/signup", SignUp)

	go WaitForShutdown(g.WaitServerShutdown(svr),
		g.RejectRequestAndWaiting)
	svr.Start(":10002")
}

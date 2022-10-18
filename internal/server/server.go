package server

import (
	"context"
	"fmt"
	"log"
	"myserver/internal/ctx"
	"net/http"
)

type Server interface {
	Routable
	Start(port string) error
	Shutdown(ctx context.Context) error
}

type MyServer struct {
	handler Handler
}

func NewServer(middlewares ...ctx.HandleFunc) Server {
	return &MyServer{
		//handler: NewMapBasedHandler(),
		handler: NewTreeBasedHandler(middlewares...),
	}
}

func (s *MyServer) Route(method, path string, hfs ...ctx.HandleFunc) {
	log.Printf("method:%s, path:%s\n", method, path)
	s.handler.Route(method, path, hfs...)
}

func (s *MyServer) Start(port string) error {
	return http.ListenAndServe(port, s.handler)
}

func (s *MyServer) Shutdown(ctx context.Context) error {
	fmt.Printf("server shutdown...\n")
	return nil
}

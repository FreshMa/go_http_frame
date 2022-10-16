package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

type HandleFunc func(c *Context)

type Server interface {
	Routable
	Start(port string) error
	Shutdown(ctx context.Context) error
}

type MyServer struct {
	handler Handler
}

func NewServer(middlewares ...HandleFunc) Server {
	return &MyServer{
		//handler: NewMapBasedHandler(),
		handler: NewTreeBasedHandler(middlewares...),
	}
}

func (s *MyServer) Route(method, path string, hfs ...HandleFunc) {
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

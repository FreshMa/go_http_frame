package service

import (
	"context"
	"log"
	"myserver/internal/ctx"
	"myserver/internal/entity/dto"
	"myserver/internal/mq"
	"net/http"
	"time"
)

type MQServiceImpl struct {
	mq *mq.RabbitMQ
}

var _ MQService = &MQServiceImpl{}

func NewMQService(mq *mq.RabbitMQ) MQService {
	return &MQServiceImpl{
		mq: mq,
	}
}

func (s *MQServiceImpl) Push(c *ctx.Context) {
	req := &dto.MQPushReq{}
	if err := c.ReadJson(req); err != nil {
		log.Printf("read json failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.mq.Push(ctx, req.ExchangeName, req.RoutingKey, []byte(req.Body)); err != nil {
		log.Printf("push failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusInternalServerError)
		return
	}

	rsp := &dto.CommonResponse{
		Code: 0,
		Msg:  "success",
	}
	if err := c.WriteJson(http.StatusOK, rsp); err != nil {
		log.Printf("write failed, err:%v\n", err)
	}
}

func (s *MQServiceImpl) Consume(c *ctx.Context) {
}

func (s *MQServiceImpl) CreateExchange(c *ctx.Context) {
	req := &dto.MQCreateExchangeReq{}
	if err := c.ReadJson(req); err != nil {
		log.Printf("read json failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.mq.CreateExchange(ctx, req.ExchangeName, req.ExchangeType); err != nil {
		log.Printf("create exchange failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusInternalServerError)
		return
	}

	rsp := &dto.CommonResponse{
		Code: 0,
		Msg:  "success",
	}
	if err := c.WriteJson(http.StatusOK, rsp); err != nil {
		log.Printf("write failed, err:%v\n", err)
	}
}

func (s *MQServiceImpl) DeclareAndBindQueue(c *ctx.Context) {
	req := &dto.MQQueueBindReq{}
	if err := c.ReadJson(req); err != nil {
		log.Printf("read json failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.mq.DeclareAndBindQueue(ctx, req.QueueName, req.BindingKey, req.ExchangeName); err != nil {
		log.Printf("declare and bind queue failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusInternalServerError)
		return
	}

	rsp := &dto.CommonResponse{
		Code: 0,
		Msg:  "success",
	}
	if err := c.WriteJson(http.StatusOK, rsp); err != nil {
		log.Printf("write failed, err:%v\n", err)
	}
}

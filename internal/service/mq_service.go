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

func NewMQService(mq *mq.RabbitMQ) MQService {
	return &MQServiceImpl{
		mq: mq,
	}
}

func (s *MQServiceImpl) Push(c *ctx.Context) {
	req := &dto.MQReq{}
	if err := c.ReadJson(req); err != nil {
		log.Printf("read json failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.mq.Push(ctx, req.Queue, []byte(req.Body)); err != nil {
		log.Printf("push failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusInternalServerError)
		return
	}

	rsp := &dto.CommonResponse{
		Code: 0,
		Msg:  "success",
	}
	if err := c.WriteJson(200, rsp); err != nil {
		log.Printf("write failed, err:%v\n", err)
	}
}

func (s *MQServiceImpl) Consume(c *ctx.Context) {
}

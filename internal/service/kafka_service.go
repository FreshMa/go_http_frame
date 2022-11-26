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

type KafkaServiceImpl struct {
	kafka *mq.KafkaCli
}

var _ KafkaService = &KafkaServiceImpl{}

func NewKafkaService(k *mq.KafkaCli) *KafkaServiceImpl {
	return &KafkaServiceImpl{
		kafka: k,
	}
}

func (k *KafkaServiceImpl) Publish(c *ctx.Context) {
	req := &dto.KafkaPublishReq{}
	if err := c.ReadJson(req); err != nil {
		log.Printf("read json failed, req:%v, err:%v\n", req, err)
		c.W.WriteHeader(http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := k.kafka.Publish(ctx, req.Topic, req.Msgs); err != nil {
		log.Printf("publish failed, req:%v, err:%v\n", req, err)
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

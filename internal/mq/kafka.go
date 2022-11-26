package mq

import (
	"context"
	"log"
	"math/rand"
	"myserver/internal/entity/dto"

	"github.com/segmentio/kafka-go"
)

var (
	charDict = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOOQRTSUVWXYZ0123456789-")
)

type KafkaCli struct {
	addr string
	conn *kafka.Conn

	writer *kafka.Writer
}

func NewKafkaCli(addr string) (*KafkaCli, error) {
	conn, err := kafka.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &KafkaCli{
		addr: addr,
		conn: conn,
		writer: &kafka.Writer{
			Addr:     kafka.TCP(addr),
			Balancer: &kafka.LeastBytes{},
		},
	}, nil
}

func (k *KafkaCli) Close() error {
	if err := k.conn.Close(); err != nil {
		return err
	}
	if err := k.writer.Close(); err != nil {
		return err
	}
	return nil
}

func (k *KafkaCli) Publish(ctx context.Context, topic string, msgs []dto.KafkaMsg) error {
	kMsgs := make([]kafka.Message, 0, len(msgs))
	for _, m := range msgs {
		kMsgs = append(kMsgs, kafka.Message{
			Key:   []byte(m.Key),
			Value: []byte(m.Value),
		})
	}
	if err := k.writer.WriteMessages(ctx, kMsgs...); err != nil {
		return err
	}
	return nil
}

// Consume 消费kafka，需要提供一个处理函数。默认使用消费者组进行消费，如果没提供groupID，生成一个随机的
func (k *KafkaCli) Consume(ctx context.Context, topic, groupID string, autoCommit bool, fn func([]byte) error) error {
	if len(groupID) == 0 {
		groupID = k.genGroupName()
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{k.addr},
		GroupID: groupID,
		Topic:   topic,
	})

	for {
		var err error

		m, err := r.FetchMessage(ctx)
		if err != nil {
			break
		}

		retryTimes := 0
		maxRetryTimes := 3
		for retryTimes < maxRetryTimes {
			err := fn(m.Value)
			if err != nil {
				continue
			} else {
				break
			}
		}

		if err != nil {
			log.Printf("consume msg failed, err:%v, msg:%s\n", err, string(m.Value))
		}
		// 失败仍然提交 offset
		r.CommitMessages(ctx, m)
	}
	return nil
}

func (k *KafkaCli) genGroupName() string {
	nameLen := 16
	dictLen := len(charDict)

	name := make([]byte, nameLen)
	for i := 0; i < nameLen; i++ {
		name[i] = charDict[rand.Intn(dictLen)]
	}

	return string(name)
}

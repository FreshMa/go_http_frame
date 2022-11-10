package mq

import (
	"context"
	"errors"
	"log"
	"myserver/internal/server"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Options struct {
	Durable bool // for queue
	AutoACK bool // for consumer
}

type Option func(*Options)

func WithDurable(durable bool) Option {
	return func(o *Options) {
		o.Durable = durable
	}
}

func WithAutoACK(autoACK bool) Option {
	return func(o *Options) {
		o.AutoACK = autoACK
	}
}

type RabbitMQ struct {
	conn    *amqp.Connection
	chs     map[string]*amqp.Channel // channelName->channel
	binding map[string]*amqp.Channel // queue->channel
}

var defaultOption = &Options{}

func NewRabbitMQ(url string, opts ...Option) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(defaultOption)
	}

	return &RabbitMQ{
		conn:    conn,
		chs:     make(map[string]*amqp.Channel),
		binding: make(map[string]*amqp.Channel),
	}, nil
}

func (c *RabbitMQ) Close() error {
	return c.conn.Close()
}

func (c *RabbitMQ) GracefulClose(ctx context.Context) error {
	doneCh := make(chan error)
	go func() {
		doneCh <- c.Close()
	}()

	select {
	case err := <-doneCh:
		return err
	case <-ctx.Done():
		return server.ErrHookTimeout
	}
}

func (c *RabbitMQ) CreateChannel(name string, prefetchCnt int) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	err = ch.Qos(prefetchCnt, 0, false)
	if err != nil {
		return err
	}

	c.chs[name] = ch
	return err
}

func (c *RabbitMQ) BindQueue(chName, queueName string) error {
	ch, ok := c.chs[chName]
	if !ok {
		return errors.New("cannot find channel")
	}

	_, err := ch.QueueDeclare(
		queueName,
		defaultOption.Durable, // durable
		false,                 // delete when unsued
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return err
	}
	c.binding[queueName] = ch
	return err
}

func (c *RabbitMQ) Push(ctx context.Context, queue string, content []byte) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ch, ok := c.binding[queue]
	if !ok {
		return errors.New("no valid queue binding")
	}

	return ch.PublishWithContext(timeoutCtx,
		"",    // exchange type
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        content,
		},
	)
}

// StartConsumer 启动指定数量的消费者，并提供对应的消费函数
func (c *RabbitMQ) StartConsumer(ctx context.Context, consumerCnt int, queue string, prefetchSize int, fn func([]byte) error) error {
	ch, ok := c.binding[queue]
	if !ok {
		return errors.New("no valid queue binding")
	}

	msgs, err := ch.Consume(
		queue,
		"",    // consumer
		false, // auto ack
		false, // exclusive,
		false, //no-local,
		false, //no-wait,
		nil,   //args
	)
	if err != nil {
		return err
	}

	for i := 0; i < consumerCnt; i++ {
		go func() {
			for d := range msgs {
				err = fn(d.Body)
				if err != nil {
					log.Printf("[%d] consume failed, content:%s, err:%v\n", i, string(d.Body), err)
				}
			}
		}()

	}
	return nil
}

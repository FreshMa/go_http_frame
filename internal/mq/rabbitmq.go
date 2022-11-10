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

type Channel struct {
	name        string
	ch          *amqp.Channel
	prefetchCnt int
}

type RabbitMQ struct {
	url     string
	conn    *amqp.Connection
	chs     map[string]Channel // channelName->channel
	binding map[string]string  // queue->channelName

	redialCh chan struct{}
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

	mq := &RabbitMQ{
		url:      url,
		conn:     conn,
		chs:      make(map[string]Channel),
		binding:  make(map[string]string),
		redialCh: make(chan struct{}, 1),
	}

	go mq.monitor()
	return mq, nil
}

// Reconnect
func (m *RabbitMQ) monitor() {
	baseDuration := 10 * time.Second
	ticker := time.NewTicker(baseDuration)
	errCnt := 0
	for {
		select {
		case <-ticker.C:
			if m.conn.IsClosed() {
				m.redialCh <- struct{}{}
				errCnt++
			} else {
				errCnt = 0
			}

			if errCnt%10 == 0 {
				ticker.Reset(baseDuration + time.Duration(errCnt/10)*time.Second)
			}
		case <-m.redialCh:
			m.reconnect()
		}
	}
}

func (m *RabbitMQ) reconnect() {
	retryCnt := 0
	maxRetry := 3
	var newConn *amqp.Connection
	var err error

	for ; retryCnt < maxRetry; retryCnt++ {
		newConn, err = amqp.Dial(m.url)
		if err != nil {
			//log.Printf("dial failed, retry...")
			continue
		}
		// 尝试关闭旧连接
		m.conn.Close()
		m.conn = newConn
		break
	}
	// 获取失败了
	if retryCnt == maxRetry {
		log.Printf("mq: rabbitmq reconnect exceed max retry cnt, stop retry...")
		return
	}

	// 获取成功，需要重置channel和queue
	m.rebindChAndQueue()
}

func (m *RabbitMQ) rebindChAndQueue() {
	// queue->channel_name
	// channel_name->channel
	newChannels := make(map[string]Channel)
	for k, v := range m.chs {
		v := v
		ch, err := m.conn.Channel()
		if err != nil {
			continue
		}
		v.ch = ch
		newChannels[k] = v
	}
	m.chs = newChannels
}

func (m *RabbitMQ) Close() error {
	return m.conn.Close()
}

func (m *RabbitMQ) GracefulClose(ctx context.Context) error {
	doneCh := make(chan error)
	go func() {
		doneCh <- m.Close()
	}()

	select {
	case err := <-doneCh:
		return err
	case <-ctx.Done():
		return server.ErrHookTimeout
	}
}

func (m *RabbitMQ) CreateChannel(name string, prefetchCnt int) error {
	ch, err := m.conn.Channel()
	if err != nil {
		return err
	}
	err = ch.Qos(prefetchCnt, 0, false)
	if err != nil {
		return err
	}

	m.chs[name] = Channel{
		name:        name,
		ch:          ch,
		prefetchCnt: prefetchCnt,
	}
	return err
}

func (m *RabbitMQ) BindQueue(chName, queueName string) error {
	channel, ok := m.chs[chName]
	if !ok {
		return errors.New("cannot find channel")
	}

	_, err := channel.ch.QueueDeclare(
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
	m.binding[queueName] = chName
	return err
}

func (m *RabbitMQ) Push(ctx context.Context, queue string, content []byte) error {
	chName, ok := m.binding[queue]
	if !ok {
		return errors.New("no valid queue binding")
	}
	channel := m.chs[chName]

	err := channel.ch.PublishWithContext(ctx,
		"",    // exchange type
		queue, // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        content,
		},
	)
	// 重连通知只需要有一个就够了
	if err != nil && errors.Is(err, amqp.ErrClosed) {
		if len(m.redialCh) < 1 {
			m.redialCh <- struct{}{}
		}
	}
	return err
}

// StartConsumer 启动指定数量的消费者，并提供对应的消费函数
func (m *RabbitMQ) StartConsumer(ctx context.Context, consumerCnt int, queue string, prefetchSize int, fn func([]byte) error) error {
	chName, ok := m.binding[queue]
	if !ok {
		return errors.New("no valid queue binding")
	}
	channel := m.chs[chName]

	msgs, err := channel.ch.Consume(
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
		go func(idx int) {
			for d := range msgs {
				err = fn(d.Body)
				if err != nil {
					log.Printf("[%d] consume failed, content:%s, err:%v\n", idx, string(d.Body), err)
				}
			}
		}(i)

	}
	return nil
}

package mq

import (
	"context"
	"errors"
	"log"
	"myserver/internal/server"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Options struct {
}

type Option func(*Options)

type RabbitMQ struct {
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel

	redialCh       chan struct{}
	lastRedialTime time.Time
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

	defaultCh, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	mq := &RabbitMQ{
		url:      url,
		conn:     conn,
		ch:       defaultCh,
		redialCh: make(chan struct{}, 1),
	}

	go mq.monitor()
	return mq, nil
}

// 监控mq server健康状况，处理重连
func (m *RabbitMQ) monitor() {
	baseDuration := 10 * time.Second
	ticker := time.NewTicker(baseDuration)
	errCnt := 0
	for {
		select {
		case <-ticker.C:
			// amqp会定期发送心跳（默认10s），如果发送失败会将conn置为Close状态
			// 这里可以直接用IsClosed来判断mq是否挂掉了/网络是否有问题
			if m.conn.IsClosed() {
				if len(m.redialCh) < 1 {
					m.redialCh <- struct{}{}
				}
				errCnt++
			} else {
				errCnt = 0
			}

			// TODO 一个不成熟的退避算法，10s一次
			if errCnt > 0 && errCnt%10 == 0 {
				ticker.Reset(baseDuration + time.Duration(errCnt/10)*time.Second)
			}
		case <-m.redialCh:
			// 防止短时间内大量重建请求
			if time.Since(m.lastRedialTime) > 10*time.Second {
				m.reconnect()
				m.lastRedialTime = time.Now()
			}
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
		break
	}
	// 获取失败了
	if retryCnt == maxRetry {
		log.Printf("mq: rabbitmq reconnect exceed max retry cnt, stop retry...")
		return
	}

	// 获取成功，需要重建channel以及绑定queue
	newCh, err := newConn.Channel()
	if err != nil {
		log.Printf("mq: recreate channel failed:%v\n", err)
		newConn.Close()
		return
	}

	// TODO 是否需要加锁
	m.conn.Close()
	m.conn = newConn
	m.ch = newCh
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

func (m *RabbitMQ) CreateExchange(ctx context.Context, name, exType string) error {
	if exType != amqp.ExchangeFanout && exType != amqp.ExchangeDirect &&
		exType != amqp.ExchangeTopic && exType != amqp.ExchangeHeaders {
		return errors.New("mq: illegal rabbitmq exchange type")
	}

	doneCh := make(chan error)
	go func() {
		doneCh <- m.ch.ExchangeDeclare(
			name,   // name
			exType, // type
			true,   // durable
			false,  // auto-deleted
			false,  // internal
			false,  // no-wait
			nil,    // arguments
		)
	}()

	select {
	case <-ctx.Done():
		return errors.New("timeout")
	case err := <-doneCh:
		return err
	}
}

func (m *RabbitMQ) DeclareAndBindQueue(ctx context.Context, queueName, bindingKey, exchangeName string) error {
	doneCh := make(chan error)

	go func() {
		_, err := m.ch.QueueDeclare(
			queueName,
			true,  // durable
			false, // delete when unsued
			false, // exclusive
			false, // no-wait
			nil,   // args
		)
		if err != nil {
			doneCh <- err
		}

		if len(exchangeName) > 0 {
			doneCh <- m.ch.QueueBind(
				queueName,    // queue name
				bindingKey,   // binding key
				exchangeName, // exchange name
				false,        // no wait
				nil,          // args
			)
		}
	}()

	select {
	case <-ctx.Done():
		return errors.New("timeout")
	case err := <-doneCh:
		return err
	}
}

func (m *RabbitMQ) Push(ctx context.Context, exchangeName, routingKey string, content []byte) error {
	err := m.ch.PublishWithContext(ctx,
		exchangeName, // exchange name
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         content,
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

// Consume 启动指定数量的消费者，并提供对应的消费函数
func (m *RabbitMQ) Consume(ctx context.Context, consumerCnt int, queue string, prefetchCnt int, fn func([]byte) error) error {
	// 针对消费者，每个消费者组使用一个channel
	ch, err := m.conn.Channel()
	if err != nil {
		return err
	}

	err = ch.Qos(prefetchCnt, 0, false)
	if err != nil {
		ch.Close()
		return err
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
		ch.Close()
		return err
	}

	var wg sync.WaitGroup
	for i := 0; i < consumerCnt; i++ {
		wg.Add(1)
		go func(idx int) {
			for d := range msgs {
				err = fn(d.Body)
				if err != nil {
					log.Printf("[%d] consume failed, content:%s, err:%v\n", idx, string(d.Body), err)
				}
			}
			// 如果走到这里，说明msgs已经关闭了，需要重新创建连接
			wg.Done()
		}(i)
	}

	// 起一个协程来负责重建消费者
	go func() {
		wg.Wait()
		if len(m.redialCh) < 1 {
			m.redialCh <- struct{}{}
		}
		m.Consume(ctx, consumerCnt, queue, prefetchCnt, fn)
	}()

	return nil
}

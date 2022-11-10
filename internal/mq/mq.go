package mq

// TODO 现在还抽不出来，看看rabbitmq的能写成啥样
//mq 的接口
type MQ interface {
	Init(addr string) error
	Send(msg interface{}) error
	Ack() error
	// 提供回调
	OnCosume() error
}

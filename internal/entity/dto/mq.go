package dto

type MQPushReq struct {
	ExchangeName string `json:"exchange_name"`
	RoutingKey   string `json:"routing_key"`
	Body         string `json:"body"`
}

type MQCreateExchangeReq struct {
	ExchangeName string `json:"exchange_name"`
	ExchangeType string `json:"exchange_type"`
}

type MQQueueBindReq struct {
	QueueName    string `json:"queue_name"`
	BindingKey   string `json:"binding_key"`
	ExchangeName string `json:"exchange_name"`
}

type KafkaMsg struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type KafkaPublishReq struct {
	Topic string     `json:"topic"`
	Msgs  []KafkaMsg `json:"msgs"`
}

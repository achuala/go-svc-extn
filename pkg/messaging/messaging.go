package messaging

import "time"

type BrokerConfig struct {
	Broker  string
	Address string
	Timeout time.Duration
}

type NatsJsSubscriberConfig struct {
	DurableName  string
	ConsumerName string
}

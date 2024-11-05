package messaging

import (
	"context"
	"time"
)

type BrokerConfig struct {
	Broker  string
	Address string
	Timeout time.Duration
}

type NatsJsConsumerConfig struct {
	DurableName  string
	ConsumerName string
	StreamName   string
	HandlerFunc  func(ctx context.Context, msg *Message) error
}

type Message struct {
	Headers map[string][]string
	Data    []byte
}

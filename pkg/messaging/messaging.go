package messaging

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
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
	Subject      string
	HandlerName  string
	HandlerFunc  func(msg *message.Message) error
}

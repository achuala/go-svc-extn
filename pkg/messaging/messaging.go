package messaging

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go/jetstream"
)

type BrokerConfig struct {
	Broker  string
	Address string
	Timeout time.Duration
}

type NatsJsConsumerConfig struct {
	DurableName   string
	ConsumerName  string
	StreamName    string
	Subject       string
	HandlerName   string
	HandlerFunc   func(msg *message.Message) error
	AckPolicy     jetstream.AckPolicy
	AckWait       time.Duration
	DeliverPolicy jetstream.DeliverPolicy
	MaxAckPending int
}

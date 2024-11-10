package nats

import (
	"time"

	watermill_nats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/achuala/go-svc-extn/pkg/messaging"
	"github.com/achuala/go-svc-extn/pkg/util/idgen"
	cloudevents "github.com/cloudevents/sdk-go"
	nc "github.com/nats-io/nats.go"

	"github.com/go-kratos/kratos/v2/log"
)

type NatsJsPublisher struct {
	publisher message.Publisher
}

func NewNatsJsPublisher(cfg *messaging.BrokerConfig, logger log.Logger) (*NatsJsPublisher, func(), error) {
	log := log.NewHelper(logger)
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
	}
	wmLogger := messaging.NewWatermillLoggerAdapter(logger)
	log.Infof("publisher connecting  to nats at - %s", cfg.Address)
	publisher, err := watermill_nats.NewPublisher(
		watermill_nats.PublisherConfig{
			URL:         cfg.Address,
			NatsOptions: options,
			Marshaler:   &watermill_nats.NATSMarshaler{},
		},
		wmLogger,
	)

	if err != nil {
		return nil, nil, err
	}
	jsPublisher := &NatsJsPublisher{publisher: publisher}
	return jsPublisher, func() {
		publisher.Close()
	}, nil
}

func (n *NatsJsPublisher) PublishEvent(topic string, event *cloudevents.Event) error {
	dataBytes, err := event.MarshalJSON()
	if err != nil {
		return err
	}

	msg := message.NewMessage(event.ID(), dataBytes)
	return n.publisher.Publish(topic, msg)
}

func (n *NatsJsPublisher) PublishMessage(topic string, msg *message.Message) error {
	return n.publisher.Publish(topic, msg)
}

func (n *NatsJsPublisher) Publish(topic string, data []byte) error {
	msg := message.NewMessage(idgen.NewId(), data)
	return n.publisher.Publish(topic, msg)
}

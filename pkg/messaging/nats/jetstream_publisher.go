package nats

import (
	"log/slog"
	"time"

	watermill_nats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/achuala/go-svc-extn/pkg/messaging"
	"github.com/achuala/go-svc-extn/pkg/util/idgen"
	cloudevents "github.com/cloudevents/sdk-go"
	nc "github.com/nats-io/nats.go"
)

type NatsJsPublisher struct {
	publisher message.Publisher
}

func NewNatsJsPublisher(cfg *messaging.BrokerConfig, logger *slog.Logger) (*NatsJsPublisher, func(), error) {
	if logger == nil {
		logger = slog.Default()
	}
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
		nc.DisconnectErrHandler(func(nc *nc.Conn, err error) {
			logger.Error("nats disconnected", "err", err)
		}),
		nc.ReconnectHandler(func(nc *nc.Conn) {
			logger.Info("nats reconnected", "server_id", nc.ConnectedServerId())
		}),
		nc.ConnectHandler(func(nc *nc.Conn) {
			logger.Info("nats connected", "server_id", nc.ConnectedServerId())
		}),
	}
	options = append(options, cfg.NatsOptions()...)
	wmLogger := messaging.NewWatermillLoggerAdapter(logger)
	logger.Info("nats js publisher connecting to nats", "address", cfg.Address)
	publisher, err := watermill_nats.NewPublisher(
		watermill_nats.PublisherConfig{
			URL:         cfg.Address,
			NatsOptions: options,
		},
		wmLogger,
	)

	if err != nil {
		return nil, nil, err
	}
	jsPublisher := &NatsJsPublisher{publisher: publisher}
	return jsPublisher, func() {
		if err := publisher.Close(); err != nil {
			logger.Warn("error closing publisher", "err", err)
		}
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

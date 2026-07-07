package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermill_nats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/achuala/go-svc-extn/pkg/messaging"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsJsConsumer struct {
	subscriber *watermill_nats.Subscriber
	router     *message.Router
	logger     *slog.Logger
}

func consumerConfigurator(consumerName, streamName, subject string) watermill_nats.ResourceInitializer {
	return func(ctx context.Context, js jetstream.JetStream, topic string) (jetstream.Consumer, func(context.Context, watermill.LoggerAdapter), error) {
		stream, err := js.Stream(ctx, streamName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get stream for topic %s: %w", subject, err)
		}
		consumer, err := stream.Consumer(ctx, consumerName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get consumer %s: %w", consumerName, err)
		}

		return consumer, nil, nil
	}
}

func NewNatsJsConsumer(cfg *messaging.BrokerConfig, subCfg *messaging.NatsJsConsumerConfig, logger *slog.Logger) (*NatsJsConsumer, func(), error) {
	if logger == nil {
		logger = slog.Default()
	}
	wmLogger := messaging.NewWatermillLoggerAdapter(logger)
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
	conn, err := nc.Connect(cfg.Address, options...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to nats: %w", err)
	}
	logger.Info("consumer connected to nats", "url", conn.ConnectedUrl(), "status", conn.Status())

	consumerConfig := func(topic string, group string) jetstream.ConsumerConfig {
		return jetstream.ConsumerConfig{
			Durable:       subCfg.DurableName,
			AckPolicy:     subCfg.AckPolicy,
			AckWait:       subCfg.AckWait,
			DeliverPolicy: subCfg.DeliverPolicy,
			FilterSubject: subCfg.Subject,
			MaxAckPending: subCfg.MaxAckPending,
		}
	}
	subscriberConfig := watermill_nats.SubscriberConfig{
		Conn:                conn,
		Logger:              wmLogger,
		ConfigureConsumer:   consumerConfig,
		ResourceInitializer: consumerConfigurator(subCfg.ConsumerName, subCfg.StreamName, subCfg.Subject),
	}
	subscriber, err := watermill_nats.NewSubscriber(subscriberConfig)
	if err != nil {
		return nil, nil, err
	}
	router, err := message.NewRouter(message.RouterConfig{CloseTimeout: 5 * time.Second}, wmLogger)
	if err != nil {
		return nil, nil, err
	}
	router.AddMiddleware(middleware.Recoverer)
	router.AddConsumerHandler(subCfg.HandlerName, subCfg.Subject, subscriber, subCfg.HandlerFunc)
	jsConsumer := &NatsJsConsumer{router: router, subscriber: subscriber, logger: logger}
	return jsConsumer, func() {
		logger.Info("closing nats js consumer", "consumer", subCfg.ConsumerName)
		if jsConsumer.subscriber != nil {
			if err := jsConsumer.subscriber.Close(); err != nil {
				logger.Warn("error closing nats js subscriber", "consumer", subCfg.ConsumerName, "err", err)
			}
		}
		if jsConsumer.router != nil {
			if err := jsConsumer.router.Close(); err != nil {
				logger.Warn("error closing nats js router", "consumer", subCfg.ConsumerName, "err", err)
			}
		}
	}, nil
}

func (c *NatsJsConsumer) Run(ctx context.Context) error {
	c.logger.Info("starting nats js router and consumer")
	return c.router.Run(ctx)
}

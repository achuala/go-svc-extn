package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermill_nats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/achuala/go-svc-extn/pkg/messaging"
	"github.com/go-kratos/kratos/v2/log"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsJsConsumer struct {
	subscriber *watermill_nats.Subscriber
	router     *message.Router
	log        *log.Helper
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

func NewNatsJsConsumer(cfg *messaging.BrokerConfig, subCfg *messaging.NatsJsConsumerConfig, logger log.Logger) (*NatsJsConsumer, func(), error) {
	log := log.NewHelper(logger)
	wmLogger := messaging.NewWatermillLoggerAdapter(logger)
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
		nc.DisconnectErrHandler(func(nc *nc.Conn, err error) {
			log.Errorf("NATS disconnected: %v", err)
		}),
		nc.ReconnectHandler(func(nc *nc.Conn) {
			log.Infof("NATS reconnected to %s", nc.ConnectedServerId)
		}),
		nc.ConnectHandler(func(nc *nc.Conn) {
			log.Infof("NATS connected to %s", nc.ConnectedServerId)
		}),
	}
	conn, err := nc.Connect(cfg.Address, options...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to nats: %w", err)
	}
	log.Infof("consumer connected to nats - %v, status - %v", conn.ConnectedUrl(), conn.Status())

	// Consumer configuration just uses the durable name, the expectation is that the stream is already created and consumer is already created
	// with necessary configuration.
	consumerConfig := func(topic string, group string) jetstream.ConsumerConfig {
		return jetstream.ConsumerConfig{
			Durable:       subCfg.DurableName,
			AckPolicy:     jetstream.AckExplicitPolicy,
			AckWait:       5 * time.Second,
			DeliverPolicy: jetstream.DeliverAllPolicy,
			FilterSubject: subCfg.Subject,
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
	jsConsumer := &NatsJsConsumer{router: router, subscriber: subscriber, log: log}
	return jsConsumer, func() {
		log.Info("closing consumer")
		if jsConsumer.subscriber != nil {
			if err := jsConsumer.subscriber.Close(); err != nil {
				log.Warnf("error closing subscriber: %v", err)
			}
		}
		if jsConsumer.router != nil {
			if err := jsConsumer.router.Close(); err != nil {
				log.Warnf("error closing router: %v", err)
			}
		}
	}, nil
}

func (c *NatsJsConsumer) Run(ctx context.Context) error {
	log.Info("starting router and consumer")
	return c.router.Run(ctx)
}

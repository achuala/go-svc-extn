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

func defaultStreamConfigurator(streamName, topic string) jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{topic},
	}
}

func consumerConfigurator(consumerName, streamName, subject string) watermill_nats.ResourceInitializer {
	return func(ctx context.Context, js jetstream.JetStream, topic string) (jetstream.Consumer, func(context.Context, watermill.LoggerAdapter), error) {
		streamConfig := defaultStreamConfigurator(streamName, subject)

		stream, err := js.Stream(ctx, streamConfig.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get stream for topic %s: %w", subject, err)
		}
		consumer, err := stream.Consumer(ctx, consumerName)

		return consumer, nil, err
	}
}

func NewNatsJsConsumer(cfg *messaging.BrokerConfig, subCfg *messaging.NatsJsConsumerConfig, logger log.Logger) (*NatsJsConsumer, func(), error) {
	log := log.NewHelper(logger)
	wmLogger := messaging.NewWatermillLoggerAdapter(logger)
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
	}
	conn, err := nc.Connect(cfg.Address, options...)
	if err != nil {
		return nil, nil, err
	}
	log.Infof("consumer connected to nats - %v, status - %v", conn.ConnectedUrl(), conn.Status())
	consumerConfig := func(topic string, group string) jetstream.ConsumerConfig {
		return jetstream.ConsumerConfig{
			Name:      subCfg.ConsumerName,
			Durable:   subCfg.DurableName,
			AckPolicy: jetstream.AckExplicitPolicy,
		}
	}
	jsConfig := watermill_nats.SubscriberConfig{
		Conn:                conn,
		Logger:              wmLogger,
		ConfigureConsumer:   consumerConfig,
		ResourceInitializer: consumerConfigurator(subCfg.ConsumerName, subCfg.StreamName, subCfg.Subject),
	}
	subscriber, err := watermill_nats.NewSubscriber(jsConfig)
	if err != nil {
		return nil, nil, err
	}
	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		return nil, nil, err
	}
	router.AddMiddleware(middleware.Recoverer)
	router.AddNoPublisherHandler(subCfg.HandlerName, subCfg.Subject, subscriber, subCfg.HandlerFunc)
	jsConsumer := &NatsJsConsumer{router: router, subscriber: subscriber, log: log}
	return jsConsumer, func() {
		log.Info("closing consumer")
		if jsConsumer.subscriber != nil {
			jsConsumer.subscriber.Close()
		}
		if jsConsumer.router != nil {
			jsConsumer.router.Close()
		}
	}, nil
}

func (c *NatsJsConsumer) Run(ctx context.Context) error {
	log.Info("starting router and consumer")
	return c.router.Run(ctx)
}

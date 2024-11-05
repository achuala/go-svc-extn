package nats

import (
	"context"
	"errors"
	"time"

	watermill_nats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/jetstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/achuala/go-svc-extn/pkg/messaging"
	"github.com/go-kratos/kratos/v2/log"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type NatsJsConsumer struct {
	subscriber    *watermill_nats.Subscriber
	router        *message.Router
	log           *log.Helper
	handlersAdded bool
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
	consumerConfig := func(topic string, group string) jetstream.ConsumerConfig {
		return jetstream.ConsumerConfig{
			Name:      subCfg.ConsumerName,
			Durable:   subCfg.ConsumerName,
			AckPolicy: jetstream.AckExplicitPolicy,
		}
	}
	jsConfig := watermill_nats.SubscriberConfig{
		Conn:              conn,
		Logger:            wmLogger,
		ConfigureConsumer: consumerConfig,
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

func (c *NatsJsConsumer) AddHandler(topic string, handler message.NoPublishHandlerFunc) {
	c.router.AddNoPublisherHandler(topic, topic, c.subscriber, handler)
	c.handlersAdded = true
}

func (c *NatsJsConsumer) Run(ctx context.Context) error {
	if !c.handlersAdded {
		return errors.New("no handlers added")
	}
	log.Info("starting router and consumer")
	return c.router.Run(ctx)
}

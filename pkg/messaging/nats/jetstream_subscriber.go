package nats

import (
	"context"
	"errors"
	"time"

	watermill_nats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/achuala/go-svc-extn/pkg/messaging"
	"github.com/go-kratos/kratos/v2/log"
	nc "github.com/nats-io/nats.go"
)

type NatsJsSubscriber struct {
	subscriber    *watermill_nats.Subscriber
	router        *message.Router
	log           *log.Helper
	handlersAdded bool
}

func NewNatsJsSubscriber(cfg *messaging.BrokerConfig, subCfg *messaging.NatsJsSubscriberConfig, logger log.Logger) (*NatsJsSubscriber, func(), error) {
	log := log.NewHelper(logger)
	wmLogger := messaging.NewWatermillLoggerAdapter(logger)
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
	}
	subscribeOptions := []nc.SubOpt{
		nc.DeliverNew(),
		nc.AckExplicit(),
	}
	jsConfig := watermill_nats.JetStreamConfig{
		SubscribeOptions: subscribeOptions,

		AutoProvision: false,
		DurablePrefix: subCfg.ConsumerName,
	}
	subscriber, err := watermill_nats.NewSubscriber(
		watermill_nats.SubscriberConfig{
			URL:            cfg.Address,
			AckWaitTimeout: 3 * time.Second,
			CloseTimeout:   30 * time.Second,
			NatsOptions:    options,
			JetStream:      jsConfig,
		},
		wmLogger,
	)
	if err != nil {
		return nil, nil, err
	}
	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		return nil, nil, err
	}
	router.AddMiddleware(middleware.Recoverer)
	jsSubscriber := &NatsJsSubscriber{router: router, subscriber: subscriber, log: log}
	return jsSubscriber, func() {
		log.Info("closing subscriber")
		if jsSubscriber.subscriber != nil {
			jsSubscriber.subscriber.Close()
		}
		if jsSubscriber.router != nil {
			jsSubscriber.router.Close()
		}
	}, nil
}

func (c *NatsJsSubscriber) AddHandler(topic string, handler message.NoPublishHandlerFunc) {
	c.router.AddNoPublisherHandler(topic, topic, c.subscriber, handler)
	c.handlersAdded = true
}

func (c *NatsJsSubscriber) AddHandlerWithPublisher(subscribeTopic string, publishTopic string, publisher *NatsJsPublisher, handler message.HandlerFunc) {
	c.router.AddHandler(subscribeTopic, publishTopic, c.subscriber, publishTopic, publisher.publisher, handler)
	c.handlersAdded = true
}

func (c *NatsJsSubscriber) Run(ctx context.Context) error {
	if !c.handlersAdded {
		return errors.New("no handlers added")
	}
	log.Info("starting router and txn event consumers")
	return c.router.Run(ctx)
}

package nats

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	watermill_nats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/achuala/go-svc-extn/pkg/messaging"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/sourcegraph/conc"
)

type NatsJsConsumer struct {
	log            *log.Helper
	isSubscribed   bool
	nc             *nats.Conn
	consumer       jetstream.Consumer
	consumeContext jetstream.ConsumeContext
	wg             *conc.WaitGroup
}

func NewNatsJsConsumer(cfg *messaging.BrokerConfig, logger log.Logger) (*NatsJsConsumer, func(), error) {
	log := log.NewHelper(logger)
	nc, err := nats.Connect(c.cfg.Address,
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Warnf("nats client disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			log.Info("nats client reconnected")
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			log.Info("nats client closed")
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			log.Errorf("nats client error: %v", err)
		}))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to connect to nats - nats.Connect(%+v) failed: %v", cfg.Address, err)
	}
	consumer := &NatsJsConsumer{log: log, nc: nc}
	return consumer, func() {
		// Cleanup here
		if consumer.isSubscribed {
			consumer.consumeContext.Drain()
			consumer.consumeContext.Stop()
		}
		consumer.nc.Close()
	}, nil
}

func (c *NatsJsConsumer) AddConsumerHandler(ctx context.Context, consumerCfg *messaging.NatsJsConsumerConfig) error {
	js, err := jetstream.New(c.nc)
	if err != nil {
		return fmt.Errorf("unable to create jetstream - jetstream.New failed: %v", err)
	}
	// Get handle to the existing stream
	stream, err := js.Stream(ctx, consumerCfg.StreamName)
	if err != nil {
		return fmt.Errorf("unable to get stream - jetstream.Stream failed: %v", err)
	}
	// retrieve consumer handle from a stream
	cons, err := stream.Consumer(ctx, consumerCfg.ConsumerName)
	if err != nil {
		if err == jetstream.ErrConsumerNotFound {
			cons, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
				Durable:   consumerCfg.ConsumerName,
				AckPolicy: jetstream.AckExplicitPolicy,
			})
			if err != nil {
				panic(err)
			}
		} else {
			return fmt.Errorf("Get- jetstream.Consumer failed: %v", err)
		}
	}
	c.consumer = cons
	c.isSubscribed = true
}

func (c *NatsJsConsumer) Start(ctx context.Context) error {

	c.nc = nc
	// create jetstream context from nats connection
	js, err := jetstream.New(nc)
	if err != nil {
		return fmt.Errorf("unable to create jetstream - jetstream.New failed: %v", err)
	}
	// Get handle to the existing stream
	stream, err := js.Stream(ctx, "UPICREDITS")
	if err != nil {
		return fmt.Errorf("unable to get stream - jetstream.Stream failed: %v", err)
	}
	// retrieve consumer handle from a stream
	cons, err := stream.Consumer(ctx, c.subCfg.ConsumerName)
	if err != nil {
		return fmt.Errorf("unable to get consumer - jetstream.Consumer failed: %v", err)
	}
	// consume messages from the consumer in callback
	cc, err := cons.Consume(func(msg jetstream.Msg) {
		data := msg.Data()
		c.wg.Go(func() {
			c.msgHandler.Handle(data)
		})
		if err = msg.Ack(); err != nil {
			log.Errorf("error during ack: %v", err)
		}
	}, jetstream.ConsumeErrHandler(func(consumeCtx jetstream.ConsumeContext, err error) {
		log.Errorf("error: %v", err)
	}))

	if err != nil {
		log.Fatalf("error: %v", err)
		os.Exit(1)
	}
	c.isSubscribed = true
	c.consumeContext = cc
}

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

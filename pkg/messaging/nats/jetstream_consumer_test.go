package nats_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/achuala/go-svc-extn/pkg/messaging"
	"github.com/achuala/go-svc-extn/pkg/messaging/nats"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/go-kratos/kratos/v2/log"
)

func TestNatsJsConsumer(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	cfg := messaging.BrokerConfig{
		Broker:  "nats",
		Address: "nats://localhost:4222",
		Timeout: time.Second * 10,
	}
	handlerFunc := func(msg *message.Message) error {
		t.Logf("received message: %v", string(msg.Payload))
		event := cloudevents.NewEvent()
		err := event.UnmarshalJSON(msg.Payload)
		if err != nil {
			t.Fatalf("failed to unmarshal event: %v", err)
		}
		t.Logf("received event: %v", event.Data)

		return nil
	}
	consumer, closeFn, err := nats.NewNatsJsConsumer(&cfg, &messaging.NatsJsConsumerConfig{
		ConsumerName: "test-consumer1",
		DurableName:  "test-durable1",
		StreamName:   "TEST",
		Subject:      "test.>",
		HandlerName:  "test-handler1",
		HandlerFunc:  handlerFunc,
	}, logger)
	if err != nil {
		t.Fatalf("failed to create consumer: %v", err)
	}
	defer closeFn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := consumer.Run(ctx); err != nil {
		t.Fatalf("failed to run consumer: %v", err)
	}
}

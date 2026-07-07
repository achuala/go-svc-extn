package nats_test

import (
	"testing"
	"time"

	"github.com/achuala/go-svc-extn/pkg/messaging"
	"github.com/achuala/go-svc-extn/pkg/messaging/nats"
	cloudevents "github.com/cloudevents/sdk-go"
)

func TestNatsJsPublisher(t *testing.T) {
	cfg := messaging.BrokerConfig{
		Broker:  "nats",
		Address: "nats://localhost:4222",
		Timeout: time.Second * 10,
	}

	publisher, closeFn, err := nats.NewNatsJsPublisher(&cfg, testLogger())
	if err != nil {
		t.Fatalf("failed to create publisher: %v", err)
	}
	defer closeFn()

	event := cloudevents.NewEvent()
	event.SetID("test-id")
	event.SetSource("test-source")
	event.SetType("test-type")
	event.SetData("test-data")
	event.SetDataContentType(cloudevents.ApplicationJSON)
	event.SetSpecVersion(cloudevents.VersionV1)
	err = publisher.PublishEvent("test.unit", &event)
	if err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}
}

package nats

import (
	"time"

	"github.com/achuala/go-svc-extn/pkg/messaging"
	cloudevents "github.com/cloudevents/sdk-go"
	"github.com/go-kratos/kratos/v2/log"
	nc "github.com/nats-io/nats.go"
)

type NatsJsPublisher struct {
	conn *nc.Conn
	js   nc.JetStream
}

func NewNatsJsPublisher(cfg *messaging.BrokerConfig, logger log.Logger) (*NatsJsPublisher, func(), error) {
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
	}
	conn, err := nc.Connect(cfg.Address, options...)
	if err != nil {
		return nil, nil, err
	}

	js, err := conn.JetStream()
	if err != nil {
		return nil, nil, err
	}

	jsPublisher := &NatsJsPublisher{conn: conn, js: js}
	return jsPublisher, func() {
		conn.Close()
	}, nil
}

func (p *NatsJsPublisher) PublishEvent(topic string, event *cloudevents.Event) error {
	dataBytes, err := event.MarshalJSON()
	if err != nil {
		return err
	}
	msg := nc.NewMsg(topic)
	msg.Data = dataBytes
	_, err = p.js.PublishMsg(msg)
	return err
}

func (p *NatsJsPublisher) Publish(topic string, data []byte) error {
	msg := nc.NewMsg(topic)
	msg.Data = data
	_, err := p.js.PublishMsg(msg)
	return err
}

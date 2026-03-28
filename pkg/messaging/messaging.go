package messaging

import (
	"crypto/tls"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type BrokerConfig struct {
	Broker  string
	Address string
	Timeout time.Duration
	// TLSConfig enables TLS for the NATS connection.
	TLSConfig *tls.Config
	// Username for user/password authentication.
	Username string
	// Password for user/password authentication.
	Password string
	// Token for token-based authentication.
	Token string
	// CredentialsFile is the path to a .creds file for JWT + NKey authentication (e.g., Synadia managed NATS).
	CredentialsFile string
	// NKeyFile is the path to an NKey seed file for NKey authentication.
	NKeyFile string
}

// NatsOptions returns the NATS connection options derived from the BrokerConfig auth and TLS fields.
func (c *BrokerConfig) NatsOptions() []nc.Option {
	var opts []nc.Option
	if c.TLSConfig != nil {
		opts = append(opts, nc.Secure(c.TLSConfig))
	}
	if c.CredentialsFile != "" {
		opts = append(opts, nc.UserCredentials(c.CredentialsFile))
	} else if c.NKeyFile != "" {
		opt, err := nc.NkeyOptionFromSeed(c.NKeyFile)
		if err == nil {
			opts = append(opts, opt)
		}
	} else if c.Token != "" {
		opts = append(opts, nc.Token(c.Token))
	} else if c.Username != "" || c.Password != "" {
		opts = append(opts, nc.UserInfo(c.Username, c.Password))
	}
	return opts
}

type NatsJsConsumerConfig struct {
	DurableName   string
	ConsumerName  string
	StreamName    string
	Subject       string
	HandlerName   string
	HandlerFunc   func(msg *message.Message) error
	AckPolicy     jetstream.AckPolicy
	AckWait       time.Duration
	DeliverPolicy jetstream.DeliverPolicy
	MaxAckPending int
}

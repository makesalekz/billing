package data

import (
	nats "github.com/nats-io/nats.go"
	"github.com/makesalekz/billing/internal/conf"
)

// NewNatsClient .
func NewNatsClient(conf *conf.Bootstrap) (*nats.Conn, func(), error) {
	nc, err := nats.Connect(conf.GetNats())
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		nc.Close()
	}

	return nc, cleanup, nil
}

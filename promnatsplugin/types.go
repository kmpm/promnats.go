package promnatsplugin

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

type PromNats struct {
	ContextName string
	ServerURL   string
	logger      *zap.Logger
	nc          *nats.Conn
}

package promnatsplugin

import (
	"time"

	"github.com/kmpm/promnats.go"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type PromNats struct {
	ContextName string
	ServerURL   string
	Interval    time.Duration
	logger      *zap.Logger
	nc          *nats.Conn
}

func (m *PromNats) refresh() {
	ticker := time.NewTicker(5 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				ctx := context.TODO()
				msgs, err := doReq(ctx, nil, "metrics", 0, m.nc)
				if err != nil {
					m.logger.Error("error getting data", zap.Error(err))
					continue
				}
				for _, msg := range msgs {
					id := msg.Header.Get(promnats.HeaderPnID)
					if id == "" {
						m.logger.Warn("response without header", zap.Any("header", msg.Header), zap.Int("length", len(msg.Data)))
						continue
					}
					// TODO: make something out of it,
					// sort by id, split by dot and make available by path
					// also add some kind of index page to ease discovery
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

package index_initiator

import (
	"context"
	"log/slog"
	"time"

	"yadro.com/course/search/core"
)

type TickerInitiator struct {
	ttl    time.Duration
	svc    *core.Service
	stopCh chan struct{}
}

func NewTickerInitiator(svc *core.Service, ttl time.Duration) *TickerInitiator {
	return &TickerInitiator{
		ttl:    ttl,
		svc:    svc,
		stopCh: make(chan struct{}),
	}
}

func (i *TickerInitiator) Start(ctx context.Context, log *slog.Logger) {
	err := i.svc.BuildIndex(ctx)
	if err != nil {
		log.Error("failed indexing", "error", err)
	}

	ticker := time.NewTicker(i.ttl)
	for {
		select {
		case <-ticker.C:
			err = i.svc.BuildIndex(ctx)
			if err != nil {
				log.Error("failed indexing", "error", err)
			}
		case <-i.stopCh:
			ticker.Stop()
			return
		}
	}
}

func (i *TickerInitiator) Stop() {
	close(i.stopCh)
}

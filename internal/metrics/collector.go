package metrics

import (
	"sync/atomic"
	"time"

	"github.com/kenanabbak/notification-management-api/internal/domain"
)

type Collector struct {
	delivered    atomic.Int64
	failed       atomic.Int64
	totalLatency atomic.Int64
	latencyCount atomic.Int64

	channelDelivered [3]atomic.Int64
	channelFailed    [3]atomic.Int64
}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) RecordDelivered(channel domain.Channel, latency time.Duration) {
	c.delivered.Add(1)
	c.totalLatency.Add(latency.Milliseconds())
	c.latencyCount.Add(1)
	c.channelDelivered[channelIndex(channel)].Add(1)
}

func (c *Collector) RecordFailed(channel domain.Channel) {
	c.failed.Add(1)
	c.channelFailed[channelIndex(channel)].Add(1)
}

type Snapshot struct {
	Delivered   int64              `json:"delivered_total"`
	Failed      int64              `json:"failed_total"`
	SuccessRate float64            `json:"success_rate"`
	AvgLatency  float64            `json:"avg_latency_ms"`
	ByChannel   map[string]Channel `json:"by_channel"`
}

type Channel struct {
	Delivered int64 `json:"delivered"`
	Failed    int64 `json:"failed"`
}

func (c *Collector) Snapshot() Snapshot {
	delivered := c.delivered.Load()
	failed := c.failed.Load()
	total := delivered + failed

	var successRate float64
	if total > 0 {
		successRate = float64(delivered) / float64(total) * 100
	}

	var avgLatency float64
	if count := c.latencyCount.Load(); count > 0 {
		avgLatency = float64(c.totalLatency.Load()) / float64(count)
	}

	channels := []domain.Channel{domain.ChannelSMS, domain.ChannelEmail, domain.ChannelPush}
	byChannel := make(map[string]Channel, 3)
	for i, ch := range channels {
		byChannel[string(ch)] = Channel{
			Delivered: c.channelDelivered[i].Load(),
			Failed:    c.channelFailed[i].Load(),
		}
	}

	return Snapshot{
		Delivered:   delivered,
		Failed:      failed,
		SuccessRate: successRate,
		AvgLatency:  avgLatency,
		ByChannel:   byChannel,
	}
}

func channelIndex(ch domain.Channel) int {
	switch ch {
	case domain.ChannelSMS:
		return 0
	case domain.ChannelEmail:
		return 1
	default:
		return 2
	}
}

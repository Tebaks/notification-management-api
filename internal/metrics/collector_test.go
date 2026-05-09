package metrics_test

import (
	"testing"
	"time"

	"github.com/kenanabbak/notification-management-api/internal/domain"
	"github.com/kenanabbak/notification-management-api/internal/metrics"
	"github.com/stretchr/testify/suite"
)

type CollectorSuite struct {
	suite.Suite
	c *metrics.Collector
}

func TestCollectorSuite(t *testing.T) {
	suite.Run(t, new(CollectorSuite))
}

func (s *CollectorSuite) SetupTest() {
	s.c = metrics.New()
}

func (s *CollectorSuite) TestSuccessRate() {
	s.c.RecordDelivered(domain.ChannelSMS, 50*time.Millisecond)
	s.c.RecordDelivered(domain.ChannelSMS, 100*time.Millisecond)
	s.c.RecordFailed(domain.ChannelSMS)

	snap := s.c.Snapshot()

	s.Equal(int64(2), snap.Delivered)
	s.Equal(int64(1), snap.Failed)
	s.InDelta(66.67, snap.SuccessRate, 0.1)
	s.InDelta(75.0, snap.AvgLatency, 0.1)
}

func (s *CollectorSuite) TestByChannel() {
	s.c.RecordDelivered(domain.ChannelSMS, 10*time.Millisecond)
	s.c.RecordDelivered(domain.ChannelEmail, 20*time.Millisecond)
	s.c.RecordFailed(domain.ChannelPush)

	snap := s.c.Snapshot()

	s.Equal(int64(1), snap.ByChannel["sms"].Delivered)
	s.Equal(int64(1), snap.ByChannel["email"].Delivered)
	s.Equal(int64(1), snap.ByChannel["push"].Failed)
}

func (s *CollectorSuite) TestZeroValues() {
	snap := s.c.Snapshot()

	s.Equal(float64(0), snap.SuccessRate)
	s.Equal(float64(0), snap.AvgLatency)
	s.Equal(int64(0), snap.Delivered)
}

func (s *CollectorSuite) TestAllChannels() {
	s.c.RecordDelivered(domain.ChannelPush, 5*time.Millisecond)
	s.c.RecordFailed(domain.ChannelSMS)
	s.c.RecordFailed(domain.ChannelEmail)

	snap := s.c.Snapshot()

	s.Equal(int64(1), snap.ByChannel["push"].Delivered)
	s.Equal(int64(1), snap.ByChannel["sms"].Failed)
	s.Equal(int64(1), snap.ByChannel["email"].Failed)
	s.InDelta(33.33, snap.SuccessRate, 0.1)
}

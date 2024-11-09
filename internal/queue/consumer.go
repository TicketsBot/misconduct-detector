package queue

import (
	"context"
	"encoding/json"
	"github.com/TicketsBot/common/eventforwarding"
	"github.com/TicketsBot/common/rpc"
	"github.com/TicketsBot/misconduct-detector/internal/config"
	"github.com/rxdn/gdl/gateway/payloads"
	"github.com/rxdn/gdl/gateway/payloads/events"
	"github.com/rxdn/gdl/objects/guild"
	"go.uber.org/zap"
	"time"
)

type Consumer struct {
	config config.Config
	logger *zap.Logger

	ch chan<- guild.Guild
}

// Still scan guilds that have joined if there is a Kafka backlog while booting
const joinDetectionThreshold = time.Hour

var _ rpc.Listener = (*Consumer)(nil)

func NewConsumer(config config.Config, logger *zap.Logger, ch chan<- guild.Guild) *Consumer {
	return &Consumer{
		config: config,
		logger: logger,
		ch:     ch,
	}
}

func (c *Consumer) BuildContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), c.config.TaskTimeout)
}

func (c *Consumer) HandleMessage(ctx context.Context, message []byte) {
	var wrapped eventforwarding.Event
	if err := json.Unmarshal(message, &wrapped); err != nil {
		c.logger.Error("Failed to unmarshal guild", zap.Error(err))
		return
	}

	var payload payloads.Payload
	if err := json.Unmarshal(wrapped.Event, &payload); err != nil {
		c.logger.Error("Failed to unmarshal payload", zap.Error(err))
		return
	}

	if payload.EventName != string(events.GUILD_CREATE) {
		return
	}

	var guild events.GuildCreate
	if err := json.Unmarshal(payload.Data, &guild); err != nil {
		c.logger.Error("Failed to unmarshal guild", zap.Error(err))
		return
	}

	if guild.JoinedAt.Before(time.Now().Add(-joinDetectionThreshold)) {
		c.logger.Debug("Ignoring guild, as we joined it too long ago", zap.Uint64("guild_id", guild.Id))
		return
	}

	c.logger.Debug("Received guild applicable for scanning", zap.Uint64("guild_id", guild.Id))

	c.ch <- guild.Guild
}

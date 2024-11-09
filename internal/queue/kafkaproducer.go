package queue

import (
	"context"
	"github.com/TicketsBot/common/rpc"
	"github.com/TicketsBot/common/rpc/model"
	"github.com/TicketsBot/misconduct-detector/internal/config"
)

type KafkaProducer struct {
	config config.Config
	client *rpc.Client
}

var _ Producer = (*KafkaProducer)(nil)

func NewKafkaProducer(config config.Config, client *rpc.Client) *KafkaProducer {
	return &KafkaProducer{
		config: config,
		client: client,
	}
}

func (k *KafkaProducer) PublishAlert(ctx context.Context, alert model.MisconductAlert) error {
	return k.client.ProduceSyncJson(ctx, k.config.Kafka.DetectionTopic, alert)
}

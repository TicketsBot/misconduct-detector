package queue

import (
	"context"
	"github.com/TicketsBot/common/rpc/model"
)

type Producer interface {
	PublishAlert(ctx context.Context, alert model.MisconductAlert) error
}

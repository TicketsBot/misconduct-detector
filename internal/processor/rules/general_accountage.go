package rules

import (
	"context"
	"github.com/TicketsBot/common/utils"
	"github.com/rxdn/gdl/objects/guild"
	"time"
)

type GeneralAccountAgeEvaluator struct {
}

var _ Evaluator = (*GeneralAccountAgeEvaluator)(nil)

func (e *GeneralAccountAgeEvaluator) Evaluate(_ context.Context, _ *AppContext, guild *guild.Guild) (int, error) {
	createdAt := utils.SnowflakeToTimestamp(guild.OwnerId)
	accountAge := time.Now().Sub(createdAt)

	if accountAge.Hours() < 24 {
		return 80, nil
	} else if accountAge.Hours() < 24*7 {
		return 50, nil
	} else if accountAge.Hours() < 24*30 {
		return 20, nil
	} else {
		return 0, nil
	}
}

func (e *GeneralAccountAgeEvaluator) Properties() EvaluatorProperties {
	return EvaluatorProperties{
		RuleName:             "Account age",
		RuleType:             RuleTypeGeneral,
		ShouldSpawnGoroutine: false,
	}
}

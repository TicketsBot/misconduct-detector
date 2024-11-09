package rules

import (
	"context"
	"github.com/rxdn/gdl/objects/guild"
)

type Evaluator interface {
	Evaluate(ctx context.Context, appContext *AppContext, guild *guild.Guild) (int, error)
	Properties() EvaluatorProperties
}

type RuleType string

type EvaluatorProperties struct {
	RuleName             string
	RuleType             RuleType
	ShouldSpawnGoroutine bool
}

const (
	RuleTypeGeneral    RuleType = "GENERAL"
	RuleTypeCryptoScam RuleType = "CRYPTO_SCAM"
	RuleTypeGameCheats RuleType = "GAME_CHEATS"
)

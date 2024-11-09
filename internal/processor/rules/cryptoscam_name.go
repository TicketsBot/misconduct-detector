package rules

import (
	"context"
	"github.com/rxdn/gdl/objects/guild"
	"strings"
)

type CryptoScamNameEvaluator struct {
}

var _ Evaluator = (*CryptoScamNameEvaluator)(nil)

var cryptoScamWords = map[string]int{
	"ticket":         50,
	"support ticket": 70,
}

func (c *CryptoScamNameEvaluator) Evaluate(_ context.Context, _ *AppContext, guild *guild.Guild) (int, error) {
	lower := strings.ToLower(guild.Name)

	var maxScore int
	for word, score := range cryptoScamWords {
		if strings.Contains(lower, word) {
			if score > maxScore {
				maxScore = score
			}
		}
	}

	return maxScore, nil
}

func (c *CryptoScamNameEvaluator) Properties() EvaluatorProperties {
	return EvaluatorProperties{
		RuleName:             "Guild name contains \"ticket\"",
		RuleType:             RuleTypeCryptoScam,
		ShouldSpawnGoroutine: false,
	}
}

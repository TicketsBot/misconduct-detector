package rules

var Ruleset = []Evaluator{
	&CryptoScamNameEvaluator{},
	&GeneralAccountAgeEvaluator{},
}

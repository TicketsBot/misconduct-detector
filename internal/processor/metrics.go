package processor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	MetricNamespace = "tickets"
	MetricSubsystem = "misconduct_detector"
)

var (
	ruleEvaluationCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricNamespace,
		Subsystem: MetricSubsystem,
		Name:      "rule_evaluations",
	}, []string{"rule"})

	ruleExecutionTimeHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: MetricNamespace,
		Subsystem: MetricSubsystem,
		Name:      "rule_execution_time",
	}, []string{"rule"})

	ruleScoreCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: MetricNamespace,
		Subsystem: MetricSubsystem,
		Name:      "rule_score",
	}, []string{"rule"})
)

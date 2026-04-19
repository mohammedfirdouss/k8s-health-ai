package metrics

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	ctrmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var diagnosesTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "diagnoses_total",
		Help: "Count of cluster diagnosis reconciliations by failure type and phase.",
	},
	[]string{"failure_type", "phase"},
)

var reconcileDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "reconcile_duration_seconds",
		Help:    "Duration of reconcile calls in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"result"},
)

var llmCallDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "llm_call_duration_seconds",
		Help:    "Duration of LLM provider calls in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"provider", "result"},
)

func init() {
	ctrmetrics.Registry.MustRegister(diagnosesTotal)
	ctrmetrics.Registry.MustRegister(reconcileDuration)
	ctrmetrics.Registry.MustRegister(llmCallDuration)
}

// RecordDiagnosis increments diagnoses_total. Phase should be lowercase: ready, error, or analyzing.
func RecordDiagnosis(failureType, phase string) {
	diagnosesTotal.WithLabelValues(failureType, strings.ToLower(phase)).Inc()
}

// ObserveReconcile records the duration of a reconcile call.
// Result should be one of: success, error, skipped.
func ObserveReconcile(result string, d time.Duration) {
	reconcileDuration.WithLabelValues(strings.ToLower(result)).Observe(d.Seconds())
}

// ObserveLLMCall records the duration of an LLM provider call.
// Result should be one of: success, error.
func ObserveLLMCall(provider, result string, d time.Duration) {
	llmCallDuration.WithLabelValues(provider, strings.ToLower(result)).Observe(d.Seconds())
}

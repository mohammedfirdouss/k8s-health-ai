package metrics

import (
	"strings"

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

func init() {
	ctrmetrics.Registry.MustRegister(diagnosesTotal)
}

// RecordDiagnosis increments diagnoses_total. Phase should be lowercase: ready, error, or analyzing.
func RecordDiagnosis(failureType, phase string) {
	diagnosesTotal.WithLabelValues(failureType, strings.ToLower(phase)).Inc()
}

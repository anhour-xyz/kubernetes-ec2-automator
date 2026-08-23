package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	namespaceLabel = "namespace"
	workloadLabel  = "workload"
)

var (
	managedWorkloadHealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "managed_workload_healthy",
			Help: "Whether a managed workload is healthy",
		},
		[]string{namespaceLabel, workloadLabel},
	)

	managedWorkloadRecoveries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "managed_workload_recoveries_total",
			Help: "Successful managed workload recoveries",
		},
		[]string{namespaceLabel, workloadLabel},
	)

	managedWorkloadRecoveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "managed_workload_recovery_duration_seconds",
			Help: "Time required for workload recovery",
		},
		[]string{namespaceLabel, workloadLabel},
	)
)

func init() {
	metrics.Registry.MustRegister(
		managedWorkloadHealthy,
		managedWorkloadRecoveries,
		managedWorkloadRecoveryDuration,
	)
}

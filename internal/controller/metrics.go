package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	managedWorkloadHealthy = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "managed_workload_healthy",
			Help: "Whether a managed workload is healthy",
		},
		[]string{"namespace", "workload"},
	)

	managedWorkloadRecoveries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "managed_workload_recoveries_total",
			Help: "Successful managed workload recoveries",
		},
		[]string{"namespace", "workload"},
	)

	managedWorkloadRecoveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "managed_workload_recovery_duration_seconds",
			Help: "Time required for workload recovery",
		},
		[]string{"namespace", "workload"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		managedWorkloadHealthy,
		managedWorkloadRecoveries,
		managedWorkloadRecoveryDuration,
	)
}

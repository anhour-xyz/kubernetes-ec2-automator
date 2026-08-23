package controller

import (
	"context"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	webappv1 "github.com/anhour-xyz/kubernetes-ec2-automator/api/v1"
)

// ManagedWorkloadReconciler reconciles a ManagedWorkload object
type ManagedWorkloadReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=webapp.cloud.com,resources=managedworkloads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.cloud.com,resources=managedworkloads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=webapp.cloud.com,resources=managedworkloads/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ManagedWorkload object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *ManagedWorkloadReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	workload := &webappv1.ManagedWorkload{}
	if err := r.Get(ctx, req.NamespacedName, workload); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	labels := map[string]string{
		"app.kubernetes.io/name":       workload.Name,
		"app.kubernetes.io/managed-by": "managedworkload-operator",
	}

	env := make([]corev1.EnvVar, 0, len(workload.Spec.Env))
	keys := make([]string, 0, len(workload.Spec.Env))

	for key := range workload.Spec.Env {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		env = append(env, corev1.EnvVar{
			Name:  key,
			Value: workload.Spec.Env[key],
		})
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.Name,
			Namespace: workload.Namespace,
		},
	}

	operation, err := controllerutil.CreateOrUpdate(
		ctx,
		r.Client,
		deployment,
		func() error {
			deployment.Labels = labels
			deployment.Spec.Replicas =
				ptr.To(workload.Spec.Replicas)

			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": workload.Name,
				},
			}

			container := corev1.Container{
				Name:  "application",
				Image: workload.Spec.Image,
				Env:   env,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("10m"),
						corev1.ResourceMemory: resource.MustParse("16Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("64Mi"),
					},
				},
			}

			if workload.Spec.Port > 0 {
				container.Ports = []corev1.ContainerPort{
					{
						Name:          "http",
						ContainerPort: workload.Spec.Port,
					},
				}

				probe := corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/",
						Port: intstr.FromString("http"),
					},
				}

				container.StartupProbe = &corev1.Probe{
					ProbeHandler:     probe,
					FailureThreshold: 30,
					PeriodSeconds:    2,
				}

				container.ReadinessProbe = &corev1.Probe{
					ProbeHandler:     probe,
					FailureThreshold: 3,
					PeriodSeconds:    5,
				}

				container.LivenessProbe = &corev1.Probe{
					ProbeHandler:     probe,
					FailureThreshold: 3,
					PeriodSeconds:    10,
				}
			}

			deployment.Spec.Template.ObjectMeta.Labels = labels
			deployment.Spec.Template.Spec.Containers =
				[]corev1.Container{container}

			return controllerutil.SetControllerReference(
				workload,
				deployment,
				r.Scheme,
			)
		},
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info(
		"Reconciled managed Deployment",
		"operation", operation,
		"deployment", deployment.Name,
	)

	return r.updateManagedWorkloadStatus(
		ctx,
		workload,
		deployment,
	)
}

func (r *ManagedWorkloadReconciler) updateManagedWorkloadStatus(
	ctx context.Context,
	workload *webappv1.ManagedWorkload,
	deployment *appsv1.Deployment,
) (ctrl.Result, error) {
	now := metav1.Now()
	healthy :=
		deployment.Status.AvailableReplicas >=
			workload.Spec.Replicas

	healthyValue := 0.0
	if healthy {
		healthyValue = 1.0
	}

	managedWorkloadHealthy.WithLabelValues(
		workload.Namespace,
		workload.Name,
	).Set(healthyValue)

	oldPhase := workload.Status.Phase
	phase := "Recovering"

	if healthy {
		phase = "Healthy"
	}

	if !healthy &&
		workload.Status.RecoveryStartedAt == nil {
		workload.Status.RecoveryStartedAt = &now
	}

	if healthy &&
		oldPhase == "Recovering" &&
		workload.Status.RecoveryStartedAt != nil {
		workload.Status.RecoveryCount++

		duration := now.Sub(
			workload.Status.RecoveryStartedAt.Time,
		)

		workload.Status.LastRecoveryDurationSeconds =
			int64(duration.Seconds())
		workload.Status.LastRecoveryTime = &now
		workload.Status.RecoveryStartedAt = nil

		managedWorkloadRecoveries.WithLabelValues(
			workload.Namespace,
			workload.Name,
		).Inc()

		managedWorkloadRecoveryDuration.WithLabelValues(
			workload.Namespace,
			workload.Name,
		).Observe(duration.Seconds())
	}

	workload.Status.Phase = phase
	workload.Status.ReadyReplicas =
		deployment.Status.ReadyReplicas
	workload.Status.AvailableReplicas =
		deployment.Status.AvailableReplicas

	conditionStatus := metav1.ConditionFalse
	reason := "ReplicasUnavailable"
	message := "Waiting for all replicas to become available"

	if healthy {
		conditionStatus = metav1.ConditionTrue
		reason = "AllReplicasAvailable"
		message = "All desired replicas are available"
	}

	meta.SetStatusCondition(
		&workload.Status.Conditions,
		metav1.Condition{
			Type:               "Available",
			Status:             conditionStatus,
			ObservedGeneration: workload.Generation,
			Reason:             reason,
			Message:            message,
		},
	)

	if err := r.Status().Update(ctx, workload); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{
		RequeueAfter: 15 * time.Second,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ManagedWorkloadReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.ManagedWorkload{}).
		Owns(&appsv1.Deployment{}).
		Named("managedworkload").
		Complete(r)
}

package controller

import (
	"context"
	"time"

	webappv1 "github.com/anhour-xyz/kubernetes-ec2-automator/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const ec2Finalizer = "webapp.cloud.com/ec2-finalizer"

// EC2InstanceReconciler reconciles an EC2Instance object.
type EC2InstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=webapp.cloud.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.cloud.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=webapp.cloud.com,resources=ec2instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EC2Instance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *EC2InstanceReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	l := logf.FromContext(ctx)

	ec2Instance := &webappv1.EC2Instance{}
	if err := r.Get(
		ctx,
		req.NamespacedName,
		ec2Instance,
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle Deletion
	if !ec2Instance.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(
			ec2Instance,
			ec2Finalizer,
		) {
			return ctrl.Result{}, nil
		}

		if ec2Instance.Status.InstanceID != "" {
			ec2Client, err := awsClient(
				ctx,
				ec2Instance.Spec.Region,
			)

			if err != nil {
				return ctrl.Result{}, err
			}

			if err := deleteEC2Instance(
				ctx,
				ec2Client,
				ec2Instance.Status.InstanceID,
			); err != nil {
				return ctrl.Result{}, err
			}
		}

		controllerutil.RemoveFinalizer(
			ec2Instance,
			ec2Finalizer,
		)

		if err := r.Update(ctx, ec2Instance); err != nil {
			return ctrl.Result{}, err
		}

		l.Info(
			"EC2 instance deleted",
			"instanceID", ec2Instance.Status.InstanceID,
		)
		return ctrl.Result{}, nil
	}

	// Add finalizer before create external resources
	if !controllerutil.ContainsFinalizer(
		ec2Instance,
		ec2Finalizer,
	) {
		controllerutil.AddFinalizer(
			ec2Instance,
			ec2Finalizer,
		)

		if err := r.Update(ctx, ec2Instance); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// Create EC2 instance only when no instance ID is recorded
	if ec2Instance.Status.InstanceID == "" {
		instanceID, err := createEC2Instance(
			ctx,
			ec2Instance,
			ec2Instance.Spec.Region,
		)
		if err != nil {
			return ctrl.Result{}, err
		}

		ec2Instance.Status.InstanceID = instanceID
		ec2Instance.Status.Phase = "running"

		if err := r.Status().Update(
			ctx,
			ec2Instance,
		); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	// Check existing EC2 instance
	exists, instance, err := checkEC2InstanceExists(
		ctx,
		ec2Instance.Status.InstanceID,
		ec2Instance,
	)

	if err != nil {
		return ctrl.Result{}, err
	}

	phase := "not-found"
	publicIP := ""

	if exists {
		if instance.State != nil {
			phase = string(instance.State.Name)
		}
		publicIP = derefString(instance.PublicIpAddress)
	}

	statusChanged := ec2Instance.Status.Phase != phase || ec2Instance.Status.PublicIP != publicIP

	if statusChanged {
		ec2Instance.Status.Phase = phase
		ec2Instance.Status.PublicIP = publicIP

		if err := r.Status().Update(
			ctx, ec2Instance,
		); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{
		RequeueAfter: 30 * time.Second,
	}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *EC2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.EC2Instance{}).
		Named("ec2instance").
		Complete(r)
}

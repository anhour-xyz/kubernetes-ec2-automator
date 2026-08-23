package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	webappv1 "github.com/anhour-xyz/kubernetes-ec2-automator/api/v1"
)

var _ = Describe("ManagedWorkload Controller", func() {
	const (
		resourceName      = "test-resource"
		resourceNamespace = "default"
	)

	ctx := context.Background()

	namespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: resourceNamespace,
	}

	BeforeEach(func() {
		resource := &webappv1.ManagedWorkload{}
		err := k8sClient.Get(
			ctx,
			namespacedName,
			resource,
		)

		if errors.IsNotFound(err) {
			resource = &webappv1.ManagedWorkload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: webappv1.ManagedWorkloadSpec{
					Image:    "nginx:1.29",
					Replicas: 2,
					Port:     80,
				},
			}

			Expect(
				k8sClient.Create(ctx, resource),
			).To(Succeed())

			return
		}

		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		deployment := &appsv1.Deployment{}
		err := k8sClient.Get(
			ctx,
			namespacedName,
			deployment,
		)

		if err == nil {
			Expect(
				k8sClient.Delete(ctx, deployment),
			).To(Succeed())
		} else {
			Expect(errors.IsNotFound(err)).To(BeTrue())
		}

		resource := &webappv1.ManagedWorkload{}
		err = k8sClient.Get(
			ctx,
			namespacedName,
			resource,
		)

		if err == nil {
			Expect(
				k8sClient.Delete(ctx, resource),
			).To(Succeed())
		} else {
			Expect(errors.IsNotFound(err)).To(BeTrue())
		}
	})

	It("creates the desired Deployment", func() {
		reconciler := &ManagedWorkloadReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		_, err := reconciler.Reconcile(
			ctx,
			reconcile.Request{
				NamespacedName: namespacedName,
			},
		)
		Expect(err).NotTo(HaveOccurred())

		deployment := &appsv1.Deployment{}
		Expect(k8sClient.Get(
			ctx,
			namespacedName,
			deployment,
		)).To(Succeed())

		Expect(*deployment.Spec.Replicas).
			To(Equal(int32(2)))

		Expect(deployment.Spec.Template.Spec.Containers).
			To(HaveLen(1))

		Expect(
			deployment.Spec.Template.Spec.Containers[0].Image,
		).To(Equal("nginx:1.29"))

		Expect(
			deployment.Spec.Template.Spec.Containers[0].
				ReadinessProbe,
		).NotTo(BeNil())

		Expect(
			deployment.Spec.Template.Spec.Containers[0].
				LivenessProbe,
		).NotTo(BeNil())
	})
})

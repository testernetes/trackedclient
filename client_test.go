package trackedclient

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("trackedclient", func() {
	When("creating a tracked client", Ordered, func() {

		var (
			trackedClient TrackedClient
			cm            *corev1.ConfigMap
			ctx           context.Context
		)

		BeforeAll(func() {
			var err error
			trackedClient, err = New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(trackedClient).ShouldNot(BeNil())

			ctx = context.Background()
			cm = &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"}}
		})

		It("should create objects", func() {
			Expect(trackedClient.Create(ctx, cm)).Should(Succeed())
		})

		It("should track objects and delete them", func() {
			Expect(trackedClient.DeleteAllTracked(ctx)).Should(Succeed())
			Expect(k8sErrors.IsNotFound(trackedClient.Get(ctx, client.ObjectKeyFromObject(cm), cm))).Should(BeTrue())
		})
	})
})

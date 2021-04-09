package controllers

import (
	"context"
	"github.com/aiven/aiven-k8s-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"
)

var _ = Describe("PG Controller", func() {
	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		pgNamespace = "default"

		timeout  = time.Minute * 20
		interval = time.Second * 10
	)

	var (
		pg          *v1alpha1.PG
		serviceName string
		ctx         context.Context
	)

	BeforeEach(func() {
		serviceName = "k8s-test-pg-acc-" + generateRandomID()

		pg = &v1alpha1.PG{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "k8s-operator.aiven.io/v1alpha1",
				Kind:       "PG",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: pgNamespace,
			},
			Spec: v1alpha1.PGSpec{
				Project:               os.Getenv("AIVEN_PROJECT_NAME"),
				ServiceName:           serviceName,
				Plan:                  "business-4",
				CloudName:             "google-europe-west1",
				MaintenanceWindowDow:  "monday",
				MaintenanceWindowTime: "10:00:00",
				PGUserConfig: v1alpha1.PGUserConfig{
					PgVersion: "12",
					PublicAccess: v1alpha1.PublicAccessUserConfig{
						Pg:         boolPointer(true),
						Prometheus: boolPointer(true),
					},
					Pg: v1alpha1.PGSubPGUserConfig{
						IdleInTransactionSessionTimeout: int64Pointer(900),
					},
				},
			},
		}
		ctx = context.Background()

		By("Creating a new PG CR instance")
		Expect(k8sClient.Create(ctx, pg)).Should(Succeed())

		pgLookupKey := types.NamespacedName{Name: serviceName, Namespace: pgNamespace}
		createdPG := &v1alpha1.PG{}
		// We'll need to retry getting this newly created PG,
		// given that creation may not immediately happen.
		By("by retrieving PG instance from k8s")
		Eventually(func() bool {
			err := k8sClient.Get(ctx, pgLookupKey, createdPG)

			return err == nil
		}, timeout, interval).Should(BeTrue())

		By("by waiting PG service status to become RUNNING")
		Eventually(func() string {
			err := k8sClient.Get(ctx, pgLookupKey, createdPG)
			if err == nil {
				return createdPG.Status.State
			}

			return ""
		}, timeout, interval).Should(Equal("RUNNING"))

		By("by checking finalizers")
		Expect(createdPG.GetFinalizers()).ToNot(BeEmpty())
	})

	Context("Validating PG reconciler behaviour", func() {
		It("should create a new PG service", func() {
			createdPG := &v1alpha1.PG{}
			pgLookupKey := types.NamespacedName{Name: serviceName, Namespace: pgNamespace}

			Expect(k8sClient.Get(ctx, pgLookupKey, createdPG)).Should(Succeed())

			// Let's make sure our PG status was properly populated.
			By("by checking that after creation PG service status fields were properly populated")
			Expect(createdPG.Status.ServiceName).Should(Equal(serviceName))
			Expect(createdPG.Status.State).Should(Equal("RUNNING"))
			Expect(createdPG.Status.Plan).Should(Equal("business-4"))
			Expect(createdPG.Status.CloudName).Should(Equal("google-europe-west1"))
			Expect(createdPG.Status.MaintenanceWindowDow).Should(Equal("monday"))
			Expect(createdPG.Status.MaintenanceWindowTime).Should(Equal("10:00:00"))
		})
	})

	AfterEach(func() {
		By("Ensures that PG instance was deleted")
		ensureDelete(ctx, pg)
	})
})
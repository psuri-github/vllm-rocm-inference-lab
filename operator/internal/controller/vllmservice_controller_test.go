/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	inferencev1alpha1 "github.com/psuri-github/vllm-rocm-inference-lab/operator/api/v1alpha1"
)

var _ = Describe("VLLMService Controller", func() {
	Context("When reconciling a resource", func() {
		const (
			resourceName      = "test-resource"
			resourceNamespace = "default"
		)

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: resourceNamespace,
		}
		vllmservice := &inferencev1alpha1.VLLMService{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind VLLMService")
			err := k8sClient.Get(ctx, typeNamespacedName, vllmservice)
			if err != nil && errors.IsNotFound(err) {
				resource := &inferencev1alpha1.VLLMService{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: resourceNamespace,
					},
					Spec: inferencev1alpha1.VLLMServiceSpec{
						Model: "test-model",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
			Expect(k8sClient.Get(ctx, typeNamespacedName, vllmservice)).To(Succeed())
		})

		AfterEach(func() {
			pvcKey := types.NamespacedName{
				Name:      resourceName + "-model-cache",
				Namespace: resourceNamespace,
			}
			pvc := &corev1.PersistentVolumeClaim{}
			if err := k8sClient.Get(ctx, pvcKey, pvc); err == nil {
				By("Cleaning up the model-cache PersistentVolumeClaim")
				Expect(k8sClient.Delete(ctx, pvc)).To(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).To(BeTrue())
			}
			resource := &inferencev1alpha1.VLLMService{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance VLLMService")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should create and own a model-cache PersistentVolumeClaim idempotently", func() {
			By("Reconciling the created resource")
			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			pvc := &corev1.PersistentVolumeClaim{}
			pvcKey := types.NamespacedName{
				Name:      resourceName + "-model-cache",
				Namespace: resourceNamespace,
			}

			Expect(k8sClient.Get(ctx, pvcKey, pvc)).To(Succeed())

			Expect(pvc.Spec.AccessModes).To(Equal(
				[]corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			))

			storageRequest, found := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
			Expect(found).To(BeTrue())
			Expect(storageRequest.String()).To(Equal("20Gi"))

			Expect(pvc.Labels).To(HaveKeyWithValue("app.kubernetes.io/name", "vllm"))
			Expect(pvc.Labels).To(HaveKeyWithValue("app.kubernetes.io/instance", resourceName))
			Expect(pvc.Labels).To(HaveKeyWithValue("app.kubernetes.io/component", "model-cache"))
			Expect(pvc.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "vllm-operator"))

			Expect(metav1.IsControlledBy(pvc, vllmservice)).To(BeTrue())

			By("Reconciling the same resource again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When reconciling a resource that does not exist", func() {
		It("should return without an error", func() {
			By("Reconciling a missing VLLMService")

			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(
				context.Background(),
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "missing-resource",
						Namespace: "default",
					},
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

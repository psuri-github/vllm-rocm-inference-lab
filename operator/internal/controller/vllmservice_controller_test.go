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
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

				if len(pvc.Finalizers) > 0 {
					pvc.Finalizers = nil
					Expect(client.IgnoreNotFound(
						k8sClient.Update(ctx, pvc),
					)).To(Succeed())
				}

				Expect(client.IgnoreNotFound(
					k8sClient.Delete(ctx, pvc),
				)).To(Succeed())

				Eventually(func() bool {
					current := &corev1.PersistentVolumeClaim{}
					err := k8sClient.Get(ctx, pvcKey, current)
					return errors.IsNotFound(err)
				}).Should(BeTrue())
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
			By("Simulating label drift")
			pvc.Labels["app.kubernetes.io/managed-by"] = "manual-change"
			pvc.Labels["example.com/custom"] = "preserve-me"
			Expect(k8sClient.Update(ctx, pvc)).To(Succeed())

			By("Reconciling the label drift")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			updatedPVC := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(ctx, pvcKey, updatedPVC)).To(Succeed())

			Expect(updatedPVC.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/managed-by",
				"vllm-operator",
			))
			Expect(updatedPVC.Labels).To(HaveKeyWithValue(
				"example.com/custom",
				"preserve-me",
			))
		})
		It("should reject a same-named PVC that it does not control", func() {
			By("Creating a same-named PVC without a controller owner reference")
			pvc := newModelCachePVC(vllmservice)
			Expect(k8sClient.Create(ctx, pvc)).To(Succeed())

			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"is not controlled by VLLMService",
			))
		})
		It("should increase the model-cache PVC storage request", func() {
			storageClassName := "expandable-test"
			allowExpansion := true

			storageClass := &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: storageClassName,
				},
				Provisioner:          "example.com/test",
				AllowVolumeExpansion: &allowExpansion,
			}
			Expect(k8sClient.Create(ctx, storageClass)).To(Succeed())

			DeferCleanup(func() {
				storageClass := &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: storageClassName,
					},
				}
				Expect(client.IgnoreNotFound(
					k8sClient.Delete(ctx, storageClass),
				)).To(Succeed())
			})

			vllmservice.Spec.StorageClassName = &storageClassName
			Expect(k8sClient.Update(ctx, vllmservice)).To(Succeed())

			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			By("Creating the model-cache PVC")
			_, err := controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			pvcKey := types.NamespacedName{
				Name:      resourceName + "-model-cache",
				Namespace: resourceNamespace,
			}
			pvc := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(ctx, pvcKey, pvc)).To(Succeed())

			By("Simulating the storage controller binding the PVC")
			pvc.Status.Phase = corev1.ClaimBound
			Expect(k8sClient.Status().Update(ctx, pvc)).To(Succeed())

			By("Increasing the requested model-cache size")
			expandedSize := apiresource.MustParse("30Gi")
			vllmservice.Spec.ModelCacheSize = &expandedSize
			Expect(k8sClient.Update(ctx, vllmservice)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			updatedPVC := &corev1.PersistentVolumeClaim{}
			Expect(k8sClient.Get(ctx, pvcKey, updatedPVC)).To(Succeed())

			actualSize := updatedPVC.Spec.Resources.Requests[corev1.ResourceStorage]
			Expect(actualSize.Cmp(expandedSize)).To(Equal(0))
		})
		It("should reject decreasing the model-cache PVC storage request", func() {
			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			By("Creating the model-cache PVC")
			_, err := controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			By("Requesting a smaller model-cache size")
			smallerSize := apiresource.MustParse("10Gi")
			vllmservice.Spec.ModelCacheSize = &smallerSize
			Expect(k8sClient.Update(ctx, vllmservice)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"cannot decrease storage request",
			))

			pvc := &corev1.PersistentVolumeClaim{}
			pvcKey := types.NamespacedName{
				Name:      resourceName + "-model-cache",
				Namespace: resourceNamespace,
			}
			Expect(k8sClient.Get(ctx, pvcKey, pvc)).To(Succeed())

			actualSize := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
			Expect(actualSize.String()).To(Equal("20Gi"))
		})
		It("should reject changing the model-cache PVC storage class", func() {
			initialStorageClassName := "initial-test"
			vllmservice.Spec.StorageClassName = &initialStorageClassName
			Expect(k8sClient.Update(ctx, vllmservice)).To(Succeed())

			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			By("Creating the model-cache PVC with the initial StorageClass")
			_, err := controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			By("Requesting a different StorageClass")
			replacementStorageClassName := "replacement-test"
			vllmservice.Spec.StorageClassName = &replacementStorageClassName
			Expect(k8sClient.Update(ctx, vllmservice)).To(Succeed())

			_, err = controllerReconciler.Reconcile(ctx, request)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				`cannot change storageClassName`,
			))
			Expect(err.Error()).To(ContainSubstring(
				`from "initial-test" to "replacement-test"`,
			))

			By("Confirming that the PVC retained its original StorageClass")
			pvc := &corev1.PersistentVolumeClaim{}
			pvcKey := types.NamespacedName{
				Name:      resourceName + "-model-cache",
				Namespace: resourceNamespace,
			}
			Expect(k8sClient.Get(ctx, pvcKey, pvc)).To(Succeed())
			Expect(pvc.Spec.StorageClassName).NotTo(BeNil())
			Expect(*pvc.Spec.StorageClassName).To(Equal(initialStorageClassName))
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

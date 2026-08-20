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
	appsv1 "k8s.io/api/apps/v1"
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
			deploymentKey := types.NamespacedName{
				Name:      resourceName,
				Namespace: resourceNamespace,
			}
			deployment := &appsv1.Deployment{}

			if err := k8sClient.Get(ctx, deploymentKey, deployment); err == nil {
				By("Cleaning up the vLLM Deployment")
				Expect(k8sClient.Delete(ctx, deployment)).To(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).To(BeTrue())
			}
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

			serviceKey := types.NamespacedName{
				Name:      resourceName,
				Namespace: resourceNamespace,
			}
			service := &corev1.Service{}

			if err := k8sClient.Get(ctx, serviceKey, service); err == nil {
				By("Cleaning up the vLLM Service")
				Expect(k8sClient.Delete(ctx, service)).To(Succeed())
			} else {
				Expect(errors.IsNotFound(err)).To(BeTrue())
			}
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
		It("should create and own a vLLM Service idempotently", func() {
			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			By("Reconciling the VLLMService")
			_, err := controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			serviceKey := types.NamespacedName{
				Name:      resourceName,
				Namespace: resourceNamespace,
			}
			service := &corev1.Service{}
			Expect(k8sClient.Get(ctx, serviceKey, service)).To(Succeed())

			Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			Expect(service.Spec.Selector).To(Equal(map[string]string{
				"app.kubernetes.io/name":      "vllm",
				"app.kubernetes.io/instance":  resourceName,
				"app.kubernetes.io/component": "server",
			}))

			Expect(service.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/managed-by",
				"vllm-operator",
			))

			Expect(service.Spec.Ports).To(HaveLen(1))
			servicePort := service.Spec.Ports[0]
			Expect(servicePort.Name).To(Equal("http"))
			Expect(servicePort.Protocol).To(Equal(corev1.ProtocolTCP))
			Expect(servicePort.Port).To(Equal(int32(8000)))
			Expect(servicePort.TargetPort.String()).To(Equal("http"))

			Expect(metav1.IsControlledBy(service, vllmservice)).To(BeTrue())

			By("Reconciling the same VLLMService again")
			_, err = controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should reject a same-named Service that it does not control", func() {
			By("Creating a same-named Service without a controller owner reference")
			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: resourceNamespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Name: "http",
							Port: 9000,
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, service)).To(Succeed())

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

			By("Confirming that the conflicting Service was not modified")
			existingService := &corev1.Service{}
			serviceKey := types.NamespacedName{
				Name:      resourceName,
				Namespace: resourceNamespace,
			}
			Expect(k8sClient.Get(ctx, serviceKey, existingService)).To(Succeed())
			Expect(existingService.OwnerReferences).To(BeEmpty())
			Expect(existingService.Spec.Ports).To(HaveLen(1))
			Expect(existingService.Spec.Ports[0].Port).To(Equal(int32(9000)))
		})
		It("should repair vLLM Service drift while preserving API-server fields", func() {
			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			By("Creating the vLLM Service")
			_, err := controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			serviceKey := types.NamespacedName{
				Name:      resourceName,
				Namespace: resourceNamespace,
			}
			service := &corev1.Service{}
			Expect(k8sClient.Get(ctx, serviceKey, service)).To(Succeed())

			originalClusterIP := service.Spec.ClusterIP
			originalClusterIPs := append([]string(nil), service.Spec.ClusterIPs...)

			By("Introducing label, type, selector, and port drift")
			service.Labels["team"] = "ai-platform"
			service.Labels["app.kubernetes.io/managed-by"] = "manual-change"
			delete(service.Labels, "app.kubernetes.io/component")

			service.Spec.Type = corev1.ServiceTypeNodePort
			service.Spec.Selector = map[string]string{
				"unexpected": "selector",
			}
			service.Spec.Ports[0].Name = "unexpected"
			service.Spec.Ports[0].Port = 9000

			Expect(k8sClient.Update(ctx, service)).To(Succeed())

			By("Changing the desired port on the VLLMService")
			Expect(k8sClient.Get(ctx, typeNamespacedName, vllmservice)).To(Succeed())
			vllmservice.Spec.Port = 8080
			Expect(k8sClient.Update(ctx, vllmservice)).To(Succeed())

			By("Reconciling the drifted Service")
			_, err = controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			updatedService := &corev1.Service{}
			Expect(k8sClient.Get(ctx, serviceKey, updatedService)).To(Succeed())

			Expect(updatedService.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/name",
				"vllm",
			))
			Expect(updatedService.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/instance",
				resourceName,
			))
			Expect(updatedService.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/component",
				"server",
			))
			Expect(updatedService.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/managed-by",
				"vllm-operator",
			))
			Expect(updatedService.Labels).To(HaveKeyWithValue(
				"team",
				"ai-platform",
			))

			Expect(updatedService.Spec.Type).To(Equal(
				corev1.ServiceTypeClusterIP,
			))
			Expect(updatedService.Spec.Selector).To(Equal(map[string]string{
				"app.kubernetes.io/name":      "vllm",
				"app.kubernetes.io/instance":  resourceName,
				"app.kubernetes.io/component": "server",
			}))

			Expect(updatedService.Spec.Ports).To(HaveLen(1))
			servicePort := updatedService.Spec.Ports[0]
			Expect(servicePort.Name).To(Equal("http"))
			Expect(servicePort.Protocol).To(Equal(corev1.ProtocolTCP))
			Expect(servicePort.Port).To(Equal(int32(8080)))
			Expect(servicePort.TargetPort.String()).To(Equal("http"))

			Expect(updatedService.Spec.ClusterIP).To(Equal(originalClusterIP))
			Expect(updatedService.Spec.ClusterIPs).To(Equal(originalClusterIPs))

			By("Reconciling again without any drift")
			resourceVersion := updatedService.ResourceVersion

			_, err = controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			reconciledAgain := &corev1.Service{}
			Expect(k8sClient.Get(ctx, serviceKey, reconciledAgain)).To(Succeed())
			Expect(reconciledAgain.ResourceVersion).To(Equal(resourceVersion))
		})
		It("should create and own a vLLM Deployment idempotently", func() {
			controllerReconciler := &VLLMServiceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}
			request := reconcile.Request{
				NamespacedName: typeNamespacedName,
			}

			By("Reconciling the VLLMService")
			_, err := controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			deploymentKey := types.NamespacedName{
				Name:      resourceName,
				Namespace: resourceNamespace,
			}
			deployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, deploymentKey, deployment)).To(Succeed())

			Expect(metav1.IsControlledBy(deployment, vllmservice)).To(BeTrue())

			Expect(deployment.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/managed-by",
				"vllm-operator",
			))
			Expect(deployment.Spec.Replicas).NotTo(BeNil())
			Expect(*deployment.Spec.Replicas).To(Equal(int32(1)))

			expectedSelector := map[string]string{
				"app.kubernetes.io/name":      "vllm",
				"app.kubernetes.io/instance":  resourceName,
				"app.kubernetes.io/component": "server",
			}
			Expect(deployment.Spec.Selector.MatchLabels).To(Equal(expectedSelector))
			Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/name",
				"vllm",
			))
			Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/instance",
				resourceName,
			))
			Expect(deployment.Spec.Template.Labels).To(HaveKeyWithValue(
				"app.kubernetes.io/component",
				"server",
			))

			Expect(deployment.Spec.Template.Spec.Containers).To(HaveLen(1))
			container := deployment.Spec.Template.Spec.Containers[0]

			Expect(container.Name).To(Equal("vllm"))
			Expect(container.Image).To(Equal(
				"vllm/vllm-openai-rocm:latest",
			))
			Expect(container.ImagePullPolicy).To(Equal(corev1.PullIfNotPresent))
			Expect(container.Args).To(Equal([]string{
				"test-model",
				"--host",
				"0.0.0.0",
				"--port",
				"8000",
				"--dtype",
				"bfloat16",
				"--max-model-len",
				"4096",
				"--gpu-memory-utilization",
				"0.5",
				"--generation-config",
				"vllm",
			}))

			Expect(container.Ports).To(HaveLen(1))
			Expect(container.Ports[0].Name).To(Equal("http"))
			Expect(container.Ports[0].ContainerPort).To(Equal(int32(8000)))
			Expect(container.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))

			amdGPU := corev1.ResourceName("amd.com/gpu")

			cpuRequest := container.Resources.Requests[corev1.ResourceCPU]
			memoryRequest := container.Resources.Requests[corev1.ResourceMemory]
			gpuRequest := container.Resources.Requests[amdGPU]

			Expect(cpuRequest.String()).To(Equal("4"))
			Expect(memoryRequest.String()).To(Equal("16Gi"))
			Expect(gpuRequest.String()).To(Equal("1"))

			cpuLimit := container.Resources.Limits[corev1.ResourceCPU]
			memoryLimit := container.Resources.Limits[corev1.ResourceMemory]
			gpuLimit := container.Resources.Limits[amdGPU]

			Expect(cpuLimit.String()).To(Equal("16"))
			Expect(memoryLimit.String()).To(Equal("64Gi"))
			Expect(gpuLimit.String()).To(Equal("1"))

			Expect(container.VolumeMounts).To(HaveLen(2))
			Expect(container.VolumeMounts[0].Name).To(Equal("model-cache"))
			Expect(container.VolumeMounts[0].MountPath).To(Equal(
				"/root/.cache/huggingface",
			))
			Expect(container.VolumeMounts[1].Name).To(Equal("shared-memory"))
			Expect(container.VolumeMounts[1].MountPath).To(Equal("/dev/shm"))

			volumes := deployment.Spec.Template.Spec.Volumes
			Expect(volumes).To(HaveLen(2))

			Expect(volumes[0].Name).To(Equal("model-cache"))
			Expect(volumes[0].PersistentVolumeClaim).NotTo(BeNil())
			Expect(volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(
				resourceName + "-model-cache",
			))

			Expect(volumes[1].Name).To(Equal("shared-memory"))
			Expect(volumes[1].EmptyDir).NotTo(BeNil())
			Expect(volumes[1].EmptyDir.Medium).To(Equal(
				corev1.StorageMediumMemory,
			))
			Expect(volumes[1].EmptyDir.SizeLimit).NotTo(BeNil())
			Expect(volumes[1].EmptyDir.SizeLimit.String()).To(Equal("8Gi"))

			Expect(container.StartupProbe).NotTo(BeNil())
			Expect(container.StartupProbe.HTTPGet.Path).To(Equal("/health"))
			Expect(container.StartupProbe.HTTPGet.Port.String()).To(Equal("http"))
			Expect(container.StartupProbe.PeriodSeconds).To(Equal(int32(10)))
			Expect(container.StartupProbe.FailureThreshold).To(Equal(int32(90)))

			Expect(container.ReadinessProbe).NotTo(BeNil())
			Expect(container.ReadinessProbe.PeriodSeconds).To(Equal(int32(5)))
			Expect(container.ReadinessProbe.FailureThreshold).To(Equal(int32(3)))

			Expect(container.LivenessProbe).NotTo(BeNil())
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(10)))
			Expect(container.LivenessProbe.FailureThreshold).To(Equal(int32(3)))

			Expect(container.SecurityContext).NotTo(BeNil())
			Expect(container.SecurityContext.SeccompProfile).NotTo(BeNil())
			Expect(container.SecurityContext.SeccompProfile.Type).To(Equal(
				corev1.SeccompProfileTypeUnconfined,
			))
			Expect(container.SecurityContext.Capabilities).NotTo(BeNil())
			Expect(container.SecurityContext.Capabilities.Add).To(ContainElement(
				corev1.Capability("SYS_PTRACE"),
			))

			By("Reconciling again without changing the desired state")
			resourceVersion := deployment.ResourceVersion

			_, err = controllerReconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			reconciledAgain := &appsv1.Deployment{}
			Expect(k8sClient.Get(
				ctx,
				deploymentKey,
				reconciledAgain,
			)).To(Succeed())
			Expect(reconciledAgain.ResourceVersion).To(Equal(resourceVersion))
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

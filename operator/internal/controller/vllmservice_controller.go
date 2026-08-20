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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	inferencev1alpha1 "github.com/psuri-github/vllm-rocm-inference-lab/operator/api/v1alpha1"
)

// VLLMServiceReconciler reconciles a VLLMService object
type VLLMServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=inference.psuri-github.github.io,resources=vllmservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=inference.psuri-github.github.io,resources=vllmservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=inference.psuri-github.github.io,resources=vllmservices/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VLLMService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *VLLMServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	vllmService := &inferencev1alpha1.VLLMService{}
	if err := r.Get(ctx, req.NamespacedName, vllmService); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, fmt.Errorf("failed to get VLLMService: %w", err)
	}

	log.Info(
		"Reconciling VLLMService",
		"name", vllmService.Name,
		"namespace", vllmService.Namespace,
		"generation", vllmService.Generation,
	)

	if err := r.ensureModelCachePVC(ctx, vllmService); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to ensure model-cache PVC: %w", err)
	}

	if err := r.ensureVLLMServerService(ctx, vllmService); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to ensure vLLM Service: %w", err)
	}

	return ctrl.Result{}, nil
}

func modelCachePVCName(vllmService *inferencev1alpha1.VLLMService) string {
	return vllmService.Name + "-model-cache"
}

func labelsForVLLMService(vllmService *inferencev1alpha1.VLLMService) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "vllm",
		"app.kubernetes.io/instance":   vllmService.Name,
		"app.kubernetes.io/managed-by": "vllm-operator",
	}
}

func labelsForModelCachePVC(vllmService *inferencev1alpha1.VLLMService) map[string]string {
	labels := labelsForVLLMService(vllmService)
	labels["app.kubernetes.io/component"] = "model-cache"
	return labels
}

func labelsForVLLMServer(vllmService *inferencev1alpha1.VLLMService) map[string]string {
	labels := labelsForVLLMService(vllmService)
	labels["app.kubernetes.io/component"] = "server"
	return labels
}

func selectorLabelsForVLLMServer(
	vllmService *inferencev1alpha1.VLLMService,
) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":      "vllm",
		"app.kubernetes.io/instance":  vllmService.Name,
		"app.kubernetes.io/component": "server",
	}
}

func newModelCachePVC(vllmService *inferencev1alpha1.VLLMService) *corev1.PersistentVolumeClaim {
	var storageClassName *string
	if vllmService.Spec.StorageClassName != nil {
		value := *vllmService.Spec.StorageClassName
		storageClassName = &value
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelCachePVCName(vllmService),
			Namespace: vllmService.Namespace,
			Labels:    labelsForModelCachePVC(vllmService),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: storageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: vllmService.Spec.ModelCacheSize.DeepCopy(),
				},
			},
		},
	}
}

func (r *VLLMServiceReconciler) ensureModelCachePVC(
	ctx context.Context,
	vllmService *inferencev1alpha1.VLLMService,
) error {
	if vllmService.Spec.ModelCacheSize == nil {
		return fmt.Errorf(
			"modelCacheSize is missing from VLLMService %s/%s",
			vllmService.Namespace,
			vllmService.Name,
		)
	}
	key := client.ObjectKey{
		Name:      modelCachePVCName(vllmService),
		Namespace: vllmService.Namespace,
	}

	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, key, pvc); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get PersistentVolumeClaim %s: %w", key, err)
		}

		pvc = newModelCachePVC(vllmService)

		if err := controllerutil.SetControllerReference(vllmService, pvc, r.Scheme); err != nil {
			return fmt.Errorf("failed to set VLLMService as PVC owner: %w", err)
		}

		if err := r.Create(ctx, pvc); err != nil {
			return fmt.Errorf("failed to create PersistentVolumeClaim %s: %w", key, err)
		}

		logf.FromContext(ctx).Info(
			"Created PersistentVolumeClaim",
			"name", pvc.Name,
			"namespace", pvc.Namespace,
		)
		return nil
	}
	if !metav1.IsControlledBy(pvc, vllmService) {
		return fmt.Errorf(
			"existing PVC %s is not controlled by VLLMService %s/%s",
			key,
			vllmService.Namespace,
			vllmService.Name,
		)
	}

	if vllmService.Spec.StorageClassName != nil {
		actualStorageClassName := "<none>"
		if pvc.Spec.StorageClassName != nil {
			actualStorageClassName = *pvc.Spec.StorageClassName
		}

		if actualStorageClassName != *vllmService.Spec.StorageClassName {
			return fmt.Errorf(
				"cannot change storageClassName for PVC %s from %q to %q",
				key,
				actualStorageClassName,
				*vllmService.Spec.StorageClassName,
			)
		}
	}

	original := pvc.DeepCopy()
	pvcChanged := false

	if pvc.Labels == nil {
		pvc.Labels = map[string]string{}
	}

	for labelKey, labelValue := range labelsForModelCachePVC(vllmService) {
		if pvc.Labels[labelKey] != labelValue {
			pvc.Labels[labelKey] = labelValue
			pvcChanged = true
		}
	}

	desiredStorage := vllmService.Spec.ModelCacheSize.DeepCopy()

	if pvc.Spec.Resources.Requests == nil {
		pvc.Spec.Resources.Requests = corev1.ResourceList{}
	}

	currentStorage, found := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !found {
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = desiredStorage
		pvcChanged = true
	} else {
		comparison := desiredStorage.Cmp(currentStorage)

		if comparison > 0 {
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = desiredStorage
			pvcChanged = true
		} else if comparison < 0 {
			return fmt.Errorf(
				"cannot decrease storage request for PVC %s from %s to %s",
				key,
				currentStorage.String(),
				desiredStorage.String(),
			)
		}
	}

	if !pvcChanged {
		return nil
	}

	if err := r.Patch(ctx, pvc, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch PersistentVolumeClaim %s: %w", key, err)
	}

	logf.FromContext(ctx).Info(
		"Updated PersistentVolumeClaim",
		"name", pvc.Name,
		"namespace", pvc.Namespace,
	)

	return nil
}

func vllmServerServiceName(
	vllmService *inferencev1alpha1.VLLMService,
) string {
	return vllmService.Name
}

func newVLLMServerService(
	vllmService *inferencev1alpha1.VLLMService,
) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vllmServerServiceName(vllmService),
			Namespace: vllmService.Namespace,
			Labels:    labelsForVLLMServer(vllmService),
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selectorLabelsForVLLMServer(vllmService),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       vllmService.Spec.Port,
					TargetPort: intstr.FromString("http"),
				},
			},
		},
	}
}

func (r *VLLMServiceReconciler) ensureVLLMServerService(
	ctx context.Context,
	vllmService *inferencev1alpha1.VLLMService,
) error {
	key := client.ObjectKey{
		Name:      vllmServerServiceName(vllmService),
		Namespace: vllmService.Namespace,
	}

	service := &corev1.Service{}
	if err := r.Get(ctx, key, service); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Service %s: %w", key, err)
		}

		service = newVLLMServerService(vllmService)

		if err := controllerutil.SetControllerReference(
			vllmService,
			service,
			r.Scheme,
		); err != nil {
			return fmt.Errorf("failed to set VLLMService as Service owner: %w", err)
		}

		if err := r.Create(ctx, service); err != nil {
			return fmt.Errorf("failed to create Service %s: %w", key, err)
		}

		logf.FromContext(ctx).Info(
			"Created Service",
			"name", service.Name,
			"namespace", service.Namespace,
		)
	}

	if !metav1.IsControlledBy(service, vllmService) {
		return fmt.Errorf(
			"existing Service %s is not controlled by VLLMService %s/%s",
			key,
			vllmService.Namespace,
			vllmService.Name,
		)
	}

	desiredService := newVLLMServerService(vllmService)
	original := service.DeepCopy()
	serviceChanged := false

	if service.Labels == nil {
		service.Labels = map[string]string{}
	}

	for labelKey, labelValue := range desiredService.Labels {
		if service.Labels[labelKey] != labelValue {
			service.Labels[labelKey] = labelValue
			serviceChanged = true
		}
	}

	if service.Spec.Type != desiredService.Spec.Type {
		service.Spec.Type = desiredService.Spec.Type
		serviceChanged = true
	}

	if !apiequality.Semantic.DeepEqual(
		service.Spec.Selector,
		desiredService.Spec.Selector,
	) {
		service.Spec.Selector = desiredService.Spec.Selector
		serviceChanged = true
	}

	if !apiequality.Semantic.DeepEqual(
		service.Spec.Ports,
		desiredService.Spec.Ports,
	) {
		service.Spec.Ports = desiredService.Spec.Ports
		serviceChanged = true
	}

	if !serviceChanged {
		return nil
	}

	if err := r.Patch(ctx, service, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch Service %s: %w", key, err)
	}

	logf.FromContext(ctx).Info(
		"Updated Service",
		"name", service.Name,
		"namespace", service.Namespace,
	)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VLLMServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&inferencev1alpha1.VLLMService{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Service{}).
		Named("vllmservice").
		Complete(r)
}

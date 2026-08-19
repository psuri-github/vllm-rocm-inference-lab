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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func newModelCachePVC(vllmService *inferencev1alpha1.VLLMService) *corev1.PersistentVolumeClaim {
	labels := labelsForVLLMService(vllmService)
	labels["app.kubernetes.io/component"] = "model-cache"

	var storageClassName *string
	if vllmService.Spec.StorageClassName != nil {
		value := *vllmService.Spec.StorageClassName
		storageClassName = &value
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      modelCachePVCName(vllmService),
			Namespace: vllmService.Namespace,
			Labels:    labels,
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
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VLLMServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&inferencev1alpha1.VLLMService{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Named("vllmservice").
		Complete(r)
}

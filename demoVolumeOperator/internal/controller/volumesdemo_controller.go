/*
Copyright 2023.

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

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	newdemov1 "demovolume/api/v1"
)

// VolumesDemoReconciler reconciles a VolumesDemo object
type VolumesDemoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=newdemo.volume.io,resources=volumesdemoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=newdemo.volume.io,resources=volumesdemoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=newdemo.volume.io,resources=volumesdemoes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VolumesDemo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *VolumesDemoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	var VolumesDemo newdemov1.VolumesDemo
	if err := r.Get(ctx, req.NamespacedName, &VolumesDemo); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if VolumesDemo.Spec.Name != VolumesDemo.Status.Name {
		VolumesDemo.Status.Name = VolumesDemo.Spec.Name
		if updateErr := r.Status().Update(ctx, &VolumesDemo); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
	}
	var pvc v1.PersistentVolumeClaim
	err := r.Get(ctx, req.NamespacedName, &pvc)
	if err == nil {
		log.Log.Info("PVC found")
		return ctrl.Result{}, nil
	}

	if !errors.IsNotFound((err)) {
		log.Log.Info("PVC not found")
		return ctrl.Result{}, nil
	}

	pvcCreated, err := r.reconcilePVC(ctx, &VolumesDemo)
	if err != nil {
		log.Log.Info(err.Error())
	}
	if err == nil {
		log.Log.Info("PVC created successfully", "pvc", pvcCreated)

	}

	log.Log.Info("Volume spec :", "Name :", VolumesDemo.Spec.Name, "Size", VolumesDemo.Spec.Size)
	return ctrl.Result{}, nil
}

func (r *VolumesDemoReconciler) reconcilePVC(ctx context.Context, VolumesDemo *newdemov1.VolumesDemo) (*v1.PersistentVolumeClaim, error) {

	storageClass := "standard"
	storageRequest, _ := resource.ParseQuantity(fmt.Sprintf("%dGi", VolumesDemo.Spec.Size))
	var pvc v1.PersistentVolumeClaim
	pvc = v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: VolumesDemo.Namespace,
			Name:      VolumesDemo.Name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(VolumesDemo, schema.GroupVersionKind{
					Group:   newdemov1.SchemeBuilder.GroupVersion.Group,
					Version: newdemov1.SchemeBuilder.GroupVersion.Version,
					Kind:    "VolumesDemo",
				}),
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClass,
			AccessModes:      []v1.PersistentVolumeAccessMode{"ReadWriteOnce"},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{"storage": storageRequest},
			},
		},
	}

	if err := r.Create(ctx, &pvc); err != nil {
		return nil, err
	}
	return &pvc, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumesDemoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&newdemov1.VolumesDemo{}).
		Complete(r)
}

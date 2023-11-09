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
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	demov1 "weatherApiOperator/api/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WeatherReportReconciler reconciles a WeatherReport object
type WeatherReportReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=demo.weatherapi.io,resources=weatherreports,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=demo.weatherapi.io,resources=weatherreports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=demo.weatherapi.io,resources=weatherreports/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WeatherReport object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *WeatherReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var weatherReport demov1.WeatherReport
	if err := r.Get(ctx, req.NamespacedName, &weatherReport); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if weatherReport.Status.State == "" || weatherReport.Status.State == "Failed" {
		// Create a Pod
		pod := createPodForWeatherReport(&weatherReport)
		if err := r.Create(ctx, pod); err != nil {
			log.Log.Error(err, "Failed to create Pod")
			weatherReport.Status.State = "Failed"
			if updateErr := r.Status().Update(ctx, &weatherReport); updateErr != nil {
				return ctrl.Result{}, updateErr
			}
			return ctrl.Result{}, err
		}

		// Update the status
		weatherReport.Status.State = "Started"
		weatherReport.Status.Pod = pod.Name
		log.Log.Info("Successful in creating Pod")
		if err := r.Status().Update(ctx, &weatherReport); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Log the spec
	log.Log.Info("WeatherReport Spec:", "City", weatherReport.Spec.City, "Days", weatherReport.Spec.Days, "State", weatherReport.Status.State, "Pod", weatherReport.Status.Pod)

	return ctrl.Result{}, nil
}
func createPodForWeatherReport(weatherReport *demov1.WeatherReport) *corev1.Pod {
	// Create a Pod definition based on the WeatherReport's specifications
	url := fmt.Sprintf("http://wttr.in/%s?%d", weatherReport.Spec.City, weatherReport.Spec.Days)
	labels := map[string]string{
		"app":  "weather-report",
		"city": strings.Replace(weatherReport.Spec.City, " ", "_", -1),
		"days": strconv.Itoa(weatherReport.Spec.Days),
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "weather-report-", // You can generate a unique name
			Namespace:    weatherReport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(weatherReport, schema.GroupVersionKind{
					Group:   demov1.SchemeBuilder.GroupVersion.Group,
					Version: demov1.SchemeBuilder.GroupVersion.Version,
					Kind:    "WeatherReport",
				}),
			},
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "weather",
					Image:   "alpine:latest", // Use the Alpine Linux image
					Command: []string{"sh", "-c", "apk --no-cache add curl && curl -s " + url + " && sleep 3600"},
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *WeatherReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.WeatherReport{}).
		Complete(r)
}

/*
Copyright 2024.

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
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	// apiv1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"

	// "time"

	ipav1alpha1 "github.com/shafinhasnat/ipa/api/v1alpha1"
	controller "github.com/shafinhasnat/ipa/internal/agent"
)

// IPAReconciler reconciles a IPA object
type IPAReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=ipa.shafinhasnat.me,resources=ipas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipa.shafinhasnat.me,resources=ipas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipa.shafinhasnat.me,resources=ipas/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IPA object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *IPAReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	ipa := &ipav1alpha1.IPA{}
	err := r.Get(ctx, req.NamespacedName, ipa)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.IPA(ctx, ipa, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	// deployment := &appsv1.Deployment{}
	return ctrl.Result{RequeueAfter: time.Duration(30 * time.Second)}, nil
}

func (r *IPAReconciler) IPA(ctx context.Context, ipa *ipav1alpha1.IPA, req ctrl.Request) error {
	apikey := ipa.Spec.Metadata.ApiKey
	prometheus := ipa.Spec.Metadata.PrometheusUri
	// defaultreplicas := ipa.Spec.Metadata.DefaultReplicas
	ipagroups := ipa.Spec.Metadata.IPAGroup
	for _, ipagroup := range ipagroups {
		deployment := &appsv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{Name: ipagroup.Deployment, Namespace: ipagroup.Namespace}, deployment)
		if err != nil {
			return err
		}
		prometheusData, err := controller.QueryPrometheus(prometheus, deployment.Name)
		if err != nil {
			return err
		}
		// fmt.Println(prometheusData)
		replicas, err := controller.AskLLM(prometheusData, apikey)
		if err != nil {
			return err
		}
		if *deployment.Spec.Replicas != replicas {
			deployment.Spec.Replicas = &replicas
			err := r.Update(ctx, deployment)
			if err != nil {
				return err
			}
		}

	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ipav1alpha1.IPA{}).
		Named("ipa").
		Complete(r)
}

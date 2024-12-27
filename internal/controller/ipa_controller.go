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

	v1 "k8s.io/api/apps/v1"
	// apiv1 "k8s.io/api/core/v1"
	"fmt"

	// "time"

	ipav1alpha1 "github.com/shafinhasnat/ipa/api/v1alpha1"
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
	err = r.ScaleDeployments(ctx, *ipa, req)
	if err != nil {
		fmt.Println(err.Error())
		return ctrl.Result{}, err
	}
	// deployment := &appsv1.Deployment{}
	return ctrl.Result{RequeueAfter: time.Duration(5 * time.Second)}, nil
}

func (r *IPAReconciler) ScaleDeployments(ctx context.Context, ipa ipav1alpha1.IPA, req ctrl.Request) error {
	ipaRules := ipa.Spec.IPARules
	for _, ipaRule := range ipaRules {
		now := time.Now().Hour()
		// fmt.Println("Current hour ->", now.Hour())
		spec_namespace := ipaRule.Namespace
		spec_deployment := ipaRule.Deployment
		deployment := &v1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{
			Namespace: spec_namespace,
			Name:      spec_deployment,
		}, deployment)
		if err != nil {
			return err
		}
		for _, rule := range ipaRule.Rules {
			if isTimeBetween(rule.From, rule.To, uint8(now)) {
				if *deployment.Spec.Replicas != rule.Replicas {
					desiredReplicas := rule.Replicas
					deployment.Spec.Replicas = &desiredReplicas
					err := r.Update(ctx, deployment)
					if err != nil {
						return err
					}
				}
			}
			// else {
			// 	fmt.Println("Out of time range")
			// 	if deployment.Spec.Replicas != &ipaRule.DefaultReplicas {
			// 		fmt.Println("Out of time range - replicas not match")
			// 		deployment.Spec.Replicas = &ipaRule.DefaultReplicas
			// 		err := r.Update(ctx, deployment)
			// 		if err != nil {
			// 			return err
			// 		}
			// 	}
			// }
		}
	}
	return nil
}
func isTimeBetween(from, to, now uint8) bool {
	if to < from {
		return now >= from || now <= to
	}
	return now >= from && now <= to
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ipav1alpha1.IPA{}).
		Named("ipa").
		Complete(r)
}

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

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"fmt"

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
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch

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
	ipa := &ipav1alpha1.IPA{}
	err := r.Get(ctx, req.NamespacedName, ipa)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.IPA(ctx, ipa, req)
	if err != nil {
		ipa.Status.Status = string(err.Error())
		err = r.Status().Update(ctx, ipa)
		if err != nil {
			// fmt.Println("ERROR UPDATING ERROR STATUS")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}
	ipa.Status.Status = "Success"
	err = r.Status().Update(ctx, ipa)
	if err != nil {
		// fmt.Println("ERROR UPDATING STATUS")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: time.Duration(1 * time.Minute)}, nil
}

func (r *IPAReconciler) IPA(ctx context.Context, ipa *ipav1alpha1.IPA, req ctrl.Request) error {
	prometheus := ipa.Spec.Metadata.PrometheusUri
	ipagroups := ipa.Spec.Metadata.IPAGroup
	for _, ipagroup := range ipagroups {
		deployment := &appsv1.Deployment{}
		err := r.Get(ctx, types.NamespacedName{Name: ipagroup.Deployment, Namespace: ipagroup.Namespace}, deployment)
		if err != nil {
			return fmt.Errorf("error getting deployment: %v", err)
		}
		var resourceInfo string
		for _, container := range deployment.Spec.Template.Spec.Containers {
			resourceInfo = fmt.Sprintf("CPU Resource Requests: %v, CPU Resource Limits: %v, Memory Resource Requests: %v, Memory Resource Limits: %v",
				container.Resources.Requests.Cpu().String(),
				container.Resources.Limits.Cpu().String(),
				container.Resources.Requests.Memory().String(),
				container.Resources.Limits.Memory().String())
		}
		podList := &corev1.PodList{}
		err = r.List(ctx, podList, client.InNamespace(ipagroup.Namespace), client.MatchingLabels(deployment.Spec.Selector.MatchLabels))
		if err != nil {
			return fmt.Errorf("error getting pods: %v", err)
		}
		var podNames []string
		var events []map[string]string
		for _, pod := range podList.Items {
			event := &corev1.EventList{}
			err = r.List(ctx, event, client.InNamespace(pod.Namespace), client.MatchingFields(map[string]string{"involvedObject.name": pod.Name}))
			if err != nil {
				return fmt.Errorf("error getting event: %v", err)
			}
			for _, item := range event.Items {
				events = append(events, map[string]string{"pod": pod.Name, "type": item.Type, "reason": item.Reason, "message": item.Message})
			}
			podNames = append(podNames, pod.Name)
		}
		prometheusData, err := controller.QueryPrometheus(prometheus, deployment.Name, podNames, ipagroup.Namespace, resourceInfo, events, ipagroup.Ingress)
		if err != nil {
			return fmt.Errorf("error querying prometheus: %v", err)
		}
		llmResponse, err := controller.GeminiAPI(ipa.Spec.Metadata.LLMAgent, prometheusData)
		if err != nil {
			return fmt.Errorf("error querying llm: %v", err)
		}
		if *deployment.Spec.Replicas != llmResponse.Config.Replicas {
			deployment.Spec.Replicas = &llmResponse.Config.Replicas
			err := r.Update(ctx, deployment)
			if err != nil {
				return fmt.Errorf("error updating deployment: %v", err)
			}
		}
		for i := range deployment.Spec.Template.Spec.Containers {
			container := &deployment.Spec.Template.Spec.Containers[i]
			container.Resources = corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(llmResponse.Config.CPURequest),
					corev1.ResourceMemory: resource.MustParse(llmResponse.Config.MemoryRequest),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(llmResponse.Config.CPULimit),
					corev1.ResourceMemory: resource.MustParse(llmResponse.Config.MemoryLimit),
				},
			}
		}
		if err := r.Update(ctx, deployment); err != nil {
			return fmt.Errorf("failed to update deployment: %v", err)
		}

	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Add field indexer for events
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Event{}, "involvedObject.name", func(obj client.Object) []string {
		event := obj.(*corev1.Event)
		return []string{event.InvolvedObject.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&ipav1alpha1.IPA{}).
		Named("ipa").
		Complete(r)
}

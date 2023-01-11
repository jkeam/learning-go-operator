/*
Copyright 2023 Jon Keam.

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

package controllers

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	hellov1 "github.com/jkeam/learning-go-operators/api/v1"
)

// GogoReconciler reconciles a Gogo object
type GogoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=hello.jonkeam.com,resources=gogoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=hello.jonkeam.com,resources=gogoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=hello.jonkeam.com,resources=gogoes/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Gogo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *GogoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting reconciliation")
	logger.Info(fmt.Sprintf("Namespace: %s", req.Namespace))

	gogo := &hellov1.Gogo{}
	err := r.Get(ctx, req.NamespacedName, gogo)
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Gogo most likely deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Gogo object")
		return ctrl.Result{}, err
	}

	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: gogo.Name, Namespace: gogo.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		deployment := r.createDeployment(gogo)
		err = r.Create(ctx, deployment)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to create new Gogo deployment"))
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Gogo deployment")
		return ctrl.Result{}, err
	}

	// check size
	size := gogo.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			logger.Error(err, "Failed to update Gogo deployment with replica count")
			return ctrl.Result{}, err
		}

		return ctrl.Result{Requeue: true}, nil
	}

	// update status with pod names
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(gogo.Namespace),
		client.MatchingLabels(labelsForGogo(gogo.Name)),
	}
	if err = r.List(ctx, podList, listOpts...); err != nil {
		logger.Error(err, "Failed to list Gogo pods")
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podList.Items)
	if !reflect.DeepEqual(podNames, gogo.Status.Pods) {
		gogo.Status.Pods = podNames
		err := r.Status().Update(ctx, gogo)
		if err != nil {
			logger.Error(err, "Failed to update Gogo status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GogoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hellov1.Gogo{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *GogoReconciler) createDeployment(m *hellov1.Gogo) *appsv1.Deployment {
	ls := labelsForGogo(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "memcached:1.4.36-alpine",
						Name:    "memcached",
						Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 11211,
							Name:          "memcached",
						}},
					}},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsForGogo return hash for labels
func labelsForGogo(name string) map[string]string {
	return map[string]string{"app": "gogo", "gogo_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

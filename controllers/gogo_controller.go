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
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilv1 "k8s.io/apimachinery/pkg/util/intstr"
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
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile the main loop
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

	// service
	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: gogo.Name, Namespace: gogo.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) && found != nil {
		service := r.createService(gogo)
		err = r.Create(ctx, service)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to create new Gogo service"))
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Gogo service")
		return ctrl.Result{}, err
	}

	// ingress
	foundRoute := &networkingv1.Ingress{}
	err = r.Get(ctx, types.NamespacedName{Name: gogo.Name, Namespace: gogo.Namespace}, foundRoute)
	if err != nil && errors.IsNotFound(err) && found != nil && foundService != nil {
		route := r.createIngress(gogo)
		err = r.Create(ctx, route)
		if err != nil {
			logger.Error(err, fmt.Sprintf("Failed to create new Gogo route"))
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Gogo route")
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
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

// createIngress create ingress object
func (r *GogoReconciler) createIngress(m *hellov1.Gogo) *networkingv1.Ingress {
	ingressPathType := networkingv1.PathTypePrefix
	ingressClassName := "openshift-default"
	ingress := &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.k8s.io/v1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: fmt.Sprintf("%s-%s.apps.%s", m.Name, m.Namespace, m.Spec.Host),
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &ingressPathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: m.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: 8080,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	ctrl.SetControllerReference(m, ingress, r.Scheme)
	return ingress
}

// createService create service
func (r *GogoReconciler) createService(m *hellov1.Gogo) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port: 8080,
				TargetPort: utilv1.IntOrString{
					Type:   utilv1.Int,
					IntVal: 8080,
				},
			}},
			Selector: map[string]string{
				"app": "gogo",
			},
		},
	}
	ctrl.SetControllerReference(m, service, r.Scheme)
	return service
}

// createDeployment create a deployment object and register it
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
						Image: "quay.io/jkeam/hello-go@sha256:96bd61e4a98f06fb677f9e0ee30a48faa5c2de6c3f9ee967966c76dd549674c3",
						Name:  "gogo",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
						}},
					}},
				},
			},
		},
	}
	// Set Gogo instance as the owner and controller
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

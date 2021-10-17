/*
Copyright 2021 Kais OMRI.

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
	"os"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-playground/validator/v10"
	"k8s.io/apimachinery/pkg/api/errors"

	komv1alpha1 "github.com/kaiso/kom-operator/api/v1alpha1"
	"github.com/kaiso/kom-operator/process"
	komutil "github.com/kaiso/kom-operator/util"
	"github.com/kaiso/kom-operator/version"
)

const (
	controllerName = "controller_microservice"
	//smartReload environment varibale
	smartReload string = "SMART_RELOAD"
)

var validate = validator.New()
var log = logf.Log.WithName(controllerName)

// MicroserviceReconciler reconciles a Microservice object
type MicroserviceReconciler struct {
	Client             client.Client
	Scheme             *runtime.Scheme
	AutoScalingHandler *process.AutoScalingHandler
}

// newReconciler returns a new reconcile.Reconciler
func NewReconciler(mgr ctrl.Manager) *MicroserviceReconciler {
	return &MicroserviceReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), AutoScalingHandler: &process.AutoScalingHandler{Manager: &mgr, Context: context.TODO(), Scheme: mgr.GetScheme()}}
}

//+kubebuilder:rbac:groups=kom.kaiso.github.io,resources=microservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kom.kaiso.github.io,resources=microservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kom.kaiso.github.io,resources=microservices/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=autoscaling,resources=*,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=services;services/finalizers;endpoints;persistentvolumeclaims;events;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=extensions,resources=ingresses,verbs=get;list;watch
//+kubebuilder:rbac:groups=extensions,resources=ingresses/status,verbs=update
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,ingressclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses/status,verbs=update


// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Microservice object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *MicroserviceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log = logf.FromContext(ctx)

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Microservice", "request", request)

	// Fetch the Microservice instance
	instance := &komv1alpha1.Microservice{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if ok, err := r.isValid(instance); !ok {
		return reconcile.Result{}, err
	}

	//fmt.Println(komutil.PrettyPrint(instance))

	// begin finalizer logic
	myFinalizerName := "router.finalizers.kom.kaiso.github.io"

	// examine DeletionTimestamp to determine if object is under deletion
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !komutil.ContainsString(instance.ObjectMeta.Finalizers, myFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if komutil.ContainsString(instance.ObjectMeta.Finalizers, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.finalize(instance); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return reconcile.Result{}, err
			}

			// remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = komutil.RemoveString(instance.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return reconcile.Result{}, nil
	}

	// End of finalizer logic

	// Define a new Pod object
	deployment, service, routers := getInstanceObjects(instance)

	// Set Microservice instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, deployment, r.Scheme); err != nil {
		return reconcile.Result{}, err
	}
	if service != nil {
		if err := controllerutil.SetControllerReference(instance, service, r.Scheme); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if this Deployment already exists
	existingDeployment := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, existingDeployment)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Client.Create(context.TODO(), deployment)
		if service != nil && err == nil {
			err = r.Client.Create(context.TODO(), service)
			if err == nil && len(routers) > 0 {
				err = (*process.GetInstance()).CreateRouter(service.Namespace, service.Name, routers)
				if err != nil {
					reqLogger.Error(err, "Error creating loadbalancer route")
				}
			}
		}

		r.udpateStatus(instance, err)

		if err != nil {
			return reconcile.Result{}, err
		}
		r.AutoScalingHandler.Run(instance, deployment)
		// Deployment created successfully - don't requeue
		reqLogger.Info("Finished Creating resources", "request", request)
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("deployment already exists", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)

	r.AutoScalingHandler.Run(instance, existingDeployment)

	deploymentChanged, serviceChanged, err := false, false, nil
	if os.Getenv(smartReload) == "false" {
		deploymentChanged, serviceChanged = true, true
	} else {
		deploymentChanged, serviceChanged, err = hasChanges(instance, existingDeployment)
		if err != nil {
			return reconcile.Result{}, err
		}
	}
	if !deploymentChanged && !serviceChanged {
		return reconcile.Result{}, nil
	}

	if deploymentChanged == true {
		reqLogger.Info("updating deployment...", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
		err = r.Client.Update(context.TODO(), deployment)
	}

	if err == nil && serviceChanged {
		reqLogger.Info("updating service...", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
		existingService := &corev1.Service{}
		err := r.Client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, existingService)
		if err == nil {
			err = r.Client.Delete(context.TODO(), existingService)
			if err == nil {
				err = (*process.GetInstance()).RemoveRouter(service.Namespace, service.Name, len(routers) == 0)
				if err == nil {
					err = r.Client.Create(context.TODO(), service)
					if err == nil && len(routers) > 0 {
						err = (*process.GetInstance()).CreateRouter(service.Namespace, service.Name, routers)
					}
				}
			}
		}
	}

	r.udpateStatus(instance, err)

	if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("Finished Updating resources", "request", request)
	return reconcile.Result{}, nil
}

func getInstanceObjects(cr *komv1alpha1.Microservice) (*appsv1.Deployment, *corev1.Service, map[int32]string) {
	labels := map[string]string{
		"app":      cr.Name,
		"provider": "kom-operator",
	}

	annotations := map[string]string{
		"creator": "kom-operator.kaiso.github.io/" + version.Version,
	}

	var replicas int32 = 1

	if (cr.Spec.Autoscaling.Min == cr.Spec.Autoscaling.Max || len(cr.Spec.Autoscaling.Scaler) == 0) && cr.Spec.Autoscaling.Min > 0 {
		replicas = cr.Spec.Autoscaling.Min
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        cr.Name,
			Namespace:   cr.Namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:           cr.Spec.Container.Image,
						Name:            cr.Name,
						Command:         cr.Spec.Container.Command,
						Args:            cr.Spec.Container.Args,
						VolumeMounts:    cr.Spec.Container.VolumeMounts,
						Resources:       cr.Spec.Container.Resources,
						ImagePullPolicy: cr.Spec.Container.ImagePullPolicy,
					}},
					Volumes:            cr.Spec.Volumes,
					ServiceAccountName: cr.Spec.ServiceAccountName,
				},
			},
		},
	}

	var size = len(cr.Spec.Container.Routing.HTTP)
	if size != 0 {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        cr.Name,
				Namespace:   cr.Namespace,
				Annotations: annotations,
				Labels:      labels,
			},
			Spec: corev1.ServiceSpec{
				Ports:    []corev1.ServicePort{},
				Selector: labels,
			},
		}
		var ports []corev1.ContainerPort
		routers := make(map[int32]string)
		for _, element := range cr.Spec.Container.Routing.HTTP {
			ports = append(ports, element.Port)
			(*service).Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
				Name:       element.Port.Name,
				Protocol:   corev1.ProtocolTCP,
				Port:       element.Port.ContainerPort,
				TargetPort: intstr.FromString(element.Port.Name),
			})
			if element.Rule != "" {
				routers[element.Port.ContainerPort] = element.Rule
			}
		}
		deployment.Spec.Template.Spec.Containers[0].Ports = ports
		return deployment, service, routers
	}
	return deployment, nil, nil
}

func (r *MicroserviceReconciler) finalize(instance *komv1alpha1.Microservice) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name)
	reqLogger.Info("finalize instance ")
	_, service, _ := getInstanceObjects(instance)
	if service != nil {
		err := (*process.GetInstance()).RemoveRouter(service.Namespace, service.Name, true)
		if err != nil {
			reqLogger.Error(err, "Error removing loadbalancer route")
			return err
		}
	}
	return nil
}

func hasChanges(instance *komv1alpha1.Microservice, existingDeployment *appsv1.Deployment) (bool, bool, error) {
	var existingReplicas int32 = *existingDeployment.Spec.Replicas
	deploymentChanged := existingReplicas != instance.Spec.Autoscaling.Min
	if deploymentChanged == false {
		deploymentChanged = existingDeployment.Spec.Template.Spec.ServiceAccountName != instance.Spec.ServiceAccountName
	}
	if deploymentChanged == false {
		found := false
		for _, container := range existingDeployment.Spec.Template.Spec.Containers {
			if instance.Spec.Container.Image == container.Image {
				found = true
				//fmt.Printf("not found  %T %v\n", instance.Spec.Args, instance.Spec.Args)
				if !reflect.DeepEqual(instance.Spec.Container.Args, container.Args) ||
					!reflect.DeepEqual(instance.Spec.Container.Command, container.Command) ||
					!reflect.DeepEqual(instance.Spec.Container.Resources, container.Resources) {
					deploymentChanged = true
				}
			}
		}
		if found == false {
			deploymentChanged = true
		}
	}
	serviceChanged := !komutil.Equals(instance.Status.Routing, instance.Spec.Container.Routing)
	return deploymentChanged, serviceChanged, nil
}

func (r *MicroserviceReconciler) udpateStatus(instance *komv1alpha1.Microservice, err error) error {
	reqLogger := log.WithValues("Request.Namespace", instance.Namespace, "Request.Name", instance.Name)

	var stampedStatus komv1alpha1.Status
	var reason string
	if err == nil {
		reason = ""
		stampedStatus = komv1alpha1.Success
	} else {
		reason = err.Error()
		stampedStatus = komv1alpha1.Failure
	}
	status := komv1alpha1.MicroserviceStatus{
		LastUpdate: metav1.Now(),
		Reason:     reason,
		Status:     stampedStatus,
		Routing:    instance.Spec.Container.Routing,
	}
	if !reflect.DeepEqual(instance.Status.Routing, status.Routing) || instance.Status.Status != status.Status {
		reqLogger.Info("Status has changed, updating...")
		instance.Status = status
		err = r.Client.Status().Update(context.Background(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Resource status ")
		}
		return err
	}
	return nil
}

func (r *MicroserviceReconciler) isValid(obj metav1.Object) (bool, error) {
	instance, ok := obj.(*komv1alpha1.Microservice)
	if !ok {
		return false, errors.NewBadRequest("Resource is not of type Microservice")
	}

	if err := validate.Struct(instance); err != nil {
		return false, err
	}

	return true, nil
}

func ignoreStatusUpdatePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() ||
				!reflect.DeepEqual(e.ObjectOld.GetFinalizers(), e.ObjectNew.GetFinalizers())
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicroserviceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if os.Getenv(smartReload) == "false" {
		log.WithValues("Operator", "KOM-Operator").Info(" Smart reload was disabled, the deployments will be replaced on every reconcile")
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&komv1alpha1.Microservice{}).
		//Owns(&appsv1.Deployment{}).
		WithEventFilter(ignoreStatusUpdatePredicate()).
		Complete(r)
}

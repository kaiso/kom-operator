package microservice

import (
	"context"
	"reflect"

	komv1alpha1 "github.com/kaiso/kom-operator/pkg/apis/kom/v1alpha1"
	"github.com/kaiso/kom-operator/pkg/process"
	komutil "github.com/kaiso/kom-operator/pkg/util"
	"github.com/kaiso/kom-operator/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_microservice")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Microservice Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMicroservice{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("microservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Microservice
	err = c.Watch(&source.Kind{Type: &komv1alpha1.Microservice{}}, &handler.EnqueueRequestForObject{}, komv1alpha1.MicroserviceChangedPredicate{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Microservice
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &komv1alpha1.Microservice{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileMicroservice implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileMicroservice{}

// ReconcileMicroservice reconciles a Microservice object
type ReconcileMicroservice struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Microservice object and makes changes based on the state read
// and what is in the Microservice.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMicroservice) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Microservice", "request", request)

	// Fetch the Microservice instance
	instance := &komv1alpha1.Microservice{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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
			if err := r.client.Update(context.Background(), instance); err != nil {
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
			if err := r.client.Update(context.Background(), instance); err != nil {
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
	if err := controllerutil.SetControllerReference(instance, deployment, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	if err := controllerutil.SetControllerReference(instance, service, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Deployment already exists
	existingDeployment := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, existingDeployment)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.client.Create(context.TODO(), deployment)
		if service != nil && err == nil {
			err = r.client.Create(context.TODO(), service)
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

		// Deployment created successfully - don't requeue
		reqLogger.Info("Finished Creating resources", "request", request)
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	reqLogger.Info("deployment already exists", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)

	deploymentChanged, serviceChanged, err := hasChanges(instance, existingDeployment)

	if !deploymentChanged && !serviceChanged {
		return reconcile.Result{}, nil
	}

	if deploymentChanged == true {
		reqLogger.Info("updating deployment...", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
		err = r.client.Update(context.TODO(), deployment)
	}

	if err == nil && serviceChanged {
		reqLogger.Info("updating service...", "Deployment.Namespace", existingDeployment.Namespace, "Deployment.Name", existingDeployment.Name)
		existingService := &corev1.Service{}
		err := r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, existingService)
		if err == nil {
			err = r.client.Delete(context.TODO(), existingService)
			if err == nil {
				err = (*process.GetInstance()).RemoveRouter(service.Namespace, service.Name, len(routers) == 0)
				if err == nil {
					err = r.client.Create(context.TODO(), service)
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

	replicas := cr.Spec.Autoscaling.Min

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
						Image:   cr.Spec.Image,
						Name:    cr.Name,
						Command: cr.Spec.Command,
						Args:    cr.Spec.Args,
					}},
				},
			},
		},
	}

	var size = len(cr.Spec.Routing.Http)
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
		for _, element := range cr.Spec.Routing.Http {
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

func (r *ReconcileMicroservice) finalize(instance *komv1alpha1.Microservice) error {
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
		found := false
		for _, container := range existingDeployment.Spec.Template.Spec.Containers {
			if instance.Spec.Image == container.Image {
				found = true
				//fmt.Printf("not found  %T %v\n", instance.Spec.Args, instance.Spec.Args)
				if !reflect.DeepEqual(instance.Spec.Args, container.Args) ||
					!reflect.DeepEqual(instance.Spec.Command, container.Command) {
					deploymentChanged = true
				}
			}
		}
		if found == false {
			deploymentChanged = true
		}
	}
	serviceChanged := !komutil.Equals(instance.Status.Routing, instance.Spec.Routing)
	return deploymentChanged, serviceChanged, nil
}

func (r *ReconcileMicroservice) udpateStatus(instance *komv1alpha1.Microservice, err error) error {
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
		Routing:    instance.Spec.Routing,
	}
	instance.Status = status
	err = r.client.Status().Update(context.Background(), instance)
	if err != nil {
		reqLogger.Error(err, "Failed to update Resource status ")
	}
	return err
}

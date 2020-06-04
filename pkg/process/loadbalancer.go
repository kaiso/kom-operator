package process

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kaiso/kom-operator/version"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/yaml"
)

// LoadBalancer type
type LoadBalancer struct {
	Name       string
	Namespace  string
	ConfigName string
	Manager    *manager.Manager
	Config     corev1.ConfigMap
	Context    context.Context
	Owner      *metav1.OwnerReference
	Deployment appsv1.Deployment
}

const (
	traefikImage          string = "traefik:v2.2.0"
	traefikConfigFileName string = "traefik.yaml"
)

var lbLogger = logf.Log.WithName("loadbalancer")
var instance *LoadBalancer
var once sync.Once
var annotations = map[string]string{
	"creator": "kom-operator.kaiso.github.io/" + version.Version,
}

//GetInstance returns the LoadBalancer singleton
func GetInstance() *LoadBalancer {
	return instance
}

//InitLoadBalancer LoadBalancer constructor
func InitLoadBalancer(ctx context.Context, name string, mgr manager.Manager) error {

	once.Do(func() {
		instance = new(LoadBalancer)
		instance.Name = name
		instance.Manager = &mgr
		instance.Context = ctx
		instance.ConfigName = "kom-operator-lbconfig"
		var err error
		if !isRunModeLocal() {
			instance.Namespace, err = k8sutil.GetOperatorNamespace()
			if err != nil {
				panic(err.Error())
			}
		} else {
			lbLogger.Info("not running in a cluster. using namespace default")
			instance.Namespace = "default"
		}
	})
	return nil
}

// Start implement Runnable.Start
func (lb *LoadBalancer) Start(<-chan struct{}) error {
	lbLogger.Info("Starting the KOM LoadBalancer Manager")
	var mgr = *lb.Manager
	if !isRunModeLocal() {
		lb.Owner, _ = myOwnerRef(lb.Context, mgr.GetClient(), lb.Namespace)
	}

	err := lb.loadConfig()
	if err != nil {
		panic(err.Error())
	}

	var cfg = NewTraefikConfig()

	if len(lb.Config.Data) == 0 {
		lbLogger.Info(fmt.Sprintf("Creating the KOM LoadBalancer configuration "))
		err = lb.writeConfig(cfg, false)
		if err != nil {
			panic(err.Error())
		}
		lbLogger.Info(fmt.Sprintf("The KOM LoadBalancer configuration was created [%v]", lb.Config.Name))
	}

	err = lb.deployLoadBalancer()
	if err != nil {
		panic(err.Error())
	}
	//to remove
	err = lb.CreateRouter("", "PathPrefix(`/api`) || PathPrefix(`/dashboard`)", "api@internal", 0)

	if err != nil {
		lbLogger.Error(err, fmt.Sprintf("Failed to create the Dashboard router"))
		panic(err.Error())
	}

	return nil

}

func myOwnerRef(ctx context.Context, client crclient.Client, ns string) (*metav1.OwnerReference, error) {
	myPod, err := k8sutil.GetPod(ctx, client, ns)
	if err != nil {
		return nil, err
	}

	owner := &metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Pod",
		Name:       myPod.ObjectMeta.Name,
		UID:        myPod.ObjectMeta.UID,
	}
	return owner, nil
}

func (lb LoadBalancer) findExistingObject(name string, namespace string, obj metav1.Object) (bool, error) {
	var found = false
	var mgr = *lb.Manager
	key := crclient.ObjectKey{Namespace: namespace, Name: name}
	err := mgr.GetClient().Get(lb.Context, key, obj.(runtime.Object))
	switch {
	case err == nil:
		if lb.Owner != nil {
			for _, existingOwner := range obj.GetOwnerReferences() {
				if existingOwner.Name == lb.Owner.Name {
					found = true
				}
			}
		} else {
			found = true
		}
		if found {
			lbLogger.Info(fmt.Sprintf("Found existing Object [%v], the KOM operator was likely restarted.", obj.GetName()))
		} else {
			lbLogger.Info(fmt.Sprintf("No pre-existing Object [%v] was found.", name))
		}
	case apierrors.IsNotFound(err):
		lbLogger.Info(fmt.Sprintf("API error NotFound, No pre-existing Object [%v] was found.", name))
	default:
		lbLogger.Error(err, fmt.Sprintf("Unknown error trying to get Object [%v]", name))
		return false, err
	}
	return found, nil
}

func (lb *LoadBalancer) deployLoadBalancer() error {
	var name = "kom-loadbalancer"
	//var mgr = *lb.Manager
	labels := map[string]string{
		"app":      name,
		"provider": "kom-operator",
	}

	var replicas int32 = 1
	lb.Deployment = appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   lb.Namespace,
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
						Image: traefikImage,
						Name:  name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Name:          "http",
						},
							{
								ContainerPort: 8080,
								Name:          "admin",
							}},
						VolumeMounts: []corev1.VolumeMount{{
							Name:      "config-volume",
							MountPath: "/etc/traefik",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "config-volume",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: lb.ConfigName,
								},
							},
						},
					}},
				},
			},
		},
	}

	var service = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   lb.Namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       80,
				TargetPort: intstr.FromString("http"),
			}},
			Selector: labels,
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	}

	if lb.Owner != nil {
		lb.Deployment.OwnerReferences = []metav1.OwnerReference{*lb.Owner}
		service.OwnerReferences = []metav1.OwnerReference{*lb.Owner}
	}

	f, err := lb.findExistingObject(lb.Deployment.Name, lb.Deployment.Namespace, &lb.Deployment)
	if err != nil {
		panic(err.Error())
	}
	if f == false {
		lbLogger.Info(fmt.Sprintf("Creating the KOM LoadBalancer..."))
		err = (*lb.Manager).GetClient().Create(lb.Context, &lb.Deployment)
		if err != nil {
			panic(err.Error())
		}
		err = (*lb.Manager).GetClient().Create(lb.Context, service)
		if err != nil {
			panic(err.Error())
		}
		lbLogger.Info(fmt.Sprintf("The KOM LoadBalancer was created [%v]", lb.Deployment.Name))
	}

	return nil
}

func (lb *LoadBalancer) loadConfig() error {
	_, err := lb.findExistingObject(lb.ConfigName, lb.Namespace, &lb.Config)
	if err != nil {
		return err
	}
	return nil
}

func (lb *LoadBalancer) writeConfig(cfg TraefikConfig, update bool) error {
	var b []byte
	var err error
	b, err = yaml.Marshal(cfg)

	if err != nil {
		lbLogger.Error(err, "Unable de marshal configuration")
		return err
	}

	labels := map[string]string{
		"provider": "kom-operator",
	}
	configStr := string(b)

	lb.Config = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        lb.ConfigName,
			Namespace:   lb.Namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Data: map[string]string{
			traefikConfigFileName: configStr,
		},
	}
	if lb.Owner != nil {
		lb.Config.OwnerReferences = []metav1.OwnerReference{*lb.Owner}
	}

	if update {
		err = (*lb.Manager).GetClient().Update(lb.Context, &lb.Config)
	} else {
		err = (*lb.Manager).GetClient().Create(lb.Context, &lb.Config)
	}
	if err != nil {
		return err
	}
	return nil
}

func (lb *LoadBalancer) reloadDeployment() error {
	payload := []PatchInterfaceValue{{
		Op:    "add",
		Path:  "/spec/template/metadata/labels/reload-date",
		Value: time.Now().Format("2006-01-02T15_04_05"),
	}}
	payloadBytes, _ := json.Marshal(payload)

	dep := lb.Deployment
	lbLogger.Info(fmt.Sprintf("reloading LoadBalancer deployment..."))
	err := (*lb.Manager).GetClient().
		Patch(lb.Context, &dep, crclient.RawPatch(types.JSONPatchType, payloadBytes))
	if err != nil {
		lbLogger.Error(err, "Failed to reload LoadBalancer")
		return err
	}
	lbLogger.Info(fmt.Sprintf("LoadBalancer deployment reloaded"))
	return nil
}

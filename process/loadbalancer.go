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

package process

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kaiso/kom-operator/util"
	k8sutil "github.com/kaiso/kom-operator/util"
	"github.com/kaiso/kom-operator/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//"k8s.io/apimachinery/pkg/runtime"
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
	traefikImage          string = "traefik:v2.5.3"
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

// NeedLeaderElection implements the LeaderElectionRunnable interface
// to indicate whether this can be started without requiring the leader lock.
func (*LoadBalancer) NeedLeaderElection() bool {
	return true
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
		if !util.IsRunModeLocal() {
			instance.Namespace, err = k8sutil.GetOperatorNamespace()
			if err != nil {
				panic(err.Error())
			}
		} else {
			lbLogger.Info("Not running in a cluster. using namespace default")
			instance.Namespace = "default"
		}
	})

	return nil
}

// Start implement Runnable.Start
func (lb *LoadBalancer) Start(context.Context) error {
	lbLogger.Info("Starting the KOM LoadBalancer Manager")

	if !util.IsRunModeLocal() {
		lb.Owner, _ = lb.myOwnerRef()
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

	return nil

}

func (lb *LoadBalancer) myOwnerRef() (*metav1.OwnerReference, error) {
	namespace, _ := k8sutil.GetOperatorNamespace()
	name, _ := k8sutil.GetOperatorName()
	ref := &corev1.Pod{}
	//err := (*lb.Manager).GetClient().Get(lb.Context, crclient.ObjectKey{Namespace: namespace, Name: name}, &ref)
	ref, err := k8sutil.GetPod(lb.Context, (*lb.Manager).GetClient(), namespace)
	if err != nil {
		lbLogger.Error(err, fmt.Sprintf("Could not find pod [%v/%v]", namespace, name))
		return nil, err
	}

	owner := &metav1.OwnerReference{
		APIVersion: "v1",
		Kind:       "Deployment",
		Name:       ref.ObjectMeta.Name,
		UID:        ref.ObjectMeta.UID,
	}
	return owner, nil
}

func (lb LoadBalancer) findExistingObject(name string, namespace string, obj metav1.Object) (bool, error) {
	var found = false
	var mgr = *lb.Manager
	key := crclient.ObjectKey{Namespace: namespace, Name: name}
	err := mgr.GetClient().Get(lb.Context, key, obj.(crclient.Object))
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

	var replicas int32 = 3
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
					ServiceAccountName: "kom-operator",
					Containers: []corev1.Container{{
						Image: traefikImage,
						Name:  name,
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "dashboard",
						},
							{
								ContainerPort: 8090,
								Name:          "primary",
							},
							{
								ContainerPort: 8091,
								Name:          "secondary",
							},
							{
								ContainerPort: 8092,
								Name:          "metrics",
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
				Name:       "primary",
				Protocol:   corev1.ProtocolTCP,
				Port:       80,
				TargetPort: intstr.FromString("primary"),
			},
				{
					Name:       "dashboard",
					Protocol:   corev1.ProtocolTCP,
					Port:       5050,
					TargetPort: intstr.FromString("dashboard"),
				}},
			Selector: labels,
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	}

	if lb.Owner != nil {
		lb.Deployment.OwnerReferences = []metav1.OwnerReference{*lb.Owner}
	}

	f, err := lb.findExistingObject(lb.Deployment.Name, lb.Deployment.Namespace, &lb.Deployment)
	if err != nil {
		lbLogger.Error(err, "Error retrieving loadbalancer deployment")
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
		lbLogger.Info(fmt.Sprintf("The KOM LoadBalancer was created "))
	} else {
		lbLogger.Info(fmt.Sprintf("Updating the KOM LoadBalancer..."))
		err = (*lb.Manager).GetClient().Update(lb.Context, &lb.Deployment)
		if err != nil {
			panic(err.Error())
		}
		existingService := &corev1.Service{}
		err := (*lb.Manager).GetClient().Get(lb.Context, types.NamespacedName{
			Name:      lb.Deployment.Name,
			Namespace: lb.Deployment.Namespace,
		}, existingService)
		if err != nil {
			lbLogger.Error(err, "Error retrieving the KOM LoadBalancer service, this is likely due to a manual intervention! "+
				"The KOM Operator will create a new service, dependening on your cloud provider this may create a new loadbalancer endpoint ")
			existingService = service
			err = (*lb.Manager).GetClient().Create(lb.Context, existingService)
			if err != nil {
				panic(err.Error())
			}
		} else {
			lbLogger.Info(fmt.Sprintf("The KOM LoadBalancer service was already created. Skipping service creation... "))
		}
		lbLogger.Info(fmt.Sprintf("The KOM LoadBalancer was updated "))
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
		Path:  "/spec/template/metadata/annotations",
		Value: map[string]interface{}{"reload-date": time.Now().Format("2006-01-02T15:04:05")},
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

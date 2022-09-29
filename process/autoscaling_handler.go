package process

import (
	"context"
	"fmt"
	"strconv"

	"k8s.io/api/autoscaling/v2beta2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	resourceapi "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	komv1alpha1 "github.com/kaiso/kom-operator/api/v1alpha1"
	version "github.com/kaiso/kom-operator/version"
)

var log = logf.Log.WithName("kom_autoscaler")

// AutoScalingHandler handler for autoscaling
type AutoScalingHandler struct {
	Manager *manager.Manager
	Context context.Context
	Scheme  *runtime.Scheme
}

// Run handles autoscaling for a deployment
func (h *AutoScalingHandler) Run(instance *komv1alpha1.Microservice, deployment *appsv1.Deployment) {
	log.Info(fmt.Sprintf("Handling AutoScaling for %v %v.%v...", deployment.Kind, deployment.Namespace, deployment.Name))
	hpa := v2beta2.HorizontalPodAutoscaler{}
	exists := true
	namespacedName := client.ObjectKey{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
	}

	if err := (*h.Manager).GetClient().Get(h.Context, namespacedName, &hpa); err != nil {
		log.Info(fmt.Sprintf("Scaler doesn't exist for %v.%v error: %v", deployment.Namespace, deployment.Name, err.Error()))
		exists = false
	}

	autoScalingEnabled := instance.Spec.Autoscaling.Min != instance.Spec.Autoscaling.Max && len(instance.Spec.Autoscaling.Scaler) > 0

	if exists {
		operatorIsOwner := false
		for _, existingOwner := range hpa.GetOwnerReferences() {
			if existingOwner.Name == instance.Name && existingOwner.Kind == instance.Kind {
				operatorIsOwner = true
				break
			}
		}
		if operatorIsOwner == false {
			log.Info(fmt.Sprintf("Found Scaler for %v.%v but not created by the KOM Operator", deployment.Namespace, deployment.Name))
			return
		}

		if autoScalingEnabled {

			scaler, err := h.createScaler(deployment, instance)
			if err != nil {
				log.Error(err, fmt.Sprintf("Invalid AutoScaler configuration for deployment %v", deployment.Name))
				return
			}
			err = (*h.Manager).GetClient().Update(h.Context, scaler)
			if err != nil && !errors.IsAlreadyExists(err) {
				log.Error(err, fmt.Sprintf("Failed to update the AutoScaler for deployment %v", deployment.Name))
				return
			}
			log.Info(fmt.Sprintf("Successfully updated AutoScaling configuration for %v.%v...", deployment.Namespace, deployment.Name))
		} else {
			err := (*h.Manager).GetClient().Delete(h.Context, &hpa)
			if err != nil {
				log.Error(err, fmt.Sprintf("Failed to delete the AutoScaler for deployment %v", deployment.Name))
				return
			}
			log.Info(fmt.Sprintf("Successfully deleted AutoScaling configuration for %v.%v...", deployment.Namespace, deployment.Name))
		}
	} else if autoScalingEnabled {
		scaler, err := h.createScaler(deployment, instance)
		if err != nil {
			log.Error(err, "Invalid AutoScaler configuration")
			return
		}
		err = (*h.Manager).GetClient().Create(h.Context, scaler)
		if err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err, fmt.Sprintf("Failed to create Autoscaler for %v", deployment.Name))
			return
		}
		log.Info(fmt.Sprintf("Successfully created AutoScaling configuration for %v.%v...", deployment.Namespace, deployment.Name))
	} else {
		log.Info(fmt.Sprintf("AutoScaling not enabled for %v.%v...", deployment.Namespace, deployment.Name))
	}
}

func (h *AutoScalingHandler) createScaler(deployment *appsv1.Deployment, instance *komv1alpha1.Microservice) (*v2beta2.HorizontalPodAutoscaler, error) {

	var metrics []v2beta2.MetricSpec
	for _, inputScaler := range instance.Spec.Autoscaling.Scaler {
		var resource corev1.ResourceName
		var target v2beta2.MetricTarget
		switch inputScaler.Resource {
		case komv1alpha1.CPU:
			resource = corev1.ResourceCPU
			if instance.Spec.Container.Resources.Limits == nil || instance.Spec.Container.Resources.Limits.Cpu().IsZero() {
				return nil, errors.NewBadRequest(fmt.Sprintf("Invalid AutoScaler configuration for deployment %v, you must specify CPU Limits at container level", deployment.Name))
			}
		case komv1alpha1.Memory:
			resource = corev1.ResourceMemory
			if instance.Spec.Container.Resources.Limits == nil || instance.Spec.Container.Resources.Limits.Memory().IsZero() {
				return nil, errors.NewBadRequest(fmt.Sprintf("Invalid AutoScaler configuration for deployment %v, you must specify Memory Limits at container level", deployment.Name))
			}
		default:
			return nil, errors.NewBadRequest(fmt.Sprintf("Unsupported Scaler Resource [%v] in AutoScaling ", inputScaler.Resource))
		}

		switch inputScaler.Type {
		case komv1alpha1.AverageValue:
			targetValue, err := resourceapi.ParseQuantity(inputScaler.Value)
			if err != nil {
				return nil, errors.NewBadRequest(fmt.Sprintf("Invalid Scaler Value: %v for deployment %v, error: %v", inputScaler.Value, deployment.Name, err.Error()))
			}
			target = v2beta2.MetricTarget{
				Type:         v2beta2.AverageValueMetricType,
				AverageValue: &targetValue,
			}
		case komv1alpha1.Value:
			targetValue, err := resourceapi.ParseQuantity(inputScaler.Value)
			if err != nil {
				return nil, errors.NewBadRequest(fmt.Sprintf("Invalid Scaler Value: %v for deployment %v, error: %v", inputScaler.Value, deployment.Name, err.Error()))
			}

			target = v2beta2.MetricTarget{
				Type:  v2beta2.ValueMetricType,
				Value: &targetValue,
			}
		case komv1alpha1.Utilization:
			int64Value, err := strconv.ParseInt(inputScaler.Value, 10, 32)
			if err != nil {
				return nil, errors.NewBadRequest(fmt.Sprintf("Invalid Scaler Value: %v for deployment %v error: %v", inputScaler.Value, deployment.Name, err.Error()))
			}
			targetValue := int32(int64Value)
			if targetValue <= 0 || targetValue > 100 {
				return nil, errors.NewBadRequest(fmt.Sprintf("Invalid Scaler Value: %v for deployment %v should be a pourcentage", inputScaler.Value, deployment.Name))
			}
			target = v2beta2.MetricTarget{
				Type:               v2beta2.UtilizationMetricType,
				AverageUtilization: &targetValue,
			}
		default:
			return nil, errors.NewBadRequest(fmt.Sprintf("Unsupported Scaler Type [%v] in AutoScaling ", inputScaler.Type))
		}
		metrics = append(metrics, v2beta2.MetricSpec{
			Type: v2beta2.ResourceMetricSourceType,
			Resource: &v2beta2.ResourceMetricSource{
				Name:   resource,
				Target: target,
			},
		})
	}

	if len(metrics) > 0 {
		annotations := map[string]string{
			"creator": "kom-operator.kaiso.github.io/" + version.Version,
		}
		minReplicas := instance.Spec.Autoscaling.Min
		maxReplicas := instance.Spec.Autoscaling.Max

		var kind = "Deployment"
		if deployment.Kind != "" {
			kind = deployment.Kind
		}

		scaler := &v2beta2.HorizontalPodAutoscaler{
			TypeMeta: metav1.TypeMeta{
				Kind:       "HorizontalPodAutoscaler",
				APIVersion: "autoscaling/v2beta2",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        deployment.Name,
				Namespace:   deployment.Namespace,
				Annotations: annotations,
			},
			Spec: v2beta2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: v2beta2.CrossVersionObjectReference{
					APIVersion: deployment.APIVersion,
					Kind:       kind,
					Name:       deployment.Name,
				},
				MinReplicas: &minReplicas,
				MaxReplicas: maxReplicas,
				Metrics:     metrics,
			},
		}
		if err := controllerutil.SetControllerReference(instance, scaler, h.Scheme); err != nil {
			return nil, errors.NewBadRequest(fmt.Sprintf("Failed to set up owner reference for AutoScaler, deployment: %v, error: %v", deployment.Name, err.Error()))
		}
		return scaler, nil
	}

	return nil, errors.NewBadRequest(fmt.Sprintf("Invalid Scaler configuration for deployment %v", deployment.Name))
}

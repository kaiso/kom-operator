package util

import (
	"encoding/json"
	"fmt"
	"reflect"

	komv1alpha1 "github.com/kaiso/kom-operator/pkg/apis/kom/v1alpha1"
	"github.com/prometheus/common/log"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

//PrettyPrint print a struct in json with indentation
func PrettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return fmt.Sprintf(string(s))
}

//Equals checks object equality
func Equals(a, b komv1alpha1.Routing) bool {
	if &a == &b {
		return true
	}

	if len(a.HTTP) != len(b.HTTP) {
		return false
	}

	for _, k := range a.HTTP {
		found := false
		for _, v := range b.HTTP {
			if v.Rule == k.Rule &&
				v.Port == k.Port {
				found = true
			}
		}
		if found == false {
			return false
		}
	}

	return true
}

//ContainsString Helper functions to check string from a slice of strings.
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

//RemoveString Helper functions to remove string from a slice of strings.
func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// MicroserviceChangedPredicate predicate to detect changes
type MicroserviceChangedPredicate struct {
	predicate.Funcs
}

// Update implements default UpdateEvent filter for validating resource version change
func (MicroserviceChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.MetaOld == nil {
		log.Error(nil, "Update event has no old metadata", "event", e)
		return false
	}
	if e.ObjectOld == nil {
		log.Error(nil, "Update event has no old runtime object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		log.Error(nil, "Update event has no new runtime object for update", "event", e)
		return false
	}
	if e.MetaNew == nil {
		log.Error(nil, "Update event has no new metadata", "event", e)
		return false
	}
	if e.MetaNew.GetGeneration() == e.MetaOld.GetGeneration() &&
		e.MetaNew.GetGeneration() != 0 &&
		reflect.DeepEqual(e.MetaNew.GetFinalizers(), e.MetaOld.GetFinalizers()) {
		return false
	}
	return true
}

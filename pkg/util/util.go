package util

import (
	"encoding/json"
	"fmt"

	komv1alpha1 "github.com/kaiso/kom-operator/pkg/apis/kom/v1alpha1"
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

	if len(a.Http) != len(b.Http) {
		return false
	}

	for _, k := range a.Http {
		found := false
		for _, v := range b.Http {
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

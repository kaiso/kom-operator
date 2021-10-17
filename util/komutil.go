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

package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	komv1alpha1 "github.com/kaiso/kom-operator/api/v1alpha1"
)

// RunModeType operator run mode
type RunModeType string

const (
	KomOperatorRunModeEnvVar = "KOM_OPERATOR_RUNMODE"
	// LocalRunMode local run mode
	LocalRunMode RunModeType = "local"
	// ClusterRunMode cluster run mode
	ClusterRunMode RunModeType = "cluster"
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

func GetOperatorSdkVersion() string {
	return os.Getenv("OPERATOR_SDK_DL_URL")
}

func ReplaceRegexInFile(path, match, replace string) error {
	matcher, err := regexp.Compile(match)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	s := matcher.ReplaceAllString(string(b), replace)
	if s == string(b) {
		return errors.New("unable to find the content to be replaced")
	}
	err = ioutil.WriteFile(path, []byte(s), info.Mode())
	if err != nil {
		return err
	}
	return nil
}

func IsRunModeLocal() bool {
	runMode, found := os.LookupEnv(KomOperatorRunModeEnvVar)
	if found != true {
		panic(fmt.Sprintf(" run mode environment variable not defined"))
	}
	return runMode == string(LocalRunMode)
}

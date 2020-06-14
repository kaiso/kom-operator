package runtime

import (
	"fmt"
	"reflect"
	"unsafe"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/kubectl/pkg/cmd/set"
	"k8s.io/kubectl/pkg/polymorphichelpers"
	"k8s.io/kubectl/pkg/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var lbLogger = logf.Log.WithName("kom-runtime")

// RestartOptions is the start of the data required to perform the operation.  As new fields are added, add them here instead of
// referencing the cmd.Flags()
type RestartOptions struct {
	Resources []string

	Builder          func() *resource.Builder
	Restarter        polymorphichelpers.ObjectRestarterFunc
	Namespace        string
	EnforceNamespace bool
	fieldManager     string
}

// RunRestart restart a resource
func RunRestart(Config *rest.Config, Namespace string, Resources ...string) error {
	lbLogger.Info(fmt.Sprintf("Restarting resource [%v] in Namespace [%v]", Resources, Namespace))
	resources := []*restmapper.APIGroupResources{
		{
			Group: metav1.APIGroup{
				Name: "apps",
				Versions: []metav1.GroupVersionForDiscovery{
					{Version: "v1"},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{Version: "v1"},
			},
			VersionedResources: map[string][]metav1.APIResource{
				"v1": {
					{Name: "replicaset", Namespaced: true, Kind: "Replicaset"},
				},
			},
		},
	}

	RestMapper := restmapper.NewDiscoveryRESTMapper(resources)

	Builder := resource.Builder{}

	restMapperFn := reflect.ValueOf(&Builder).Elem().FieldByName("restMapperFn")
	SetUnexportedField(restMapperFn, func() (meta.RESTMapper, error) {
		return RestMapper, nil
	})

	clientConfigFn := reflect.ValueOf(&Builder).Elem().FieldByName("clientConfigFn")
	SetUnexportedField(clientConfigFn, func() (*rest.Config, error) {
		return Config, nil
	})

	Restarter := polymorphichelpers.ObjectRestarterFn
	EnforceNamespace := true
	r := Builder.
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		NamespaceParam(Namespace).DefaultNamespace().
		FilenameParam(EnforceNamespace, &resource.FilenameOptions{}).
		ResourceTypeOrNameArgs(true, Resources...).
		ContinueOnError().
		Latest().
		Flatten().
		Do()
	if err := r.Err(); err != nil {
		return err
	}

	allErrs := []error{}
	infos, err := r.Infos()
	if err != nil {
		// restore previous command behavior where
		// an error caused by retrieving infos due to
		// at least a single broken object did not result
		// in an immediate return, but rather an overall
		// aggregation of errors.
		allErrs = append(allErrs, err)
	}

	for _, patch := range set.CalculatePatches(infos, scheme.DefaultJSONEncoder(), set.PatchFn(Restarter)) {
		info := patch.Info

		if patch.Err != nil {
			resourceString := info.Mapping.Resource.Resource
			if len(info.Mapping.Resource.Group) > 0 {
				resourceString = resourceString + "." + info.Mapping.Resource.Group
			}
			allErrs = append(allErrs, fmt.Errorf("error: %s %q %v", resourceString, info.Name, patch.Err))
			continue
		}

		if string(patch.Patch) == "{}" || len(patch.Patch) == 0 {
			allErrs = append(allErrs, fmt.Errorf("failed to create patch for %v: empty patch", info.Name))
		}

		obj, err := resource.NewHelper(info.Client, info.Mapping).
			Patch(info.Namespace, info.Name, types.StrategicMergePatchType, patch.Patch, nil)
		if err != nil {
			allErrs = append(allErrs, fmt.Errorf("failed to patch: %v", err))
			continue
		}

		info.Refresh(obj, true)

	}

	return utilerrors.NewAggregate(allErrs)
}

func GetUnexportedField(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func SetUnexportedField(field reflect.Value, value interface{}) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).
		Elem().
		Set(reflect.ValueOf(value))
}

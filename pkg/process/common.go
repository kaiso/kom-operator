package process

import "os"

// RunModeType operator run mode
type RunModeType string

const (
	// LocalRunMode local run mode
	LocalRunMode RunModeType = "local"
	// ClusterRunMode cluster run mode
	ClusterRunMode RunModeType = "cluster"

	//ForceRunModeEnv  force run mode env
	ForceRunModeEnv string = "OSDK_FORCE_RUN_MODE"
)

//PatchStringValue specifies a patch operation for a string.
type PatchStringValue struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

//PatchUInt32Value specifies a patch operation for a uint32.
type PatchUInt32Value struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value uint32 `json:"value"`
}

//PatchInterfaceValue specifies a patch operation for a string.
type PatchInterfaceValue struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func isRunModeLocal() bool {
	return os.Getenv(ForceRunModeEnv) == string(LocalRunMode)
}

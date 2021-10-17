package process

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

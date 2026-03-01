package cnpg

type ClusterInfo struct {
	InstanceID string            `json:"instance_id"`
	ServiceID  string            `json:"service_id"`
	PlanID     string            `json:"plan_id"`
	Namespace  string            `json:"namespace"`
	Status     string            `json:"status"`
	Instances  int64             `json:"instances"`
	Ready      int64             `json:"ready_instances"`
	Phase      string            `json:"phase"`
	CPU        string            `json:"cpu"`
	Memory     string            `json:"memory"`
	Storage    string            `json:"storage"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type ClusterStatus struct {
	Exists         bool   `json:"exists"`
	Phase          string `json:"phase"`
	Ready          int64  `json:"ready_instances"`
	Total          int64  `json:"total_instances"`
	IsReady        bool   `json:"is_ready"`
	IsProvisioning bool   `json:"is_provisioning"`
	IsFailed       bool   `json:"is_failed"`
	FailureReason  string `json:"failure_reason,omitempty"`
	SpecInstances  int64  `json:"spec_instances"`
	SpecCPU        string `json:"spec_cpu"`
	SpecMemory     string `json:"spec_memory"`
	SpecStorage    string `json:"spec_storage"`
	SpecPlanID     string `json:"spec_plan_id"`
}

type NamespaceStatus struct {
	Exists        bool `json:"exists"`
	IsTerminating bool `json:"is_terminating"`
}

package cnpg

type ClusterInfo struct {
	Exists         bool              `json:"exists"`
	InstanceID     string            `json:"instance_id"`
	ServiceID      string            `json:"service_id"`
	PlanID         string            `json:"plan_id"`
	Namespace      string            `json:"namespace"`
	Name           string            `json:"name"`
	TotalInstances int64             `json:"total_instances"`
	ReadyInstances int64             `json:"ready_instances"`
	Phase          string            `json:"phase"`
	IsReady        bool              `json:"is_ready"`
	IsProvisioning bool              `json:"is_provisioning"`
	IsFailed       bool              `json:"is_failed"`
	FailureReason  string            `json:"failure_reason,omitempty"`
	Instances      int64             `json:"instances"`
	CPU            string            `json:"cpu"`
	Memory         string            `json:"memory"`
	Storage        string            `json:"storage"`
	Labels         map[string]string `json:"labels,omitempty"`
}

type NamespaceStatus struct {
	Exists        bool `json:"exists"`
	IsTerminating bool `json:"is_terminating"`
}

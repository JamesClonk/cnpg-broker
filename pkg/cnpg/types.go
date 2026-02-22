package cnpg

type ClusterInfo struct {
	InstanceID string            `json:"instance_id"`
	Namespace  string            `json:"namespace"`
	Status     string            `json:"status"`
	Instances  int64             `json:"instances"`
	Ready      int64             `json:"ready_instances"`
	Phase      string            `json:"phase"`
	Labels     map[string]string `json:"labels,omitempty"`
}

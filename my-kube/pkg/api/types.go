package api

// PodStatus represents the status of a Pod
type PodStatus string

const (
	PodPending   PodStatus = "Pending"
	PodRunning   PodStatus = "Running"
	PodSucceeded PodStatus = "Succeeded"
	PodFailed    PodStatus = "Failed"
)

// Pod represents a simplified Kubernetes Pod
type Pod struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Command  []string  `json:"command"` // e.g. ["python3", "-m", "http.server"]
	Status   PodStatus `json:"status"`
	NodeID   string    `json:"node_id,omitempty"` // Which node is running this pod
	PodIP    string    `json:"pod_ip,omitempty"`
}

// Node represents a Worker Node (Kubelet)
type Node struct {
	ID            string `json:"id"`
	IP            string `json:"ip"`
	MemoryTotalMB int    `json:"memory_total_mb"`
	MemoryUsedMB  int    `json:"memory_used_mb"`
}

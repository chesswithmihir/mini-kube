package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"my-kube/pkg/api"
)

// MockRuntime
type MockRuntime struct {
	StartedPods []string
}

func (m *MockRuntime) RunPod(pod api.Pod) error {
	m.StartedPods = append(m.StartedPods, pod.ID)
	return nil
}
func (m *MockRuntime) StopPod(id string) error { return nil }
func (m *MockRuntime) ListRunningPods() ([]string, error) { return nil, nil }

func TestAgent_Sync(t *testing.T) {
	// 1. Setup Mock Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Expect GET /nodes/node-1/pods
		if r.URL.Path == "/nodes/node-1/pods" {
			pods := []api.Pod{{ID: "pod-1", Command: []string{"echo"}}}
			json.NewEncoder(w).Encode(pods)
			return
		}
		http.Error(w, "not found", 404)
	}))
	defer server.Close()
	
	// 2. Setup Agent
	mockRuntime := &MockRuntime{}
	agent := NewAgent("node-1", server.URL, mockRuntime)
	
	// 3. Run Sync
	err := agent.sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	
	// 4. Verify Pod was "Started"
	if len(mockRuntime.StartedPods) != 1 {
		t.Errorf("Expected 1 pod started, got %d", len(mockRuntime.StartedPods))
	}
	if mockRuntime.StartedPods[0] != "pod-1" {
		t.Errorf("Expected pod-1, got %s", mockRuntime.StartedPods[0])
	}
}

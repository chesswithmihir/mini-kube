package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"my-kube/pkg/api"
)

func TestAPIServer_HandlePods(t *testing.T) {
	store := NewMemoryStore()
	server := NewAPIServer(store)
	
	// 1. Create a Pod
	pod := api.Pod{ID: "pod-1", Name: "nginx", Command: []string{"nginx"}}
	body, _ := json.Marshal(pod)
	req := httptest.NewRequest("POST", "/pods", bytes.NewReader(body))
	w := httptest.NewRecorder()
	
	server.ServeHTTP(w, req)
	
	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201 Created, got %d", w.Code)
	}
	
	// 2. List Pods
	req = httptest.NewRequest("GET", "/pods", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	
	var pods []api.Pod
	json.NewDecoder(w.Body).Decode(&pods)
	if len(pods) != 1 {
		t.Errorf("Expected 1 pod, got %d", len(pods))
	}
}

func TestAPIServer_NodePolling(t *testing.T) {
	store := NewMemoryStore()
	server := NewAPIServer(store)
	
	// Setup: 1 Node, 1 Pod assigned to it
	node := api.Node{ID: "node-1"}
	store.RegisterNode(node)
	pod := api.Pod{ID: "pod-1", NodeID: "node-1", Command: []string{"sleep"}}
	store.AddPod(pod)
	
	// Act: Node polls for work
	// Note: We need to use the full path including /nodes prefix because ServeHTTP uses mux
	// However, our manual mux implementation in ServeHTTP expects the request to hit the root mux
	req := httptest.NewRequest("GET", "/nodes/node-1/pods", nil)
	w := httptest.NewRecorder()
	
	// In the real main(), we strip prefixes or use a proper router.
	// In ServeHTTP above: mux.HandleFunc("/nodes", s.handleNodes)
	// So requests to /nodes/node-1/pods WILL match /nodes prefix
	server.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}
	
	var pods []api.Pod
	json.NewDecoder(w.Body).Decode(&pods)
	if len(pods) != 1 {
		t.Errorf("Expected 1 assigned pod, got %d", len(pods))
	}
	if pods[0].ID != "pod-1" {
		t.Errorf("Expected pod-1, got %s", pods[0].ID)
	}
}

package server

import (
	"my-kube/pkg/api"
	"testing"
)

func TestMemoryStore_PodOperations(t *testing.T) {
	store := NewMemoryStore()
	pod := api.Pod{
		ID:      "pod-1",
		Name:    "test-pod",
		Command: []string{"echo", "hello"},
	}

	// 1. Add Pod
	if err := store.AddPod(pod); err != nil {
		t.Fatalf("Failed to add pod: %v", err)
	}

	// 2. Get Pod
	fetched, exists := store.GetPod("pod-1")
	if !exists {
		t.Fatal("Pod should exist")
	}
	if fetched.Name != "test-pod" {
		t.Errorf("Expected name test-pod, got %s", fetched.Name)
	}
	if fetched.Status != api.PodPending {
		t.Errorf("Expected default status Pending, got %s", fetched.Status)
	}

	// 3. Update Status
	store.UpdatePodStatus("pod-1", api.PodRunning)
	fetched, _ = store.GetPod("pod-1")
	if fetched.Status != api.PodRunning {
		t.Errorf("Expected status Running, got %s", fetched.Status)
	}
	
	// 4. List Pods
	list := store.ListPods()
	if len(list) != 1 {
		t.Errorf("Expected 1 pod, got %d", len(list))
	}
}

func TestMemoryStore_NodeAssignment(t *testing.T) {
	store := NewMemoryStore()
	pod := api.Pod{ID: "pod-1"}
	node := api.Node{ID: "node-1"}

	store.AddPod(pod)
	store.RegisterNode(node)

	// 1. Assign
	if err := store.AssignPodToNode("pod-1", "node-1"); err != nil {
		t.Fatalf("Failed to assign: %v", err)
	}

	// 2. Verify
	fetched, _ := store.GetPod("pod-1")
	if fetched.NodeID != "node-1" {
		t.Errorf("Expected NodeID node-1, got %s", fetched.NodeID)
	}
}

func TestMemoryStore_ThreadSafety(t *testing.T) {
	store := NewMemoryStore()
	
	// Run concurrent adds
	go func() {
		for i := 0; i < 100; i++ {
			store.AddPod(api.Pod{ID: "pod-a"}) // Will error on duplicates, that's fine
		}
	}()
	
	go func() {
		for i := 0; i < 100; i++ {
			store.ListPods()
		}
	}()
	
	// If this crashes with "concurrent map read/write", test fails
}

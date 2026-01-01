package server

import (
	"my-kube/pkg/api"
	"testing"
)

func TestScheduler_ScheduleAll(t *testing.T) {
	// 1. Setup Store with 1 Node and 1 Pending Pod
	store := NewMemoryStore()
	node := api.Node{ID: "node-1"}
	store.RegisterNode(node)
	
	pod := api.Pod{ID: "pod-1", Status: api.PodPending}
	store.AddPod(pod)

	// 2. Run Scheduler
	scheduler := NewScheduler(store)
	scheduler.ScheduleAll()

	// 3. Verify Assignment
	updatedPod, exists := store.GetPod("pod-1")
	if !exists {
		t.Fatal("Pod should exist")
	}
	if updatedPod.NodeID != "node-1" {
		t.Errorf("Expected pod to be assigned to node-1, got %q", updatedPod.NodeID)
	}
}

func TestScheduler_NoNodes(t *testing.T) {
	// 1. Setup Store with 0 Nodes and 1 Pending Pod
	store := NewMemoryStore()
	pod := api.Pod{ID: "pod-1", Status: api.PodPending}
	store.AddPod(pod)

	// 2. Run Scheduler
	scheduler := NewScheduler(store)
	scheduler.ScheduleAll()

	// 3. Verify No Assignment
	updatedPod, _ := store.GetPod("pod-1")
	if updatedPod.NodeID != "" {
		t.Errorf("Expected pod to remain unassigned, got %q", updatedPod.NodeID)
	}
}

func TestScheduler_AlreadyAssigned(t *testing.T) {
	// 1. Setup Store with 1 Node and 1 Assigned Pod
	store := NewMemoryStore()
	store.RegisterNode(api.Node{ID: "node-1"})
	store.RegisterNode(api.Node{ID: "node-2"}) // Another node exists
	
	pod := api.Pod{ID: "pod-1", Status: api.PodPending, NodeID: "node-1"}
	store.AddPod(pod)

	// 2. Run Scheduler
	scheduler := NewScheduler(store)
	scheduler.ScheduleAll()

	// 3. Verify it wasn't moved to node-2 (because it's already assigned)
	updatedPod, _ := store.GetPod("pod-1")
	if updatedPod.NodeID != "node-1" {
		t.Errorf("Expected pod to stay on node-1, got %q", updatedPod.NodeID)
	}
}

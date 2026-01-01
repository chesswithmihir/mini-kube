package server

import (
	"fmt"
	"time"
	"my-kube/pkg/api"
)

// Scheduler watches for pending pods and assigns them to nodes.
type Scheduler struct {
	store Store
}

func NewScheduler(store Store) *Scheduler {
	return &Scheduler{store: store}
}

// Run starts the scheduler loop.
func (s *Scheduler) Run() {
	for {
		time.Sleep(5 * time.Second)
		s.ScheduleAll()
	}
}

// ScheduleAll attempts to schedule all pending pods.
func (s *Scheduler) ScheduleAll() {
	pods := s.store.ListPods()
	nodes := s.store.ListNodes()

	if len(nodes) == 0 {
		return
	}

	for _, pod := range pods {
		if pod.NodeID == "" && pod.Status == api.PodPending {
			s.schedulePod(&pod, nodes)
		}
	}
}

// schedulePod selects a node for a single pod.
func (s *Scheduler) schedulePod(pod *api.Pod, nodes []api.Node) {
	// Simple Round-Robin or Random strategy could go here.
	// For now, sticking to the "pick first node" strategy.
	node := nodes[0]

	fmt.Printf("Scheduler: Assigning pod %s to node %s\n", pod.ID, node.ID)
	if err := s.store.AssignPodToNode(pod.ID, node.ID); err != nil {
		fmt.Printf("Scheduler: Failed to assign pod %s: %v\n", pod.ID, err)
	}
}

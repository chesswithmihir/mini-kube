package server

import (
	"fmt"
	"sync"
	"my-kube/pkg/api"
)

// Store interface for mocking
type Store interface {
	AddPod(pod api.Pod) error
	GetPod(id string) (api.Pod, bool)
	ListPods() []api.Pod
	UpdatePodStatus(id string, status api.PodStatus) error
	AssignPodToNode(podID string, nodeID string) error
	
	RegisterNode(node api.Node) error
	GetNode(id string) (api.Node, bool)
	ListNodes() []api.Node
}

// MemoryStore is an in-memory implementation of Store
type MemoryStore struct {
	pods  map[string]api.Pod
	nodes map[string]api.Node
	mu    sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		pods:  make(map[string]api.Pod),
		nodes: make(map[string]api.Node),
	}
}

func (s *MemoryStore) AddPod(pod api.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.pods[pod.ID]; exists {
		return fmt.Errorf("pod %s already exists", pod.ID)
	}
	// Default status
	if pod.Status == "" {
		pod.Status = api.PodPending
	}
	s.pods[pod.ID] = pod
	return nil
}

func (s *MemoryStore) GetPod(id string) (api.Pod, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pod, exists := s.pods[id]
	return pod, exists
}

func (s *MemoryStore) ListPods() []api.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]api.Pod, 0, len(s.pods))
	for _, p := range s.pods {
		list = append(list, p)
	}
	return list
}

func (s *MemoryStore) UpdatePodStatus(id string, status api.PodStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pod, exists := s.pods[id]
	if !exists {
		return fmt.Errorf("pod %s not found", id)
	}
	pod.Status = status
	s.pods[id] = pod
	return nil
}

func (s *MemoryStore) AssignPodToNode(podID string, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	pod, exists := s.pods[podID]
	if !exists {
		return fmt.Errorf("pod %s not found", podID)
	}
	if _, nodeExists := s.nodes[nodeID]; !nodeExists {
		return fmt.Errorf("node %s not found", nodeID)
	}
	
	pod.NodeID = nodeID
	s.pods[podID] = pod
	return nil
}

func (s *MemoryStore) RegisterNode(node api.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node.ID] = node
	return nil
}

func (s *MemoryStore) GetNode(id string) (api.Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, exists := s.nodes[id]
	return node, exists
}

func (s *MemoryStore) ListNodes() []api.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]api.Node, 0, len(s.nodes))
	for _, n := range s.nodes {
		list = append(list, n)
	}
	return list
}

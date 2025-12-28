package server

import (
	"encoding/json"
	"net/http"
	"strings"
	
	"my-kube/pkg/api"
)

type APIServer struct {
	store Store
}

func NewAPIServer(store Store) *APIServer {
	return &APIServer{store: store}
}

func (s *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()
	
	mux.HandleFunc("/pods", s.handlePods)
	mux.HandleFunc("/nodes", s.handleNodes)
	mux.HandleFunc("/nodes/", s.handleNodes)
	
	mux.ServeHTTP(w, r)
}

func (s *APIServer) handlePods(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var pod api.Pod
		if err := json.NewDecoder(r.Body).Decode(&pod); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Simple validation
		if pod.ID == "" || len(pod.Command) == 0 {
			http.Error(w, "ID and Command are required", http.StatusBadRequest)
			return
		}
		
		if err := s.store.AddPod(pod); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(pod)
		
	case http.MethodGet:
		pods := s.store.ListPods()
		json.NewEncoder(w).Encode(pods)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *APIServer) handleNodes(w http.ResponseWriter, r *http.Request) {
	// Check if this is a request for /nodes/{id}/pods
	// Path: /nodes
	// r.URL.Path might be /nodes/node-1/pods
	
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	// ["nodes"] -> List/Register
	// ["nodes", "node-1", "pods"] -> Get Pods for Node
	
	if len(parts) == 3 && parts[2] == "pods" && r.Method == http.MethodGet {
		nodeID := parts[1]
		s.handleGetNodePods(w, r, nodeID)
		return
	}

	switch r.Method {
	case http.MethodPost:
		var node api.Node
		if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.store.RegisterNode(node); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		
	case http.MethodGet:
		nodes := s.store.ListNodes()
		json.NewEncoder(w).Encode(nodes)
		
	default:
		// If it wasn't captured above, it's 404 or 405
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetNodePods returns the list of pods assigned to a specific node
func (s *APIServer) handleGetNodePods(w http.ResponseWriter, r *http.Request, nodeID string) {
	allPods := s.store.ListPods()
	assigned := []api.Pod{}
	
	for _, p := range allPods {
		if p.NodeID == nodeID {
			assigned = append(assigned, p)
		}
	}
	
	json.NewEncoder(w).Encode(assigned)
}

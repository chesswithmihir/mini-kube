package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"my-kube/pkg/api"
	"my-kube/pkg/server"
)

func main() {
	fmt.Println("Starting my-kube-server (Control Plane)...")

	store := server.NewMemoryStore()
	apiServer := server.NewAPIServer(store)

	// Start Scheduler in background
	go runScheduler(store)

	// Start HTTP Server
	fmt.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe(":8080", apiServer)) // ServeHTTP method makes it a Handler
}

func runScheduler(s server.Store) {
	for {
		time.Sleep(5 * time.Second)
		
pods := s.ListPods()
nodes := s.ListNodes()
		
		if len(nodes) == 0 {
			continue
		}

		for _, pod := range pods {
			if pod.NodeID == "" && pod.Status == api.PodPending {
				// Round-robin or random? 
				// Dumbest scheduler ever: Always pick the first node
				node := nodes[0]
				
				fmt.Printf("Scheduler: Assigning pod %s to node %s\n", pod.ID, node.ID)
				s.AssignPodToNode(pod.ID, node.ID)
			}
		}
	}
}
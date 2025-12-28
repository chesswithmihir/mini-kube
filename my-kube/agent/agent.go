package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	
	"my-kube/pkg/api"
)

type Agent struct {
	NodeID     string
	ServerURL  string
	Runtime    Runtime
	Client     *http.Client
	knownPods  map[string]bool // Set of pod IDs we have started
}

func NewAgent(nodeID, serverURL string, runtime Runtime) *Agent {
	return &Agent{
		NodeID:    nodeID,
		ServerURL: serverURL,
		Runtime:   runtime,
		Client:    &http.Client{Timeout: 5 * time.Second},
		knownPods: make(map[string]bool),
	}
}

func (a *Agent) Register() error {
	node := api.Node{
		ID:            a.NodeID,
		MemoryTotalMB: 1024,
	}
	
	body, _ := json.Marshal(node)
	resp, err := a.Client.Post(a.ServerURL+"/nodes", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed: %d", resp.StatusCode)
	}
	return nil
}

func (a *Agent) SyncLoop() {
	for {
		if err := a.sync(); err != nil {
			fmt.Printf("Sync error: %v\n", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (a *Agent) sync() error {
	// 1. Get assigned pods
	resp, err := a.Client.Get(fmt.Sprintf("%s/nodes/%s/pods", a.ServerURL, a.NodeID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	var desiredPods []api.Pod
	if err := json.NewDecoder(resp.Body).Decode(&desiredPods); err != nil {
		return err
	}
	
	// 2. Diff and Act
	for _, pod := range desiredPods {
		if !a.knownPods[pod.ID] {
			fmt.Printf("Found new pod %s. Starting...\n", pod.ID)
			if err := a.Runtime.RunPod(pod); err != nil {
				fmt.Printf("Failed to run pod %s: %v\n", pod.ID, err)
				continue
			}
			a.knownPods[pod.ID] = true
		}
	}
	
	return nil
}

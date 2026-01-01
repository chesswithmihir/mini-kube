//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"my-kube/pkg/api"
)

func TestE2E_FullFlow(t *testing.T) {
				// 1. Build Binaries
				rootDir, err := filepath.Abs("../..")
				if err != nil {			t.Fatalf("Failed to get root dir: %v", err)
		}
	
		// Clean bin dir to ensure fresh build
		cmd := exec.Command("make", "clean")
		cmd.Dir = rootDir
		cmd.Run()
	
		cmd = exec.Command("make", "all")
		cmd.Dir = rootDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("Failed to build: %v\n%s", err, out)
		}
	
		binDir := filepath.Join(rootDir, "bin")
		serverBin := filepath.Join(binDir, "my-kube-server")
		agentBin := filepath.Join(binDir, "my-kubelet")
		mockRunc := filepath.Join(rootDir, "my-kube", "e2e", "mock-runc.sh")
		mockLog := "/tmp/mini-kube-e2e-runc.log"

	// Cleanup previous logs
	os.Remove(mockLog)

	// 2. Start Server
	serverPort := 58080
	serverURL := fmt.Sprintf("http://localhost:%d", serverPort)
	serverCmd := exec.Command(serverBin, "-port", fmt.Sprintf("%d", serverPort))
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		serverCmd.Process.Kill()
		serverCmd.Wait()
	}()

	// Wait for server to be up
	waitForServer(t, serverURL)

	// 3. Start Agent
	nodeID := "e2e-worker-1"
	agentCmd := exec.Command(agentBin, 
		"-node", nodeID, 
		"-server", serverURL,
		"-runc", mockRunc,
	)
	agentCmd.Stdout = os.Stdout
	agentCmd.Stderr = os.Stderr
	if err := agentCmd.Start(); err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}
	defer func() {
		agentCmd.Process.Kill()
		agentCmd.Wait()
	}()

	// Wait for node to register
	waitForNode(t, serverURL, nodeID)

	// 4. Submit a Pod
	podID := "e2e-pod-1"
	pod := api.Pod{
		ID:      podID,
		Name:    "test-pod",
		Command: []string{"echo", "hello-world"},
	}
	body, _ := json.Marshal(pod)
	resp, err := http.Post(serverURL+"/pods", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d", resp.StatusCode)
	}

	// 5. Wait for Pod Assignment and Execution
	// The scheduler runs every 5 seconds. The agent syncs every 5 seconds.
	// We give it some time.
	deadline := time.Now().Add(15 * time.Second)
	success := false
	for time.Now().Before(deadline) {
		// Check 1: Is it assigned on the server?
		if isAssigned(t, serverURL, podID, nodeID) {
			// Check 2: Did mock-runc execute?
			if checkMockLog(mockLog, "hello-world") {
				success = true
				break
			}
		}
		time.Sleep(1 * time.Second)
	}

	if !success {
		t.Fatalf("Timed out waiting for pod execution. Logs might be in %s", mockLog)
	}
}

func waitForServer(t *testing.T, url string) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, err := http.Get(url + "/pods")
		if err == nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("Server failed to start")
}

func waitForNode(t *testing.T, url, nodeID string) {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url + "/nodes")
		if err == nil {
			var nodes []api.Node
			json.NewDecoder(resp.Body).Decode(&nodes)
			resp.Body.Close()
			for _, n := range nodes {
				if n.ID == nodeID {
					return
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	t.Fatal("Node failed to register")
}

func isAssigned(t *testing.T, url, podID, nodeID string) bool {
	resp, err := http.Get(url + "/pods")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var pods []api.Pod
	json.NewDecoder(resp.Body).Decode(&pods)
	for _, p := range pods {
		if p.ID == podID && p.NodeID == nodeID {
			return true
		}
	}
	return false
}

func checkMockLog(logPath, expectedContent string) bool {
	content, err := os.ReadFile(logPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), expectedContent)
}

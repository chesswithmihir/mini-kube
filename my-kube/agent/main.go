package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	nodeID := flag.String("node", "worker-1", "Node ID")
	serverURL := flag.String("server", "http://localhost:8080", "Server URL")
	runcPath := flag.String("runc", "./my-runc", "Path to my-runc binary")
	flag.Parse()

	fmt.Printf("Starting my-kubelet %s...\n", *nodeID)
	
	// Verify runc exists
	if _, err := os.Stat(*runcPath); os.IsNotExist(err) {
		fmt.Printf("Warning: runc binary not found at %s\n", *runcPath)
	}

	runtime := NewMyRuncRuntime(*runcPath)
	agent := NewAgent(*nodeID, *serverURL, runtime)
	
	fmt.Println("Registering with Control Plane...")
	if err := agent.Register(); err != nil {
		fmt.Printf("Registration failed: %v. Retrying in loop...\n", err)
	} else {
		fmt.Println("Registration successful.")
	}

	agent.SyncLoop()
}
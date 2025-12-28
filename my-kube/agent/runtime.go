package main

import (
	"fmt"
	"os"
	"os/exec"
	"my-kube/pkg/api"
)

// Runtime defines the interface for running containers
type Runtime interface {
	RunPod(pod api.Pod) error
	StopPod(podID string) error
	ListRunningPods() ([]string, error)
}

// MyRuncRuntime implements Runtime using the my-runc binary
type MyRuncRuntime struct {
	BinaryPath string
}

func NewMyRuncRuntime(path string) *MyRuncRuntime {
	return &MyRuncRuntime{BinaryPath: path}
}

func (r *MyRuncRuntime) RunPod(pod api.Pod) error {
	// my-runc run --ip <ip> <command>
	// We need to construct the args
	
	args := []string{"run"}
	if pod.PodIP != "" {
		args = append(args, "--ip", pod.PodIP)
	}
	
	args = append(args, pod.Command...)
	
	cmd := exec.Command(r.BinaryPath, args...)
	// For now, we connect stdout/stderr to the agent's output
	// In production, we'd log to files
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	fmt.Printf("Agent: Executing %s %v\n", r.BinaryPath, args)
	
	// Start in background?
	// my-runc blocks until container exits.
	// Kubelet needs to manage multiple pods.
	// So we should Start() it and let it run.
	
	return cmd.Start()
}

func (r *MyRuncRuntime) StopPod(podID string) error {
	// TODO: Store PIDs so we can kill them
	return nil
}

func (r *MyRuncRuntime) ListRunningPods() ([]string, error) {
	// TODO: Check valid PIDs
	return []string{}, nil
}

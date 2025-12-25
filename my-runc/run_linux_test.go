//go:build linux

package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

var myRuncBinary = "./my-runc"

func TestMain(m *testing.M) {
	// Build the my-runc binary
	// We need to build it because the tests execute it as a subprocess
	cmd := exec.Command("go", "build", "-o", myRuncBinary, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to build my-runc binary: %v\nOutput: %s", err, output)
	}

	// Run the tests
	code := m.Run()

	// Clean up
	os.Remove(myRuncBinary)
	
	os.Exit(code)
}

func TestHostnameIsolation(t *testing.T) {
	// 1. Get Hostname of Host
	hostHostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("Failed to get host hostname: %v", err)
	}

	// 2. Run container and change hostname inside
	// Use Output() to capture only stdout (hostname), ignoring stderr (logs)
	cmd := exec.Command(myRuncBinary, "run", "sh", "-c", "hostname container-test && hostname")
	output, err := cmd.Output()
	if err != nil {
		// If it fails, capture the stderr to see why
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("Failed to run container: %v, Stderr: %s", err, exitErr.Stderr)
		}
		t.Fatalf("Failed to run container: %v", err)
	}

	containerHostname := strings.TrimSpace(string(output))

	// 3. Verify container hostname is different/changed
	if containerHostname != "container-test" {
		t.Errorf("Expected container hostname to be 'container-test', got '%s'", containerHostname)
	}

	// 4. Verify host hostname hasn't changed
	currentHostHostname, _ := os.Hostname()
	if currentHostHostname != hostHostname {
		t.Errorf("Host hostname was modified! Expected '%s', got '%s'", hostHostname, currentHostHostname)
	}
}

func TestPidIsolation(t *testing.T) {
	cmd := exec.Command(myRuncBinary, "run", "ps", "aux")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run ps in container: %v, Output: %s", err, output)
	}
	
	outputStr := string(output)

	if strings.Contains(outputStr, "sshd") {
		t.Error("PID isolation failed: found 'sshd' process from host inside container")
	}
	
	if !strings.Contains(outputStr, " 1 ") && !strings.Contains(outputStr, "\t1\t") {
		t.Error("PID isolation weirdness: No PID 1 found in container ps output")
	}
}

func TestFilesystemIsolation(t *testing.T) {
	// Check if /.pivot_root exists. It should NOT exist if unmount was successful.
	checkCmd := exec.Command(myRuncBinary, "run", "ls", "/.pivot_root")
	err := checkCmd.Run()
	
	// ls should FAIL because the file/dir shouldn't exist
	if err == nil {
		t.Error("Filesystem isolation failed: /.pivot_root still exists and is accessible")
	}
}

func TestCgroupsMemoryLimit(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found, skipping cgroup memory test")
	}

	// Python script: continuously allocate memory until OOM
	// We append 1MB chunks to a list.
	pythonScript := "l = []\nwhile True:\n l.append(' ' * 1024 * 1024)"
	
	cmd := exec.Command(myRuncBinary, "run", "python3", "-c", pythonScript)
	
	output, err := cmd.CombinedOutput()
	
	if err == nil {
		t.Fatal("Cgroup memory limit failed: Process finished successfully but should have been killed")
	}
	
	// The parent process (my-runc) will exit with status 1 if the child is killed.
	// It logs "Failed to run command in container: ... signal: killed"
	// So we check the output for evidence of the kill.
	if strings.Contains(string(output), "signal: killed") || strings.Contains(string(output), "Killed") {
		// Passed!
		return
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		status := exitErr.Sys().(syscall.WaitStatus)
		// If the parent itself was killed (unlikely), that's also a pass
		if status.Signaled() && status.Signal() == syscall.SIGKILL {
			return
		}
		t.Fatalf("Process failed with %v (Signal: %v), expected SIGKILL (OOM).\nOutput:\n%s", err, status.Signal(), output)
	} else {
		t.Fatalf("Process failed but not with ExitError: %v", err)
	}
}
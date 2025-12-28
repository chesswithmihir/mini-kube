//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

// TestNetworkSetupIntegration tests the full bridge setup.
// It requires root and the 'ip' command.
func TestNetworkSetupIntegration(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Skipping network test; requires root privileges")
	}

	// 1. Prepare: Pick a test IP and dummy PID
	// To test "real" network setup, we actually need a process with a netns.
	// So we will spawn a simple 'sleep' process in a new netns.
	cmd := exec.Command("sleep", "5")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNET,
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start dummy process: %v", err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()
	
	pid := cmd.Process.Pid
	containerIP := "10.244.0.100/16"
	bridgeName := "test-bridge0"
	bridgeIP := "10.244.0.1/16"

	// 2. Execute: Run setupNetwork (we will define this function later)
	// We pass a custom bridge name to avoid messing with the default one during tests
	err := setupNetwork(pid, containerIP, bridgeName, bridgeIP)
	if err != nil {
		t.Fatalf("setupNetwork failed: %v", err)
	}

	// 3. Verify: Check Host Side
	// Check if bridge exists
	out, err := exec.Command("ip", "link", "show", bridgeName).CombinedOutput()
	if err != nil {
		t.Errorf("Bridge %s not found: %s", bridgeName, out)
	}

	// Check if veth exists on host (it should be attached to bridge)
	// The naming convention in our implementation will be veth<PID>
	vethHost := fmt.Sprintf("veth%d", pid)
	out, err = exec.Command("ip", "link", "show", vethHost).CombinedOutput()
	if err != nil {
		t.Errorf("Host veth %s not found: %s", vethHost, out)
	}

	// 4. Verify: Check Container Side
	// We use nsenter to look inside the child's namespace
	// ping the gateway (bridge) to verify connectivity
	// We need to wait a tiny bit for the link to come up? Usually instant.
	
	// Check if interface exists inside
	checkCmd := exec.Command("nsenter", "-t", fmt.Sprintf("%d", pid), "-n", "ip", "addr", "show", "eth0")
	out, checkErr := checkCmd.CombinedOutput()
	if checkErr != nil {
		t.Errorf("Container eth0 not found: %s", out)
	} else {
		if !strings.Contains(string(out), "10.244.0.100") {
			t.Errorf("Container eth0 does not have correct IP. Got: %s", out)
		}
	}

	// Cleanup
	exec.Command("ip", "link", "delete", bridgeName).Run()
}

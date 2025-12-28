//go:build linux

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// setupNetwork configures the bridge networking for the container.
// It creates a veth pair, moves one end to the container, and assigns IPs.
func setupNetwork(pid int, containerIP string, bridgeName string, bridgeIP string) error {
	// Define interface names
	vethHost := fmt.Sprintf("veth%d", pid)
	vethContainer := fmt.Sprintf("veth-c%d", pid) // Temporary name before we move it

	// 1. Setup Bridge
	// We allow these to fail (e.g., if bridge already exists)
	exec.Command("ip", "link", "add", "name", bridgeName, "type", "bridge").Run()
	exec.Command("ip", "addr", "add", bridgeIP, "dev", bridgeName).Run()
	exec.Command("ip", "link", "set", bridgeName, "up").Run()

	// 2. Create Veth Pair
	// ip link add veth<PID> type veth peer name veth-c<PID>
	if out, err := exec.Command("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethContainer).CombinedOutput(); err != nil {
		return fmt.Errorf("creating veth pair failed: %s: %w", out, err)
	}

	// 3. Attach Host end to Bridge
	if out, err := exec.Command("ip", "link", "set", vethHost, "master", bridgeName).CombinedOutput(); err != nil {
		return fmt.Errorf("attaching veth to bridge failed: %s: %w", out, err)
	}
	if out, err := exec.Command("ip", "link", "set", vethHost, "up").CombinedOutput(); err != nil {
		return fmt.Errorf("setting veth up failed: %s: %w", out, err)
	}

	// 4. Move Container end to the new Namespace
	pidStr := strconv.Itoa(pid)
	if out, err := exec.Command("ip", "link", "set", vethContainer, "netns", pidStr).CombinedOutput(); err != nil {
		return fmt.Errorf("moving veth to namespace %s failed: %s: %w", pidStr, out, err)
	}

	// 5. Configure Interface INSIDE the Namespace (using nsenter)
	// We rename the random interface name to 'eth0' for standard looking containers
	if err := nsRun(pidStr, "ip", "link", "set", "dev", vethContainer, "name", "eth0"); err != nil {
		return err
	}
	if err := nsRun(pidStr, "ip", "addr", "add", containerIP, "dev", "eth0"); err != nil {
		return err
	}
	if err := nsRun(pidStr, "ip", "link", "set", "eth0", "up"); err != nil {
		return err
	}
	// Bring up loopback so localhost works
	if err := nsRun(pidStr, "ip", "link", "set", "lo", "up"); err != nil {
		return err
	}
	
	// Set Default Gateway (The Bridge IP)
	gateway := strings.Split(bridgeIP, "/")[0]
	if err := nsRun(pidStr, "ip", "route", "add", "default", "via", gateway); err != nil {
		return err
	}

	return nil
}

// nsRun is a helper to run a command inside a specific namespace using nsenter
func nsRun(pidStr string, cmdParts ...string) error {
	// nsenter -t <pid> -n <command...>
	// -t: target pid
	// -n: enter network namespace
	args := append([]string{"-t", pidStr, "-n"}, cmdParts...)
	cmd := exec.Command("nsenter", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("nsenter command %v failed: %s: %w", cmdParts, out, err)
	}
	return nil
}

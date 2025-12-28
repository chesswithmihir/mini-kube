//go:build linux

package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func run(commandToRun []string, ip string) {
	log.Printf("Running command: %s with IP: %s", commandToRun, ip)

	// 1. Create Synchronization Pipe
	// r: Read end (passed to child)
	// w: Write end (kept by parent)
	r, w, err := os.Pipe()
	if err != nil {
		log.Fatalf("Failed to create pipe: %v", err)
	}

	// Re-execute my-runc with the "child" command
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, commandToRun...)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWUSER,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	// Pass the pipe read-end to the child
	// It will be available as FD 3
	cmd.ExtraFiles = []*os.File{r}

	// 2. Start the Child (Fork/Clone)
	// The child will start but block reading from FD 3
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to run container: %v", err)
	}

	// Close parent's copy of the read-end
	r.Close()

	// 3. Setup Network (if requested)
	if ip != "" {
		// Use default bridge name and IP for now
		// Future: Make bridge name configurable
		// We append /16 to ensure the container IP is in the same subnet as the bridge (10.244.0.1/16)
		if err := setupNetwork(cmd.Process.Pid, ip+"/16", "my-bridge0", "10.244.0.1/16"); err != nil {
			log.Printf("Setup network failed: %v", err)
			cmd.Process.Kill()
			os.Exit(1)
		}
	}

	// 4. Signal Child to Continue
	// Write "OK" to unblock the child
	w.Write([]byte("OK"))
	w.Close()

	// 5. Wait for Child to Finish
	if err := cmd.Wait(); err != nil {
		log.Fatalf("Container process failed: %v", err)
	}
}

//go:build linux

package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func run(commandToRun []string) {
	log.Printf("Running command: %s", commandToRun)

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

	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run container: %v", err)
	}
}
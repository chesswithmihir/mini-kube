//go:build linux

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

// setupCgroups sets up cgroups for resource limiting
func setupCgroups() error {
	log.Println("Setting up cgroups...")

	cgroupPath := "/sys/fs/cgroup"
	var memPath string
	var limitFile string
	
	// Check if we are on Cgroups v1 (memory controller mounted at /sys/fs/cgroup/memory)
	// or Cgroups v2 (unified hierarchy at /sys/fs/cgroup)
	if _, err := os.Stat(filepath.Join(cgroupPath, "memory")); err == nil {
		// Cgroups v1
		log.Println("Detected Cgroups v1")
		memPath = filepath.Join(cgroupPath, "memory", "my-container")
		limitFile = "memory.limit_in_bytes"
	} else {
		// Cgroups v2
		log.Println("Detected Cgroups v2")
		memPath = filepath.Join(cgroupPath, "my-container")
		limitFile = "memory.max"
	}

	if err := os.Mkdir(memPath, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create cgroup directory: %w", err)
	}

	// Move the current process to the new cgroup
	pid := os.Getpid()
	procsFile := filepath.Join(memPath, "cgroup.procs")
	if err := os.WriteFile(procsFile, []byte(fmt.Sprintf("%d", pid)), 0700); err != nil {
		return fmt.Errorf("failed to write to cgroup.procs: %w", err)
	}

	// Set a memory limit of 100MB
	limitFilePath := filepath.Join(memPath, limitFile)
	if err := os.WriteFile(limitFilePath, []byte("100000000"), 0700); err != nil {
		return fmt.Errorf("failed to write to memory limit file: %w", err)
	}

	log.Println("Cgroups setup complete")
	return nil
}

// setupRootFS sets up a minimal root filesystem for the container
func setupRootFS(rootfs string) error {
	log.Println("Setting up root filesystem...")

	// 1. Create a location for the new root
	// We use a temporary directory to act as the new root mount point.
	newRoot := "/tmp/my-runc-root"
	if err := os.MkdirAll(newRoot, 0755); err != nil {
		return fmt.Errorf("failed to create new root dir: %w", err)
	}

	// 2. Bind mount the desired rootfs (e.g. "/") to the new location.
	// This makes 'newRoot' a mount point with the content of 'rootfs'.
	if err := syscall.Mount(rootfs, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to bind mount rootfs to new root: %w", err)
	}

	// 3. Make the new root private to this namespace
	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to make new root private: %w", err)
	}

	// 4. Create directory for the old root inside the new root
	putOld := filepath.Join(newRoot, ".pivot_root")
	if err := os.Mkdir(putOld, 0777); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create put_old directory: %w", err)
	}

	// 5. Pivot Root
	// Moves the root filesystem to putOld and makes newRoot the new root filesystem.
	if err := syscall.PivotRoot(newRoot, putOld); err != nil {
		return fmt.Errorf("failed to pivot_root: %w", err)
	}

	// 6. Change the current working directory to the new root ("/")
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to change directory to new root: %w", err)
	}

	// 7. Unmount the old root
	// It is now available at /.pivot_root
	if err := syscall.Unmount("/.pivot_root", syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("failed to unmount old root: %w", err)
	}

	// 8. Remove the temporary directory
	if err := os.Remove("/.pivot_root"); err != nil {
		return fmt.Errorf("failed to remove put_old directory: %w", err)
	}

	// 9. Mount proc filesystem
	// This is essential for tools like 'ps' to see the process list of the container (PID namespace)
	// instead of the host's process list.
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("failed to mount proc: %w", err)
	}

	log.Println("Root filesystem setup complete")
	return nil
}
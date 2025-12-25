//go:build !linux

package main

import "log"

// setupCgroups sets up cgroups for resource limiting
func setupCgroups() error {
	log.Println("Setting up cgroups is only supported on Linux.")
	return nil
}

// setupRootFS sets up a minimal root filesystem for the container
func setupRootFS(rootfs string) error {
	log.Println("Setting up root filesystem is only supported on Linux.")
	return nil
}
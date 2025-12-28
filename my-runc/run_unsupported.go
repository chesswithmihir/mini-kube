//go:build !linux

package main

import "log"

func run(commandToRun []string, ip string) {
	log.Fatal("Containerization is only supported on Linux.")
}

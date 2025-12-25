package main

import (
	"log"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: my-runc <command> [args...]")
	}

	command := os.Args[1]

	log.Printf("my-runc command: %s", command)

	switch command {
	case "run":
		// Run a container with the specified command
		if len(os.Args) < 3 {
			log.Fatal("Usage: my-runc run <command>")
		}
		// Parse the command to run
		commandToRun := os.Args[2:]
		run(commandToRun)

	case "child":
		// This is the child process, run inside the new namespaces
		if len(os.Args) < 3 {
			log.Fatal("Usage: my-runc child <command>")
		}
		commandToRun := os.Args[2:]
		log.Printf("Running command in child: %s", commandToRun)

		// Setup cgroups
		if err := setupCgroups(); err != nil {
			log.Fatalf("Failed to setup cgroups: %v", err)
		}

		// Setup root filesystem
		if err := setupRootFS("/"); err != nil {
			log.Fatalf("Failed to setup root filesystem: %v", err)
		}

		// Execute the command
		cmd := exec.Command(commandToRun[0], commandToRun[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Fatalf("Failed to run command in container: %v", err)
		}

	case "spec":
		// Generate a container spec
		log.Println("Generating container specification...")
		log.Println("This would create a container configuration file")

	case "version":
		// Show version information
		log.Println("my-runc version 0.1.0")

	default:
		log.Printf("Unknown command: %s", command)
		log.Println("Available commands: run, child, spec, version")
		os.Exit(1)
	}
}
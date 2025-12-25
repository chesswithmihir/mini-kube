//go:build !linux

package main

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var myRuncBinary = "./my-runc"

func TestMain(m *testing.M) {
	// Build the my-runc binary
	cmd := exec.Command("go", "build", "-o", myRuncBinary, ".")
	cmd.Dir = "./" // Current directory
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to build my-runc binary: %v\nOutput: %s", err, output)
	}

	// Run the tests
	code := m.Run()

	// Clean up the binary
	err = os.Remove(myRuncBinary)
	if err != nil {
		log.Printf("Failed to remove my-runc binary: %v", err)
	}

	os.Exit(code)
}

func TestRunUnsupported(t *testing.T) {
	// Run the my-runc command with the "run" argument
	cmd := exec.Command(myRuncBinary, "run", "echo", "hello")
	output, err := cmd.CombinedOutput()

	// Check that the command exited with a non-zero status code
	if err == nil {
		t.Fatal("Expected command to fail, but it succeeded")
	}

	// Check that the output contains the expected error message
	expected := "Containerization is only supported on Linux."
	if !strings.Contains(string(output), expected) {
		t.Fatalf("Expected output to contain %q, but got %q", expected, string(output))
	}

	// Check that the exit code is non-zero
	if exitError, ok := err.(*exec.ExitError); ok {
		if exitError.Success() {
			t.Fatalf("Expected command to exit with non-zero status, but got 0")
		}
	} else {
		t.Fatalf("Expected command to return an ExitError, but got %v", err)
	}
}

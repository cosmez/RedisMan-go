//go:build integration

package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestOneShotCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/redisman", "--command", "PING")
	cmd.Dir = "../../" // Run from the root of the project
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v\nStderr: %s", err, stderr.String())
	}

	output := strings.TrimSpace(out.String())
	if output != "PONG" {
		t.Errorf("Expected PONG, got %q", output)
	}
}

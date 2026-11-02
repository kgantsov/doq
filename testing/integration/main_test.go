//go:build integration

// Package integration runs DOQ as real processes: each test starts its own
// cluster of doq binaries on free ports, talks to it over HTTP, and can kill
// or restart nodes to exercise failover and recovery.
//
// Run with `make test_integration`. Set DOQ_BINARY to test a prebuilt binary
// (e.g. an older build) instead of building ./cmd/server.
package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// doqBinary is the path of the server binary every test cluster runs.
var doqBinary string

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	if bin := os.Getenv("DOQ_BINARY"); bin != "" {
		doqBinary = bin
		return m.Run()
	}

	dir, err := os.MkdirTemp("", "doq-integration-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)

	doqBinary = filepath.Join(dir, "doq")
	// The server embeds the web UI, so cmd/server/assets and index.html must
	// exist (make build_web).
	build := exec.Command("go", "build", "-o", doqBinary, "./cmd/server")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "building doq failed (did you run make build_web?): %v\n%s", err, out)
		return 1
	}

	return m.Run()
}

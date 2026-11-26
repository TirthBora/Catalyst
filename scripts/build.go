package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type target struct {
	goos   string
	goarch string
	output string
}

func main() {
	version := "1.0.0"
	buildDate := time.Now().UTC().Format(time.RFC3339)

	// Retrieve git commit hash
	commit := "unknown"
	if gitCmd := exec.Command("git", "rev-parse", "--short", "HEAD"); gitCmd != nil {
		if out, err := gitCmd.Output(); err == nil {
			commit = strings.TrimSpace(string(out))
		}
	}

	ldflags := fmt.Sprintf("-s -w -X catalyst/internal/cli.Version=%s -X catalyst/internal/cli.Commit=%s -X catalyst/internal/cli.BuildDate=%s",
		version, commit, buildDate)

	binDir := "bin"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		fmt.Printf("Failed to create bin directory: %v\n", err)
		os.Exit(1)
	}

	targets := []target{
		{goos: "windows", goarch: "amd64", output: filepath.Join(binDir, "catalyst-windows-amd64.exe")},
		{goos: "linux", goarch: "amd64", output: filepath.Join(binDir, "catalyst-linux-amd64")},
		{goos: "darwin", goarch: "arm64", output: filepath.Join(binDir, "catalyst-darwin-arm64")},
		{goos: "darwin", goarch: "amd64", output: filepath.Join(binDir, "catalyst-darwin-amd64")},
	}

	// Also build local host binary as bin/catalyst(.exe)
	hostExt := ""
	if runtime.GOOS == "windows" {
		hostExt = ".exe"
	}
	hostTarget := target{
		goos:   runtime.GOOS,
		goarch: runtime.GOARCH,
		output: filepath.Join(binDir, "catalyst"+hostExt),
	}
	targets = append([]target{hostTarget}, targets...)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  BUILDING CATALYST v%s (%s)\n", version, commit)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, t := range targets {
		fmt.Printf("  • Compiling %s/%s -> %s ... ", t.goos, t.goarch, t.output)

		cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", t.output, "./cmd/catalyst")
		cmd.Env = append(os.Environ(),
			"GOOS="+t.goos,
			"GOARCH="+t.goarch,
			"CGO_ENABLED=0",
		)

		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("FAILED\n%s\n", string(out))
		} else {
			fmt.Println("OK")
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Build complete. Binaries are available in ./bin/")
}

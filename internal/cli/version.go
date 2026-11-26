package cli

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current release version of Catalyst.
	Version = "1.0.0"

	// Commit is the git commit hash injected at build time.
	Commit = "none"

	// BuildDate is the date and time Catalyst was compiled.
	BuildDate = "unknown"
)

// Info contains structured version and build metadata.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// GetInfo returns complete runtime version and platform metadata.
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// FormatVersion formats the version information for terminal display.
func FormatVersion() string {
	info := GetInfo()
	return fmt.Sprintf("Catalyst v%s (%s) built on %s with %s on %s",
		info.Version, info.Commit, info.BuildDate, info.GoVersion, info.Platform)
}

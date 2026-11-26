package plugin_test

import (
	"testing"

	"catalyst/internal/plugin"
)

func TestHookDispatcher(t *testing.T) {
	d := plugin.NewHookDispatcher()

	d.RegisterHook(plugin.Hook{
		Type:     plugin.HookOnFileChanged,
		Pattern:  "*.proto",
		ActionID: "protoc-gen",
	})

	d.RegisterHook(plugin.Hook{
		Type:     plugin.HookOnFileChanged,
		Pattern:  "*.go",
		ActionID: "go-vet",
	})

	d.RegisterHook(plugin.Hook{
		Type:     plugin.HookOnStart,
		ActionID: "db-seed",
	})

	// 1. File matches
	protoMatches := d.MatchFileChanged("models/user.proto")
	if len(protoMatches) != 1 || protoMatches[0] != "protoc-gen" {
		t.Errorf("expected [protoc-gen] for user.proto, got %+v", protoMatches)
	}

	goMatches := d.MatchFileChanged("cmd/server.go")
	if len(goMatches) != 1 || goMatches[0] != "go-vet" {
		t.Errorf("expected [go-vet] for server.go, got %+v", goMatches)
	}

	mdMatches := d.MatchFileChanged("README.md")
	if len(mdMatches) != 0 {
		t.Errorf("expected 0 matches for README.md, got %+v", mdMatches)
	}

	// 2. OnStart matches
	startMatches := d.MatchOnStart()
	if len(startMatches) != 1 || startMatches[0] != "db-seed" {
		t.Errorf("expected [db-seed] for OnStart, got %+v", startMatches)
	}
}

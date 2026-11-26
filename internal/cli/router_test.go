package cli_test

import (
	"strings"
	"testing"

	"catalyst/internal/cli"
)

func TestParseArgs(t *testing.T) {
	// 1. Default (no args)
	opts, err := cli.ParseArgs(nil)
	if err != nil || opts.Action != cli.ActionDev {
		t.Errorf("expected ActionDev for empty args, got %s", opts.Action)
	}

	// 2. Subcommands
	tests := []struct {
		args     []string
		expected cli.Action
	}{
		{[]string{"dev"}, cli.ActionDev},
		{[]string{"start"}, cli.ActionDev},
		{[]string{"stop"}, cli.ActionStop},
		{[]string{"status"}, cli.ActionStatus},
		{[]string{"doctor"}, cli.ActionDoctor},
		{[]string{"version"}, cli.ActionVersion},
		{[]string{"--version"}, cli.ActionVersion},
		{[]string{"help"}, cli.ActionHelp},
		{[]string{"--help"}, cli.ActionHelp},
		{[]string{"-h"}, cli.ActionHelp},
		{[]string{"--doctor"}, cli.ActionDoctor},
	}

	for _, tt := range tests {
		o, err := cli.ParseArgs(tt.args)
		if err != nil {
			t.Errorf("failed to parse %v: %v", tt.args, err)
		}
		if o.Action != tt.expected {
			t.Errorf("for args %v expected %s, got %s", tt.args, tt.expected, o.Action)
		}
	}

	// 3. Flags with subcommand
	statusOpts, err := cli.ParseArgs([]string{"status", "-p", "8888"})
	if err != nil || statusOpts.Action != cli.ActionStatus || statusOpts.Port != "8888" {
		t.Errorf("expected status on port 8888, got %+v", statusOpts)
	}

	// 4. Flags without subcommand (defaults to dev)
	devOpts, err := cli.ParseArgs([]string{"-p", "7777", "-d", "/workspace"})
	if err != nil || devOpts.Action != cli.ActionDev || devOpts.Port != "7777" || devOpts.Dir != "/workspace" {
		t.Errorf("expected dev on port 7777 dir /workspace, got %+v", devOpts)
	}
}

func TestHelpText(t *testing.T) {
	help := cli.HelpText()
	if !strings.Contains(help, "Usage:") || !strings.Contains(help, "Available Commands:") {
		t.Errorf("expected structured help text, got: %s", help)
	}
}

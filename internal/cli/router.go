package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Action represents the resolved CLI command to execute.
type Action string

const (
	ActionDev     Action = "dev"
	ActionStop    Action = "stop"
	ActionStatus  Action = "status"
	ActionDoctor  Action = "doctor"
	ActionVersion Action = "version"
	ActionHelp    Action = "help"
)

// Options holds common command-line options parsed by the CLI.
type Options struct {
	Action     Action
	Port       string
	Dir        string
	RelayURL   string
	RelayToken string
}

// HelpText returns the formatted Catalyst CLI usage manual.
func HelpText() string {
	return `Catalyst - Second-Screen Development Workspace Companion

Usage:
  catalyst [command] [flags]

Available Commands:
  dev        Start the Catalyst development daemon (default)
  stop       Stop a running Catalyst daemon
  status     Check status of the running Catalyst daemon
  doctor     Diagnose local development toolchains and workspace health
  version    Print Catalyst version and build info
  help       Show this help message

Flags:
  -p, -port string          Port to bind HTTP/WebSocket server (default: 9999)
  -d, -dir  string          Workspace directory to watch and preview (default: .)
  -r, -relay string         CatalystRelay WebSocket URL for WAN tunneling (optional)
  -relay-token string       Authentication token for the cloud relay server
  -h, -help                 Show help information
`
}

// ParseArgs parses CLI arguments into structured Options.
func ParseArgs(args []string) (Options, error) {
	opts := Options{
		Action: ActionDev,
		Port:   "9999",
		Dir:    ".",
	}

	if len(args) == 0 {
		return opts, nil
	}

	// Check first argument for subcommand
	subcmd := args[0]
	var flagArgs []string

	switch subcmd {
	case "dev", "start":
		opts.Action = ActionDev
		flagArgs = args[1:]
	case "stop":
		opts.Action = ActionStop
		flagArgs = args[1:]
	case "status":
		opts.Action = ActionStatus
		flagArgs = args[1:]
	case "doctor":
		opts.Action = ActionDoctor
		flagArgs = args[1:]
	case "version", "-v", "--version":
		opts.Action = ActionVersion
		return opts, nil
	case "help", "-h", "--help":
		opts.Action = ActionHelp
		return opts, nil
	default:
		// No subcommand matched, treat all arguments as flags for dev mode
		flagArgs = args
	}

	fs := flag.NewFlagSet("catalyst", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var port string
	var dir string
	var relayURL string
	var relayToken string
	var help bool
	var doctorFlag bool

	defaultPort := os.Getenv("CATALYST_PORT")
	if defaultPort == "" {
		defaultPort = "9999"
	}

	defaultRelayURL := os.Getenv("CATALYST_RELAY_URL")
	defaultRelayToken := os.Getenv("CATALYST_RELAY_TOKEN")

	fs.StringVar(&port, "port", defaultPort, "")
	fs.StringVar(&port, "p", defaultPort, "")
	fs.StringVar(&dir, "dir", ".", "")
	fs.StringVar(&dir, "d", ".", "")
	fs.StringVar(&relayURL, "relay", defaultRelayURL, "")
	fs.StringVar(&relayURL, "r", defaultRelayURL, "")
	fs.StringVar(&relayToken, "relay-token", defaultRelayToken, "")
	fs.BoolVar(&help, "help", false, "")
	fs.BoolVar(&help, "h", false, "")
	fs.BoolVar(&doctorFlag, "doctor", false, "")

	if err := fs.Parse(flagArgs); err != nil {
		return opts, fmt.Errorf("invalid flags: %w", err)
	}

	if help {
		opts.Action = ActionHelp
		return opts, nil
	}

	if doctorFlag {
		opts.Action = ActionDoctor
		return opts, nil
	}

	opts.Port = port
	opts.Dir = dir
	opts.RelayURL = relayURL
	opts.RelayToken = relayToken

	return opts, nil
}

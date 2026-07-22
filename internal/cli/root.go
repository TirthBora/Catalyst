package cli

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "catalyst",
	Short: "Catalyst is a local development companion.",
	Long: `Catalyst accelerates the local development workflow by
automatically rebuilding and restarting Go applications whenever
source files change.`,
}

func Execute() error {
	return rootCmd.Execute()
}

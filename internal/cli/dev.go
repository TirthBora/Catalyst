package cli

import (
	"github.com/TirthBora/catalyst/internal/dev"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the development server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return dev.Run()
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
}

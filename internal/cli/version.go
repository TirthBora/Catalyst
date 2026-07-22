package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print Catalyst version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Catalyst v0.1.0")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

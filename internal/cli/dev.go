package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the development server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Development mode is not implemented yet.")
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your development environment",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Doctor is not implemented yet.")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

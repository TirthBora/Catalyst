package cli

import (
	"github.com/TirthBora/catalyst/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your development environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return doctor.Run()
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

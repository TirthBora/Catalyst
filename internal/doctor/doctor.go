package doctor

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/TirthBora/catalyst/internal/project"
)

func Run() error {
	fmt.Println("Catalyst Doctor")
	fmt.Println()

	// Go
	if out, err := exec.Command("go", "version").Output(); err == nil {
		fmt.Printf("[✓] Go installed: %s", out)
	} else {
		fmt.Println("[✗] Go is not installed")
	}

	// Git
	if out, err := exec.Command("git", "--version").Output(); err == nil {
		fmt.Printf("[✓] %s", out)
	} else {
		fmt.Println("[✗] Git is not installed")
	}

	// Project
	if _, err := os.Stat("go.mod"); err == nil {
		fmt.Println("[✓] go.mod found")
	} else {
		fmt.Println("[✗] go.mod not found")
	}

	// Entry point
	if proj, err := project.Detect(); err == nil {
		fmt.Printf("[✓] Entry point: %s\n", proj.EntryPoint)
	} else {
		fmt.Printf("[✗] %v\n", err)
	}

	fmt.Println()
	fmt.Println("Doctor completed.")

	return nil
}

package runner

import (
	"os/exec"

	"github.com/TirthBora/catalyst/internal/project"
)

func New(proj *project.Project) *exec.Cmd {
	return exec.Command(
		"go",
		"run",
		proj.EntryPoint,
	)
}

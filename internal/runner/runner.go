package runner

import (
	"os"
	"os/exec"

	"github.com/TirthBora/catalyst/internal/project"
)

func Command(proj *project.Project) *exec.Cmd {
	cmd := exec.Command("go", "run", proj.EntryPoint)

	cmd.Dir = proj.Root
	cmd.Env = os.Environ()

	return cmd
}

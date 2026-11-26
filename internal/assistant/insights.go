package assistant

import (
	"os"
	"path/filepath"

	"catalyst/internal/project"
)

// ProjectInsight represents a proactive recommendation for a project workspace.
type ProjectInsight struct {
	Severity string `json:"severity"` // "info", "warning", "tip"
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Action   string `json:"action,omitempty"`
}

// EvaluateWorkspace inspects the workspace configuration and generates proactive insights.
func EvaluateWorkspace(workDir string, info project.Info) []ProjectInsight {
	var insights []ProjectInsight

	if workDir == "" {
		return insights
	}

	// 1. Check for missing .gitignore in git repository
	gitDir := filepath.Join(workDir, ".git")
	gitignore := filepath.Join(workDir, ".gitignore")
	if exists(gitDir) && !exists(gitignore) {
		insights = append(insights, ProjectInsight{
			Severity: "warning",
			Title:    "Missing .gitignore",
			Detail:   "Git repository detected without a .gitignore file. Build artifacts and credentials may be accidentally committed.",
			Action:   "Create a .gitignore file to exclude node_modules, binaries, and environment secrets.",
		})
	}

	// 2. Check for missing node_modules when package.json is present
	pkgPath := filepath.Join(workDir, "package.json")
	nodeModules := filepath.Join(workDir, "node_modules")
	if exists(pkgPath) && !exists(nodeModules) {
		insights = append(insights, ProjectInsight{
			Severity: "warning",
			Title:    "Dependencies Uninstalled",
			Detail:   "package.json was found but node_modules is missing.",
			Action:   "Run 'npm install' to install project dependencies.",
		})
	}

	// 3. Check for missing dev/start script in Node projects
	if exists(pkgPath) && len(info.Scripts) == 0 {
		insights = append(insights, ProjectInsight{
			Severity: "tip",
			Title:    "No Runnable Scripts Configured",
			Detail:   "No scripts defined in package.json.",
			Action:   "Add a 'dev' or 'start' script to package.json to enable one-click dev runner.",
		})
	}

	// 4. Framework-specific tips
	if info.Type == project.TypeNext {
		insights = append(insights, ProjectInsight{
			Severity: "info",
			Title:    "Next.js Fast Refresh Active",
			Detail:   "Next.js detected. Catalyst reverse proxy will preserve Hot Module Replacement over /proxy/3000/.",
		})
	} else if info.Type == project.TypeVite {
		insights = append(insights, ProjectInsight{
			Severity: "info",
			Title:    "Vite HMR Active",
			Detail:   "Vite detected. Ensure server binds to localhost or use Catalyst reverse proxy on /proxy/5173/.",
		})
	}

	return insights
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

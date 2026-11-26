package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectType identifies the detected runtime environment.
type ProjectType string

const (
	TypeGo      ProjectType = "Go"
	TypeNode    ProjectType = "Node.js"
	TypeNext    ProjectType = "Next.js"
	TypeReact   ProjectType = "React"
	TypeVite    ProjectType = "Vite"
	TypePython  ProjectType = "Python"
	TypeRust    ProjectType = "Rust"
	TypeStatic  ProjectType = "Static Web"
	TypeUnknown ProjectType = "Generic"
)

// Script defines a runnable development task extracted from project configuration.
type Script struct {
	Name    string   `json:"name"`    // e.g. "dev", "start", "run"
	Command string   `json:"command"` // e.g. "npm", "go", "cargo"
	Args    []string `json:"args"`    // e.g. ["run", "dev"]
}

// Info contains metadata about the detected project.
type Info struct {
	Name        string      `json:"name"`
	Type        ProjectType `json:"type"`
	Framework   string      `json:"framework,omitempty"`
	DefaultPort string      `json:"defaultPort,omitempty"`
	RootDir     string      `json:"rootDir"`
	EntryFile   string      `json:"entryFile,omitempty"`
	Scripts     []Script    `json:"scripts"`
}

type packageJSON struct {
	Name            string            `json:"name"`
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// Detect inspects rootDir and identifies the project type, framework, and likely dev port.
func Detect(rootDir string) Info {
	baseName := filepath.Base(rootDir)
	info := Info{
		Name:    baseName,
		Type:    TypeUnknown,
		RootDir: rootDir,
		Scripts: make([]Script, 0),
	}

	// 1. Check for Go (go.mod)
	if exists(filepath.Join(rootDir, "go.mod")) {
		info.Type = TypeGo
		info.Framework = "Standard Go"
		info.DefaultPort = "8080"
		info.Scripts = []Script{
			{Name: "run", Command: "go", Args: []string{"run", "."}},
			{Name: "test", Command: "go", Args: []string{"test", "./..."}},
		}
		return info
	}

	// 2. Check for Node.js / JavaScript / TypeScript (package.json)
	pkgPath := filepath.Join(rootDir, "package.json")
	if exists(pkgPath) {
		info.Type = TypeNode
		info.DefaultPort = "3000"

		if data, err := os.ReadFile(pkgPath); err == nil {
			var pkg packageJSON
			if err := json.Unmarshal(data, &pkg); err == nil {
				if pkg.Name != "" {
					info.Name = pkg.Name
				}

				allDeps := make(map[string]bool)
				for k := range pkg.Dependencies {
					allDeps[k] = true
				}
				for k := range pkg.DevDependencies {
					allDeps[k] = true
				}

				// Extract package.json runnable scripts
				for scriptName := range pkg.Scripts {
					info.Scripts = append(info.Scripts, Script{
						Name:    scriptName,
						Command: "npm",
						Args:    []string{"run", scriptName},
					})
				}

				switch {
				case allDeps["next"]:
					info.Type = TypeNext
					info.Framework = "Next.js"
					info.DefaultPort = "3000"
				case allDeps["vite"]:
					info.Type = TypeVite
					info.Framework = "Vite"
					info.DefaultPort = "5173"
				case allDeps["react"]:
					info.Type = TypeReact
					info.Framework = "React"
					info.DefaultPort = "3000"
				case allDeps["vue"]:
					info.Framework = "Vue"
					info.DefaultPort = "5173"
				case allDeps["svelte"]:
					info.Framework = "Svelte"
					info.DefaultPort = "5173"
				case allDeps["express"]:
					info.Framework = "Express"
					info.DefaultPort = "3000"
				}
			}
		}
		return info
	}

	// 3. Check for Python (pyproject.toml, requirements.txt, Pipfile)
	if exists(filepath.Join(rootDir, "pyproject.toml")) ||
		exists(filepath.Join(rootDir, "requirements.txt")) ||
		exists(filepath.Join(rootDir, "Pipfile")) {
		info.Type = TypePython
		info.Framework = "Python"
		info.DefaultPort = "8000"

		if exists(filepath.Join(rootDir, "app.py")) {
			info.Scripts = append(info.Scripts, Script{Name: "run", Command: "python", Args: []string{"app.py"}})
		} else if exists(filepath.Join(rootDir, "main.py")) {
			info.Scripts = append(info.Scripts, Script{Name: "run", Command: "python", Args: []string{"main.py"}})
		}

		if data, err := os.ReadFile(filepath.Join(rootDir, "requirements.txt")); err == nil {
			content := strings.ToLower(string(data))
			if strings.Contains(content, "flask") {
				info.Framework = "Flask"
				info.DefaultPort = "5000"
				info.Scripts = append(info.Scripts, Script{Name: "flask", Command: "flask", Args: []string{"run"}})
			} else if strings.Contains(content, "django") {
				info.Framework = "Django"
				info.DefaultPort = "8000"
				info.Scripts = append(info.Scripts, Script{Name: "django", Command: "python", Args: []string{"manage.py", "runserver"}})
			} else if strings.Contains(content, "fastapi") {
				info.Framework = "FastAPI"
				info.DefaultPort = "8000"
				info.Scripts = append(info.Scripts, Script{Name: "fastapi", Command: "uvicorn", Args: []string{"main:app", "--reload"}})
			}
		}
		return info
	}

	// 4. Check for Rust (Cargo.toml)
	if exists(filepath.Join(rootDir, "Cargo.toml")) {
		info.Type = TypeRust
		info.Framework = "Cargo"
		info.DefaultPort = "8080"
		info.Scripts = []Script{
			{Name: "run", Command: "cargo", Args: []string{"run"}},
			{Name: "test", Command: "cargo", Args: []string{"test"}},
		}
		return info
	}

	// 5. Check for Static Web (index.html)
	if exists(filepath.Join(rootDir, "index.html")) {
		info.Type = TypeStatic
		info.Framework = "HTML5 / CSS / JS"
		info.EntryFile = "index.html"
		return info
	}

	return info
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

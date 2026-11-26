package analyzer

import (
	"path/filepath"
	"strings"
)

// ImpactLevel represents the estimated severity and ripple effect of a file change.
type ImpactLevel string

const (
	ImpactCritical   ImpactLevel = "CRITICAL"   // Entry points, core configs, root schemas
	ImpactHigh       ImpactLevel = "HIGH"       // Global styles, state management, routes
	ImpactMedium     ImpactLevel = "MEDIUM"     // UI components, controllers, views
	ImpactLow        ImpactLevel = "LOW"        // Utilities, helpers, local assets
	ImpactNegligible ImpactLevel = "NEGLIGIBLE" // Tests, fixtures, markdown docs
)

// ImpactResult details the evaluated importance and scope of a modified file.
type ImpactResult struct {
	Level       ImpactLevel `json:"level"`
	Score       int         `json:"score"`
	Affected    string      `json:"affected"`
	Strategy    string      `json:"strategy"` // "hard_refresh", "hot_reload", "log_only"
	Description string      `json:"description"`
}

// EvaluateChange computes the blast radius, affected scope, and reload strategy for a changed file.
func EvaluateChange(relPath string) ImpactResult {
	base := strings.ToLower(filepath.Base(relPath))
	ext := strings.ToLower(filepath.Ext(relPath))
	clean := strings.ToLower(filepath.ToSlash(relPath))

	// 1. Critical Impact: Project roots & configurations
	if base == "package.json" || base == "go.mod" || base == "tsconfig.json" || base == "cargo.toml" || base == ".env" {
		return ImpactResult{
			Level:       ImpactCritical,
			Score:       95,
			Affected:    "Dependencies & Core Config",
			Strategy:    "hard_refresh",
			Description: "Root configuration modified; rebuild or restart may be required.",
		}
	}
	if base == "index.html" || base == "main.go" || base == "app.tsx" || base == "main.tsx" || base == "app.jsx" || base == "main.jsx" {
		return ImpactResult{
			Level:       ImpactCritical,
			Score:       90,
			Affected:    "Root Entry Point & DOM",
			Strategy:    "hard_refresh",
			Description: "Primary application bootstrap file changed.",
		}
	}

	// 2. High Impact: Routing, global state, stylesheets
	if strings.Contains(clean, "router") || strings.Contains(clean, "routes") || strings.Contains(clean, "store") || strings.Contains(clean, "context") {
		return ImpactResult{
			Level:       ImpactHigh,
			Score:       80,
			Affected:    "Routing & Global State",
			Strategy:    "hot_reload",
			Description: "Application state or navigation architecture updated.",
		}
	}
	if ext == ".css" || ext == ".scss" || ext == ".less" || strings.Contains(base, "tailwind") {
		return ImpactResult{
			Level:       ImpactHigh,
			Score:       75,
			Affected:    "Global Stylesheet & Design",
			Strategy:    "hot_reload",
			Description: "Visual styling updated across the application.",
		}
	}

	// 3. Negligible Impact: Non-runtime files (markdown, tests, specs)
	if ext == ".md" || strings.Contains(clean, "test") || strings.Contains(clean, "spec") || strings.HasPrefix(base, ".") {
		return ImpactResult{
			Level:       ImpactNegligible,
			Score:       15,
			Affected:    "Documentation & Test Suite",
			Strategy:    "log_only",
			Description: "Non-runtime file modified.",
		}
	}

	// 4. Medium Impact: UI Components & Views
	if ext == ".tsx" || ext == ".jsx" || ext == ".vue" || ext == ".svelte" || strings.Contains(clean, "components") {
		return ImpactResult{
			Level:       ImpactMedium,
			Score:       60,
			Affected:    "UI Components & Views",
			Strategy:    "hot_reload",
			Description: "Interactive component modified.",
		}
	}

	// 5. Low Impact: General source files
	return ImpactResult{
		Level:       ImpactLow,
		Score:       40,
		Affected:    "Application Logic",
		Strategy:    "hot_reload",
		Description: "General source file modified.",
	}
}

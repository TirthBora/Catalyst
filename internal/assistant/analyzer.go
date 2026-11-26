package assistant

import (
	"regexp"
	"strings"
	"time"
)

// Category classifies the identified development issue.
type Category string

const (
	CategoryModuleMissing Category = "missing_dependency"
	CategorySyntaxError   Category = "syntax_error"
	CategoryTypeMismatch  Category = "type_error"
	CategoryPortConflict  Category = "port_conflict"
	CategoryPanic         Category = "runtime_panic"
	CategoryGeneric       Category = "general_error"
)

// FixSuggestion represents an actionable step to resolve an error.
type FixSuggestion struct {
	Title   string `json:"title"`
	Command string `json:"command,omitempty"`
	Action  string `json:"action,omitempty"`
}

// ErrorAnalysis contains structured insights about a process or build failure.
type ErrorAnalysis struct {
	Matched     bool          `json:"matched"`
	Category    Category      `json:"category"`
	Summary     string        `json:"summary"`
	File        string        `json:"file,omitempty"`
	Line        int           `json:"line,omitempty"`
	Suggestions []FixSuggestion `json:"suggestions"`
	Timestamp   time.Time     `json:"timestamp"`
}

var (
	// Node/Webpack: Cannot find module 'xyz'
	nodeMissingModRe = regexp.MustCompile(`Cannot find module ['"]([^'"]+)['"]`)
	// Node: EADDRINUSE :::3000
	portConflictRe = regexp.MustCompile(`(?:EADDRINUSE|address already in use|bind: address already in use).*?(\d{2,5})`)
	// Go: undefined: xyz
	goUndefinedRe = regexp.MustCompile(`(?:\.\/)?([^\s:]+):(\d+):(?:\d+)?: undefined: (\w+)`)
	// Python: ModuleNotFoundError: No module named 'xyz'
	pythonMissingModRe = regexp.MustCompile(`ModuleNotFoundError: No module named ['"]([^'"]+)['"]`)
	// Python: Traceback (most recent call last): File "xyz.py", line 42
	pythonTracebackRe = regexp.MustCompile(`File ["']([^"']+)["'], line (\d+)`)
	// TypeScript: TS2304: Cannot find name 'xyz'
	tsMissingNameRe = regexp.MustCompile(`(?:TS\d+:)?\s*Cannot find name ['"]([^'"]+)['"]`)
)

// AnalyzeLogs inspects console lines and returns structured error diagnosis.
func AnalyzeLogs(logs []string) ErrorAnalysis {
	combined := strings.Join(logs, "\n")

	// 1. Port conflicts
	if matches := portConflictRe.FindStringSubmatch(combined); len(matches) > 1 {
		port := matches[1]
		return ErrorAnalysis{
			Matched:  true,
			Category: CategoryPortConflict,
			Summary:  "Port " + port + " is already in use by another process.",
			Suggestions: []FixSuggestion{
				{
					Title:   "Kill process using port " + port,
					Command: "catalyst stop -p " + port,
				},
				{
					Title:  "Change development port in settings",
					Action: "Open ⚙️ Settings and specify a different port",
				},
			},
			Timestamp: time.Now().UTC(),
		}
	}

	// 2. Node.js missing dependencies
	if matches := nodeMissingModRe.FindStringSubmatch(combined); len(matches) > 1 {
		mod := matches[1]
		return ErrorAnalysis{
			Matched:  true,
			Category: CategoryModuleMissing,
			Summary:  "Missing Node.js dependency: " + mod,
			Suggestions: []FixSuggestion{
				{
					Title:   "Install missing module via npm",
					Command: "npm install " + mod,
				},
				{
					Title:   "Reinstall all workspace dependencies",
					Command: "npm install",
				},
			},
			Timestamp: time.Now().UTC(),
		}
	}

	// 3. Python missing dependencies
	if matches := pythonMissingModRe.FindStringSubmatch(combined); len(matches) > 1 {
		mod := matches[1]
		return ErrorAnalysis{
			Matched:  true,
			Category: CategoryModuleMissing,
			Summary:  "Missing Python module: " + mod,
			Suggestions: []FixSuggestion{
				{
					Title:   "Install module via pip",
					Command: "pip install " + mod,
				},
			},
			Timestamp: time.Now().UTC(),
		}
	}

	// 4. Go undefined identifier
	if matches := goUndefinedRe.FindStringSubmatch(combined); len(matches) > 3 {
		file := matches[1]
		ident := matches[3]
		return ErrorAnalysis{
			Matched:  true,
			Category: CategorySyntaxError,
			Summary:  "Undefined symbol '" + ident + "' in " + file,
			File:     file,
			Suggestions: []FixSuggestion{
				{
					Title:  "Check import declaration or spelling",
					Action: "Verify that package or variable '" + ident + "' is imported and exported",
				},
			},
			Timestamp: time.Now().UTC(),
		}
	}

	// 5. TypeScript missing identifier
	if matches := tsMissingNameRe.FindStringSubmatch(combined); len(matches) > 1 {
		name := matches[1]
		return ErrorAnalysis{
			Matched:  true,
			Category: CategoryTypeMismatch,
			Summary:  "TypeScript cannot find name '" + name + "'",
			Suggestions: []FixSuggestion{
				{
					Title:  "Check import statement",
					Action: "Ensure '" + name + "' is imported from the correct module",
				},
			},
			Timestamp: time.Now().UTC(),
		}
	}

	// 6. Generic panic or traceback
	if strings.Contains(combined, "panic:") || strings.Contains(combined, "Traceback") {
		summary := "Runtime panic or unhandled exception detected."
		analysis := ErrorAnalysis{
			Matched:   true,
			Category:  CategoryPanic,
			Summary:   summary,
			Timestamp: time.Now().UTC(),
		}
		if pyMatches := pythonTracebackRe.FindStringSubmatch(combined); len(pyMatches) > 1 {
			analysis.File = pyMatches[1]
		}
		return analysis
	}

	return ErrorAnalysis{
		Matched:   false,
		Category:  CategoryGeneric,
		Summary:   "No specific error pattern identified",
		Timestamp: time.Now().UTC(),
	}
}

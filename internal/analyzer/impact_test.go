package analyzer_test

import (
	"testing"

	"catalyst/internal/analyzer"
)

func TestEvaluateChange(t *testing.T) {
	// 1. Critical
	pkgRes := analyzer.EvaluateChange("package.json")
	if pkgRes.Level != analyzer.ImpactCritical || pkgRes.Strategy != "hard_refresh" {
		t.Errorf("expected Critical impact for package.json, got %+v", pkgRes)
	}

	indexRes := analyzer.EvaluateChange("public/index.html")
	if indexRes.Level != analyzer.ImpactCritical {
		t.Errorf("expected Critical impact for index.html, got %+v", indexRes)
	}

	// 2. High
	cssRes := analyzer.EvaluateChange("styles/globals.css")
	if cssRes.Level != analyzer.ImpactHigh || cssRes.Strategy != "hot_reload" {
		t.Errorf("expected High impact for css, got %+v", cssRes)
	}

	routerRes := analyzer.EvaluateChange("src/routes/user.go")
	if routerRes.Level != analyzer.ImpactHigh {
		t.Errorf("expected High impact for router file, got %+v", routerRes)
	}

	// 3. Medium
	compRes := analyzer.EvaluateChange("src/components/Button.tsx")
	if compRes.Level != analyzer.ImpactMedium {
		t.Errorf("expected Medium impact for component, got %+v", compRes)
	}

	// 4. Negligible
	docRes := analyzer.EvaluateChange("docs/README.md")
	if docRes.Level != analyzer.ImpactNegligible || docRes.Strategy != "log_only" {
		t.Errorf("expected Negligible impact for docs, got %+v", docRes)
	}
}

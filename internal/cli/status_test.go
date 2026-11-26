package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"catalyst/internal/cli"
)

func TestCheckStatusNotRunning(t *testing.T) {
	res := cli.CheckStatus("59991")
	if res.Running {
		t.Error("expected Running to be false on closed port")
	}

	output := cli.FormatStatus(res)
	if !strings.Contains(output, "Catalyst is not running") {
		t.Errorf("expected 'Catalyst is not running' message, got: %s", output)
	}
}

func TestCheckStatusRunning(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/workspace", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":      "MyProject",
			"rootDir":   "/projects/myproject",
			"framework": "React / Vite",
			"type":      "Node.js",
		})
	})

	mux.HandleFunc("/api/pairing", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"devicesCount": 2,
		})
	})

	mux.HandleFunc("/api/system", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"uptimeSec": 120,
			"memory": map[string]interface{}{
				"usedPct": 42.5,
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	res := cli.CheckStatus(u.Port())
	if !res.Running {
		t.Fatal("expected Running to be true")
	}

	if res.Workspace != "MyProject" {
		t.Errorf("expected workspace MyProject, got %s", res.Workspace)
	}
	if res.Framework != "React / Vite" {
		t.Errorf("expected framework React / Vite, got %s", res.Framework)
	}
	if res.DevicesCount != 2 {
		t.Errorf("expected 2 devices, got %d", res.DevicesCount)
	}
	if res.RAMUsedPct != 42.5 {
		t.Errorf("expected 42.5%% RAM, got %f", res.RAMUsedPct)
	}

	formatted := cli.FormatStatus(res)
	if !strings.Contains(formatted, "ACTIVE") || !strings.Contains(formatted, "MyProject") {
		t.Errorf("unexpected formatted status output:\n%s", formatted)
	}
}

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StatusResult contains the aggregated health and operational details of a running Catalyst daemon.
type StatusResult struct {
	Running      bool   `json:"running"`
	Port         string `json:"port"`
	Workspace    string `json:"workspace,omitempty"`
	RootDir      string `json:"rootDir,omitempty"`
	Framework    string `json:"framework,omitempty"`
	ProjectType  string `json:"projectType,omitempty"`
	DevicesCount int    `json:"devicesCount,omitempty"`
	RAMUsedPct   float64`json:"ramUsedPct,omitempty"`
	UptimeSec    int64  `json:"uptimeSec,omitempty"`
}

// CheckStatus queries a local Catalyst instance on the specified port.
func CheckStatus(port string) StatusResult {
	if port == "" {
		port = "9999"
	}

	client := &http.Client{Timeout: 800 * time.Millisecond}

	// 1. Query Workspace API
	wsResp, err := client.Get(fmt.Sprintf("http://localhost:%s/api/workspace", port))
	if err != nil {
		return StatusResult{Running: false, Port: port}
	}
	defer wsResp.Body.Close()

	if wsResp.StatusCode != http.StatusOK {
		return StatusResult{Running: false, Port: port}
	}

	var wsData struct {
		Name      string `json:"name"`
		RootDir   string `json:"rootDir"`
		Framework string `json:"framework"`
		Type      string `json:"type"`
	}
	bodyBytes, _ := io.ReadAll(wsResp.Body)
	_ = json.Unmarshal(bodyBytes, &wsData)

	res := StatusResult{
		Running:     true,
		Port:        port,
		Workspace:   wsData.Name,
		RootDir:     wsData.RootDir,
		Framework:   wsData.Framework,
		ProjectType: wsData.Type,
	}

	// 2. Query Pairing API for connected device count
	if pairResp, err := client.Get(fmt.Sprintf("http://localhost:%s/api/pairing", port)); err == nil {
		defer pairResp.Body.Close()
		var pairData struct {
			DevicesCount int `json:"devicesCount"`
		}
		if pBytes, err := io.ReadAll(pairResp.Body); err == nil {
			_ = json.Unmarshal(pBytes, &pairData)
			res.DevicesCount = pairData.DevicesCount
		}
	}

	// 3. Query System API for resource utilization
	if sysResp, err := client.Get(fmt.Sprintf("http://localhost:%s/api/system", port)); err == nil {
		defer sysResp.Body.Close()
		var sysData struct {
			UptimeSec int64 `json:"uptimeSec"`
			Memory    struct {
				UsedPct float64 `json:"usedPct"`
			} `json:"memory"`
		}
		if sBytes, err := io.ReadAll(sysResp.Body); err == nil {
			_ = json.Unmarshal(sBytes, &sysData)
			res.UptimeSec = sysData.UptimeSec
			res.RAMUsedPct = sysData.Memory.UsedPct
		}
	}

	return res
}

// FormatStatus formats the StatusResult for CLI printing.
func FormatStatus(res StatusResult) string {
	if !res.Running {
		return fmt.Sprintf("Catalyst is not running on port %s. (Run 'catalyst dev' to start)", res.Port)
	}

	fw := res.Framework
	if fw == "" {
		fw = res.ProjectType
	}

	return fmt.Sprintf(`Catalyst Status: ACTIVE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Workspace : %s (%s)
  Project   : %s
  Port      : %s
  Devices   : %d connected
  Host RAM  : %.1f%% used
  Uptime    : %ds
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`,
		res.Workspace, res.RootDir, fw, res.Port, res.DevicesCount, res.RAMUsedPct, res.UptimeSec)
}

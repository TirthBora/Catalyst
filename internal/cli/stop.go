package cli

import (
	"fmt"
	"net/http"
	"time"
)

// StopDaemon sends a graceful shutdown request to the local Catalyst daemon on the specified port.
func StopDaemon(port string) error {
	if port == "" {
		port = "9999"
	}

	client := &http.Client{Timeout: 1 * time.Second}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%s/api/shutdown", port), nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("catalyst is not currently running on port %s", port)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("unexpected status from shutdown endpoint: %d", resp.StatusCode)
}

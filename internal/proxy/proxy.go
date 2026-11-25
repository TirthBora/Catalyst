package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

// ReverseProxy forwards requests to local development servers on behalf of secondary devices.
type ReverseProxy struct{}

func NewReverseProxy() *ReverseProxy {
	return &ReverseProxy{}
}

// ServeHTTP handles requests directed to /proxy/{port}/[path...]
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Trim the /proxy/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/proxy")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		http.Error(w, "Proxy target port required. Usage: /proxy/{port}/", http.StatusBadRequest)
		return
	}

	parts := strings.SplitN(path, "/", 2)
	portStr := parts[0]

	// Validate port
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		http.Error(w, fmt.Sprintf("Invalid port %q", portStr), http.StatusBadRequest)
		return
	}

	targetPath := "/"
	if len(parts) > 1 {
		targetPath = "/" + parts[1]
	}

	targetURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		http.Error(w, "Failed to parse target URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = targetPath
		req.URL.RawQuery = r.URL.RawQuery
		// Rewrite Host header so Vite / Next.js accept the request
		req.Host = fmt.Sprintf("localhost:%d", port)
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", "http")
	}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		rw.WriteHeader(http.StatusBadGateway)
		_, _ = rw.Write([]byte(fmt.Sprintf(
			"Catalyst Proxy Error: Unable to reach development server on port %d.\nEnsure your dev server is running.\nDetails: %v\n",
			port, err,
		)))
	}

	proxy.ServeHTTP(w, r)
}

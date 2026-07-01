// Command healthprobe is a tiny, dependency-free readiness probe for the
// distroless cloudflared image.
//
// Why this exists:
//   The official cloudflare/cloudflared image is fully distroless — its only
//   binary is /usr/local/bin/cloudflared (no shell, no curl, no wget). A Docker
//   HEALTHCHECK runs *inside* the container, so there is nothing there to probe
//   cloudflared's own /ready endpoint with. This ~2 MB static binary is the
//   single tool we add to close that gap while keeping the base distroless.
//
// What it checks:
//   cloudflared exposes /ready on its metrics server (enabled via
//   `--metrics <addr>`). It returns HTTP 200 once the tunnel has at least one
//   established edge connection, and a 5xx / connection-refused otherwise.
//   Exit 0 on 200 → container "healthy"; any other outcome → exit 1.
//
// The target URL defaults to the metrics address baked into the image CMD and
// can be overridden via the HEALTHPROBE_URL environment variable.
package main

import (
	"net/http"
	"os"
	"time"
)

func main() {
	url := os.Getenv("HEALTHPROBE_URL")
	if url == "" {
		url = "http://127.0.0.1:2000/ready"
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
}

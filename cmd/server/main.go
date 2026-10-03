// Local development server. In production the same router runs inside the
// Vercel function in api/index.go.
package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"standort-check/backend"
)

func main() {
	loadDotEnv(".env")

	app, err := backend.NewApp(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Bind to loopback only: the dev server must not be reachable from the
	// network (and Windows does not show a firewall prompt for it).
	addr := "127.0.0.1:8080"
	log.Printf("API listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, app))
}

// loadDotEnv is a tiny .env reader for local development so we don't need an
// extra dependency. Existing environment variables take precedence.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}

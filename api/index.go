// Package handler is the single Vercel serverless function. vercel.json
// rewrites every /api/* request here; chi then routes on the original path.
package handler

import (
	"context"
	"log"
	"net/http"
	"sync"

	"standort-check/backend"
)

var (
	once    sync.Once
	app     http.Handler
	initErr error
)

// Handler is the entry point Vercel invokes. The app (and its DB pool) is
// created once per warm instance and reused across requests.
func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		app, initErr = backend.NewApp(context.Background())
		if initErr != nil {
			log.Printf("init failed: %v", initErr)
		}
	})
	if initErr != nil {
		http.Error(w, `{"error":"Server ist nicht korrekt konfiguriert"}`, http.StatusInternalServerError)
		return
	}
	app.ServeHTTP(w, r)
}

// Package server provides a simple HTTP server.
package server

import (
	"errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"nginx-reaper/internal/log"
	"time"
)

// CreateServer creates an HTTP server configured with the specified address, metrics, and default timeouts.
func CreateServer(addr string, metrics ...prometheus.Collector) *http.Server {
	// Handler for configuration requests.
	var mux http.ServeMux
	mux.HandleFunc(configPath, configHandler)

	// Handler for metrics requests.
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics...)
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// Default handler for all other requests.
	mux.HandleFunc("/", defaultHandler)

	return &http.Server{
		Addr:              addr,
		Handler:           &mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// StartServer starts a specified HTTP server to listen and respond to incoming requests.
func StartServer(server *http.Server) {
	// ListenAndServe always returns a non-nil error.
	log.Infof("Server listening on %q", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Infof("Server stopped listening on %q, %v", server.Addr, err)
		} else {
			log.Panicf("Server failed: %v", err)
		}
	}
}

// defaultHandler responds with "ready" to all incoming requests and logs any errors that occur.
// E.g. "GET /healthz".
func defaultHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Request %v", r)
	if _, err := w.Write([]byte("ready")); err != nil {
		log.Errorf("Request %v failed: %v", r, err)
	}
}

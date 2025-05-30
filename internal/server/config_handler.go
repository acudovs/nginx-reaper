package server

import (
	"net/http"
	"nginx-reaper/internal/log"
)

const (
	configPath  = "/config"
	keyLogLevel = "log-level"
)

// configHandler responds to requests to the configPath endpoint.
// E.g. "PUT /config?log-level=debug".
func configHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Request %v", r)
	if r.URL.Path != configPath {
		w.WriteHeader(http.StatusNotFound)
	} else if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
	} else if err := setLogLevel(r); err != nil {
		log.Errorf("Request %v failed: %v", r, err)
		w.WriteHeader(http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// setLogLevel sets the log level to the value specified in the keyLogLevel query parameter.
func setLogLevel(r *http.Request) error {
	level := r.URL.Query().Get(keyLogLevel)
	l, err := log.ParseLevel(level)
	if err != nil {
		return err
	}
	log.SetLevel(l)
	return nil
}

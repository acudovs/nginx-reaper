package reaper

import (
	"fmt"
	"nginx-reaper/internal/log"
	"nginx-reaper/internal/procps"
	"nginx-reaper/internal/ticker"
	"os"
	"os/signal"
	"time"
)

type ShutdownHandler struct {
	shutdownInterval time.Duration
	shutdownTimeout  time.Duration
}

// Interval at which the ShutdownHandler checks whether a Nginx master process is still running.
func (s *ShutdownHandler) Interval() time.Duration {
	return s.shutdownInterval
}

// String returns a string representation of the ShutdownHandler.
func (s *ShutdownHandler) String() string {
	return fmt.Sprintf("Nginx Reaper shutdown handler with interval %v and timeout %v",
		s.shutdownInterval, s.shutdownTimeout)
}

// Run checks whether a Nginx master process is still running.
// Decreases shutdownTimeout by shutdownInterval on each call.
func (s *ShutdownHandler) Run() bool {
	s.shutdownTimeout -= s.shutdownInterval
	return s.shutdownTimeout >= s.shutdownInterval && nginxMasterRunning()
}

// nginxMasterRunning returns a bool indicating whether a Nginx master process is still running.
func nginxMasterRunning() bool {
	masters := procpsPgrep(OptionNginxMaster)
	if len(masters) == 0 {
		return false
	}
	for _, master := range masters {
		log.Infof("Nginx master process is still running %v", procps.NewProcessInfo(master))
	}
	return true
}

// WaitShutdown listens for the specified signal and starts the graceful shutdown process when received.
func WaitShutdown(shutdownInterval time.Duration, shutdownTimeout time.Duration, sig os.Signal) {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, sig)

	// Block until a signal is received.
	received := <-channel
	log.Infof("Nginx Reaper %v", received)

	if nginxMasterRunning() {
		handler := &ShutdownHandler{
			shutdownInterval: shutdownInterval,
			shutdownTimeout:  shutdownTimeout,
		}
		ticker.Start(handler)
	}
}

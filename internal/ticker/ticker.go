// Package ticker provides a convenient way to execute a Job at a regular interval.
package ticker

import (
	"nginx-reaper/internal/log"
	"time"
)

// Job is an interface that defines a Job to be executed at a regular interval.
type Job interface {
	Interval() time.Duration // Interval returns the interval at which the Job runs.
	String() string          // String returns a string representation of the Job.
	Run() bool               // Run executes the Job logic while true.
}

// Start starts a ticker that executes the provided Job at a regular interval.
func Start(job Job) {
	ticker := time.NewTicker(job.Interval())
	defer ticker.Stop()

	log.Infof("Scheduled %v", job)
	for range ticker.C {
		log.Infof("Executing %v", job)
		if !job.Run() {
			break
		}
	}
}

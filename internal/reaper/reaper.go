// Package reaper provides a Reaper to terminate shutting down Nginx worker processes.
package reaper

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"nginx-reaper/internal/log"
	"nginx-reaper/internal/procps"
	"nginx-reaper/internal/procps/option"
	"time"
)

const (
	NginxMaster         = "nginx: master process"
	NginxWorker         = "nginx: worker process"
	NginxWorkerShutdown = "nginx: worker process is shutting down"

	LabelActive     = "active"
	LabelShutdown   = "shutdown"
	LabelError      = "error"
	LabelTerminated = "terminated"
)

var (
	OptionNginxMaster         = option.Cmdline(NginxMaster)
	OptionNginxWorker         = option.Cmdline(NginxWorker)
	OptionNginxWorkerShutdown = option.Cmdline(NginxWorkerShutdown)

	procpsFilter        = procps.Filter
	procpsPgrep         = procps.Pgrep
	procpsTerminate     = procps.Terminate
	procpsNewMemoryInfo = procps.NewMemoryInfo
)

type Reaper struct {
	interval               time.Duration
	maxShutdownWorkers     int
	availableMemoryPercent int

	// Metrics
	collectorRunning  *prometheus.GaugeVec
	collectorShutdown *prometheus.CounterVec
}

// NewReaper creates a new Reaper instance with the specified configuration parameters.
func NewReaper(interval time.Duration, maxShutdownWorkers int, availableMemoryPercent int) *Reaper {
	if interval <= 0 {
		log.Panicf("Non-positive interval %v", interval)
	}
	if maxShutdownWorkers <= 0 {
		log.Panicf("Non-positive maxShutdownWorkers %v", maxShutdownWorkers)
	}
	if availableMemoryPercent < 0 || availableMemoryPercent > 100 {
		log.Panicf("Invalid availableMemoryPercent %v", availableMemoryPercent)
	}

	nginxReaper := &Reaper{
		interval:               interval,
		maxShutdownWorkers:     maxShutdownWorkers,
		availableMemoryPercent: availableMemoryPercent,

		collectorRunning: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nginx_workers_running_current",
				Help: "Current number of running Nginx workers by status",
			},
			[]string{"status"},
		),

		collectorShutdown: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nginx_workers_shutdown_total",
				Help: "Total number of shutdown Nginx workers by status",
			},
			[]string{"status"},
		),
	}

	// Initialize Prometheus metrics to zero values.
	nginxReaper.collectorRunning.WithLabelValues(LabelActive).Add(0)
	nginxReaper.collectorRunning.WithLabelValues(LabelShutdown).Add(0)
	nginxReaper.collectorShutdown.WithLabelValues(LabelError).Add(0)
	nginxReaper.collectorShutdown.WithLabelValues(LabelTerminated).Add(0)

	return nginxReaper
}

// Interval returns the interval at which the Reaper runs.
func (r *Reaper) Interval() time.Duration {
	return r.interval
}

// String returns a string representation of the Reaper.
func (r *Reaper) String() string {
	return fmt.Sprintf(
		"Nginx Reaper with configuration: interval %v, max workers to keep %v, target available memory %v%%",
		r.interval, r.maxShutdownWorkers, r.availableMemoryPercent,
	)
}

// Metrics returns a slice of Prometheus collectors managed by the Reaper.
func (r *Reaper) Metrics() []prometheus.Collector {
	return []prometheus.Collector{r.collectorRunning, r.collectorShutdown}
}

// Run executes the Reaper logic.
func (r *Reaper) Run() bool {
	for _, master := range procpsPgrep(OptionNginxMaster) {

		workers := procpsPgrep(OptionNginxWorker, option.Parent(master.Pid))
		workersShutdown := procpsFilter(workers, OptionNginxWorkerShutdown)

		r.collectorRunning.WithLabelValues(LabelActive).Set(float64(len(workers) - len(workersShutdown)))
		r.collectorRunning.WithLabelValues(LabelShutdown).Set(float64(len(workersShutdown)))

		// Maybe terminate workers.
		for i, l := 0, len(workersShutdown); i < l && r.shouldTerminate(int(master.Pid), l-i); i++ {
			if i == 0 {
				// Sort workers by creation time once, to terminate the oldest first.
				procps.SortByCreateTime(workersShutdown)
			}
			worker := workersShutdown[i]
			log.Warningf("Terminating nginx worker process %v", procps.NewProcessInfo(worker))
			err := procpsTerminate(worker)
			if err == nil {
				r.collectorShutdown.WithLabelValues(LabelTerminated).Inc()
				time.Sleep(1 * time.Second)
			} else {
				r.collectorShutdown.WithLabelValues(LabelError).Inc()
				log.Errorf("Failed to terminate nginx worker process %v: %v", worker.Pid, err)
			}
		}
	}
	return true
}

// shouldTerminate returns a bool indicating whether Nginx workers should be terminated.
func (r *Reaper) shouldTerminate(pid int, workers int) bool {
	// Check the number of workers.
	if workers > r.maxShutdownWorkers {
		log.Warningf("Number of nginx workers shutting down %d exceeds limit %d", workers, r.maxShutdownWorkers)
		return true
	}
	log.Debugf("Number of nginx workers shutting down %d within limit %d", workers, r.maxShutdownWorkers)

	// Check available memory.
	m := procpsNewMemoryInfo(pid)
	percent := m.AvailableMemoryPercent()
	if percent < r.availableMemoryPercent {
		log.Warningf("Available memory %d/%d bytes is %d%% and less than %d%% limit",
			m.Available, m.Total, percent, r.availableMemoryPercent)
		return true
	}
	log.Debugf("Available memory %d/%d bytes is %d%% and within %d%% limit",
		m.Available, m.Total, percent, r.availableMemoryPercent)

	return false
}

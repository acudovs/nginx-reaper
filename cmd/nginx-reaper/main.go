// Package main contains the entry point for the nginx-reaper application.
package main

import (
	"nginx-reaper/internal/env"
	"nginx-reaper/internal/log"
	"nginx-reaper/internal/reaper"
	"nginx-reaper/internal/server"
	"nginx-reaper/internal/ticker"
	"syscall"
)

// Supported environment variables
const (
	envLogLevel               = "LOG_LEVEL"
	envReaperInterval         = "REAPER_INTERVAL"
	envMaxShutdownWorkers     = "MAX_SHUTDOWN_WORKERS"
	envAvailableMemoryPercent = "AVAILABLE_MEMORY_PERCENT"
	envServerAddr             = "SERVER_ADDR"
	envShutdownInterval       = "SHUTDOWN_INTERVAL"
	envShutdownTimeout        = "SHUTDOWN_TIMEOUT"
)

// Get environment variables or default values.
var (
	logLevel               = env.GetLogLevel(envLogLevel, "INFO")
	reaperInterval         = env.GetDuration(envReaperInterval, "30s")
	maxShutdownWorkers     = env.GetInt(envMaxShutdownWorkers, "255")
	availableMemoryPercent = env.GetInt(envAvailableMemoryPercent, "0")
	serverAddr             = env.GetString(envServerAddr, ":11254")
	shutdownInterval       = env.GetDuration(envShutdownInterval, "10s")
	shutdownTimeout        = env.GetDuration(envShutdownTimeout, "5m")
)

func main() {
	// Set the log level.
	log.SetLevel(logLevel)

	// Start the Reaper as a goroutine at a regular interval.
	nginxReaper := reaper.NewReaper(reaperInterval, maxShutdownWorkers, availableMemoryPercent)
	go ticker.Start(nginxReaper)

	// Start the HTTP Server as a goroutine.
	httpServer := server.CreateServer(serverAddr, nginxReaper.Metrics()...)
	go server.StartServer(httpServer)

	// Wait for SIGTERM for a graceful shutdown.
	reaper.WaitShutdown(shutdownInterval, shutdownTimeout, syscall.SIGTERM)
}

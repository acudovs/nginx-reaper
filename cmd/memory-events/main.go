// Package main contains the entry point for the memory-events application.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/cgroups/v3/cgroup2"
	"golang.org/x/sys/unix"
	"nginx-reaper/internal/log"
	"os"
	"os/signal"
	"path"
	"syscall"
)

const SysFsCgroup = "/sys/fs/cgroup"

type Event = cgroup2.Event

func main() {
	pid := flag.Int("pid", 1, "cgroup PID to monitor")
	flag.Parse()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	if cgroups.Mode() == cgroups.Unified {
		processV2Events(*pid, sigCh)
	} else {
		processV1Events(*pid, sigCh)
	}
}

func processV1Events(pid int, sigCh <-chan os.Signal) {
	var cgroupPath cgroup1.Path

	// Check if the full cgroup v1 path exists, otherwise try the root path (cgroup namespace).
	pidPath, err := cgroup1.PidPath(pid)(cgroup1.Memory)
	if err != nil {
		panic(err)
	}
	checkPath := path.Join(SysFsCgroup, string(cgroup1.Memory), pidPath)
	if _, err = os.Stat(checkPath); err == nil {
		cgroupPath = cgroup1.PidPath(pid)
	} else {
		cgroupPath = cgroup1.RootPath
	}

	p, _ := cgroupPath(cgroup1.Memory)
	log.Infof("Using cgroup path: %s", path.Join(SysFsCgroup, string(cgroup1.Memory), p))
	manager, err := cgroup1.Load(cgroupPath)
	if err != nil {
		panic(err)
	}

	eventCh, errCh := v1EventChan(manager)
	processEvents(eventCh, errCh, sigCh)
}

func processV2Events(pid int, sigCh <-chan os.Signal) {
	cgroupPath, err := cgroup2.PidGroupPath(pid)
	if err != nil {
		panic(err)
	}
	log.Infof("Using cgroup path: %s", path.Join(SysFsCgroup, cgroupPath))
	manager, err := cgroup2.Load(cgroupPath)
	if err != nil {
		panic(err)
	}

	eventCh, errCh := manager.EventChan()
	processEvents(eventCh, errCh, sigCh)
}

func processEvents(eventCh <-chan Event, errCh <-chan error, sigCh <-chan os.Signal) {
	for {
		log.Infof("Waiting for memory event...")
		select {
		case event := <-eventCh:
			log.Infof("Received memory event: %v", event)
		case err := <-errCh:
			log.Errorf("Error receiving memory event: %v", err)
			return
		case sig := <-sigCh:
			log.Infof("Received signal: %v", sig)
			return
		}
	}
}

func v1EventChan(manager cgroup1.Cgroup) (<-chan Event, <-chan error) {
	eventCh := make(chan Event)
	errCh := make(chan error, 1)

	go func() {
		efd, err := manager.OOMEventFD()
		if err != nil {
			panic(err)
		}
		defer close(eventCh)
		defer close(errCh)
		defer unix.Close(int(efd))

		events := Event{}

		eventSize := 8
		buf := make([]byte, eventSize)
		for {
			n, err := unix.Read(int(efd), buf)
			log.Infof("Read %d bytes from eventfd: %v", n, buf)
			if err != nil {
				errCh <- err
				break
			}
			if n != eventSize {
				errCh <- unix.EINVAL
				break
			}
			val, n := binary.Uvarint(buf)
			if n <= 0 {
				errCh <- fmt.Errorf("invalid eventfd value: %v", buf)
			}
			events.OOMKill += val
			eventCh <- Event{OOMKill: events.OOMKill}
		}
	}()

	return eventCh, errCh
}

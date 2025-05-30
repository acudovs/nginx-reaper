package procps

import (
	"errors"
	"github.com/shirou/gopsutil/v3/process"
	"nginx-reaper/internal/procps/option"
	"os"
	"sort"
)

var (
	processes   = Processes
	processPids = process.Pids
)

// Pgrep searches for processes that match the provided options.
func Pgrep(options ...option.Option) []*process.Process {
	var result []*process.Process

	procs, err := processes()
	if err != nil {
		return result
	}

	for _, proc := range procs {
		if All(proc, options...) {
			result = append(result, proc)
		}
	}

	return result
}

// Processes retrieve a list of all running processes.
// Note: Using process.Processes() has a huge performance impact, so use process.Pids() instead.
func Processes() ([]*process.Process, error) {
	var result []*process.Process

	pids, err := processPids()
	if err != nil {
		return result, err
	}

	for _, pid := range pids {
		result = append(result, &process.Process{Pid: pid})
	}

	return result, nil
}

// All returns a bool indicating whether the process matches all the options provided.
func All(proc *process.Process, options ...option.Option) bool {
	for _, match := range options {
		if !match(proc) {
			return false
		}
	}
	return true
}

// Filter takes a slice of processes and returns a slice of processes that match all the options provided.
func Filter(procs []*process.Process, options ...option.Option) []*process.Process {
	var matched []*process.Process
	for _, proc := range procs {
		if All(proc, options...) {
			matched = append(matched, proc)
		}
	}
	return matched
}

// SortByCreateTime sorts a slice of processes based on their creation time.
func SortByCreateTime(procs []*process.Process) {
	sort.Slice(procs, func(i, j int) bool {
		cti, ei := procs[i].CreateTime()
		ctj, ej := procs[j].CreateTime()
		return ei == nil && ej == nil && cti < ctj
	})
}

// Terminate the specified process. If the process is already terminated, no error is returned.
func Terminate(proc *process.Process) error {
	err := proc.Terminate()
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}

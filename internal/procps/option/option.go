package option

import (
	"github.com/shirou/gopsutil/v3/process"
	"strings"
)

// Option is a function type that matches a process based on specific criteria.
type Option func(*process.Process) bool

// Cmdline returns an Option that matches a process whose command-line contains the specified name.
func Cmdline(name string) Option {
	return func(proc *process.Process) bool {
		cmdline, err := proc.Cmdline()
		return err == nil && strings.Contains(cmdline, name)
	}
}

// Parent returns an Option that matches a process whose parent's PID matches the specified value.
func Parent(ppid int32) Option {
	return func(proc *process.Process) bool {
		parent, err := proc.Parent()
		return err == nil && parent.Pid == ppid
	}
}

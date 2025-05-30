package procps

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

type fields struct {
	Pid        int32
	Name       string
	Cmdline    string
	CreateTime int64
}

func TestNewProcessInfo(t *testing.T) {
	current := &process.Process{Pid: int32(os.Getpid())}
	parent, _ := current.Parent()

	tests := []struct {
		name string
		proc *process.Process
		want *fields
	}{
		{
			name: "CurrentProcessInfo",
			proc: current,
		},
		{
			name: "ParentProcessInfo",
			proc: parent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewProcessInfo(tt.proc)
			parent, _ = tt.proc.Parent()
			testProcessInfo(t, tt.proc, got, parent)
		})
	}
}

func testProcessInfo(t assert.TestingT, proc *process.Process, pi *ProcessInfo, parentProc *process.Process) {
	f := fieldsFrom(proc)
	assert.Equal(t, f.Pid, pi.Pid)
	assert.Equal(t, f.Name, pi.Name)
	assert.Equal(t, f.Cmdline, pi.Cmdline)
	assert.Equal(t, f.CreateTime, pi.CreateTime)
	if pi.Parent != nil {
		testProcessInfo(t, parentProc, pi.Parent, nil)
	}
}

func fieldsFrom(proc *process.Process) *fields {
	procName, _ := proc.Name()
	procCmdline, _ := proc.Cmdline()
	procCreateTime, _ := proc.CreateTime()

	return &fields{
		Pid:        proc.Pid,
		Name:       procName,
		Cmdline:    procCmdline,
		CreateTime: procCreateTime,
	}
}

func TestProcessInfo_String(t *testing.T) {
	proc := &process.Process{Pid: 1}
	t.Run("String", func(t *testing.T) {
		got := NewProcessInfo(proc)
		assert.Equal(t, stringFrom(got), got.String())
	})
}

func stringFrom(pi *ProcessInfo) string {
	return fmt.Sprintf(
		"{\"pid\":%d,\"name\":\"%s\",\"cmdline\":\"%s\",\"createTime\":%d,\"rss\":%d,\"vms\":%d}",
		pi.Pid, pi.Name, pi.Cmdline, pi.CreateTime, pi.RSS, pi.VMS)
}

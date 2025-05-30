package procps

import (
	"encoding/json"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo represents process information.
type ProcessInfo struct {
	Pid        int32        `json:"pid"`
	Name       string       `json:"name,omitempty"`
	Cmdline    string       `json:"cmdline,omitempty"`
	CreateTime int64        `json:"createTime,omitempty"`
	RSS        uint64       `json:"rss,omitempty"`
	VMS        uint64       `json:"vms,omitempty"`
	Parent     *ProcessInfo `json:"parent,omitempty"`
}

// NewProcessInfo creates a new ProcessInfo instance from the provided *process.Process instance.
func NewProcessInfo(proc *process.Process) *ProcessInfo {
	p := FromProcess(proc)
	if parent, err := proc.Parent(); err == nil {
		p.Parent = FromProcess(parent)
	}
	return p
}

// FromProcess creates a ProcessInfo instance from the provided *process.Process instance.
func FromProcess(proc *process.Process) *ProcessInfo {
	p := &ProcessInfo{Pid: proc.Pid}
	if name, err := proc.Name(); err == nil {
		p.Name = name
	}
	if cmdline, err := proc.Cmdline(); err == nil {
		p.Cmdline = cmdline
	}
	if createTime, err := proc.CreateTime(); err == nil {
		p.CreateTime = createTime
	}
	if memoryInfo, err := proc.MemoryInfo(); err == nil {
		p.RSS = memoryInfo.RSS
		p.VMS = memoryInfo.VMS
	}
	return p
}

// String returns the JSON representation of the ProcessInfo instance.
func (p *ProcessInfo) String() string {
	s, _ := json.Marshal(p)
	return string(s)
}

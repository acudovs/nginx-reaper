package procps

import (
	"encoding/json"
	"errors"
	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/prometheus/procfs"
	"math"
	"nginx-reaper/internal/env"
	"nginx-reaper/internal/log"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	envProcMountPoint   = "PROC_MOUNT_POINT"
	envCgroupMountPoint = "CGROUP_MOUNT_POINT"

	v1LimitFile = "memory.limit_in_bytes"
	v1UsageFile = "memory.usage_in_bytes"
	v2LimitFile = "memory.max"
	v2UsageFile = "memory.current"
)

var (
	cgroupsMode         = cgroups.Mode
	cgroup1PidPath      = cgroup1.PidPath
	cgroup2PidGroupPath = cgroup2.PidGroupPath
)

// MemoryInfo represents a memory information.
type MemoryInfo struct {
	Total     uint64 `json:"total"`
	Available uint64 `json:"available"`
}

// NewMemoryInfo creates a new NewMemoryInfo instance by reading system and cgroup memory of the specified pid.
func NewMemoryInfo(pid int) *MemoryInfo {
	var m = &MemoryInfo{}
	var err error

	m.Total, m.Available, err = ReadSystemMemory()
	if err != nil {
		log.Errorf("Failed to read system memory: %v, using default %v", err, m)
		return m
	}

	limit, available, err := ReadCgroupMemory(pid)
	if err != nil {
		log.Errorf("Failed to read cgroup memory: %v, using system %v", err, m)
		return m
	}
	m.Total = min(m.Total, limit)
	m.Available = min(m.Available, available)

	return m
}

// AvailableMemoryPercent calculates the percentage of available memory.
func (m *MemoryInfo) AvailableMemoryPercent() int {
	if m.Total == 0 || m.Available >= m.Total {
		return 100
	}
	return int(float64(m.Available) / float64(m.Total) * 100)
}

// String returns the JSON representation of the MemoryInfo instance.
func (m *MemoryInfo) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}

// ReadSystemMemory reads the system memory information.
// Returns the total and available memory in bytes, and error, if any.
// See https://www.kernel.org/doc/Documentation/filesystems/proc.txt
func ReadSystemMemory() (uint64, uint64, error) {
	procMountPoint := env.GetString(envProcMountPoint, "/proc")

	fs, err := procfs.NewFS(procMountPoint)
	if err != nil {
		return 0, 0, err
	}

	meminfo, err := fs.Meminfo()
	if err != nil {
		return 0, 0, err
	}

	if meminfo.MemTotal == nil || meminfo.MemAvailable == nil {
		return 0, 0, errors.New("error reading meminfo")
	}

	return *meminfo.MemTotal * 1024, *meminfo.MemAvailable * 1024, nil
}

// ReadCgroupMemory reads the cgroup memory information of the specified pid.
// Returns the limit and available memory in bytes, and error, if any.
// See https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt
// See https://www.kernel.org/doc/Documentation/cgroup-v2.txt
func ReadCgroupMemory(pid int) (uint64, uint64, error) {
	cgroupMountPoint := env.GetString(envCgroupMountPoint, "/sys/fs/cgroup")

	var limitFile, usageFile string

	if cgroupsMode() == cgroups.Unified {
		cgroupPath, err := cgroup2PidGroupPath(pid)
		if err != nil {
			return 0, 0, err
		}
		limitFile = path.Join(cgroupMountPoint, cgroupPath, v2LimitFile)
		usageFile = path.Join(cgroupMountPoint, cgroupPath, v2UsageFile)
	} else {
		subsystem := cgroup1.Memory
		cgroupPath, err := cgroup1PidPath(pid)(subsystem)
		if err != nil {
			return 0, 0, err
		}
		// Check if the full cgroup v1 path exists, otherwise try the root path (cgroup namespace).
		if _, err = os.Stat(path.Join(cgroupMountPoint, string(subsystem), cgroupPath)); err != nil {
			cgroupPath, _ = cgroup1.RootPath(subsystem)
		}
		limitFile = path.Join(cgroupMountPoint, string(subsystem), cgroupPath, v1LimitFile)
		usageFile = path.Join(cgroupMountPoint, string(subsystem), cgroupPath, v1UsageFile)
	}

	limit, err := readContentUint64(limitFile)
	if err != nil {
		return 0, 0, err
	}
	usage, err := readContentUint64(usageFile)
	if err != nil {
		return 0, 0, err
	}

	// Usage may temporarily exceed the limit.
	available := max(0, limit-usage)
	return limit, available, nil
}

// readContentUint64 reads uint64 value from the specified file.
// Returns parsed uint64 value and error, if any.
func readContentUint64(filePath string) (uint64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// We need a maximum of 20 bytes for an uint64 string, using next power of 2.
	buf := make([]byte, 32)
	n, err := f.Read(buf)
	if err != nil {
		return 0, err
	}

	value := strings.TrimSpace(string(buf[:n]))
	if value == "max" {
		return math.MaxUint64, nil
	}

	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, err
	}
	return result, nil
}

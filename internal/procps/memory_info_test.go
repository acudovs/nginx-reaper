package procps

import (
	"errors"
	"fmt"
	"github.com/containerd/cgroups/v3"
	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/containerd/cgroups/v3/cgroup2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math"
	"math/rand"
	"nginx-reaper/internal/log"
	"os"
	"path"
	"testing"
)

// See https://github.com/stretchr/testify#mock-package
type MockCgroupsMode struct {
	mock.Mock
}

func (m *MockCgroupsMode) Call() cgroups.CGMode {
	args := m.Called()
	return args.Get(0).(cgroups.CGMode)
}

type MockCgroup1PidPath struct {
	mock.Mock
}

func (m *MockCgroup1PidPath) Call(int) cgroup1.Path {
	args := m.Called()
	return func(subsystem cgroup1.Name) (string, error) {
		return args.String(0), args.Error(1)
	}
}

type MockCgroup2PidGroupPath struct {
	mock.Mock
}

func (m *MockCgroup2PidGroupPath) Call(int) (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func TestNewMemoryInfo(t *testing.T) {
	totalKB := rand.Uint64()
	availableKB := rand.Uint64()
	limit := rand.Uint64()
	usage := rand.Uint64()

	type data struct {
		totalKB     uint64
		availableKB uint64
		limit       uint64
		usage       uint64
		cgroup      bool
	}
	tests := []struct {
		name string
		data data
		want *MemoryInfo
	}{
		{
			name: "SystemMemory",
			data: data{
				totalKB:     totalKB,
				availableKB: availableKB,
			},
			want: &MemoryInfo{
				Total:     totalKB * 1024,
				Available: availableKB * 1024,
			},
		},
		{
			name: "CgroupMemory",
			data: data{
				totalKB:     totalKB,
				availableKB: availableKB,
				limit:       limit,
				usage:       usage,
				cgroup:      true,
			},
			want: &MemoryInfo{
				Total:     min(totalKB*1024, limit),
				Available: min(availableKB*1024, max(0, limit-usage)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			meminfoPath := path.Join(tempDir, "meminfo")
			writeFile(
				t,
				meminfoPath,
				fmt.Sprintf("MemTotal: %d kB\nMemAvailable: %d kB\n", tt.data.totalKB, tt.data.availableKB),
			)
			t.Setenv(envProcMountPoint, tempDir)

			if tt.data.cgroup {
				var mockCgroupsMode MockCgroupsMode
				mockCgroupsMode.On("Call").Return(cgroups.Unified)
				cgroupsMode = mockCgroupsMode.Call
				defer func() { cgroupsMode = cgroups.Mode }()

				cgroupPath := "/"
				var mockCgroup2PidGroupPath MockCgroup2PidGroupPath
				mockCgroup2PidGroupPath.On("Call").Return(cgroupPath, nil)
				cgroup2PidGroupPath = mockCgroup2PidGroupPath.Call
				defer func() { cgroup2PidGroupPath = cgroup2.PidGroupPath }()

				writeFile(t, path.Join(tempDir, cgroupPath, v2LimitFile), fmt.Sprintln(tt.data.limit))
				writeFile(t, path.Join(tempDir, cgroupPath, v2UsageFile), fmt.Sprintln(tt.data.usage))
			}
			t.Setenv(envCgroupMountPoint, tempDir)

			assert.Equal(t, tt.want, NewMemoryInfo(os.Getpid()))
		})
	}
	t.Run("Default", func(t *testing.T) {
		tempDir := t.TempDir()
		t.Setenv(envProcMountPoint, tempDir)
		assert.Equal(t, &MemoryInfo{}, NewMemoryInfo(os.Getpid()))
	})
}

func TestMemoryInfo_AvailableMemoryPercent(t *testing.T) {
	value := rand.Uint64()

	type fields struct {
		Total     uint64
		Available uint64
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "100%Available",
			want: 100,
		},
		{
			name: "100%Available",
			fields: fields{
				Total:     value,
				Available: value,
			},
			want: 100,
		},
		{
			name: "100%Available",
			fields: fields{
				Total:     value / 2,
				Available: value,
			},
			want: 100,
		},
		{
			name: "50%Available",
			fields: fields{
				Total:     value,
				Available: value / 2,
			},
			want: 50,
		},
		{
			name: "10%Available",
			fields: fields{
				Total:     value,
				Available: value / 10,
			},
			want: 10,
		},
		{
			name: "0%Available",
			fields: fields{
				Total:     value,
				Available: 0,
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemoryInfo{
				Total:     tt.fields.Total,
				Available: tt.fields.Available,
			}
			assert.Equal(t, tt.want, m.AvailableMemoryPercent())
		})
	}
}

func TestMemoryInfo_String(t *testing.T) {
	total := rand.Uint64()
	available := rand.Uint64()
	want := fmt.Sprintf(`{"total":%d,"available":%d}`, total, available)

	t.Run("String", func(t *testing.T) {
		m := &MemoryInfo{
			Total:     total,
			Available: available,
		}
		assert.Equal(t, want, m.String())
	})
}

func TestReadSystemMemory(t *testing.T) {
	totalKB := rand.Uint64()
	availableKB := rand.Uint64()

	type want struct {
		total     uint64
		available uint64
	}
	tests := []struct {
		name    string
		meminfo string
		want    want
		wantErr bool
	}{
		{
			name:    "ReadSystemMemory",
			meminfo: fmt.Sprintf("MemTotal: %d kB\nMemAvailable: %d kB\n", totalKB, availableKB),
			want: want{
				total:     totalKB * 1024,
				available: availableKB * 1024,
			},
		},
		{
			name:    "ReadNoMemTotal",
			meminfo: fmt.Sprintf("MemAvailable: %d kB\n", availableKB),
			wantErr: true,
		},
		{
			name:    "ReadNoMemAvailable",
			meminfo: fmt.Sprintf("MemTotal: %d kB\n", totalKB),
			wantErr: true,
		},
		{
			name:    "ReadInvalid",
			meminfo: "MemTotal: X kB\nMemAvailable: Y kB\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			meminfoPath := path.Join(tempDir, "meminfo")
			writeFile(t, meminfoPath, tt.meminfo)
			t.Setenv(envProcMountPoint, tempDir)

			got1, got2, err := ReadSystemMemory()
			if tt.wantErr {
				log.Error(err)
				assert.Zero(t, got1)
				assert.Zero(t, got2)
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want.total, got1)
				assert.Equal(t, tt.want.available, got2)
				assert.NoError(t, err)
			}
		})
	}
	t.Run("ReadNoDir", func(t *testing.T) {
		t.Setenv(envProcMountPoint, string(rand.Int31()))
		got1, got2, err := ReadSystemMemory()
		log.Error(err)
		assert.Zero(t, got1)
		assert.Zero(t, got2)
		assert.Error(t, err)
	})
	t.Run("ReadNoFile", func(t *testing.T) {
		t.Setenv(envProcMountPoint, t.TempDir())
		got1, got2, err := ReadSystemMemory()
		log.Error(err)
		assert.Zero(t, got1)
		assert.Zero(t, got2)
		assert.Error(t, err)
	})
}

func TestReadCgroupMemory(t *testing.T) {
	limit := rand.Uint64()
	usage := rand.Uint64()

	type data struct {
		limit string
		usage string
	}
	type want struct {
		limit     uint64
		available uint64
	}
	type wantErr struct {
		path    bool
		content bool
	}
	tests := []struct {
		name    string
		data    data
		mode    cgroups.CGMode
		want    want
		wantErr wantErr
	}{
		{
			name: "ReadCgroupMemoryV1",
			data: data{
				limit: fmt.Sprintln(limit),
				usage: fmt.Sprintln(usage),
			},
			mode: cgroups.Legacy,
			want: want{
				limit:     limit,
				available: max(0, limit-usage),
			},
		},
		{
			name: "ReadCgroupMemoryV2",
			data: data{
				limit: fmt.Sprintln(limit),
				usage: fmt.Sprintln(usage),
			},
			mode: cgroups.Unified,
			want: want{
				limit:     limit,
				available: max(0, limit-usage),
			},
		},
		{
			name: "V1PathError",
			data: data{
				limit: fmt.Sprintln(limit),
				usage: fmt.Sprintln(usage),
			},
			mode: cgroups.Legacy,
			wantErr: wantErr{
				path: true,
			},
		},
		{
			name: "V2PathError",
			data: data{
				limit: fmt.Sprintln(limit),
				usage: fmt.Sprintln(usage),
			},
			mode: cgroups.Unified,
			wantErr: wantErr{
				path: true,
			},
		},
		{
			name: "LimitError",
			data: data{
				usage: fmt.Sprintln(usage),
			},
			mode: cgroups.Legacy,
			wantErr: wantErr{
				content: true,
			},
		},
		{
			name: "UsageError",
			data: data{
				limit: fmt.Sprintln(limit),
			},
			mode: cgroups.Unified,
			wantErr: wantErr{
				content: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockCgroupsMode MockCgroupsMode
			mockCgroupsMode.On("Call").Return(tt.mode)
			cgroupsMode = mockCgroupsMode.Call
			defer func() { cgroupsMode = cgroups.Mode }()

			tempDir := t.TempDir()
			t.Setenv(envCgroupMountPoint, tempDir)

			cgroupPath := "/"
			if tt.mode == cgroups.Unified {
				writeFile(t, path.Join(tempDir, cgroupPath, v2LimitFile), tt.data.limit)
				writeFile(t, path.Join(tempDir, cgroupPath, v2UsageFile), tt.data.usage)

				var mockCgroup2PidGroupPath MockCgroup2PidGroupPath
				if tt.wantErr.path {
					mockCgroup2PidGroupPath.On("Call").Return("", errors.New(tt.name))
				} else {
					mockCgroup2PidGroupPath.On("Call").Return(cgroupPath, nil)
				}
				cgroup2PidGroupPath = mockCgroup2PidGroupPath.Call
				defer func() { cgroup2PidGroupPath = cgroup2.PidGroupPath }()
			} else {
				subsystem := cgroup1.Memory
				writeFile(t, path.Join(tempDir, string(subsystem), cgroupPath, v1LimitFile), tt.data.limit)
				writeFile(t, path.Join(tempDir, string(subsystem), cgroupPath, v1UsageFile), tt.data.usage)

				var mockCgroup1PidPath MockCgroup1PidPath
				if tt.wantErr.path {
					mockCgroup1PidPath.On("Call").Return("", errors.New(tt.name))
				} else {
					mockCgroup1PidPath.On("Call").Return(cgroupPath+"missing", nil)
				}
				cgroup1PidPath = mockCgroup1PidPath.Call
				defer func() { cgroup1PidPath = cgroup1.PidPath }()
			}

			got1, got2, err := ReadCgroupMemory(os.Getpid())
			if tt.wantErr.path || tt.wantErr.content {
				log.Error(err)
				assert.Zero(t, got1)
				assert.Zero(t, got2)
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want.limit, got1)
				assert.Equal(t, tt.want.available, got2)
				assert.NoError(t, err)
			}
		})
	}
}

func Test_readCgroupFile(t *testing.T) {
	value := rand.Uint64()

	tests := []struct {
		name    string
		content string
		want    uint64
		wantErr bool
	}{
		{
			name:    "ReadValue",
			content: fmt.Sprintln(value),
			want:    value,
		},
		{
			name:    "ReadWithSpaces",
			content: fmt.Sprintf(" %d ", value),
			want:    value,
		},
		{
			name:    "ReadMax",
			content: "max",
			want:    math.MaxUint64,
		},
		{
			name:    "ReadNoNewline",
			content: fmt.Sprintf("%d", value),
			want:    value,
		},
		{
			name:    "ReadNegative",
			content: fmt.Sprintf("-%d", value),
			wantErr: true,
		},
		{
			name:    "ReadEmpty",
			content: "",
			wantErr: true,
		},
		{
			name:    "ReadInvalid",
			content: "X",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tempFile(t.TempDir())
			writeFile(t, filePath, tt.content)
			got, err := readContentUint64(filePath)
			if tt.wantErr {
				log.Error(err)
				assert.Zero(t, got)
				assert.Error(t, err)
			} else {
				assert.Equal(t, tt.want, got)
				assert.NoError(t, err)
			}
		})
	}
	t.Run("ReadNoFile", func(t *testing.T) {
		filePath := tempFile(t.TempDir())
		got, err := readContentUint64(filePath)
		log.Error(err)
		assert.Zero(t, got)
		assert.Error(t, err)
	})
}

func tempFile(dir string) string {
	return path.Join(dir, fmt.Sprint(rand.Uint32()))
}

func writeFile(t *testing.T, filePath string, content string) {
	err := os.MkdirAll(path.Dir(filePath), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(filePath, []byte(content), 0644)
	assert.NoError(t, err)
}

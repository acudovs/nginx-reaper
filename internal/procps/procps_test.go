package procps

import (
	"errors"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/rand"
	"nginx-reaper/internal/log"
	"nginx-reaper/internal/procps/option"
	"os"
	"os/exec"
	"testing"
)

// See https://github.com/stretchr/testify#mock-package
type MockProcesses struct {
	mock.Mock
}

func (m *MockProcesses) Call() ([]*process.Process, error) {
	args := m.Called()
	var result []*process.Process
	if args.Get(0) != nil {
		result = args.Get(0).([]*process.Process)
	}
	return result, args.Error(1)
}

func TestPgrep(t *testing.T) {
	tests := []struct {
		name    string
		options []option.Option
		want    []*process.Process
		wantErr bool
	}{
		{
			name: "Pid1",
			options: []option.Option{
				func(proc *process.Process) bool {
					return proc.Pid == 1
				},
			},
			want: []*process.Process{
				{Pid: 1},
			},
		},
		{
			name: "CurrentPid",
			options: []option.Option{
				func(proc *process.Process) bool {
					return proc.Pid == int32(os.Getpid())
				},
			},
			want: []*process.Process{
				{Pid: int32(os.Getpid())},
			},
		},
		{
			name: "Pid1CurrentPid",
			options: []option.Option{
				func(proc *process.Process) bool {
					return proc.Pid == 1 || proc.Pid == int32(os.Getpid())
				},
			},
			want: []*process.Process{
				{Pid: 1},
				{Pid: int32(os.Getpid())},
			},
		},
		{
			name: "NotMatch",
			options: []option.Option{
				func(proc *process.Process) bool {
					return false
				},
			},
		},
		{
			name:    "ProcessesError",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcesses := MockProcesses{}
			if tt.wantErr {
				// Mock processes function to return an error
				mockProcesses.On("Call").Return(nil, errors.New(tt.name))
				processes = mockProcesses.Call
				defer func() { processes = Processes }()
			}
			got := Pgrep(tt.options...)
			if tt.wantErr {
				mockProcesses.AssertCalled(t, "Call")
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

// See https://github.com/stretchr/testify#mock-package
type MockPids struct {
	mock.Mock
}

func (m *MockPids) Call() ([]int32, error) {
	args := m.Called()
	var pids []int32
	if args.Get(0) != nil {
		pids = args.Get(0).([]int32)
	}
	return pids, args.Error(1)
}

func TestProcesses(t *testing.T) {
	tests := []struct {
		name    string
		want    []*process.Process
		wantErr bool
	}{
		{
			name: "ProcessPids",
			want: []*process.Process{
				{Pid: rand.Int31()},
				{Pid: rand.Int31()},
				{Pid: rand.Int31()},
			},
		},
		{
			name:    "ProcessPidsError",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPids := MockPids{}
			if tt.wantErr {
				// Mock the processPids function to return an error
				mockPids.On("Call").Return(nil, errors.New(tt.name))
				processPids = mockPids.Call
				defer func() { processPids = process.Pids }()
			} else {
				// Mock the processPids function to return the expected result
				mockPids.On("Call").Return(pidsFrom(tt.want), nil)
				processPids = mockPids.Call
				defer func() { processPids = process.Pids }()
			}
			got, err := Processes()
			mockPids.AssertCalled(t, "Call")
			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				log.Error(err)
				assert.Zero(t, len(got))
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func pidsFrom(procs []*process.Process) []int32 {
	var pids []int32
	for _, proc := range procs {
		pids = append(pids, proc.Pid)
	}
	return pids
}

func TestAll(t *testing.T) {
	currentProc := &process.Process{Pid: int32(os.Getpid())}
	currentProcCmdline, _ := currentProc.Cmdline()
	parentProc, _ := currentProc.Parent()

	type args struct {
		proc    *process.Process
		options []option.Option
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "MatchProcCmdline",
			args: args{
				proc: currentProc,
				options: []option.Option{
					option.Cmdline(currentProcCmdline),
				},
			},
			want: true,
		},
		{
			name: "MatchProcCmdlineAndParent",
			args: args{
				proc: currentProc,
				options: []option.Option{
					option.Cmdline(currentProcCmdline),
					option.Parent(parentProc.Pid),
				},
			},
			want: true,
		},
		{
			name: "NotMatchProc",
			args: args{
				proc: currentProc,
				options: []option.Option{
					option.Cmdline(currentProcCmdline),
					option.Parent(currentProc.Pid),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := All(tt.args.proc, tt.args.options...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilter(t *testing.T) {
	pid1Proc := &process.Process{Pid: 1}
	pid1Cmdline, _ := pid1Proc.Cmdline()
	currentProc := &process.Process{Pid: int32(os.Getpid())}
	currentCmdline, _ := currentProc.Cmdline()
	parentProc, _ := currentProc.Parent()

	type args struct {
		procs   []*process.Process
		options []option.Option
	}
	tests := []struct {
		name string
		args args
		want []*process.Process
	}{
		{
			name: "FilterPid1Proc",
			args: args{
				procs:   []*process.Process{pid1Proc, currentProc},
				options: []option.Option{option.Cmdline(pid1Cmdline)},
			},
			want: []*process.Process{pid1Proc},
		},
		{
			name: "FilterCurrentProc",
			args: args{
				procs: []*process.Process{pid1Proc, currentProc},
				options: []option.Option{
					option.Cmdline(currentCmdline),
					option.Parent(parentProc.Pid),
				},
			},
			want: []*process.Process{currentProc},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Filter(tt.args.procs, tt.args.options...)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSortByCreateTime(t *testing.T) {
	pid1Proc := &process.Process{Pid: 1}
	currentProc := &process.Process{Pid: int32(os.Getpid())}
	parentProc, _ := currentProc.Parent()

	tests := []struct {
		name      string
		processes []*process.Process
		want      []*process.Process
	}{
		{
			name:      "SortByCreateTime",
			processes: []*process.Process{parentProc, currentProc, pid1Proc},
			want:      []*process.Process{pid1Proc, parentProc, currentProc},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortByCreateTime(tt.processes)
			assert.Equal(t, tt.want, tt.processes)
		})
	}
}

func TestTerminate(t *testing.T) {
	cmd := exec.Command("sleep", "100")
	assert.NoError(t, cmd.Start())

	tests := []struct {
		name    string
		proc    *process.Process
		wantErr bool
	}{
		{
			name: "Terminate",
			proc: &process.Process{Pid: int32(cmd.Process.Pid)},
		},
		{
			name: "ErrProcessDone",
			proc: &process.Process{Pid: -2},
		},
		{
			name:    "TerminateError",
			proc:    &process.Process{Pid: 0},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Terminate(tt.proc)
			if tt.wantErr {
				log.Error(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

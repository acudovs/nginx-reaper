package reaper

import (
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math"
	"math/rand"
	"nginx-reaper/internal/procps"
	"nginx-reaper/internal/procps/option"
	"testing"
	"time"
)

type MockProcpsPgrep struct {
	mock.Mock
}

func (m *MockProcpsPgrep) Call(...option.Option) []*process.Process {
	args := m.Called()
	return args.Get(len(m.Calls) - 1).([]*process.Process)
}

func TestReaper(t *testing.T) {
	interval := time.Duration(rand.Intn(math.MaxInt-1) + 1)
	maxShutdownWorkers := rand.Intn(math.MaxInt-1) + 1
	availableMemoryPercent := rand.Intn(101)

	type args struct {
		interval               time.Duration
		maxShutdownWorkers     int
		availableMemoryPercent int
	}
	tests := []struct {
		name      string
		args      args
		want      *Reaper
		wantPanic bool
	}{
		{
			name: "NewReaper",
			args: args{
				interval:               interval,
				maxShutdownWorkers:     maxShutdownWorkers,
				availableMemoryPercent: availableMemoryPercent,
			},
			want: &Reaper{
				interval:               interval,
				maxShutdownWorkers:     maxShutdownWorkers,
				availableMemoryPercent: availableMemoryPercent,
			},
		},
		{
			name: "NonPositiveInterval",
			args: args{
				interval:               -interval,
				maxShutdownWorkers:     maxShutdownWorkers,
				availableMemoryPercent: availableMemoryPercent,
			},
			wantPanic: true,
		},
		{
			name: "NonPositiveMaxShutdownWorkers",
			args: args{
				interval:               interval,
				maxShutdownWorkers:     -maxShutdownWorkers,
				availableMemoryPercent: availableMemoryPercent,
			},
			wantPanic: true,
		},
		{
			name: "NegativeMemoryPercent",
			args: args{
				interval:               interval,
				maxShutdownWorkers:     maxShutdownWorkers,
				availableMemoryPercent: -availableMemoryPercent - 1,
			},
			wantPanic: true,
		},
		{
			name: "InvalidMemoryPercent",
			args: args{
				interval:               interval,
				maxShutdownWorkers:     maxShutdownWorkers,
				availableMemoryPercent: 101,
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() {
					NewReaper(tt.args.interval, tt.args.maxShutdownWorkers, tt.args.availableMemoryPercent)
				})
			} else {
				got := NewReaper(tt.args.interval, tt.args.maxShutdownWorkers, tt.args.availableMemoryPercent)
				assert.Equal(t, tt.want.interval, got.Interval())
				assert.Equal(t, tt.want.maxShutdownWorkers, got.maxShutdownWorkers)
				assert.Equal(t, stringFrom(got), got.String())
				assert.Equal(t, 2, len(got.Metrics()))
			}
		})
	}
}

func stringFrom(r *Reaper) string {
	return fmt.Sprintf(
		"Nginx Reaper with configuration: interval %v, max workers to keep %v, target available memory %v%%",
		r.interval, r.maxShutdownWorkers, r.availableMemoryPercent,
	)
}

type MockProcpsFilter struct {
	mock.Mock
}

func (m *MockProcpsFilter) Call([]*process.Process, ...option.Option) []*process.Process {
	args := m.Called()
	return args.Get(0).([]*process.Process)
}

type MockProcpsTerminate struct {
	mock.Mock
}

func (m *MockProcpsTerminate) Call(*process.Process) error {
	args := m.Called()
	return args.Error(0)
}

func TestReaper_Run(t *testing.T) {
	type fields struct {
		interval               time.Duration
		maxShutdownWorkers     int
		availableMemoryPercent int
	}
	type procs struct {
		masters         []*process.Process
		workers         []*process.Process
		workersShutdown []*process.Process
	}
	tests := []struct {
		name    string
		fields  fields
		procs   procs
		want    int
		wantErr bool
	}{
		{
			name: "Terminate",
			fields: fields{
				interval:           1,
				maxShutdownWorkers: 1,
			},
			procs: procs{
				masters:         []*process.Process{{Pid: 0}},
				workers:         []*process.Process{{Pid: 0}, {Pid: 0}, {Pid: 0}, {Pid: 0}},
				workersShutdown: []*process.Process{{Pid: 0}, {Pid: 0}, {Pid: 0}},
			},
			want: 2,
		},
		{
			name: "TerminateError",
			fields: fields{
				interval:           1,
				maxShutdownWorkers: 1,
			},
			procs: procs{
				masters:         []*process.Process{{Pid: 0}},
				workers:         []*process.Process{{Pid: 0}, {Pid: 0}, {Pid: 0}, {Pid: 0}},
				workersShutdown: []*process.Process{{Pid: 0}, {Pid: 0}, {Pid: 0}},
			},
			want:    2,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcpsPgrep := MockProcpsPgrep{}
			mockProcpsPgrep.On("Call").Return(tt.procs.masters, tt.procs.workers)
			procpsPgrep = mockProcpsPgrep.Call
			defer func() { procpsPgrep = procps.Pgrep }()

			mockProcpsFilter := MockProcpsFilter{}
			mockProcpsFilter.On("Call").Return(tt.procs.workersShutdown)
			procpsFilter = mockProcpsFilter.Call
			defer func() { procpsFilter = procps.Filter }()

			mockProcpsTerminate := MockProcpsTerminate{}
			if tt.wantErr {
				mockProcpsTerminate.On("Call").Return(errors.New(tt.name))
			} else {
				mockProcpsTerminate.On("Call").Return(nil)
			}
			procpsTerminate = mockProcpsTerminate.Call
			defer func() { procpsTerminate = procps.Terminate }()

			r := NewReaper(tt.fields.interval, tt.fields.maxShutdownWorkers, tt.fields.availableMemoryPercent)

			assert.True(t, r.Run())
			mockProcpsTerminate.AssertNumberOfCalls(t, "Call", tt.want)

			active := len(tt.procs.workers) - len(tt.procs.workersShutdown)
			shutdown := len(tt.procs.workersShutdown)
			terminated := shutdown - tt.fields.maxShutdownWorkers
			if tt.wantErr {
				assert.Equal(t, active, getGaugeValueInt(r.collectorRunning, LabelActive))
				assert.Equal(t, shutdown, getGaugeValueInt(r.collectorRunning, LabelShutdown))
				assert.Equal(t, terminated, getCounterValueInt(r.collectorShutdown, LabelError))
				assert.Equal(t, 0, getCounterValueInt(r.collectorShutdown, LabelTerminated))
			} else {
				assert.Equal(t, active, getGaugeValueInt(r.collectorRunning, LabelActive))
				assert.Equal(t, shutdown, getGaugeValueInt(r.collectorRunning, LabelShutdown))
				assert.Equal(t, 0, getCounterValueInt(r.collectorShutdown, LabelError))
				assert.Equal(t, terminated, getCounterValueInt(r.collectorShutdown, LabelTerminated))
			}
		})
	}
}

func getCounterValueInt(metric *prometheus.CounterVec, label string) int {
	m := &dto.Metric{}
	if err := metric.WithLabelValues(label).Write(m); err != nil {
		return 0
	}
	return int(m.Counter.GetValue())
}

func getGaugeValueInt(metric *prometheus.GaugeVec, label string) int {
	m := &dto.Metric{}
	if err := metric.WithLabelValues(label).Write(m); err != nil {
		return 0
	}
	return int(m.Gauge.GetValue())
}

// See https://github.com/stretchr/testify#mock-package
type MockNewMemoryInfo struct {
	mock.Mock
}

func (m *MockNewMemoryInfo) Call(int) *procps.MemoryInfo {
	args := m.Called()
	return args.Get(0).(*procps.MemoryInfo)
}

func TestReaper_shouldTerminate(t *testing.T) {
	type fields struct {
		maxShutdownWorkers     int
		availableMemoryPercent int
		Total                  uint64
		Available              uint64
	}
	tests := []struct {
		name    string
		fields  fields
		workers int
		want    bool
	}{
		{
			name: "Default",
			fields: fields{
				maxShutdownWorkers: 255,
			},
			workers: 0,
		},
		{
			name: "MaxShutdownWorkersEquals",
			fields: fields{
				maxShutdownWorkers: 3,
			},
			workers: 3,
		},
		{
			name: "MaxShutdownWorkersLess",
			fields: fields{
				maxShutdownWorkers: 2,
			},
			workers: 3,
			want:    true,
		},
		{
			name: "MaxShutdownWorkersZero",
			fields: fields{
				maxShutdownWorkers: 0,
			},
			workers: 3,
			want:    true,
		},
		{
			name: "AvailableMemoryPercentEquals",
			fields: fields{
				maxShutdownWorkers:     255,
				availableMemoryPercent: 50,
				Total:                  100,
				Available:              50,
			},
			workers: 3,
		},
		{
			name: "AvailableMemoryPercentLess",
			fields: fields{
				maxShutdownWorkers:     255,
				availableMemoryPercent: 50,
				Total:                  100,
				Available:              25,
			},
			workers: 3,
			want:    true,
		},
		{
			name: "AvailableMemoryPercentZero",
			fields: fields{
				maxShutdownWorkers:     255,
				availableMemoryPercent: 50,
				Total:                  100,
				Available:              0,
			},
			workers: 3,
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reaper{
				maxShutdownWorkers:     tt.fields.maxShutdownWorkers,
				availableMemoryPercent: tt.fields.availableMemoryPercent,
			}
			var mockNewMemoryInfo MockNewMemoryInfo
			mockNewMemoryInfo.On("Call").Return(
				&procps.MemoryInfo{
					Total:     tt.fields.Total,
					Available: tt.fields.Available,
				},
			)
			procpsNewMemoryInfo = mockNewMemoryInfo.Call
			defer func() { procpsNewMemoryInfo = procps.NewMemoryInfo }()

			assert.Equal(t, tt.want, r.shouldTerminate(0, tt.workers))
		})
	}
}

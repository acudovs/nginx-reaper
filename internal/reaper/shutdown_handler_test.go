package reaper

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"nginx-reaper/internal/procps"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestShutdownHandler(t *testing.T) {
	shutdownInterval := time.Duration(rand.Int())
	shutdownTimeout := time.Duration(rand.Int())
	str := fmt.Sprintf("Nginx Reaper shutdown handler with interval %v and timeout %v",
		shutdownInterval, shutdownTimeout)

	t.Run("ShutdownHandler", func(t *testing.T) {
		s := &ShutdownHandler{
			shutdownInterval: shutdownInterval,
			shutdownTimeout:  shutdownTimeout,
		}
		assert.Equal(t, shutdownInterval, s.Interval())
		assert.Equal(t, str, s.String())
	})
}

func TestShutdownHandler_Run(t *testing.T) {
	timeout := time.Duration(rand.Int())

	type fields struct {
		shutdownInterval time.Duration
		shutdownTimeout  time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		masters []*process.Process
		want    bool
	}{
		{
			name: "NotRun",
		},
		{
			name: "IntervalEqualsTimeout",
			fields: fields{
				shutdownInterval: timeout,
				shutdownTimeout:  timeout,
			},
			masters: []*process.Process{{Pid: 0}},
		},
		{
			name: "IntervalLessTimeout",
			fields: fields{
				shutdownInterval: timeout / 2,
				shutdownTimeout:  timeout,
			},
			masters: []*process.Process{{Pid: 0}},
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ShutdownHandler{
				shutdownInterval: tt.fields.shutdownInterval,
				shutdownTimeout:  tt.fields.shutdownTimeout,
			}
			mockProcpsPgrep := MockProcpsPgrep{}
			mockProcpsPgrep.On("Call").Return(tt.masters)
			procpsPgrep = mockProcpsPgrep.Call
			defer func() { procpsPgrep = procps.Pgrep }()

			assert.Equal(t, tt.want, s.Run())
			assert.Equal(t, s.shutdownTimeout, tt.fields.shutdownTimeout-tt.fields.shutdownInterval)
		})
	}
}

func Test_nginxMasterRunning(t *testing.T) {
	tests := []struct {
		name    string
		masters []*process.Process
		want    bool
	}{
		{
			name: "NoMasters",
		},
		{
			name:    "HasMaster",
			masters: []*process.Process{{Pid: 0}},
			want:    true,
		},
		{
			name:    "HasMasters",
			masters: []*process.Process{{Pid: 0}, {Pid: 0}},
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcpsPgrep := MockProcpsPgrep{}
			mockProcpsPgrep.On("Call").Return(tt.masters)
			procpsPgrep = mockProcpsPgrep.Call
			defer func() { procpsPgrep = procps.Pgrep }()

			assert.Equal(t, tt.want, nginxMasterRunning())
		})
	}
}

func TestWaitShutdown(t *testing.T) {
	type args struct {
		shutdownInterval time.Duration
		shutdownTimeout  time.Duration
		sig              syscall.Signal
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Timeout",
			args: args{
				shutdownInterval: 1 * time.Millisecond,
				shutdownTimeout:  1 * time.Millisecond,
				sig:              syscall.SIGTERM,
			},
			want: 1,
		},
		{
			name: "MasterNotRunning",
			args: args{
				shutdownInterval: 1 * time.Millisecond,
				shutdownTimeout:  1 * time.Second,
				sig:              syscall.SIGTERM,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProcpsPgrep := MockProcpsPgrep{}
			mockProcpsPgrep.On("Call").Return([]*process.Process{{Pid: 0}}, []*process.Process{})
			procpsPgrep = mockProcpsPgrep.Call
			defer func() { procpsPgrep = procps.Pgrep }()

			go func() {
				time.Sleep(1 * time.Second)
				assert.NoError(t, syscall.Kill(os.Getpid(), tt.args.sig))
			}()

			WaitShutdown(tt.args.shutdownInterval, tt.args.shutdownTimeout, tt.args.sig)

			mockProcpsPgrep.AssertNumberOfCalls(t, "Call", tt.want)
		})
	}
}

package option

import (
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"

	"os"
	"testing"
)

var (
	pid1Proc       = &process.Process{Pid: 1}
	pid1Cmdline, _ = pid1Proc.Cmdline()
	currentProc    = &process.Process{Pid: int32(os.Getpid())}
	parentProc, _  = currentProc.Parent()
)

type args struct {
	option Option
	proc   *process.Process
}

func TestCmdline(t *testing.T) {
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "MatchCmdline",
			args: args{
				option: Cmdline(pid1Cmdline),
				proc:   pid1Proc,
			},
			want: true,
		},
		{
			name: "MatchPartial",
			args: args{
				option: Cmdline(pid1Cmdline[1 : len(pid1Cmdline)-1]),
				proc:   pid1Proc,
			},
			want: true,
		},
		{
			name: "NotMatchCmdline",
			args: args{
				option: Cmdline(pid1Cmdline),
				proc:   currentProc,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.option(tt.args.proc)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParent(t *testing.T) {
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "MatchParent",
			args: args{
				option: Parent(parentProc.Pid),
				proc:   currentProc,
			},
			want: true,
		},
		{
			name: "NotMatchParent",
			args: args{
				option: Parent(currentProc.Pid),
				proc:   parentProc,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.args.option(tt.args.proc)
			assert.Equal(t, tt.want, got)
		})
	}
}

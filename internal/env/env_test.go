package env

import (
	"github.com/stretchr/testify/assert"
	"nginx-reaper/internal/log"
	"testing"
	"time"
)

const (
	envName  = "TEST_ENV"
	nilValue = "[NIL]"
)

type args struct {
	envName      string
	envValue     string
	defaultValue string
}

func TestGetDuration(t *testing.T) {
	tests := []struct {
		name      string
		args      args
		want      time.Duration
		wantPanic bool
	}{
		{
			name: "NilValue",
			args: args{
				envName:      envName,
				envValue:     nilValue,
				defaultValue: "10s",
			},
			want: 10 * time.Second,
		},
		{
			name: "ValidValue",
			args: args{
				envName:      envName,
				envValue:     "5s",
				defaultValue: "10s",
			},
			want: 5 * time.Second,
		},
		{
			name: "InvalidValue",
			args: args{
				envName:      envName,
				envValue:     "invalid",
				defaultValue: "10s",
			},
			want: 10 * time.Second,
		},
		{
			name: "EmptyValue",
			args: args{
				envName:      envName,
				envValue:     "",
				defaultValue: "10s",
			},
			want: 10 * time.Second,
		},
		{
			name: "InvalidDefaultValue",
			args: args{
				envName:      envName,
				envValue:     "5s",
				defaultValue: "invalid",
			},
			wantPanic: true,
		},
		{
			name: "EmptyDefaultValue",
			args: args{
				envName:      envName,
				envValue:     "5s",
				defaultValue: "",
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.envValue != nilValue {
				t.Setenv(tt.args.envName, tt.args.envValue)
			}
			if tt.wantPanic {
				assert.Panics(t, func() { GetDuration(tt.args.envName, tt.args.defaultValue) })
			} else {
				got := GetDuration(tt.args.envName, tt.args.defaultValue)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	tests := []struct {
		name      string
		args      args
		want      int
		wantPanic bool
	}{
		{
			name: "NilValue",
			args: args{
				envName:      envName,
				envValue:     nilValue,
				defaultValue: "10",
			},
			want: 10,
		},
		{
			name: "ValidValue",
			args: args{
				envName:      envName,
				envValue:     "5",
				defaultValue: "10",
			},
			want: 5,
		},
		{
			name: "InvalidValue",
			args: args{
				envName:      envName,
				envValue:     "invalid",
				defaultValue: "10",
			},
			want: 10,
		},
		{
			name: "EmptyValue",
			args: args{
				envName:      envName,
				envValue:     "",
				defaultValue: "10",
			},
			want: 10,
		},
		{
			name: "InvalidDefaultValue",
			args: args{
				envName:      envName,
				envValue:     "5",
				defaultValue: "invalid",
			},
			wantPanic: true,
		},
		{
			name: "EmptyDefaultValue",
			args: args{
				envName:      envName,
				envValue:     "5",
				defaultValue: "",
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.envValue != nilValue {
				t.Setenv(tt.args.envName, tt.args.envValue)
			}
			if tt.wantPanic {
				assert.Panics(t, func() { GetInt(tt.args.envName, tt.args.defaultValue) })
			} else {
				got := GetInt(tt.args.envName, tt.args.defaultValue)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		args      args
		want      log.Level
		wantPanic bool
	}{
		{
			name: "NilValue",
			args: args{
				envName:      envName,
				envValue:     nilValue,
				defaultValue: "INFO",
			},
			want: log.InfoLevel,
		},
		{
			name: "ValidValue",
			args: args{
				envName:      envName,
				envValue:     "DEBUG",
				defaultValue: "INFO",
			},
			want: log.DebugLevel,
		},
		{
			name: "InvalidValue",
			args: args{
				envName:      envName,
				envValue:     "INVALID",
				defaultValue: "INFO",
			},
			want: log.InfoLevel,
		},
		{
			name: "EmptyValue",
			args: args{
				envName:      envName,
				envValue:     "",
				defaultValue: "INFO",
			},
			want: log.InfoLevel,
		},
		{
			name: "InvalidDefaultValue",
			args: args{
				envName:      envName,
				envValue:     "INFO",
				defaultValue: "INVALID",
			},
			wantPanic: true,
		},
		{
			name: "EmptyDefaultValue",
			args: args{
				envName:      envName,
				envValue:     "INFO",
				defaultValue: "",
			},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.envValue != nilValue {
				t.Setenv(tt.args.envName, tt.args.envValue)
			}
			if tt.wantPanic {
				assert.Panics(t, func() { GetLogLevel(tt.args.envName, tt.args.defaultValue) })
			} else {
				got := GetLogLevel(tt.args.envName, tt.args.defaultValue)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "NilValue",
			args: args{
				envName:      envName,
				envValue:     nilValue,
				defaultValue: "defaultValue",
			},
			want: "defaultValue",
		},
		{
			name: "ValidValue",
			args: args{
				envName:      envName,
				envValue:     "value",
				defaultValue: "defaultValue",
			},
			want: "value",
		},
		{
			name: "EmptyValue",
			args: args{
				envName:      envName,
				envValue:     "",
				defaultValue: "defaultValue",
			},
			want: "",
		},
		{
			name: "EmptyDefaultValue",
			args: args{
				envName:      envName,
				envValue:     "value",
				defaultValue: "",
			},
			want: "value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.envValue != nilValue {
				t.Setenv(tt.args.envName, tt.args.envValue)
			}
			got := GetString(tt.args.envName, tt.args.defaultValue)
			assert.Equal(t, tt.want, got)
		})
	}
}

package log

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		want    Level
		wantErr bool
	}{
		{
			name:    "Empty",
			level:   "",
			wantErr: true,
		},
		{
			name:    "Invalid",
			level:   "none",
			wantErr: true,
		},
		{
			name:  "Panic",
			level: "panic",
		},
		{
			name:  "Error",
			level: "ERROR",
			want:  1,
		},
		{
			name:  "Warning",
			level: "Warning",
			want:  2,
		},
		{
			name:  "Info",
			level: "infO",
			want:  3,
		},
		{
			name:  "Debug",
			level: "DeBuG",
			want:  4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLevel(tt.level)
			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				Error(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name  string
		level Level
		want  Level
	}{
		{
			name:  "Panic",
			level: 0,
			want:  PanicLevel,
		},
		{
			name:  "Error",
			level: 1,
			want:  ErrorLevel,
		},
		{
			name:  "Warning",
			level: 2,
			want:  WarningLevel,
		},
		{
			name:  "Info",
			level: 3,
			want:  InfoLevel,
		},
		{
			name:  "Debug",
			level: 4,
			want:  DebugLevel,
		},
		{
			name:  "Default",
			level: 5,
			want:  DefaultLevel,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer SetLevel(DefaultLevel)
			go func() {
				time.Sleep(10 * time.Millisecond)
				SetLevel(tt.level)
			}()
			assert.Eventually(t, func() bool { return level == tt.want }, time.Second, 10*time.Millisecond)
		})
	}
}

func TestLog(t *testing.T) {
	type args struct {
		level Level
		v     []any
	}
	tests := []struct {
		name      string
		f         func(v ...any)
		args      args
		want      string
		wantPanic bool
	}{
		{
			name: "Panic",
			f:    Panic,
			args: args{
				level: PanicLevel,
				v:     []any{"panic ", errors.New("message")},
			},
			want:      "PANIC panic message\n",
			wantPanic: true,
		},
		{
			name: "Error",
			f:    Error,
			args: args{
				level: ErrorLevel,
				v:     []any{errors.New("error"), errors.New("message")},
			},
			want: "ERROR error message\n",
		},
		{
			name: "Warning",
			f:    Warning,
			args: args{
				level: WarningLevel,
				v:     []any{"warning ", errors.New("message")},
			},
			want: "WARNING warning message\n",
		},
		{
			name: "Info",
			f:    Info,
			args: args{
				level: InfoLevel,
				v:     []any{"info ", "message"},
			},
			want: "INFO info message\n",
		},
		{
			name: "Debug",
			f:    Debug,
			args: args{
				level: DebugLevel,
				v:     []any{"debug message"},
			},
			want: "DEBUG debug message\n",
		},
		{
			name: "Disabled",
			f:    Debug,
			args: args{
				level: InfoLevel,
				v:     []any{"debug message"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.args.level)
			defer SetLevel(DefaultLevel)

			builder := strings.Builder{}
			log.SetOutput(&builder)
			defer log.SetOutput(os.Stderr)

			log.SetFlags(0)
			defer log.SetFlags(log.LstdFlags)

			if tt.wantPanic {
				assert.Panics(t, func() { tt.f(tt.args.v...) })
			} else {
				tt.f(tt.args.v...)
			}
			assert.Equal(t, tt.want, builder.String())
		})
	}
}

func TestLogf(t *testing.T) {
	type args struct {
		level  Level
		format string
		v      []any
	}
	tests := []struct {
		name      string
		f         func(format string, v ...any)
		args      args
		want      string
		wantPanic bool
	}{
		{
			name: "Panicf",
			f:    Panicf,
			args: args{
				level:  PanicLevel,
				format: "%v %v",
				v:      []any{"panic", errors.New("message")},
			},
			want:      "PANIC panic message\n",
			wantPanic: true,
		},
		{
			name: "Errorf",
			f:    Errorf,
			args: args{
				level:  ErrorLevel,
				format: "%v %v",
				v:      []any{errors.New("error"), errors.New("message")},
			},
			want: "ERROR error message\n",
		},
		{
			name: "Warningf",
			f:    Warningf,
			args: args{
				level:  WarningLevel,
				format: "%v %v",
				v:      []any{"warning", errors.New("message")},
			},
			want: "WARNING warning message\n",
		},
		{
			name: "Infof",
			f:    Infof,
			args: args{
				level:  InfoLevel,
				format: "%v %v",
				v:      []any{"info", "message"},
			},
			want: "INFO info message\n",
		},
		{
			name: "Debugf",
			f:    Debugf,
			args: args{
				level:  DebugLevel,
				format: "%v",
				v:      []any{"debug message"},
			},
			want: "DEBUG debug message\n",
		},
		{
			name: "Disabled",
			f:    Debugf,
			args: args{
				level:  InfoLevel,
				format: "%v",
				v:      []any{"debug message"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.args.level)
			defer SetLevel(DefaultLevel)

			builder := strings.Builder{}
			log.SetOutput(&builder)
			defer log.SetOutput(os.Stderr)

			log.SetFlags(0)
			defer log.SetFlags(log.LstdFlags)

			if tt.wantPanic {
				assert.Panics(t, func() { tt.f(tt.args.format, tt.args.v...) })
			} else {
				tt.f(tt.args.format, tt.args.v...)
			}
			assert.Equal(t, tt.want, builder.String())
		})
	}
}

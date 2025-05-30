package server

import (
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"nginx-reaper/internal/log"
	"testing"
	_ "unsafe"
)

func Test_configHandler(t *testing.T) {
	type args struct {
		method string
		target string
	}
	type want struct {
		code int
		body string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "NotFound",
			args: args{
				method: http.MethodGet,
				target: "/",
			},
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name: "MethodNotAllowed",
			args: args{
				method: http.MethodGet,
				target: configPath,
			},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name: "BadRequest",
			args: args{
				method: http.MethodPut,
				target: configPath + "?" + keyLogLevel + "=xxx",
			},
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "NoContent",
			args: args{
				method: http.MethodPut,
				target: configPath + "?" + keyLogLevel + "=debug",
			},
			want: want{
				code: http.StatusNoContent,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := httptest.NewRecorder()
			configHandler(writer, httptest.NewRequest(tt.args.method, tt.args.target, nil))
			resp := writer.Result()
			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.code, resp.StatusCode)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}

//go:linkname logLevel internal/log.level
var logLevel log.Level

func Test_setLogLevel(t *testing.T) {
	type want struct {
		level   log.Level
		wantErr bool
	}
	tests := []struct {
		name string
		url  *url.URL
		want want
	}{
		{
			name: "Unset",
			url:  &url.URL{RawQuery: ""},
			want: want{
				level:   log.DefaultLevel,
				wantErr: true,
			},
		},
		{
			name: "Empty",
			url:  &url.URL{RawQuery: keyLogLevel + "="},
			want: want{
				level:   log.DefaultLevel,
				wantErr: true,
			},
		},
		{
			name: "Invalid",
			url:  &url.URL{RawQuery: keyLogLevel + "=xxx"},
			want: want{
				level:   log.DefaultLevel,
				wantErr: true,
			},
		},
		{
			name: "Panic",
			url:  &url.URL{RawQuery: keyLogLevel + "=panic"},
			want: want{
				level: log.PanicLevel,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer log.SetLevel(log.DefaultLevel)
			err := setLogLevel(&http.Request{URL: tt.url})

			if tt.want.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.want.level, log.DefaultLevel)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.level, logLevel)
			}
		})
	}
}

package server

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCreateServer(t *testing.T) {
	t.Run("DefaultServer", func(t *testing.T) {
		server := CreateServer(":12345")
		assert.Equal(t, ":12345", server.Addr)
		assert.NotNil(t, server.Handler)
		assert.Equal(t, 10*time.Second, server.ReadTimeout)
		assert.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
		assert.Equal(t, 10*time.Second, server.WriteTimeout)
		assert.Equal(t, 120*time.Second, server.IdleTimeout)
	})
}

func TestStartServer(t *testing.T) {
	type args struct {
		server *http.Server
		url    string
		method string
		body   io.Reader
	}
	type want struct {
		code      int
		body      string
		wantPanic bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Root",
			args: args{
				server: CreateServer(":11254"),
				url:    "http://localhost:11254/",
				method: http.MethodGet,
			},
			want: want{
				code: http.StatusOK,
				body: "ready",
			},
		},
		{
			name: "Failed",
			args: args{
				server: CreateServer(":-1"),
			},
			want: want{
				wantPanic: true,
			},
		},
		{
			name: "Healthz",
			args: args{
				server: CreateServer(":11255"),
				url:    "http://localhost:11255/healtz",
				method: http.MethodGet,
			},
			want: want{
				code: http.StatusOK,
				body: "ready",
			},
		},
		{
			name: "ConfigMethodNotAllowed",
			args: args{
				server: CreateServer(":11256"),
				url:    "http://localhost:11256" + configPath,
				method: http.MethodGet,
			},
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name: "ConfigBadRequest",
			args: args{
				server: CreateServer(":11256"),
				url:    "http://localhost:11256" + configPath,
				method: http.MethodPut,
			},
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name: "ConfigNoContent",
			args: args{
				server: CreateServer(":11256"),
				url:    "http://localhost:11256" + configPath + "?" + keyLogLevel + "=debug",
				method: http.MethodPut,
			},
			want: want{
				code: http.StatusNoContent,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want.wantPanic {
				assert.Panics(t, func() { StartServer(tt.args.server) })
				return
			}
			// Start the HTTP server as a goroutine.
			go StartServer(tt.args.server)
			defer func() { _ = tt.args.server.Close() }()
			time.Sleep(100 * time.Millisecond)

			// Send a request to the HTTP server.
			req, err := http.NewRequest(tt.args.method, tt.args.url, tt.args.body)
			assert.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.code, resp.StatusCode)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}

type ErrorWriter struct {
	mock.Mock
}

func (e *ErrorWriter) Header() http.Header {
	panic("not implemented")
}

func (e *ErrorWriter) Write([]byte) (int, error) {
	return 0, errors.New("ErrorWriter")
}

func (e *ErrorWriter) WriteHeader(int) {
	panic("not implemented")
}

func Test_defaultHandler(t *testing.T) {
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
			name: "GET",
			args: args{
				method: http.MethodGet,
				target: "/",
			},
			want: want{
				code: http.StatusOK,
				body: "ready",
			},
		},
		{
			name: "HEAD",
			args: args{
				method: http.MethodHead,
				target: "/healthz",
			},
			want: want{
				code: http.StatusOK,
				body: "ready",
			},
		},
		{
			name: "POST",
			args: args{
				method: http.MethodPost,
				target: "/xxx",
			},
			want: want{
				code: http.StatusOK,
				body: "ready",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := httptest.NewRecorder()
			defaultHandler(writer, httptest.NewRequest(tt.args.method, tt.args.target, nil))
			resp := writer.Result()
			body, err := io.ReadAll(resp.Body)
			assert.NoError(t, err)
			assert.Equal(t, tt.want.code, resp.StatusCode)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
	t.Run("WriteError", func(t *testing.T) {
		defaultHandler(&ErrorWriter{}, httptest.NewRequest("", "/", nil))
	})
}

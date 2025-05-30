package ticker

import (
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

type MockJob struct {
	mock.Mock
}

func (m *MockJob) Interval() time.Duration {
	args := m.Called()
	return time.Duration(args.Int(0))
}

func (m *MockJob) Run() bool {
	args := m.Called()
	return args.Bool(len(m.Calls) - 2)
}

func TestStart(t *testing.T) {
	runOnce := &MockJob{}
	runOnce.On("Interval").Return(1)
	runOnce.On("Run").Return(false)

	runTwice := &MockJob{}
	runTwice.On("Interval").Return(1)
	runTwice.On("Run").Return(true, false)

	type want struct {
		method string
		calls  int
	}
	tests := []struct {
		name string
		job  *MockJob
		want []want
	}{
		{
			name: "RunOnce",
			job:  runOnce,
			want: []want{
				{
					method: "Interval",
					calls:  1,
				},
				{
					method: "Run",
					calls:  1,
				},
			},
		},
		{
			name: "RunTwice",
			job:  runTwice,
			want: []want{
				{
					method: "Interval",
					calls:  1,
				},
				{
					method: "Run",
					calls:  2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Start(tt.job)
			tt.job.AssertExpectations(t)
			for _, call := range tt.want {
				tt.job.AssertNumberOfCalls(t, call.method, call.calls)
			}
		})
	}
}

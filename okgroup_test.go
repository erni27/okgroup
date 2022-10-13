package okgroup

import (
	"context"
	"errors"
	"testing"
	"time"
)

type Result string

type Executor struct {
	fn func() (Result, error)
}

func (e Executor) Execute() (Result, error) {
	return e.fn()
}

func TestWait_WithContext(t *testing.T) {
	err1, err2, err3 := errors.New("executor_1 failed"), errors.New("executor_2 failed"), errors.New("executor_3 failed")
	tests := []struct {
		name      string
		executors []Executor
		want      Result
		errors    []error
	}{
		{
			name: "only ok responses",
			executors: []Executor{
				{fn: func() (Result, error) { return "executor_1", nil }},
				{fn: func() (Result, error) { time.Sleep(time.Millisecond * 100); return "executor_2", nil }},
				{fn: func() (Result, error) { time.Sleep(time.Millisecond * 100); return "executor_3", nil }},
			},
			want: "executor_1",
		},
		{
			name: "1 ok response",
			executors: []Executor{
				{fn: func() (Result, error) { return "", err1 }},
				{fn: func() (Result, error) { return "", err2 }},
				{fn: func() (Result, error) { return "executor_3", nil }},
			},
			want: "executor_3",
		},
		{
			name: "only errors",
			executors: []Executor{
				{fn: func() (Result, error) { return "", err1 }},
				{fn: func() (Result, error) { return "", err2 }},
				{fn: func() (Result, error) { return "", err3 }},
			},
			want:   "",
			errors: []error{err1, err2, err3},
		},
	}
	for _, tc := range tests {
		g, ctx := WithContext[Result](context.Background())
		for _, executor := range tc.executors {
			g.Go(executor.Execute)
		}
		got, err := g.Wait()
		if (err != nil) != (len(tc.errors) > 0) {
			t.Fatalf("want nil err, got %v", err)
		}
		for _, wanterr := range tc.errors {
			if !errors.Is(err, wanterr) {
				t.Errorf("got err %v, want err %v", err, wanterr)
			}
		}
		if got != tc.want {
			t.Errorf("got %v, want %v", got, tc.want)
		}
		select {
		case <-ctx.Done():
		default:
			t.Errorf("want ctx canceled")
		}
	}
}

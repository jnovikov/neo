package queue

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"testing"
	"time"

	"neo/pkg/testutils"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSimpleQueue_Add(t *testing.T) {
	makeQueue := func(s int) *simpleQueue {
		q := NewSimpleQueue(1).(*simpleQueue)
		q.c = make(chan Task, s)
		return q
	}
	for _, tc := range []struct {
		q        *simpleQueue
		t        *Task
		wantTask *Task
		wantErr  error
	}{
		{q: makeQueue(100), t: &Task{executable: "1"}, wantErr: nil},
		{q: makeQueue(1), t: &Task{executable: "1"}, wantErr: nil},
		{q: makeQueue(0), t: &Task{executable: "1"}, wantErr: ErrQueueFull},
	} {
		err := tc.q.Add(*tc.t)
		if !errors.Is(err, tc.wantErr) {
			t.Errorf("simpleQueue.Add(): got error = %v, want = %v", err, tc.wantErr)
			continue
		}
		if err != nil {
			continue
		}
		if tk := <-tc.q.c; tk.executable != tc.t.executable {
			t.Errorf("simpleQueue.Add(): got unexpected data = %v, want = %v", tk, tc.t)
		}
	}
}

func TestSimpleQueue_runExploit(t *testing.T) {
	closedQueue := func() *simpleQueue {
		q := NewSimpleQueue(1).(*simpleQueue)
		q.Stop()
		return q
	}
	for _, tc := range []struct {
		q        *simpleQueue
		t        Task
		ctx      context.Context
		wantErr  error
		wantData []byte
	}{
		{
			q:        NewSimpleQueue(1).(*simpleQueue),
			t:        Task{name: "echo", executable: "echo", teamID: "id", teamIP: "ip", timeout: time.Second * 5},
			wantErr:  nil,
			wantData: []byte("ip\n"),
			ctx:      context.Background(),
		},
		{
			q:        NewSimpleQueue(1).(*simpleQueue),
			t:        Task{name: "bad executable", executable: "notfoundcli", teamID: "id", teamIP: "ip", timeout: time.Second * 5},
			wantErr:  &exec.Error{},
			wantData: nil,
			ctx:      context.Background(),
		},
		{
			q:        NewSimpleQueue(1).(*simpleQueue),
			t:        Task{name: "cancelled ctx", executable: "echo", teamID: "id", teamIP: "ip", timeout: time.Second * 5},
			wantErr:  context.Canceled,
			wantData: nil,
			ctx:      testutils.CanceledContext(),
		},
		{
			q:        NewSimpleQueue(1).(*simpleQueue),
			t:        Task{name: "timed out ctx", executable: "echo", teamID: "id", teamIP: "ip", timeout: time.Second * 5},
			wantErr:  context.DeadlineExceeded,
			wantData: nil,
			ctx:      testutils.TimedOutContext(),
		},
		{
			q:        NewSimpleQueue(1).(*simpleQueue),
			t:        Task{name: "zero timeout", executable: "echo", teamID: "id", teamIP: "ip", timeout: time.Second * 0},
			wantErr:  context.DeadlineExceeded,
			wantData: nil,
			ctx:      context.Background(),
		},
		{
			q:        closedQueue(),
			t:        Task{name: "closed queue", executable: "echo", teamID: "id", teamIP: "ip", timeout: time.Second * 5},
			wantErr:  context.Canceled,
			wantData: nil,
			ctx:      context.Background(),
		},
	} {
		data, err := tc.q.runExploit(tc.ctx, tc.t)
		cmpErr := func(e1, e2 error) bool {
			if errors.Is(e1, e2) {
				return true
			}
			return reflect.TypeOf(e1) == reflect.TypeOf(e2)
		}
		if !cmpErr(err, tc.wantErr) {
			t.Errorf("simpleQueue.runExploit() [%s]: got unexpected err = %v, want = %v", tc.t.name, err, tc.wantErr)
		}

		if diff := cmp.Diff(data, tc.wantData, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("simpleQueue.runExploit() returned data mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestSimpleQueue_Start(t *testing.T) {
	q := NewSimpleQueue(10)
	task := Task{name: "kek", executable: "echo", dir: "", teamID: "id", teamIP: "ip", timeout: time.Second * 2}
	if err := q.Add(task); err != nil {
		t.Errorf("simpleQueue.Add(): got unexpected error = %v", err)
	}

	var out *Output
	ctx, cancel := context.WithCancel(context.Background())
	q.Start(ctx)
	defer q.Stop()
	out = <-q.Results()
	cancel()

	if out.Name != task.name {
		t.Errorf("simpleQueue.Start(): got unexpected result name: got = %v, want = %v", out.Name, task.name)
	}
	if out.Team != task.teamID {
		t.Errorf("simpleQueue.Start(): got unexpected result team: got = %v, want = %v", out.Team, task.teamID)
	}
	if string(out.Out) != task.teamIP+"\n" {
		t.Errorf("simpleQueue.Start(): got unexpected result: got = %v, want = %v", out.Out, task.teamIP+"\n")
	}
}

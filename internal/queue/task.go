package queue

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func NewTask(taskID, executable, dir, teamID, teamIP string, environ []string, timeout time.Duration) Task {
	return Task{
		name:       taskID,
		executable: executable,
		dir:        dir,
		teamID:     teamID,
		teamIP:     teamIP,
		timeout:    timeout,
		environ:    environ,
	}
}

type Task struct {
	name       string
	executable string
	dir        string
	teamID     string
	teamIP     string
	environ    []string
	timeout    time.Duration
}

func (et Task) String() string {
	return fmt.Sprintf(
		"Exploit(path=%s, target=%s (%s), timeout=%v, environ=%+v)",
		et.executable,
		et.teamID,
		et.teamIP,
		et.timeout,
		et.environ,
	)
}

func (et Task) Command(ctx context.Context) *exec.Cmd {
	cmd := exec.CommandContext(ctx, et.executable, et.teamIP)
	if et.dir != "" {
		cmd.Dir = et.dir
	}

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, et.environ...)

	// disable buffering in python scripts
	cmd.Env = append(cmd.Env, "PYTHONUNBUFFERED=1")
	return cmd
}

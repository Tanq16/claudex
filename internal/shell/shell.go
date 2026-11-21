package shell

import (
	"bytes"
	"cmp"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	Timeout   = 60 * time.Second
	MaxOutput = 1 << 20
)

type Result struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exitCode"`
	DurationMs int64  `json:"durationMs"`
	TimedOut   bool   `json:"timedOut"`
	Truncated  bool   `json:"truncated"`
}

func Run(ctx context.Context, dir string, env []string, command string) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmp.Or(os.Getenv("SHELL"), "/bin/sh"), "-c", command)
	cmd.Dir = dir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
	var stdout, stderr capped
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	start := time.Now()
	if err := cmd.Run(); cmd.ProcessState == nil {
		return Result{}, err
	}
	return Result{
		Stdout:     strings.ToValidUTF8(stdout.buf.String(), "�"),
		Stderr:     strings.ToValidUTF8(stderr.buf.String(), "�"),
		ExitCode:   cmd.ProcessState.ExitCode(),
		DurationMs: time.Since(start).Milliseconds(),
		TimedOut:   errors.Is(ctx.Err(), context.DeadlineExceeded),
		Truncated:  stdout.over || stderr.over,
	}, nil
}

type capped struct {
	buf  bytes.Buffer
	over bool
}

func (c *capped) Write(p []byte) (int, error) {
	room := MaxOutput - c.buf.Len()
	if len(p) > room {
		c.over = true
		c.buf.Write(p[:max(room, 0)])
		return len(p), nil
	}
	return c.buf.Write(p)
}

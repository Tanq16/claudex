package session

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const Name = "claudex"

var ErrNotRunning = errors.New("no session is running")

type Window struct {
	PID  int
	Dead bool
}

func tmux(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return "", fmt.Errorf("tmux %s: %s: %w", args[0], detail, err)
		}
		return "", fmt.Errorf("tmux %s: %w", args[0], err)
	}
	return string(out), nil
}

func hasSession() (bool, error) {
	err := exec.Command("tmux", "has-session", "-t", "="+Name).Run()
	if err == nil {
		return true, nil
	}
	if _, exited := errors.AsType[*exec.ExitError](err); exited {
		return false, nil
	}
	return false, err
}

func target(window string) string {
	return "=" + Name + ":=" + window
}

func Windows() (map[string]Window, error) {
	windows := map[string]Window{}
	ok, err := hasSession()
	if err != nil || !ok {
		return windows, err
	}
	out, err := tmux("list-windows", "-t", "="+Name, "-F", "#{window_name}\t#{pane_pid}\t#{pane_dead}")
	if err != nil {
		return windows, err
	}
	for line := range strings.Lines(out) {
		fields := strings.Split(strings.TrimRight(line, "\n"), "\t")
		if len(fields) != 3 {
			continue
		}
		pid, _ := strconv.Atoi(fields[1])
		windows[fields[0]] = Window{PID: pid, Dead: fields[2] == "1"}
	}
	return windows, nil
}

func Active(windows map[string]Window, name string) bool {
	w, ok := windows[name]
	return ok && !w.Dead
}

func newWindow(name, dir string, argv []string) error {
	ok, err := hasSession()
	if err != nil {
		return err
	}
	args := []string{"new-session", "-d", "-s", Name, "-n", name, "-c", dir, "--"}
	if ok {
		args = []string{"new-window", "-d", "-t", "=" + Name + ":", "-n", name, "-c", dir, "--"}
	}
	_, err = tmux(append(args, argv...)...)
	return err
}

func respawn(name, dir string, argv []string) error {
	_, err := tmux(append([]string{"respawn-pane", "-k", "-t", target(name), "-c", dir, "--"}, argv...)...)
	return err
}

func kill(name string) error {
	_, err := tmux("kill-window", "-t", target(name))
	return err
}

func Stop(name string) error {
	windows, err := Windows()
	if err != nil {
		return err
	}
	w, ok := windows[name]
	if !ok {
		return ErrNotRunning
	}
	if err := kill(name); err != nil {
		return err
	}
	waitExit(w.PID)
	return nil
}

func waitExit(pid int) {
	if pid <= 0 {
		return
	}
	for range 50 {
		if syscall.Kill(pid, 0) != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func StartTimes(pids []int) map[int]time.Time {
	times := map[int]time.Time{}
	if len(pids) == 0 {
		return times
	}
	list := make([]string, len(pids))
	for i, pid := range pids {
		list[i] = strconv.Itoa(pid)
	}
	cmd := exec.Command("ps", "-o", "pid=,lstart=", "-p", strings.Join(list, ","))
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, _ := cmd.Output()
	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		t, err := time.ParseInLocation("Mon Jan _2 15:04:05 2006", strings.Join(fields[1:], " "), time.Local)
		if err == nil {
			times[pid] = t
		}
	}
	return times
}

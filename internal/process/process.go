package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fnune/pocket-tunnel/internal/app"
	"github.com/fnune/pocket-tunnel/internal/tunnel"
)

type PIDFile struct {
	path   string
	marker string
}

func NewPIDFile(stateDir string) *PIDFile {
	return &PIDFile{path: filepath.Join(stateDir, "tunnel.pid"), marker: stateDir}
}

var _ app.PIDLock = (*PIDFile)(nil)

func (p *PIDFile) Held() (int, bool) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 || syscall.Kill(pid, 0) != nil {
		return 0, false
	}
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil || !strings.Contains(string(cmdline), p.marker) {
		return 0, false
	}
	return pid, true
}

func (p *PIDFile) Acquire() error {
	if err := os.WriteFile(p.path, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return fmt.Errorf("write %s: %w", p.path, err)
	}
	return nil
}

func (p *PIDFile) Release() {
	_ = os.Remove(p.path)
}

type Tunnel struct {
	command []string
	logFile string
	iface   string
	pid     *PIDFile
}

func NewTunnel(command []string, logFile, iface string, pid *PIDFile) *Tunnel {
	return &Tunnel{command: command, logFile: logFile, iface: iface, pid: pid}
}

var _ app.Tunnel = (*Tunnel)(nil)

func (t *Tunnel) Running() bool {
	_, held := t.pid.Held()
	return held
}

func (t *Tunnel) Start() error {
	if t.Running() {
		return nil
	}
	logFile, err := os.OpenFile(t.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open %s: %w", t.logFile, err)
	}
	defer func() { _ = logFile.Close() }()

	cmd := exec.Command(t.command[0], t.command[1:]...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", t.command[0], err)
	}
	return cmd.Process.Release()
}

func (t *Tunnel) Stop() error {
	pid, held := t.pid.Held()
	if !held {
		t.pid.Release()
		return nil
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal PID %d: %w", pid, err)
	}
	for range 50 {
		if syscall.Kill(pid, 0) != nil {
			t.pid.Release()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("the tunnel process did not exit within 5 seconds")
}

func (t *Tunnel) LastHandshake() (time.Time, error) {
	st, err := tunnel.Query(t.iface)
	if err != nil {
		return time.Time{}, err
	}
	return st.LastHandshake, nil
}

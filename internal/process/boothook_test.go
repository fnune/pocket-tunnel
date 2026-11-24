package process

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const otherHooks = "#!/bin/sh\n\ntest -f ssh-on-boot && ssh-on-boot # SSH Server.pak-on-boot\n"

func hook(t *testing.T) *BootHook {
	return NewBootHook(
		filepath.Join(t.TempDir(), "auto.sh"),
		[]string{"/mnt/SDCARD/Tools/tg5040/Pocket Tunnel.pak/pocket-tunnel", "up", "--state-dir", "/mnt/SDCARD/.userdata/tg5040/Pocket Tunnel"},
		"/mnt/SDCARD/.userdata/tg5040/logs/Pocket Tunnel-tunnel.txt",
	)
}

func TestEnablingStartAtBootTwiceAddsOneHook(t *testing.T) {
	b := hook(t)
	if err := os.WriteFile(b.autoSh, []byte(otherHooks), 0755); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := b.Enable(); err != nil {
			t.Fatal(err)
		}
	}
	content, _ := os.ReadFile(b.autoSh)
	if n := strings.Count(string(content), markerStart); n != 1 {
		t.Errorf("hooks = %d, want 1:\n%s", n, content)
	}
	if !strings.Contains(string(content), "'up' '--state-dir' '/mnt/SDCARD/.userdata/tg5040/Pocket Tunnel'") {
		t.Errorf("the hook does not start the tunnel:\n%s", content)
	}
}

func TestDisablingStartAtBootKeepsOtherPaksHooks(t *testing.T) {
	b := hook(t)
	if err := os.WriteFile(b.autoSh, []byte(otherHooks), 0755); err != nil {
		t.Fatal(err)
	}
	if err := b.Enable(); err != nil {
		t.Fatal(err)
	}
	if err := b.Disable(); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(b.autoSh)
	if string(content) != otherHooks {
		t.Errorf("auto.sh = %q, want %q", content, otherHooks)
	}
}

func TestEnablingStartAtBootCreatesAutoShWhenNoPakHasOne(t *testing.T) {
	b := hook(t)
	if err := b.Enable(); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(b.autoSh)
	if !strings.HasPrefix(string(content), "#!/bin/sh\n") {
		t.Errorf("auto.sh lacks a shebang:\n%s", content)
	}
}

func TestAStalePIDFileDoesNotCountAsRunning(t *testing.T) {
	dir := t.TempDir()
	p := NewPIDFile(dir)
	if err := os.WriteFile(filepath.Join(dir, "tunnel.pid"), []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, held := p.Held(); held {
		t.Error("PID 1 counted as the tunnel although its command line does not name the state dir")
	}
}

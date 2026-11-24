package app

import (
	"strings"
	"testing"
	"time"
)

func newMenu(ui *fakeUI, store *fakeStore, tunnel *fakeTunnel, boot *fakeBootHook, clock *fakeClock) *Menu {
	return NewMenu(ui, store, tunnel, boot, &fakeNetwork{online: true}, clock)
}

func shown(item MenuItem) string {
	return item.Options[item.Selected]
}

func TestTurningOnAnUnconfiguredTunnelNamesTheMissingFieldsInsteadOfStarting(t *testing.T) {
	ui := &fakeUI{choices: []int{itemTunnel}}
	tunnel := &fakeTunnel{}
	if err := newMenu(ui, &fakeStore{cfg: template()}, tunnel, &fakeBootHook{}, &fakeClock{}).Run(); err != nil {
		t.Fatal(err)
	}
	if tunnel.starts != 0 {
		t.Error("an unconfigured tunnel was started")
	}
	if len(ui.messages) != 1 || !strings.Contains(ui.messages[0], "Peer Endpoint") || !strings.Contains(ui.messages[0], "/state/wg0.conf") {
		t.Errorf("messages = %q, want the missing fields and the config path", ui.messages)
	}
	if got := shown(ui.menus[0][itemStatus]); got != "Not configured" {
		t.Errorf("status = %q", got)
	}
}

func TestTurningOnAConfiguredTunnelStartsItAndShowsItAsEnabled(t *testing.T) {
	ui := &fakeUI{choices: []int{itemTunnel}}
	tunnel := &fakeTunnel{}
	if err := newMenu(ui, &fakeStore{cfg: complete()}, tunnel, &fakeBootHook{}, &fakeClock{}).Run(); err != nil {
		t.Fatal(err)
	}
	if tunnel.starts != 1 {
		t.Fatalf("starts = %d, want 1", tunnel.starts)
	}
	if got := shown(ui.menus[1][itemTunnel]); got != "Enabled" {
		t.Errorf("tunnel item after starting = %q", got)
	}
}

func TestConnectingShowsProgressUntilTheFirstHandshakeInsteadOfABlankScreen(t *testing.T) {
	now := time.Date(2026, 11, 24, 18, 0, 0, 0, time.UTC)
	clock := &fakeClock{now: now}
	tunnel := &fakeTunnel{handshakeAfter: 2 * time.Second, clock: clock}
	ui := &fakeUI{choices: []int{itemTunnel}}
	if err := newMenu(ui, &fakeStore{cfg: complete()}, tunnel, &fakeBootHook{}, clock).Run(); err != nil {
		t.Fatal(err)
	}
	if len(ui.progress) != 1 || ui.progress[0] != "Connecting" {
		t.Errorf("progress = %q, want Connecting", ui.progress)
	}
	if ui.progressOpen {
		t.Error("the progress message is still on screen when the menu comes back")
	}
	if waited := clock.now.Sub(now); waited > 3*time.Second {
		t.Errorf("waited %s after the handshake arrived at 2s", waited)
	}
	if got := shown(ui.menus[1][itemStatus]); got != "Connected" {
		t.Errorf("status after connecting = %q", got)
	}
}

func TestStatusIsConnectedOnlyWithARecentHandshake(t *testing.T) {
	now := time.Date(2026, 11, 24, 18, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name      string
		handshake time.Time
		want      string
		color     string
	}{
		{"no handshake yet", time.Time{}, "Connecting", colorProgress},
		{"handshake a minute ago", now.Add(-time.Minute), "Connected", colorSuccess},
		{"handshake ten minutes ago", now.Add(-10 * time.Minute), "Connecting", colorProgress},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ui := &fakeUI{}
			tunnel := &fakeTunnel{running: true, handshake: tc.handshake}
			if err := newMenu(ui, &fakeStore{cfg: complete()}, tunnel, &fakeBootHook{}, &fakeClock{now: now}).Run(); err != nil {
				t.Fatal(err)
			}
			status := ui.menus[0][itemStatus]
			if got := shown(status); got != tc.want || status.BackgroundColor != tc.color {
				t.Errorf("status = %q in %s, want %q in %s", got, status.BackgroundColor, tc.want, tc.color)
			}
		})
	}
}

func TestAutostartTogglesTheBootHook(t *testing.T) {
	ui := &fakeUI{choices: []int{itemAutostart, itemAutostart}}
	boot := &fakeBootHook{}
	if err := newMenu(ui, &fakeStore{cfg: complete()}, &fakeTunnel{}, boot, &fakeClock{}).Run(); err != nil {
		t.Fatal(err)
	}
	if boot.enabled {
		t.Error("boot hook still enabled after toggling twice")
	}
	if got := shown(ui.menus[1][itemAutostart]); got != "Enabled" {
		t.Errorf("after the first toggle = %q", got)
	}
}

func TestThePublicKeyIsSplitAcrossTwoLinesToFitTheScreen(t *testing.T) {
	ui := &fakeUI{choices: []int{itemPublicKey}}
	if err := newMenu(ui, &fakeStore{cfg: template()}, &fakeTunnel{}, &fakeBootHook{}, &fakeClock{}).Run(); err != nil {
		t.Fatal(err)
	}
	if len(ui.messages) != 1 {
		t.Fatalf("messages = %q", ui.messages)
	}
	message := ui.messages[0]
	if !strings.Contains(message, "u/UQQiP5I9hrf9r67hSrw7\n0bm33504B4uVEpP2gF+GQ=") {
		t.Errorf("message does not split the key in two lines:\n%s", message)
	}
	for _, line := range strings.Split(message, "\n") {
		if len(line) > 30 && !strings.Contains(line, "/") {
			t.Errorf("line too long for the screen: %q", line)
		}
	}
}

func TestStatusSaysNoNetworkWhenTheDeviceIsOffline(t *testing.T) {
	ui := &fakeUI{}
	menu := NewMenu(ui, &fakeStore{cfg: complete()}, &fakeTunnel{running: true}, &fakeBootHook{}, &fakeNetwork{online: false}, &fakeClock{})
	if err := menu.Run(); err != nil {
		t.Fatal(err)
	}
	status := ui.menus[0][itemStatus]
	if got := shown(status); got != "No network" || status.BackgroundColor != colorSecondary {
		t.Errorf("status = %q in %s, want No network in grey", got, status.BackgroundColor)
	}
}

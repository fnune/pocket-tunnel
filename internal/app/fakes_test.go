package app

import (
	"context"
	"strings"
	"time"

	"github.com/fnune/pocket-tunnel/internal/conf"
)

type fakeUI struct {
	choices      []int
	menus        [][]MenuItem
	messages     []string
	progress     []string
	progressOpen bool
}

func (f *fakeUI) ShowProgress(text string) error {
	f.progress = append(f.progress, text)
	f.progressOpen = true
	return nil
}

func (f *fakeUI) CloseProgress() { f.progressOpen = false }

func (f *fakeUI) Menu(_ string, items []MenuItem) (int, error) {
	f.menus = append(f.menus, items)
	if len(f.choices) == 0 {
		return -1, ErrBack
	}
	choice := f.choices[0]
	f.choices = f.choices[1:]
	return choice, nil
}

func (f *fakeUI) Message(text string) error {
	f.messages = append(f.messages, text)
	return nil
}

type fakeStore struct {
	cfg *conf.Config
}

func (f *fakeStore) Load() (*conf.Config, error) { return f.cfg, nil }
func (f *fakeStore) ConfigPath() string          { return "/state/wg0.conf" }
func (f *fakeStore) PublicKeyPath() string       { return "/state/public.key" }

type fakeTunnel struct {
	running        bool
	handshake      time.Time
	starts         int
	handshakeAfter time.Duration
	clock          *fakeClock
}

func (f *fakeTunnel) Running() bool { return f.running }
func (f *fakeTunnel) Start() error {
	f.starts++
	f.running = true
	if f.clock != nil {
		f.handshake = f.clock.now.Add(f.handshakeAfter)
	}
	return nil
}
func (f *fakeTunnel) Stop() error {
	f.running = false
	return nil
}
func (f *fakeTunnel) LastHandshake() (time.Time, error) {
	if f.clock != nil && f.clock.now.Before(f.handshake) {
		return time.Time{}, nil
	}
	return f.handshake, nil
}

type fakeBootHook struct{ enabled bool }

func (f *fakeBootHook) Enabled() bool  { return f.enabled }
func (f *fakeBootHook) Enable() error  { f.enabled = true; return nil }
func (f *fakeBootHook) Disable() error { f.enabled = false; return nil }

type fakeClock struct{ now time.Time }

func (f *fakeClock) Now() time.Time        { return f.now }
func (f *fakeClock) Sleep(d time.Duration) { f.now = f.now.Add(d) }

type fakeLock struct {
	heldBy   int
	acquired bool
	released bool
}

func (f *fakeLock) Held() (int, bool) { return f.heldBy, f.heldBy != 0 }
func (f *fakeLock) Acquire() error    { f.acquired = true; return nil }
func (f *fakeLock) Release()          { f.released = true }

type fakeRunner struct{ ran *conf.Config }

func (f *fakeRunner) Run(_ context.Context, cfg *conf.Config) error {
	f.ran = cfg
	return nil
}

func complete() *conf.Config {
	cfg, err := conf.Parse(strings.NewReader(`[Interface]
PrivateKey = AGLuYofEZv6kppTTyYs4baKYFi1sV65heniL1Tcg+WQ=
Address = 10.100.0.7/32
[Peer]
PublicKey = aVdLGe76+qHh25zQBy3zfb3qp35IeZU4DWZvrowUX0g=
Endpoint = vpn.example.com:51820
AllowedIPs = 10.100.0.0/24
`))
	if err != nil {
		panic(err)
	}
	return cfg
}

func template() *conf.Config {
	cfg, err := conf.Parse(strings.NewReader(conf.Template("AGLuYofEZv6kppTTyYs4baKYFi1sV65heniL1Tcg+WQ=")))
	if err != nil {
		panic(err)
	}
	return cfg
}

type fakeNetwork struct{ online bool }

func (f *fakeNetwork) Online() bool { return f.online }

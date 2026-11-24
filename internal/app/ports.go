package app

import (
	"context"
	"errors"
	"time"

	"github.com/fnune/pocket-tunnel/internal/conf"
)

type MenuItem struct {
	Label           string
	Options         []string
	Selected        int
	Unselectable    bool
	BackgroundColor string
	ConfirmText     string
}

var ErrBack = errors.New("back")

type UI interface {
	Menu(title string, items []MenuItem) (int, error)
	Message(text string) error
	ShowProgress(text string) error
	CloseProgress()
}

type ConfigStore interface {
	Load() (*conf.Config, error)
	ConfigPath() string
	PublicKeyPath() string
}

type Tunnel interface {
	Running() bool
	Start() error
	Stop() error
	LastHandshake() (time.Time, error)
}

type Network interface {
	Online() bool
}

type BootHook interface {
	Enabled() bool
	Enable() error
	Disable() error
}

type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type PIDLock interface {
	Held() (pid int, held bool)
	Acquire() error
	Release()
}

type TunnelRunner interface {
	Run(ctx context.Context, cfg *conf.Config) error
}

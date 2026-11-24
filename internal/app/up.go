package app

import (
	"context"
	"fmt"
	"strings"
)

type Up struct {
	store  ConfigStore
	lock   PIDLock
	runner TunnelRunner
}

func NewUp(store ConfigStore, lock PIDLock, runner TunnelRunner) *Up {
	return &Up{store: store, lock: lock, runner: runner}
}

func (u *Up) Run(ctx context.Context) error {
	if pid, held := u.lock.Held(); held {
		return fmt.Errorf("the tunnel is already running as PID %d", pid)
	}
	cfg, err := u.store.Load()
	if err != nil {
		return err
	}
	if missing := cfg.Missing(); len(missing) > 0 {
		return fmt.Errorf("%s is missing %s", u.store.ConfigPath(), strings.Join(missing, ", "))
	}
	if err := u.lock.Acquire(); err != nil {
		return err
	}
	defer u.lock.Release()
	return u.runner.Run(ctx, cfg)
}

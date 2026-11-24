package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fnune/pocket-tunnel/internal/app"
	"github.com/fnune/pocket-tunnel/internal/conf"
)

type Store struct {
	dir        string
	iface      string
	randomness io.Reader
}

func New(dir, iface string, randomness io.Reader) *Store {
	return &Store{dir: dir, iface: iface, randomness: randomness}
}

var _ app.ConfigStore = (*Store)(nil)

func (s *Store) ConfigPath() string {
	return filepath.Join(s.dir, s.iface+".conf")
}

func (s *Store) PublicKeyPath() string {
	return filepath.Join(s.dir, "public.key")
}

func (s *Store) Load() (*conf.Config, error) {
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return nil, fmt.Errorf("create %s: %w", s.dir, err)
	}
	if err := s.writeTemplateIfMissing(); err != nil {
		return nil, err
	}

	f, err := os.Open(s.ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", s.ConfigPath(), err)
	}
	defer func() { _ = f.Close() }()
	cfg, err := conf.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", s.ConfigPath(), err)
	}

	if public, err := cfg.PublicKey(); err == nil {
		if err := s.savePublicKey(public); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

func (s *Store) savePublicKey(public string) error {
	content := public + "\n"
	if saved, err := os.ReadFile(s.PublicKeyPath()); err == nil && string(saved) == content {
		return nil
	}
	if err := os.WriteFile(s.PublicKeyPath(), []byte(content), 0644); err != nil {
		return fmt.Errorf("write %s: %w", s.PublicKeyPath(), err)
	}
	return nil
}

func (s *Store) writeTemplateIfMissing() error {
	_, err := os.Stat(s.ConfigPath())
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("check %s: %w", s.ConfigPath(), err)
	}
	key, err := conf.GenerateKey(s.randomness)
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.ConfigPath(), []byte(conf.Template(key)), 0600); err != nil {
		return fmt.Errorf("write %s: %w", s.ConfigPath(), err)
	}
	return nil
}

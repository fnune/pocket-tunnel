package store

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestTheFirstLoadGeneratesAPrivateKeyAndSavesItsPublicKey(t *testing.T) {
	s := New(t.TempDir(), "wg0", bytes.NewReader(make([]byte, 32)))
	cfg, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	public, err := cfg.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(s.PublicKeyPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(saved)) != public {
		t.Errorf("public.key = %q, want %q", saved, public)
	}
	info, err := os.Stat(s.ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("wg0.conf mode = %v, want 0600 because it holds the private key", mode)
	}
}

func TestAConfigTheUserProvidedIsUsedAsIs(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, "wg0", bytes.NewReader(nil))
	provided := "[Interface]\nPrivateKey = AGLuYofEZv6kppTTyYs4baKYFi1sV65heniL1Tcg+WQ=\nAddress = 10.100.0.7/32\n"
	if err := os.WriteFile(s.ConfigPath(), []byte(provided), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(s.ConfigPath())
	if string(after) != provided {
		t.Errorf("the provided config was rewritten:\n%s", after)
	}
}

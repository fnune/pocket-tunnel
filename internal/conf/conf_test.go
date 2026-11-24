package conf

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
)

const bilboStyle = `[Interface]
PrivateKey = AGLuYofEZv6kppTTyYs4baKYFi1sV65heniL1Tcg+WQ=
Address = 10.100.0.7/32
DNS = 1.1.1.1

[Peer]
PublicKey = aVdLGe76+qHh25zQBy3zfb3qp35IeZU4DWZvrowUX0g=
Endpoint = wireguard.example.com:51820 # the router forwards this
AllowedIPs = 10.100.0.0/24, 10.200.0.0/24
PersistentKeepalive = 25
`

func TestParseReadsAWgQuickStyleConfigAndIgnoresComments(t *testing.T) {
	cfg, err := Parse(strings.NewReader(bilboStyle))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Peers) != 1 {
		t.Fatalf("peers = %d, want 1", len(cfg.Peers))
	}
	p := cfg.Peers[0]
	if p.Endpoint != "wireguard.example.com:51820" {
		t.Errorf("endpoint = %q", p.Endpoint)
	}
	if !slices.Equal(p.AllowedIPs, []string{"10.100.0.0/24", "10.200.0.0/24"}) {
		t.Errorf("allowed IPs = %v", p.AllowedIPs)
	}
	if p.Keepalive != 25 {
		t.Errorf("keepalive = %d", p.Keepalive)
	}
	if missing := cfg.Missing(); len(missing) != 0 {
		t.Errorf("missing = %v, want none", missing)
	}
}

func TestAFreshTemplateHasAKeyButIsMissingTheFieldsOnlyTheUserKnows(t *testing.T) {
	key, err := GenerateKey(bytes.NewReader(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(strings.NewReader(Template(key)))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PrivateKey != key {
		t.Errorf("private key not kept in the template")
	}
	want := []string{"Interface Address", "Peer PublicKey", "Peer Endpoint", "Peer AllowedIPs"}
	if got := cfg.Missing(); !slices.Equal(got, want) {
		t.Errorf("missing = %v, want %v", got, want)
	}
}

func TestPublicKeyMatchesWgPubkey(t *testing.T) {
	got, err := PublicKey("AGLuYofEZv6kppTTyYs4baKYFi1sV65heniL1Tcg+WQ=")
	if err != nil {
		t.Fatal(err)
	}
	if want := "u/UQQiP5I9hrf9r67hSrw70bm33504B4uVEpP2gF+GQ="; got != want {
		t.Errorf("public key = %s, want %s", got, want)
	}
}

func TestUAPIUsesHexKeysAndTheResolvedEndpoint(t *testing.T) {
	cfg, err := Parse(strings.NewReader(bilboStyle))
	if err != nil {
		t.Fatal(err)
	}
	settings, err := cfg.UAPI(func(string) (string, error) { return "192.0.2.1:51820", nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"private_key=0062ee6287c466fea4a694d3c98b386da298162d6c57ae617a788bd53720f964\n",
		"endpoint=192.0.2.1:51820\n",
		"persistent_keepalive_interval=25\n",
		"allowed_ip=10.200.0.0/24\n",
	} {
		if !strings.Contains(settings, want) {
			t.Errorf("settings lack %q:\n%s", want, settings)
		}
	}
}

func TestAnUnresolvableEndpointIsAResolveErrorSoTheTunnelRetries(t *testing.T) {
	cfg, err := Parse(strings.NewReader(bilboStyle))
	if err != nil {
		t.Fatal(err)
	}
	_, err = cfg.UAPI(func(string) (string, error) { return "", errors.New("no network yet") })
	var resolveErr *ResolveError
	if !errors.As(err, &resolveErr) {
		t.Fatalf("err = %v, want a ResolveError", err)
	}
}

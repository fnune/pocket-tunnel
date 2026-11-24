package tunnel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fnune/pocket-tunnel/internal/conf"
)

func TestParseStatusTakesTheNewestHandshakeAcrossPeers(t *testing.T) {
	reply := "private_key=00\nlisten_port=51820\npublic_key=aa\nlast_handshake_time_sec=1700000000\nlast_handshake_time_nsec=0\npublic_key=bb\nlast_handshake_time_sec=1700000100\nerrno=0\n\n"
	st, err := ParseStatus(strings.NewReader(reply))
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Unix(1700000100, 0); !st.LastHandshake.Equal(want) {
		t.Errorf("last handshake = %v, want %v", st.LastHandshake, want)
	}
}

func TestParseStatusReportsNoHandshakeBeforeTheFirstOne(t *testing.T) {
	st, err := ParseStatus(strings.NewReader("public_key=aa\nlast_handshake_time_sec=0\nerrno=0\n\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !st.LastHandshake.IsZero() {
		t.Errorf("last handshake = %v, want none", st.LastHandshake)
	}
}

func TestTheTunnelWaitsForTheEndpointToResolveBecauseWiFiMayNotBeUpAtBoot(t *testing.T) {
	cfg, err := conf.Parse(strings.NewReader("[Interface]\nPrivateKey = AGLuYofEZv6kppTTyYs4baKYFi1sV65heniL1Tcg+WQ=\nAddress = 10.100.0.7/32\n[Peer]\nPublicKey = aVdLGe76+qHh25zQBy3zfb3qp35IeZU4DWZvrowUX0g=\nEndpoint = vpn.example.com:51820\nAllowedIPs = 10.100.0.0/24\n"))
	if err != nil {
		t.Fatal(err)
	}
	attempts, waits := 0, 0
	resolve := func(string) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("no such host")
		}
		return "192.0.2.1:51820", nil
	}
	wait := func(context.Context, time.Duration) error { waits++; return nil }
	r := NewRunner("wg0", resolve, nil, wait)

	settings, err := r.uapiWhenResolvable(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 3 || waits != 2 {
		t.Errorf("attempts = %d, waits = %d, want 3 and 2", attempts, waits)
	}
	if !strings.Contains(settings, "endpoint=192.0.2.1:51820") {
		t.Errorf("settings lack the resolved endpoint:\n%s", settings)
	}
}

func TestTheTunnelStopsWaitingForTheEndpointWhenAskedToStop(t *testing.T) {
	cfg, _ := conf.Parse(strings.NewReader("[Interface]\nPrivateKey = AGLuYofEZv6kppTTyYs4baKYFi1sV65heniL1Tcg+WQ=\n[Peer]\nPublicKey = aVdLGe76+qHh25zQBy3zfb3qp35IeZU4DWZvrowUX0g=\nEndpoint = vpn.example.com:51820\n"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := NewRunner("wg0", func(string) (string, error) { return "", errors.New("no such host") }, nil, Wait)
	if _, err := r.uapiWhenResolvable(ctx, cfg); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

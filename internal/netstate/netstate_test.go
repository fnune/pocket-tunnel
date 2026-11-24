package netstate

import (
	"net"
	"testing"
)

func TestOnlineNeedsAnInterfaceOtherThanTheTunnelWithAnIPv4Address(t *testing.T) {
	loopback := Interface{Name: "lo", Up: true, Loopback: true, Addrs: []net.IP{net.ParseIP("127.0.0.1")}}
	tunnel := Interface{Name: "wg0", Up: true, Addrs: []net.IP{net.ParseIP("10.100.0.7")}}
	for _, tc := range []struct {
		name   string
		ifaces []Interface
		want   bool
	}{
		{"Wi-Fi with an address", []Interface{loopback, {Name: "wlan0", Up: true, Addrs: []net.IP{net.ParseIP("192.168.1.92")}}}, true},
		{"Wi-Fi down", []Interface{loopback, {Name: "wlan0", Up: false, Addrs: []net.IP{net.ParseIP("192.168.1.92")}}}, false},
		{"Wi-Fi up without an address", []Interface{loopback, {Name: "wlan0", Up: true}}, false},
		{"Wi-Fi with only a link-local address", []Interface{loopback, {Name: "wlan0", Up: true, Addrs: []net.IP{net.ParseIP("169.254.3.4"), net.ParseIP("fe80::1")}}}, false},
		{"only the tunnel itself", []Interface{loopback, tunnel}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Online(tc.ifaces, "wg0"); got != tc.want {
				t.Errorf("Online = %v, want %v", got, tc.want)
			}
		})
	}
}

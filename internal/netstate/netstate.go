package netstate

import (
	"net"

	"github.com/fnune/pocket-tunnel/internal/app"
)

type Interface struct {
	Name     string
	Up       bool
	Loopback bool
	Addrs    []net.IP
}

type Network struct {
	tunnel string
}

func New(tunnel string) *Network {
	return &Network{tunnel: tunnel}
}

var _ app.Network = (*Network)(nil)

func (n *Network) Online() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	var states []Interface
	for _, iface := range ifaces {
		state := Interface{
			Name:     iface.Name,
			Up:       iface.Flags&net.FlagUp != 0,
			Loopback: iface.Flags&net.FlagLoopback != 0,
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if prefix, ok := addr.(*net.IPNet); ok {
				state.Addrs = append(state.Addrs, prefix.IP)
			}
		}
		states = append(states, state)
	}
	return Online(states, n.tunnel)
}

func Online(ifaces []Interface, tunnel string) bool {
	for _, iface := range ifaces {
		if !iface.Up || iface.Loopback || iface.Name == tunnel {
			continue
		}
		for _, ip := range iface.Addrs {
			if ip.To4() != nil && !ip.IsLinkLocalUnicast() && !ip.IsLoopback() {
				return true
			}
		}
	}
	return false
}

package app

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fnune/pocket-tunnel/internal/conf"
)

const (
	connectedWithin = 3 * time.Minute
	handshakeWait   = 10 * time.Second
	handshakePoll   = 500 * time.Millisecond
)

const (
	colorSecondary = "#484949"
	colorProgress  = "#1a65a6"
	colorSuccess   = "#0f7a52"
)

const (
	statusNotConfigured = iota
	statusNoNetwork
	statusOff
	statusConnecting
	statusConnected
)

var statusOptions = []string{"Not configured", "No network", "Off", "Connecting", "Connected"}

var statusColors = []string{colorSecondary, colorSecondary, colorSecondary, colorProgress, colorSuccess}

var toggleOptions = []string{"Enabled", "Disabled"}

const (
	itemStatus = iota
	itemTunnel
	itemAutostart
	itemPublicKey
	itemAddress
)

type Menu struct {
	ui      UI
	store   ConfigStore
	tunnel  Tunnel
	boot    BootHook
	network Network
	clock   Clock
}

func NewMenu(ui UI, store ConfigStore, tunnel Tunnel, boot BootHook, network Network, clock Clock) *Menu {
	return &Menu{ui: ui, store: store, tunnel: tunnel, boot: boot, network: network, clock: clock}
}

func (m *Menu) Run() error {
	for {
		cfg, err := m.store.Load()
		if err != nil {
			_ = m.ui.Message("Pocket Tunnel could not read its configuration\n\n" + err.Error())
			return err
		}
		choice, err := m.ui.Menu("Pocket Tunnel", m.items(cfg))
		if errors.Is(err, ErrBack) {
			return nil
		}
		if err != nil {
			return err
		}
		switch choice {
		case itemTunnel:
			m.toggleTunnel(cfg)
		case itemAutostart:
			m.toggleAutostart()
		case itemPublicKey:
			m.showPublicKey(cfg)
		}
	}
}

func (m *Menu) items(cfg *conf.Config) []MenuItem {
	status := m.status(cfg)
	address := "Not set"
	if len(cfg.Addresses) > 0 {
		address = strings.Join(cfg.Addresses, ", ")
	}
	return []MenuItem{
		itemStatus:    {Label: "Status", Options: statusOptions, Selected: status, Unselectable: true, BackgroundColor: statusColors[status]},
		itemTunnel:    toggle("Tunnel", m.tunnel.Running()),
		itemAutostart: toggle("Autostart", m.boot.Enabled()),
		itemPublicKey: {Label: "Show public key"},
		itemAddress:   {Label: "Address: " + address, Unselectable: true},
	}
}

func toggle(label string, enabled bool) MenuItem {
	selected := 1
	if enabled {
		selected = 0
	}
	return MenuItem{Label: label, Options: toggleOptions, Selected: selected, ConfirmText: "Confirm"}
}

func (m *Menu) status(cfg *conf.Config) int {
	if len(cfg.Missing()) > 0 {
		return statusNotConfigured
	}
	if !m.network.Online() {
		return statusNoNetwork
	}
	if !m.tunnel.Running() {
		return statusOff
	}
	if !m.recentHandshake() {
		return statusConnecting
	}
	return statusConnected
}

func (m *Menu) recentHandshake() bool {
	last, err := m.tunnel.LastHandshake()
	return err == nil && !last.IsZero() && m.clock.Now().Sub(last) <= connectedWithin
}

func (m *Menu) toggleTunnel(cfg *conf.Config) {
	if m.tunnel.Running() {
		_ = m.ui.ShowProgress("Disconnecting")
		defer m.ui.CloseProgress()
		if err := m.tunnel.Stop(); err != nil {
			m.ui.CloseProgress()
			_ = m.ui.Message("The tunnel did not stop\n\n" + err.Error())
		}
		return
	}
	if missing := cfg.Missing(); len(missing) > 0 {
		_ = m.ui.Message(fmt.Sprintf("Fill in %s\n\nin %s", strings.Join(missing, ", "), m.store.ConfigPath()))
		return
	}
	_ = m.ui.ShowProgress("Connecting")
	defer m.ui.CloseProgress()
	if err := m.tunnel.Start(); err != nil {
		m.ui.CloseProgress()
		_ = m.ui.Message("The tunnel did not start\n\n" + err.Error())
		return
	}
	for waited := time.Duration(0); waited < handshakeWait && !m.recentHandshake(); waited += handshakePoll {
		m.clock.Sleep(handshakePoll)
	}
}

func (m *Menu) toggleAutostart() {
	var err error
	if m.boot.Enabled() {
		err = m.boot.Disable()
	} else {
		err = m.boot.Enable()
	}
	if err != nil {
		_ = m.ui.Message("Autostart did not change\n\n" + err.Error())
	}
}

func (m *Menu) showPublicKey(cfg *conf.Config) {
	public, err := cfg.PublicKey()
	if err != nil {
		_ = m.ui.Message("There is no valid PrivateKey in\n" + m.store.ConfigPath())
		return
	}
	half := len(public) / 2
	_ = m.ui.Message(fmt.Sprintf("Public key\n\n%s\n%s\n\nAlso saved in\n%s", public[:half], public[half:], m.store.PublicKeyPath()))
}

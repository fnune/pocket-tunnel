package tunnel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"

	"github.com/fnune/pocket-tunnel/internal/app"
	"github.com/fnune/pocket-tunnel/internal/conf"
)

const resolveRetryInterval = 5 * time.Second

type Runner struct {
	name    string
	resolve conf.Resolver
	ip      func(args ...string) error
	wait    func(ctx context.Context, d time.Duration) error
}

func NewRunner(name string, resolve conf.Resolver, ip func(args ...string) error, wait func(ctx context.Context, d time.Duration) error) *Runner {
	return &Runner{name: name, resolve: resolve, ip: ip, wait: wait}
}

var _ app.TunnelRunner = (*Runner)(nil)

func (r *Runner) Run(ctx context.Context, cfg *conf.Config) error {
	name := r.name
	settings, err := r.uapiWhenResolvable(ctx, cfg)
	if err != nil {
		return err
	}

	tunDev, err := tun.CreateTUN(name, device.DefaultMTU)
	if err != nil {
		return fmt.Errorf("create tun: %w", err)
	}
	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), device.NewLogger(device.LogLevelError, fmt.Sprintf("(%s) ", name)))
	defer dev.Close()

	if err := dev.IpcSet(settings); err != nil {
		return fmt.Errorf("configure device: %w", err)
	}
	if err := dev.Up(); err != nil {
		return fmt.Errorf("bring device up: %w", err)
	}

	if uapiFile, err := ipc.UAPIOpen(name); err != nil {
		log.Printf("control socket unavailable, status will not show handshakes: %v", err)
	} else if listener, err := ipc.UAPIListen(name, uapiFile); err != nil {
		log.Printf("control socket unavailable, status will not show handshakes: %v", err)
	} else {
		defer func() { _ = listener.Close() }()
		go func() {
			for {
				c, err := listener.Accept()
				if err != nil {
					return
				}
				go dev.IpcHandle(c)
			}
		}()
	}

	for _, addr := range cfg.Addresses {
		if err := r.ip("address", "add", addr, "dev", name); err != nil {
			return err
		}
	}
	if err := r.ip("link", "set", name, "up"); err != nil {
		return err
	}
	for _, p := range cfg.Peers {
		for _, allowed := range p.AllowedIPs {
			if err := r.ip("route", "replace", allowed, "dev", name); err != nil {
				return err
			}
		}
	}
	log.Printf("%s is up", name)

	select {
	case <-ctx.Done():
	case <-dev.Wait():
	}
	log.Printf("%s is down", name)
	return nil
}

func (r *Runner) uapiWhenResolvable(ctx context.Context, cfg *conf.Config) (string, error) {
	for {
		settings, err := cfg.UAPI(r.resolve)
		if err == nil {
			return settings, nil
		}
		var resolveErr *conf.ResolveError
		if !errors.As(err, &resolveErr) {
			return "", err
		}
		log.Printf("%v, retrying in %s", err, resolveRetryInterval)
		if err := r.wait(ctx, resolveRetryInterval); err != nil {
			return "", err
		}
	}
}

func IP(args ...string) error {
	out, err := exec.Command("ip", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ip %s: %w: %s", strings.Join(args, " "), err, out)
	}
	return nil
}

type Status struct {
	LastHandshake time.Time
}

func Query(name string) (*Status, error) {
	c, err := net.DialTimeout("unix", filepath.Join("/var/run/wireguard", name+".sock"), time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = c.Close() }()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(c, "get=1\n\n"); err != nil {
		return nil, err
	}
	return ParseStatus(c)
}

func ParseStatus(r io.Reader) (*Status, error) {
	status := &Status{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		key, value, _ := strings.Cut(line, "=")
		if key != "last_handshake_time_sec" {
			continue
		}
		sec, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("last_handshake_time_sec: %w", err)
		}
		if sec > 0 {
			if t := time.Unix(sec, 0); t.After(status.LastHandshake) {
				status.LastHandshake = t
			}
		}
	}
	return status, scanner.Err()
}

func ResolveUDP(endpoint string) (string, error) {
	addr, err := net.ResolveUDPAddr("udp4", endpoint)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}

func Wait(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

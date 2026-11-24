package conf

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/curve25519"
)

type Peer struct {
	PublicKey  string
	Endpoint   string
	AllowedIPs []string
	Keepalive  int
}

type Config struct {
	PrivateKey string
	Addresses  []string
	Peers      []Peer
}

func Parse(r io.Reader) (*Config, error) {
	cfg := &Config{}
	var section string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.ToLower(strings.Trim(line, "[]"))
			if section == "peer" {
				cfg.Peers = append(cfg.Peers, Peer{})
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("malformed line: %q", line)
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		switch section {
		case "interface":
			switch key {
			case "privatekey":
				cfg.PrivateKey = value
			case "address":
				cfg.Addresses = append(cfg.Addresses, splitList(value)...)
			}
		case "peer":
			p := &cfg.Peers[len(cfg.Peers)-1]
			switch key {
			case "publickey":
				p.PublicKey = value
			case "endpoint":
				p.Endpoint = value
			case "allowedips":
				p.AllowedIPs = append(p.AllowedIPs, splitList(value)...)
			case "persistentkeepalive":
				if value == "" {
					continue
				}
				n, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("persistentkeepalive: %w", err)
				}
				p.Keepalive = n
			}
		}
	}
	return cfg, scanner.Err()
}

func splitList(value string) []string {
	var items []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func (c *Config) Missing() []string {
	var missing []string
	if c.PrivateKey == "" {
		missing = append(missing, "Interface PrivateKey")
	}
	if len(c.Addresses) == 0 {
		missing = append(missing, "Interface Address")
	}
	if len(c.Peers) == 0 {
		return append(missing, "a [Peer] section")
	}
	for _, p := range c.Peers {
		if p.PublicKey == "" {
			missing = append(missing, "Peer PublicKey")
		}
		if p.Endpoint == "" {
			missing = append(missing, "Peer Endpoint")
		}
		if len(p.AllowedIPs) == 0 {
			missing = append(missing, "Peer AllowedIPs")
		}
	}
	return missing
}

func GenerateKey(random io.Reader) (string, error) {
	var key [32]byte
	if _, err := io.ReadFull(random, key[:]); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	key[0] &= 248
	key[31] = (key[31] & 127) | 64
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

func PublicKey(privateKey string) (string, error) {
	private, err := decodeKey(privateKey)
	if err != nil {
		return "", err
	}
	public, err := curve25519.X25519(private, curve25519.Basepoint)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(public), nil
}

func decodeKey(s string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key is %d bytes, want 32", len(key))
	}
	return key, nil
}

func Template(privateKey string) string {
	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address =

[Peer]
PublicKey =
Endpoint =
AllowedIPs =
PersistentKeepalive = 25
`, privateKey)
}

type Resolver func(endpoint string) (string, error)

func (c *Config) PublicKey() (string, error) {
	return PublicKey(c.PrivateKey)
}

type ResolveError struct {
	Endpoint string
	Err      error
}

func (e *ResolveError) Error() string {
	return fmt.Sprintf("resolve %s: %v", e.Endpoint, e.Err)
}

func (e *ResolveError) Unwrap() error {
	return e.Err
}

func (c *Config) UAPI(resolve Resolver) (string, error) {
	var b strings.Builder
	private, err := decodeKey(c.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("private key: %w", err)
	}
	fmt.Fprintf(&b, "private_key=%s\nreplace_peers=true\n", hex.EncodeToString(private))
	for _, p := range c.Peers {
		public, err := decodeKey(p.PublicKey)
		if err != nil {
			return "", fmt.Errorf("peer public key: %w", err)
		}
		fmt.Fprintf(&b, "public_key=%s\n", hex.EncodeToString(public))
		if p.Endpoint != "" {
			endpoint, err := resolve(p.Endpoint)
			if err != nil {
				return "", &ResolveError{Endpoint: p.Endpoint, Err: err}
			}
			fmt.Fprintf(&b, "endpoint=%s\n", endpoint)
		}
		if p.Keepalive > 0 {
			fmt.Fprintf(&b, "persistent_keepalive_interval=%d\n", p.Keepalive)
		}
		b.WriteString("replace_allowed_ips=true\n")
		for _, ip := range p.AllowedIPs {
			fmt.Fprintf(&b, "allowed_ip=%s\n", ip)
		}
	}
	return b.String(), nil
}

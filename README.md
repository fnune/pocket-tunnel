# Pocket Tunnel

A NextUI pak that connects a handheld to your home network over a VPN tunnel, so it can reach your own services from anywhere, such as a sync server. It runs the tunnel in userspace with the [wireguard-go](https://git.zx2c4.com/wireguard-go) library, because handhelds such as the TrimUI Brick run kernels without the tunnel built in.

Tested on the TrimUI Brick. The pak is marked experimental until other devices are tested.

## Install

Install Pocket Tunnel from the Pak Store, or download `Pocket.Tunnel.pak.zip` from the [latest release](https://github.com/fnune/pocket-tunnel/releases/latest) and extract it into `Tools/<platform>/Pocket Tunnel.pak/` on the SD card, such as `Tools/tg5040/Pocket Tunnel.pak/` for the Brick.

## Set up

The pak keeps its configuration in `.userdata/<platform>/Pocket Tunnel/wg0.conf` on the SD card. There are two ways to fill it:

- Write the whole file on your computer and copy it there, with the usual `[Interface]` and `[Peer]` sections.
- Open Pocket Tunnel once. It generates a key pair on the device and writes a `wg0.conf` with its private key filled in. Add the device's public key to your server, then fill in `Address` and the `[Peer]` fields.

```ini
[Interface]
PrivateKey = <generated on the device>
Address = 10.100.0.7/32

[Peer]
PublicKey = <your server's public key>
Endpoint = vpn.example.com:51820
AllowedIPs = 10.100.0.0/24
PersistentKeepalive = 25
```

Only traffic for `AllowedIPs` goes through the tunnel, so it can stay on.

The SD card's filesystem has no permissions, so anyone with the card can read the private key in `wg0.conf`.

## Use

The menu shows:

- **Status**: Not configured, No network, Off, Connecting or Connected.
- **Tunnel**: turns the tunnel on or off.
- **Autostart**: starts the tunnel when the device boots, through a marked block in `.userdata/<platform>/auto.sh`. Turning it off removes only that block.
- **Show public key**: the device's public key, also saved in `public.key` next to `wg0.conf`.
- **Address**: the tunnel address from `wg0.conf`.

Logs are in `.userdata/<platform>/logs/Pocket Tunnel.txt` for the menu and `Pocket Tunnel-tunnel.txt` for the tunnel.

## Development

`just check` runs formatting, lint and the tests. `nix build .#checks.x86_64-linux.vm` runs a NixOS VM test with a server and a handheld running the pak behind fake menu tools. `just deploy` copies the pak to a Brick over SSH. See [ENGINEERING.md](ENGINEERING.md) for how the code is built.

## Releases and the Pak Store

`just release-create 0.2.0` sets the version in `pak.json`, writes `CHANGELOG.md` from the Conventional Commits since the last tag, pushes `main` and tags `v0.2.0`. The GitHub mirror runs CI on the tag, then publishes a release with `Pocket.Tunnel.pak.zip` and `SHA256SUMS`.

The NextUI Pak Store reads `pak.json` from the root of `main` on GitHub once an hour and downloads `releases/download/<version>/<release_filename>`. The listing breaks if `pak.json` moves, if `release_filename` stops matching the uploaded zip, or if the zip stops holding the pak's contents at its root.

## License

MIT. "WireGuard" and the "WireGuard" logo are registered trademarks of Jason A. Donenfeld. Pocket Tunnel is not affiliated with or endorsed by the WireGuard project.

from datetime import timedelta
import json


def menu(state_dir: str, *choices: str) -> None:
    handheld.succeed(f"printf '%s\\n' {' '.join(repr(c) for c in choices)} > /tmp/choices" if choices else "rm -f /tmp/choices")
    handheld.succeed(
        "pocket-tunnel menu"
        f" --state-dir {state_dir}"
        " --bin-dir /etc/handheld/bin"
        " --auto-sh /var/lib/auto.sh"
        " --log-file /var/lib/tunnel.log",
        timeout=timedelta(seconds=60),
    )


def last_menu() -> list[str]:
    return json.loads(handheld.succeed("tail -n 1 /tmp/menus.log"))


def messages() -> str:
    return handheld.succeed("cat /tmp/messages.log 2>/dev/null || true")


start_all()
server.wait_for_unit("wireguard-wg0.service")
handheld.wait_for_unit("multi-user.target")
handheld.succeed("printf '#!/bin/sh\\necho other pak hook\\n' > /var/lib/auto.sh")

with subtest("an unconfigured device generates its own key and names the missing fields"):
    menu("/var/lib/fresh", "Tunnel", "Show public key")
    handheld.succeed("test -s /var/lib/fresh/wg0.conf")
    public_key = handheld.succeed("cat /var/lib/fresh/public.key").strip()
    assert "Fill in" in messages(), messages()
    assert "Peer Endpoint" in messages(), messages()
    assert public_key in messages().replace("\n", ""), messages()
    handheld.fail("ip link show wg0")
    handheld.succeed("rm /tmp/messages.log")

with subtest("a provided config brings the tunnel up and the server answers through it"):
    handheld.succeed("mkdir -p /var/lib/wg && cp /etc/handheld/wg0.conf /var/lib/wg/wg0.conf")
    menu("/var/lib/wg", "Tunnel")
    handheld.wait_until_succeeds("ping -c 1 -W 1 10.100.0.1", timeout=timedelta(seconds=60))
    server.succeed("ping -c 1 -W 1 10.100.0.7")

with subtest("the menu shows the tunnel as connected"):
    menu("/var/lib/wg")
    items = last_menu()
    assert "Status=Connected" in items, items
    assert "Tunnel=Enabled" in items, items

with subtest("the menu says no network while the device is offline"):
    handheld.succeed("ip link set eth0 down", "ip link set eth1 down")
    menu("/var/lib/wg")
    items = last_menu()
    assert "Status=No network" in items, items
    handheld.succeed("ip link set eth0 up", "ip link set eth1 up")
    handheld.wait_until_succeeds("ping -c 1 -W 1 10.100.0.1", timeout=timedelta(seconds=60))

with subtest("the public key shown is the one derived from the provided private key"):
    menu("/var/lib/wg", "Show public key")
    assert handheld_public_key in messages().replace("\n", ""), messages()

with subtest("autostart adds a hook that brings the tunnel back after it stops"):
    menu("/var/lib/wg", "Autostart", "Tunnel")
    handheld.wait_until_fails("ip link show wg0", timeout=timedelta(seconds=30))
    auto_sh = handheld.succeed("cat /var/lib/auto.sh")
    assert "echo other pak hook" in auto_sh, auto_sh
    assert "# BEGIN POCKET-TUNNEL" in auto_sh, auto_sh
    handheld.succeed("sh /var/lib/auto.sh")
    handheld.wait_until_succeeds("ping -c 1 -W 1 10.100.0.1", timeout=timedelta(seconds=60))

with subtest("turning autostart off removes only its own hook"):
    menu("/var/lib/wg", "Autostart")
    auto_sh = handheld.succeed("cat /var/lib/auto.sh")
    assert "POCKET-TUNNEL" not in auto_sh, auto_sh
    assert "echo other pak hook" in auto_sh, auto_sh

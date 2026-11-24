{ pkgs, pocket-tunnel }:
let
  server = {
    privateKey = "mNg5f8LFWWRfR60M/h0RjY4OACGwcmrxCul7EJ3JNlQ=";
    publicKey = "tpTEk2doFZtqgD7tNSDQU8AqEMxSg+eD6Ue78CNfJVE=";
  };
  handheld = {
    privateKey = "IGD2fBjq4B7ynK1rCOTEBg2QTzNz4vEg39M8X/BLQGU=";
    publicKey = "QIzme5QcgVXmr/W/jZWE+iuSMRtaH3cKoXPVFenERG4=";
  };
  pakBin = pkgs.runCommand "pak-bin" { } ''
    mkdir -p $out
    cp ${pkgs.writeShellApplication { name = "minui-list"; runtimeInputs = [ pkgs.jq pkgs.gnused ]; text = builtins.readFile ./fake-minui-list.sh; }}/bin/minui-list $out/
    cp ${pkgs.writeShellApplication { name = "minui-presenter"; text = builtins.readFile ./fake-minui-presenter.sh; }}/bin/minui-presenter $out/
  '';
  handheldConfig = pkgs.writeText "wg0.conf" ''
    [Interface]
    PrivateKey = ${handheld.privateKey}
    Address = 10.100.0.7/32

    [Peer]
    PublicKey = ${server.publicKey}
    Endpoint = server:51820
    AllowedIPs = 10.100.0.0/24
    PersistentKeepalive = 5
  '';
in
pkgs.testers.runNixOSTest {
  name = "pocket-tunnel";

  nodes.server = {
    networking.firewall.allowedUDPPorts = [ 51820 ];
    networking.wireguard.interfaces.wg0 = {
      ips = [ "10.100.0.1/24" ];
      listenPort = 51820;
      privateKeyFile = toString (pkgs.writeText "server.key" server.privateKey);
      peers = [
        {
          inherit (handheld) publicKey;
          allowedIPs = [ "10.100.0.7/32" ];
        }
      ];
    };
  };

  nodes.handheld = {
    environment.systemPackages = [
      pocket-tunnel
      pkgs.iproute2
    ];
    environment.etc."handheld/wg0.conf".source = handheldConfig;
    environment.etc."handheld/bin".source = pakBin;
  };

  testScript = ''
    handheld_public_key = "${handheld.publicKey}"
    ${builtins.readFile ./vm-test.py}
  '';
}

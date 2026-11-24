{
  description = "VPN tunnel pak for NextUI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        tools = with pkgs; [
          go
          gopls
          golangci-lint
          just
          jq
          zip
          git-cliff
        ];
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "pocket-tunnel";
          version = "0.1.0";
          src = pkgs.lib.cleanSource ./.;
          vendorHash = "sha256-IBeOESqRfgOFLa9AOLFwfYvzjawthwFc7wEii4bLBKw=";
          subPackages = [ "cmd/pocket-tunnel" ];
          env.CGO_ENABLED = 0;
          meta.mainProgram = "pocket-tunnel";
        };

        checks = pkgs.lib.optionalAttrs pkgs.stdenv.isLinux {
          vm = import ./nix/vm-test.nix {
            inherit pkgs;
            pocket-tunnel = self.packages.${system}.default;
          };
        };

        devShells.default = pkgs.mkShell { buildInputs = tools; };
        devShells.ci = pkgs.mkShell { buildInputs = tools; };
      });
}

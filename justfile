minui_list_version := "0.16.0"
minui_presenter_version := "0.14.0"
platforms := "tg5040 tg5050 h700"

build_dir := "dist"
cache_dir := ".cache"
pak_dir := build_dir / "Pocket Tunnel.pak"
pak_zip := build_dir / "Pocket.Tunnel.pak.zip"

brick := env("POCKET_TUNNEL_HOST", "root@192.168.1.92")
ssh_key := env("POCKET_TUNNEL_SSH_KEY", home_directory() / ".ssh/id_ed25519_brick")
ssh_options := "-i " + ssh_key + " -o BatchMode=yes -o IdentitiesOnly=yes -o LogLevel=ERROR -o ConnectTimeout=5 -o ServerAliveInterval=5 -o ServerAliveCountMax=3"

default:
    @just --list

check: lint test

lint:
    test -z "$(gofmt -l .)"
    golangci-lint run ./...

test:
    CGO_ENABLED=0 go test ./...

# Build Pocket Tunnel.pak for every platform in pak.json, and zip it for the Pak Store
build:
    #!/usr/bin/env bash
    set -euo pipefail
    rm -rf "{{ pak_dir }}" "{{ pak_zip }}"
    mkdir -p "{{ pak_dir }}" "{{ cache_dir }}"
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o "{{ pak_dir }}/pocket-tunnel" ./cmd/pocket-tunnel
    for platform in {{ platforms }}; do
        mkdir -p "{{ pak_dir }}/bin/$platform"
        just _helper minui-list {{ minui_list_version }} "$platform"
        just _helper minui-presenter {{ minui_presenter_version }} "$platform"
    done
    cp build/launch.sh pak.json "{{ pak_dir }}/"
    chmod +x "{{ pak_dir }}/launch.sh" "{{ pak_dir }}/pocket-tunnel" "{{ pak_dir }}"/bin/*/*
    (cd "{{ pak_dir }}" && zip -qr ../Pocket.Tunnel.pak.zip .)
    echo "Built {{ pak_dir }} and {{ pak_zip }}"

# Copy the built pak to a TrimUI Brick over SSH
deploy: build
    tar -C "{{ pak_dir }}" -cf - . | ssh {{ ssh_options }} {{ brick }} 'mkdir -p "/mnt/SDCARD/Tools/tg5040/Pocket Tunnel.pak" && tar -C "/mnt/SDCARD/Tools/tg5040/Pocket Tunnel.pak" -xf - && sync'

clean:
    rm -rf {{ build_dir }} {{ cache_dir }}

_helper name version platform:
    #!/usr/bin/env bash
    set -euo pipefail
    declare -A checksums=(
        [minui-list-tg5040]=0888e85d727c24aa96b99c71e29566fce7e818d7c85152082b460be31f7bfab1
        [minui-list-tg5050]=0e77554b59158df6f7618074b16f9346865653614db76e0d469bf4e20f63afa2
        [minui-list-h700]=c579322919f069d9e8d50751e9c22a23238a84ecef4d6c5be48661b9312baa2d
        [minui-presenter-tg5040]=264785d9aa3e5f9ac2074c539e480ea1fd6dbe220ee3a52734511aa9ec759830
        [minui-presenter-tg5050]=7e585c6bd6a96b2715e48029a860e0fe72cd29cb581372cf962f50595275fce8
        [minui-presenter-h700]=bd52303c41ef6a865c9742fe8dc5068b3230725d85ca98e3034beddf2e493fe1
    )
    cache="{{ cache_dir }}/{{ name }}-{{ version }}-{{ platform }}"
    if [ ! -f "$cache" ]; then
        curl --fail --silent --show-error --location --output "$cache.partial" \
            "https://github.com/josegonzalez/{{ name }}/releases/download/{{ version }}/{{ name }}-{{ platform }}-nextui"
        mv "$cache.partial" "$cache"
    fi
    echo "${checksums[{{ name }}-{{ platform }}]}  $cache" | sha256sum --check --quiet
    cp "$cache" "{{ pak_dir }}/bin/{{ platform }}/{{ name }}"

release-create version:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -n "$(jj diff --summary)" ]; then
        echo "The working copy has changes. Commit them first."
        exit 1
    fi
    jj git fetch
    if [ "$(jj log --no-graph -r 'main' -T commit_id)" != "$(jj log --no-graph -r 'main@origin' -T commit_id)" ]; then
        echo "Local main differs from origin. Push or rebase first."
        exit 1
    fi
    jj new main
    jq --indent 2 '.version = "v{{ version }}"' pak.json > pak.json.tmp
    mv pak.json.tmp pak.json
    git-cliff --tag "v{{ version }}" --unreleased --prepend CHANGELOG.md
    jj commit -m "chore(release): release v{{ version }}"
    jj bookmark set main -r @-
    jj git push --bookmark main
    git tag "v{{ version }}" "$(jj log --no-graph -r main -T commit_id)"
    git push origin "v{{ version }}"

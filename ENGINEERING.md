# Engineering principles

How this pak is built. These apply to every change, whoever or whatever writes it.

## Principles

### Calm tool

The pak is a small tool on a handheld. Using it should feel calm: simple, clear, predictable, and respectful of the person holding the device.

- Simplicity is the default. A feature earns its place or it does not ship.
- Clarity over cleverness. Plain menu labels, ordinary words, predictable defaults.
- Respect user agency. Nothing turns on by itself: the tunnel and the boot hook start only when the user turns them on, and stay off when turned off.
- Leave other paks alone. The boot hook lives in a marked block of `auto.sh`, and enabling or disabling it never touches another pak's lines.
- Failures are shown calmly. Messages on screen say what failed and what to do next, such as which fields of `wg0.conf` to fill in. They never blame the user.

### Testability via dependency injection

Reference: [Mark Seemann](https://blog.ploeh.dk/), [*Dependency Injection Principles, Practices, and Patterns*](https://www.manning.com/books/dependency-injection-principles-practices-patterns).

- Constructor Injection is the default: a `New...` function takes every dependency the type needs. Use other forms only when justified.
- Inject Volatile Dependencies: anything with side effects (file system, processes, network, the TUN device, `ip`), nondeterminism (time, randomness), runtime configuration, or that does not exist yet.
- Do not inject Stable Dependencies: the standard library's deterministic parts, value types and pure functions, such as parsing `wg0.conf` or deriving a public key.
- The Composition Root is `cmd/pocket-tunnel/main.go`. It is the only place that knows the concrete implementations and wires them together.
- Prefer Pure DI: hand-wired object graphs in the Composition Root, no container.
- Service Locator is an anti-pattern. Receive dependencies as constructor parameters instead of looking them up.
- Ambient Context is an anti-pattern: no package-level clocks, global configuration or calls to `time.Now()`, `rand.Reader` or `os.Getenv` below the Composition Root. Inject them.
- A growing constructor argument list is a sign that the type does too much.
- Tests follow the seams DI creates: real implementations of Stable Dependencies, hand-written in-memory fakes for Volatile ones. Code that owns a file format or the file system, such as the config store and the boot hook, is tested against a real temporary directory rather than a mock.

### Layered architecture and abstraction ownership

- `internal/app` holds the use cases: the on-device menu and the `up` command. It declares every interface it consumes (`UI`, `ConfigStore`, `Tunnel`, `BootHook`, `Clock`, `PIDLock`, `TunnelRunner`).
- `internal/store`, `internal/process`, `internal/tunnel` and `internal/minui` implement those interfaces. They import `app`; `app` never imports them.
- `internal/conf` is pure: parsing, validation, the config template, key handling and the UAPI text. Randomness and name resolution are parameters.
- Reject naive layering, where each layer depends on the one below it. The interface belongs to the consumer: this is the Dependency Inversion Principle applied at the package boundary.
- Each implementation asserts that it satisfies its interface at compile time, such as `var _ app.Tunnel = (*Tunnel)(nil)`.

### Strict typing

- The compiler and `go vet` are the floor, and `golangci-lint` runs on every change.
- Escape hatches are code smells to fix, not to paper over: `any` where a concrete type fits, unchecked type assertions, ignored errors without a reason, and `//nolint` without an explanation.
- Validate at every process boundary: `wg0.conf`, command-line flags, the control socket's replies and the output of `minui-list`. Parse into typed values once, then pass those around.

### Code comments

- Code is self-evident. Comments are a last resort.
- A comment is only justified by a non-obvious why: a hidden constraint, a subtle invariant, a workaround for a specific bug.
- Comments that restate what the code does are noise. Prefer a test whose name states the reason, such as `TestTheTunnelWaitsForTheEndpointToResolveBecauseWiFiMayNotBeUpAtBoot`.

### Dependency hygiene

- Start with the standard library: `flag`, `os/exec`, `encoding/json`, `net`, `log`, `testing`.
- The tunnel uses [wireguard-go](https://git.zx2c4.com/wireguard-go), WireGuard's own userspace implementation. Do not swap it for a third-party implementation.
- Vet every dependency: can the standard library do it, how large is its transitive tree, is it maintained.
- Prefer governance-neutral dependencies over open source that exists to feed a vendor's hosted service, where the roadmap follows the vendor's revenue rather than user need. Use such dependencies only when no neutral alternative exists, and say why in the commit.
- If it is under 50 lines, write it rather than depend on it.
- Vendor utilities under 100 lines with attribution and the original license.
- Commit `go.sum`. Treat `go.mod` changes as code changes.
- Budget: under 5 direct dependencies. Exceeding it needs a written reason in the commit.
- Pin exactly. Go modules pin by version or pseudo-version, the Nix dev shell pins through `flake.lock`, and every binary downloaded at build time, such as `minui-list` and `minui-presenter`, is pinned by version and checked against a SHA-256 in the `justfile`.

### Operational discipline

- Error handling and failure modes. No silent catches. Wrap errors with context using `%w`: what was attempted, with what inputs, what failed. Lower layers return errors; the menu decides what to show on screen, and `up` decides what to log.
- Observability. The menu logs to `$LOGS_PATH/Pocket Tunnel.txt` and the tunnel to `$LOGS_PATH/Pocket Tunnel-tunnel.txt`. If a failure cannot be seen in one of them, it cannot be fixed.
- Security baseline. The private key never leaves the device unless the user puts it there: it is generated on the device and never logged. Only the public key is shown on screen or written to `public.key`. `wg0.conf` is written with mode `0600`, but SD cards are exFAT, which has no permissions, so anyone with the card can read the key. Say so in user-facing docs rather than implying otherwise.
- State stays in `$USERDATA_PATH/Pocket Tunnel`, never in the pak folder, which the Pak Store replaces on every update.

### Local checks before pushing

- Run the cheap checks locally: `just check` runs formatting, lint and the tests.
- Choose checks based on what changed. `just test` alone is enough for a change confined to tests. A change to packaging also needs `just build`.
- Test on a device before a release: `just deploy` installs the pak on a TrimUI Brick over SSH.

### Git conventions

- [Conventional Commits](https://www.conventionalcommits.org/) for every commit message. Types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `perf`, `build`, `ci`, `revert`. Scope optional and lowercase, such as `feat(menu):` or `fix(tunnel):`.
- Imperative mood. "Add the X", not "Added the X".
- Subject line under 72 characters.
- The body explains why when the diff does not.
- No value-judgement words: "improve", "better", "cleaner", "robust", "elegant". State the change in concrete terms.
- No vague qualifiers: "various", "several minor", "ensure", "make sure".
- No AI attribution and no Co-Authored-By for AI tools.
- `git-cliff` writes `CHANGELOG.md` from these messages at release time, so they are read by users.

### Continuous integration

- Every push runs the same baseline as `just check`, plus `just build` to package the pak. Same commands locally and in CI.
- CI runs on the GitHub mirror through `fnune/ci`, pinned to a commit, inside the same Nix dev shell as local development.
- A tag `vX.Y.Z` runs the release workflow, which refuses to publish unless `pak.json` has the same version.

### Reproducible, one-command developer environments

- Same Go version and tools on every machine and in CI.
- One command to set up: `nix develop`, or `direnv allow` once.
- `flake.nix` defines the tools and `flake.lock` pins them. CI uses the `ci` shell from the same flake.

### Toolchain choices

- Go for everything. wireguard-go is a Go library, and `CGO_ENABLED=0` builds one static arm64 binary that needs nothing from the device.
- [just](https://just.systems/) for every task, so the commands in this document, in CI and in a terminal are the same.
- `gofmt` for formatting and [golangci-lint](https://golangci-lint.run/) for linting.
- `go test` for tests. No assertion or mocking frameworks: the standard library and hand-written fakes are enough.
- [git-cliff](https://git-cliff.org/) for the changelog, from Conventional Commits.

### Writing style

- Sentence case in headings.
- Bold sparingly.
- No em-dashes. Use a colon, comma, semicolon, period, or rephrase.
- No rhetorical triplets ("one X, one Y, one Z" and similar three-beat patterns).
- On-screen text follows the same rules, and stays short enough to read on a handheld.

## Use of AI and LLMs

This project accepts code written with LLMs. The tools change quickly, but the code is held to the same standards as code written without them.

- Tests at the unit and integration levels, and a check on a real device before every release.
- Typed values and validated boundaries.
- Dependency hygiene as set out above.
- Review of every change, AI-assisted or not.

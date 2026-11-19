# stamper

Go CLI (module `github.com/pipejesus/stamper`; cobra + charm fang/huh/lipgloss v2 from `charm.land/...`) that stamps Lando environments from modular recipes and installs embedded agent skills. README.md covers usage, layout and architecture. Open work is in BACKLOG.md.

## Working on it
- The Go toolchain is at `~/.local/sdk/go/bin` (not on PATH): `export PATH=$HOME/.local/sdk/go/bin:$PATH`, or `make build GO=~/.local/sdk/go/bin/go`.
- Push over SSH (`git@github.com:pipejesus/stamper.git`). The HTTPS credentials and `GH_TOKEN` on this machine belong to another account, so `gh` fails; read CI with unauthenticated `curl https://api.github.com/repos/pipejesus/stamper/actions/runs`.
- CI (`.github/workflows/ci.yml`) runs gofmt check + vet + test. A `v*` tag runs `release.yml`, which publishes the `stamper_<os>_<arch>` assets `stamper self-update` downloads. No release has been tagged yet.
- Tests: `go test ./...`. End-to-end: stamp into a scratch dir, `lando start`, verify, then `lando destroy -y`.

## Invariants
- Recipe `Option` schemas are the single source of truth: flags (`internal/cli/new.go`), TUI (`internal/tui`), validation and defaults all derive from them. Never hand-write per-option flags or form fields.
- `.lando.yml` edits go through `lando.Apply` with the recipe's `owned` list; anything not owned must survive untouched. Owned list items are tagged `# stamper` (scalars only) or reference `.lando/stamper/`. A new Landofile key must be added to `owned` in `internal/recipe/wordpress/plan.go`, or `stamper config` can't clean it up.
- Support-file policies (`recipe.File.Policy`): `Always` for files in `.lando/stamper/`; `IfManaged` for root files like `wp-cli.yml`, rewritten only while they contain `Managed by stamper`; `Once` for root `.gitignore`. `Plan.Remove` only deletes files that carry the marker.
- Default layout: web root `public_html/`, with `wp-config.php` and `logs/` one level up and `wp-cli.yml` at the root. `--webroot .` gives the flat layout. Both are tested.
- `wp-setup.sh.tmpl` must stay idempotent and must never bootstrap WordPress before it has fixed `wp-config` constants and stale object-cache drop-ins. A leftover Redis drop-in pointing at a removed service broke `wp core is-installed` once.
- Freshness is a product requirement. When skill content or recipe tables/templates change, bump the skill's `updated:` in `internal/skills/library/manifest.yml` and/or `revision` (date+time, UTC) in `internal/recipe/wordpress/wordpress.go`.
- Version tables mirror the installed Lando plugins (`~/.lando/plugins/@lando/*/builders/*.js`; Lando 3.26.9, @lando/php 1.12 → PHP ≤ 8.5). Check there before adding versions.
- Skills follow the SKILL.md format (`name` + `description` frontmatter; quote the description if it contains `: `), are kept lean with `references/` files loaded on demand, and install to `.claude/skills/` and `.agents/skills/` (Codex).

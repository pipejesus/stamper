# stamper: agent guide

Instructions for any coding agent (Codex, Claude Code ≥ 2.1.277, Cursor, Copilot, Gemini CLI, …) and for human contributors. This is the only instructions file: don't add a `CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md`. Claude Code reads AGENTS.md only when none of those exist.

Go CLI (module `github.com/pipejesus/stamper`; cobra + charm fang/huh/lipgloss v2 from `charm.land/...`) that stamps Lando environments from modular recipes and installs embedded agent skills. README.md covers usage, layout and architecture. Open work is in BACKLOG.md.

## Working on it
- Needs Go (version in `go.mod`), Docker and Lando 3 for end-to-end checks. If `go` is not on PATH, point make at it: `make build GO=/path/to/go`.
- Build `make build` (→ `bin/stamper`), test `go test ./...`, lint `make lint`. CI fails on unformatted code, so run `gofmt -w .` before committing.
- Work on `main` with small, verified commits. Remote: `git@github.com:pipejesus/stamper.git` (SSH). CI status without auth: `curl -s https://api.github.com/repos/pipejesus/stamper/actions/runs?per_page=1`.
- CI (`.github/workflows/ci.yml`) runs gofmt check + vet + test. A `v*` tag runs `release.yml`, which publishes the `stamper_<os>_<arch>` assets `stamper self-update` downloads. No release has been tagged yet.
- End-to-end check for recipe changes: stamp into a scratch dir (`bin/stamper new wordpress e2e.lndo.site -C /tmp/e2e --yes`), `lando start`, verify the site, `lando wp`, and whatever you changed, then `lando destroy -y`.
- Start from BACKLOG.md for open work; record new open items there. Ask the maintainer (GitHub `pipejesus`) before tagging releases or changing the license.

## Invariants
- Recipe `Option` schemas are the single source of truth: flags (`internal/cli/new.go`), TUI (`internal/tui`), validation and defaults all derive from them. Never hand-write per-option flags or form fields.
- `.lando.yml` edits go through `lando.Apply` with the recipe's `owned` list; anything not owned must survive untouched. Owned list items are tagged `# stamper` (scalars only) or reference `.lando/stamper/`. A new Landofile key must be added to `owned` in `internal/recipe/wordpress/plan.go`, or `stamper config` can't clean it up.
- Support-file policies (`recipe.File.Policy`): `Always` for files in `.lando/stamper/`; `IfManaged` for root files like `wp-cli.yml`, rewritten only while they contain `Managed by stamper`; `Once` for root `.gitignore`. `Plan.Remove` only deletes files that carry the marker.
- Default layout: web root `public_html/`, with `wp-config.php` and `logs/` one level up and `wp-cli.yml` at the root. `--webroot .` gives the flat layout. Both are tested.
- `wp-setup.sh.tmpl` must stay idempotent and must never bootstrap WordPress before it has fixed `wp-config` constants and stale object-cache drop-ins. A leftover Redis drop-in pointing at a removed service broke `wp core is-installed` once.
- Freshness is a product requirement. When skill content or recipe tables/templates change, bump the skill's `updated:` in `internal/skills/library/manifest.yml` and/or `revision` (date+time, UTC) in `internal/recipe/wordpress/wordpress.go`.
- Version tables mirror the installed Lando plugins (`~/.lando/plugins/@lando/*/builders/*.js`; Lando 3.26.9, @lando/php 1.12 → PHP ≤ 8.5). Check there before adding versions.
- Skills follow the SKILL.md format (`name` + `description` frontmatter; quote the description if it contains `: `), are kept lean with `references/` files loaded on demand, and install to `.claude/skills/` and `.agents/skills/` (Codex).

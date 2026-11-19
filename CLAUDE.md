# stamper

Go CLI (cobra + charm fang/huh/lipgloss v2 from `charm.land/...`) that stamps Lando environments from modular recipes and installs embedded agent skills. See README.md for usage and architecture.

- Go toolchain lives at `~/.local/sdk/go/bin` (not on PATH by default); `make build GO=~/.local/sdk/go/bin/go`.
- Recipe options are the single source of truth: flags (`internal/cli/new.go`), TUI (`internal/tui`) and validation are generated from `Option` schemas. Don't hand-write per-option flags.
- `.lando.yml` edits go through `lando.Apply` with the recipe's `owned` list: anything not owned must survive untouched. Owned list items are tagged `# stamper` (scalars) or reference `.lando/stamper/`. Add new Landofile keys to `owned` in `internal/recipe/wordpress/plan.go`, or `stamper config` can't clean them up.
- Freshness is a product requirement: when skill content or recipe version tables change, bump `internal/skills/library/manifest.yml` `updated:` and the recipe `revision` (`internal/recipe/wordpress/wordpress.go`). Boot checks in `internal/freshness` depend on them.
- Version tables mirror the installed Lando plugins (`~/.lando/plugins/@lando/*/builders/*.js`, e.g. PHP ≤ 8.5 in @lando/php 1.12). Verify there before adding versions.
- Tests: `go test ./...`. End-to-end: stamp in a scratch dir, `lando start`, then `lando destroy -y`.

# stamper

Stamp out production-matching [Lando](https://lando.dev) environments, change them later without
hand-editing YAML, and install curated agent skills for Claude Code and Codex.

Everything works both ways: **interactive** (charmbracelet forms) when run in a terminal with
missing information, **flags only** (`--yes`) for scripts and CI.

```sh
stamper                                        # interactive menu
stamper new wordpress myshop.lndo.site         # defaults: Apache, PHP 8.4, MariaDB 11.4, Mailpit
stamper new wordpress shop.lndo.site -w nginx -p 8.3 -d mysql:8.0 -c redis:8 \
        --page-cache varnish:6 -e igbinary,pcov --db-admin phpmyadmin --yes --start
stamper config                                 # interactive editor for the current project
stamper config set php=8.5 object-cache=valkey:8 --rebuild
stamper skills install --all                   # into ./.claude/skills and ./.agents/skills
```

## Install

```sh
go install github.com/pipejesus/stamper/cmd/stamper@latest
```

or download a binary from [Releases](https://github.com/pipejesus/stamper/releases)
(`stamper_<os>_<arch>`), then keep it current with `stamper self-update`.

## Build

```sh
make build                     # → bin/stamper
make install                   # → $GOBIN/stamper
make release                   # dist/stamper_<os>_<arch>, the assets self-update expects
git tag v0.2.0 && git push --tags  # CI builds and publishes the release
```

## WordPress recipe

| Area | Choices |
|---|---|
| Web server | Apache 2.4 (mod_php), Nginx 1.29 + PHP-FPM, OpenLiteSpeed 1.9.4 + LSPHP (experimental) |
| PHP | 8.5, 8.4 (default), 8.3, 8.2, 8.1, 8.0, 7.4 (OLS: 8.2–8.5) |
| Database | MariaDB 11.8 / 11.4 / 10.11 / 10.6, MySQL 8.4 / 8.0 / 5.7 |
| Object cache | Redis 8 / 7.4, Valkey 8, Memcached 1.6, APCu (redis, memcached, apcu, igbinary-ready PHP extensions are built in) |
| Page cache | Varnish 6 on `edge.<url>` |
| PHP | extra extensions (`install-php-extensions`), Xdebug mode, OPcache profile (dev / prod / prod + JIT), memory and upload limits |
| Tools | Mailpit or MailHog, phpMyAdmin, Node 22/24/26 with `lando npm/npx/node` |
| WordPress | auto download + install, multisite (subdirectory / subdomain with wildcard proxy), locale, admin credentials |

`stamper recipes wordpress` (or `stamper config options`) lists every key and value.

### What gets written

```
.lando.yml                  # Landofile; stamper only reconciles the keys it owns
.lando/stamper/state.yml    # recipe + chosen values (source of truth for `stamper config`)
.lando/stamper/php.ini      # limits, OPcache/JIT, APCu, Xdebug trigger mode
.lando/stamper/nginx.conf   # Nginx only: vhost with upload limit and multisite rewrites
.lando/stamper/wp-setup.sh  # idempotent bootstrap, runs on every `lando start` / `lando wp-setup`
```

The setup script downloads and installs WordPress if needed, then sets `WP_ENVIRONMENT_TYPE=local`
and the debug constants, and wires up the object cache (`WP_REDIS_HOST`, plus the redis-cache drop-in unless
you pass `--cache-plugin=false`). It writes `.htaccess` for Apache/OLS. It hashes its inputs and exits
straight away when nothing changed.

### Changing an environment

`stamper config set k=v…` / `stamper config edit` re-render the recipe and reconcile `.lando.yml`.
Your own services, tooling, overrides, list items and comments survive, while stale stamper keys are removed
(e.g. switching Redis → none removes the service, its tooling and the drop-in). List items stamper owns
carry a `# stamper` comment or reference `.lando/stamper/`. Options that need more than a rebuild
(database engine, web server) print what to run. `--dry-run` prints the result without writing.

## Agent skills

| Skill | Covers |
|---|---|
| `wp-gutenberg-blocks` | Native block development (block.json, PHP registration/render, JS edit/save, Interactivity API, bindings, tooling), as of WP 7.x |
| `css-modern` | 2026 CSS: container/style/scroll-state queries, typography, color, JS-free UI (popover, anchor positioning, customizable select…) |
| `php-modern` | PHP 8.0 → 8.6 features, safety, performance, WordPress compatibility |

Skills are embedded in the binary. Each one has a short `SKILL.md` and `references/` files the agent loads
only when it needs them. `stamper skills install` writes them to `.claude/skills/` (Claude Code) and
`.agents/skills/` (Codex) in the target folder (`-C`, default: current folder), or into `$HOME`
with `--global`. A `.stamper-skill.json` marker tracks the installed version, so `stamper skills update`
refreshes only untouched skills. Skills you edited locally are protected unless you pass `--force`.

## Staying fresh

On every run stamper checks (offline, plus a release lookup on GitHub that is cached for a day):

- bundled skills older than 90 days, recipe version tables older than 120 days → `stamper self-update`
- installed skills in this project or `$HOME` with a newer bundled version → `stamper skills update`
- a Landofile rendered by an older recipe revision → `stamper config upgrade`
- a newer stamper release → `stamper self-update`

Disable with `--no-update-check` or `STAMPER_NO_UPDATE_CHECK=1`. `stamper doctor` shows everything at once.

## Architecture

```
cmd/stamper               entry point (version via -ldflags)
internal/cli              cobra commands (styled by charm fang); flags generated from recipe options
internal/tui              huh forms generated from recipe options; lipgloss styles
internal/recipe           Recipe interface, Option schema, validation, registry
internal/recipe/wordpress the WordPress recipe + templates/
internal/lando            Landofile load/save, YAML builders, ownership-aware reconcile
internal/project          project dir + state file + applying plans
internal/skills           embedded skill library (library/<name>/, manifest.yml) and installer
internal/freshness        boot-time staleness checks
internal/selfupdate       GitHub-release self update
```

### Adding a recipe

1. Create `internal/recipe/<name>/` implementing `recipe.Recipe`: `Options()` (the schema drives flags,
   TUI and validation), `Normalize`, `Check`, `Plan` (returns the managed Landofile tree, the owned
   paths, support files and notes), `Read` and `Owns`. Optionally implement `Revision()` for freshness.
2. Register it in `internal/cli/root.go` (`recipe.Register(<name>.New())`).

`stamper new <name>`, `stamper config …` and the interactive menu pick it up automatically.

### Adding or refreshing a skill

Put it in `internal/skills/library/<name>/` (`SKILL.md` with `name` + `description` frontmatter,
plus `references/`), and set its `updated:` date in `internal/skills/library/manifest.yml`.

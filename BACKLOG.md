# Backlog

Open items as of 2026-11-19 (end of the first session). Newest decisions are in git history.

## Small fixes
- [ ] **memcache on PHP 8.5:** PECL memcache doesn't compile on 8.5; stamper now skips it with a plan warning. Re-test when it ships 8.5 support and drop the warning.
- [ ] **WP-CLI deprecation noise on PHP 8.5:** every `lando wp` prints `Case statements followed by a semicolon` from WP-CLI's bundled react/promise. Quiet `E_DEPRECATED` for WP-CLI only (e.g. a `wp` tooling override running `php -d error_reporting=24575 /usr/local/bin/wp`, used by the setup script too), and keep deprecations visible for the site.
- [ ] **MariaDB 11 healthcheck retries:** Lando's default healthcheck calls the deprecated `mysql` binary and retries a few times. Set `services.database.healthcheck` to `mariadb-admin ping …` for mariadb ≥ 10.4 (and add the path to `owned`).
- [ ] **PHP 8.6 status:** GA was scheduled for 2026-11-19, but php.net still showed 8.5.12 at 11:20 UTC. Once released, update the status line in `internal/skills/library/php-modern/references/php-8.6.md` (and SKILL.md if it says "scheduled"), then bump `manifest.yml`. Add 8.6 to the PHP choices once @lando/php ships images.

## Release / repo
- [ ] Tag `v0.1.0` (triggers the release workflow) — ask the user first.
- [ ] Add a LICENSE: the user's choice; the repo is public.

## Unverified / experimental
- [ ] **OpenLiteSpeed stack:** never run end to end. Verify the vhost `sed` rewrite, the `/entrypoint.sh true` command under Lando's entrypoint, file ownership for `wp` runs, and HTTPS through the proxy.
- [ ] **Interactive TUI:** renders correctly in a pty, but a full click-through of `new`, `config edit` and `skills install` hasn't been done.
- [ ] **Self-update:** untested until a release exists.

## Ideas
- More recipes (the modular system is ready: implement `recipe.Recipe`, register in `internal/cli/root.go`).
- More skills (add `library/<name>/`, plus a manifest entry).

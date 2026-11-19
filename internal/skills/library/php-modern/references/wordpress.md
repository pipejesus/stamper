# Modern PHP in WordPress plugins & themes

## Facts (verified 2026-11-19)

| Item | Value | Source |
|---|---|---|
| WP minimum PHP | 7.4 (raised from 7.2.24 in WP 7.0) | make.wordpress.org/core/2026/01/09/dropping-support-for-php-7-2-and-7-3; `wp-includes/version.php` `$required_php_version = '7.4'` |
| WP recommended PHP | 8.3+ | wordpress.org/about/requirements |
| WP 7.1 (2026-08-19) compatibility | full: 8.2–8.5; works: 8.0, 8.1 (EOL) | make.wordpress.org/hosting/handbook/compatibility/version/7-1 |
| "Beta support" label | retired May 2026 (retroactively) | make.wordpress.org/core "PHP support clarification, spring 2026 edition" |
| PHP 8.6 support | not yet documented | — |
| Raising minimum to 8.x | open discussion, Trac #64494 (≈ <5% users left behind rule) | core.trac.wordpress.org/ticket/64494 |
| Latest stable / trunk | 7.1.5 / 7.2-RC2 | wordpress-develop tags |

## Gate by PHP version

| Where | Header | Enforcement |
|---|---|---|
| Plugin main file | `Requires PHP: 8.1` | WP ≥ 5.2 refuses activation (`validate_plugin_requirements()`) |
| Plugin `readme.txt` | `Requires PHP: 8.1` | wordpress.org directory / update UI flag incompatible versions |
| Theme `style.css` | `Requires PHP: 8.1` (required for wordpress.org themes) | WP ≥ 5.5 refuses switch (`validate_theme_requirements()`) |

Headers don't protect an already-active plugin when the host later downgrades PHP → keep the main file parseable on the WP minimum and guard:

```php
<?php
/**
 * Plugin Name: Acme
 * Requires PHP: 8.2
 */
if ( PHP_VERSION_ID < 80200 ) {
	add_action( 'admin_notices', static function () {
		echo '<div class="notice notice-error"><p>' . esc_html__( 'Acme requires PHP 8.2+.', 'acme' ) . '</p></div>';
	} );
	return;
}
require __DIR__ . '/src/bootstrap.php'; // modern syntax only from here
```

Choose the floor by the highest **syntax** feature used (syntax can't be polyfilled). Functions can be polyfilled.

## Functions WordPress core already polyfills (`wp-includes/compat.php`)

| Function | Native | WP polyfill since |
|---|---|---|
| `str_contains()`, `str_starts_with()`, `str_ends_with()` | 8.0 | 5.9 |
| `array_is_list()` | 8.1 | 6.5 |
| `array_find()`, `array_find_key()`, `array_any()`, `array_all()` | 8.4 | 6.8 |
| `array_first()`, `array_last()` | 8.5 | 6.9 |
| `clamp()` | 8.6 | 7.1 (throws `InvalidArgumentException` instead of `ValueError` on PHP 7.4) |

Set `Requires at least:` to the WP version providing the polyfill. Other functions: `symfony/polyfill-php80` … `polyfill-php86` via Composer — scope vendor code (Strauss / PHP-Scoper) to avoid clashes with other plugins.

## WordPress Coding Standards vs modern PHP

WPCS 3.4.1 (2026-07-27). Handbook rules that touch modern syntax:

- Long arrays `array( … )` required in WPCS (`[]` flagged). Examples in this skill use `[]` for brevity — convert when the project uses WPCS.
- No short ternary `?:`. Yoda conditions for `==`/`===`/`!=`/`!==` with a literal.
- Explicit visibility everywhere; modifier order: `abstract|final` → `readonly` (classes); visibility → `static|readonly` → type (properties).
- `new Foo()` always with parentheses (also required for 8.4 `new Foo()->m()`).
- Enum/class names `Capitalized_Words`; one OO structure per file.
- Named arguments: core itself "cannot yet use those" (7.4 floor); since PHP 8.0 core treats parameter renames as breaking, so named args into WP functions are acceptable in 8.0+ plugins.
- Type declarations allowed per PHP version; `declare(strict_types=1)` not addressed — allowed.
- Compatibility scan: `PHPCompatibilityWP` (stable 2.1.x; 3.0.0-alpha3 / PHPCompatibility 10.0.0-alpha3 released 2026-11-17 cover PHP 8.x sniffs):
  `vendor/bin/phpcs -p . --standard=PHPCompatibilityWP --runtime-set testVersion 8.1-`

## WordPress-specific pitfalls

- **Removing hooks with first-class callables fails**: `$this->save(...)` creates a new `Closure` each time; `remove_action('save_post', $this->save(...))` matches nothing. Use `[ $this, 'save' ]` (or store the closure) for removable hooks; closures/arrow fns as hook callbacks are discouraged by the handbook for the same reason.
- **Strict types don't reach hook callbacks** (core is non-strict); typed callback params get `TypeError` on unexpected `null`/arrays from other plugins → accept `mixed` at the boundary (see types-and-safety.md).
- **PHP 8.1 null-to-internal deprecations**: `get_option()`, `get_post_meta()`, `$_GET[...] ?? null` often yield `false`/`null`/`''` → cast: `trim( (string) get_option( 'x', '' ) )`.
- **PHP 8.2 dynamic properties**: don't add props to `WP_Post`, `WP_User`, `WP_Query` or other plugins' objects; use a `WeakMap` or post meta. Declare all props in your own classes.
- **Persisted data**: `update_option()`/meta `serialize()` objects. Store enum `->value`, arrays or JSON; never readonly/enum objects (renames/autoload breaks unserialize).
- **PHP 8.6 session defaults** (strict mode, HttpOnly, SameSite=Lax) affect plugins using native `$_SESSION` across cross-site POST (payment returns, SSO).
- **PHP 8.6 `ValueError` for NUL bytes** in `file_exists()`/`is_file()` etc.: sanitize upload/path input (`sanitize_file_name()`, `validate_file()`) before filesystem calls.
- `mysqli` throws by default since 8.1, but `wpdb` calls `mysqli_report( MYSQLI_REPORT_OFF )` (process-global) → inside WP, raw mysqli calls return `false` instead of throwing; check results. Prefer `$wpdb->prepare()` over `mysqli::quote_string()` (8.6).
- `#[\SensitiveParameter]` on functions receiving API keys/passwords keeps them out of `WP_DEBUG_LOG` traces.
- Keep templates (theme `*.php` views) simple; modern constructs belong in `src/` classes autoloaded via Composer PSR-4.

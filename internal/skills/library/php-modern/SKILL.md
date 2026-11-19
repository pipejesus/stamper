---
name: php-modern
description: Modern PHP 8.0–8.6 idioms, minimum-version table, deprecations/BC breaks and performance/safety rules (match, enums, readonly, property hooks, asymmetric visibility, pipe operator, partial function application, clamp, array_find…); use when writing, refactoring, reviewing or upgrading PHP 8+ code, choosing a `Requires PHP` floor, or fixing PHP 8.x deprecations — including WordPress plugins and themes.
---

# Modern PHP 8.x

## Workflow

1. Find the PHP floor before writing code: `composer.json` `require.php` / `config.platform.php`, plugin header or `readme.txt` `Requires PHP`, theme `style.css` `Requires PHP`, `.phpcs.xml` `testVersion`, CI matrix. No floor found → ask, or assume the WordPress minimum (7.4) for distributed WP code.
2. Use only features ≤ floor (table below). Syntax can't be polyfilled; functions can.
3. Refactor toward the "prefer" column when touching code; don't mass-rewrite stable code just for style.
4. When upgrading the floor, walk each version file in `references/` for BC breaks + deprecations in between.
5. Verify: `php -l`, PHPStan/Psalm, PHPCompatibility(WP) with `testVersion <floor>-`, tests on floor and latest PHP.

## Prefer → over

| Prefer | Over | Min |
|---|---|---|
| `match ($x) { 'a', 'b' => 1, default => 0 }` (strict, expression, throws `UnhandledMatchError`) | `switch` assigning a variable | 8.0 |
| Constructor promotion `__construct(private Repo $repo)` | property + assignment boilerplate | 8.0 |
| Named args for booleans/optional params `json_encode($d, flags: JSON_THROW_ON_ERROR)` | positional `null, null, true` | 8.0 |
| Nullsafe `$a?->b()?->c` | nested `isset`/`&&` chains | 8.0 |
| `str_contains/str_starts_with/str_ends_with` | `strpos() !== false`, `substr()` compares | 8.0 |
| `throw` in expressions `$x ?? throw new E()` | `if (!$x) { throw … }` boilerplate | 8.0 |
| Union types / `mixed` in signatures | `@param` docblock-only types | 8.0 |
| `get_debug_type($v)` in error messages | `gettype()` / `get_class()` | 8.0 |
| Attributes `#[Attr]` | annotation docblocks | 8.0 |
| `WeakMap` for per-object caches/extra data | `SplObjectStorage`, dynamic props, `spl_object_hash` keys | 8.0 |
| Backed/pure `enum` with methods | class constants + `in_array()` validation | 8.1 |
| `readonly` properties / `readonly class` (8.2) for value objects | private prop + getter | 8.1 |
| First-class callable `strlen(...)`, `$this->m(...)` | `'strlen'`, `[$this, 'm']`, `Closure::fromCallable()` (except removable WP hooks) | 8.1 |
| `new` in initializers `Logger $l = new NullLogger()` | `?Logger $l = null` + `??=` | 8.1 |
| `never` return for throw/exit helpers | `void` + docblock | 8.1 |
| `array_is_list()` | `array_keys($a) === range(…)` | 8.1 |
| Spread with string keys `[...$defaults, ...$args]` | `array_merge()` for literals | 8.1 |
| `#[\SensitiveParameter]` on secrets | leaking passwords in traces | 8.2 |
| `Random\Randomizer` | `mt_rand`/`rand`/`shuffle`/`lcg_value` | 8.2 |
| DNF `(A&B)\|null`, `true`/`false`/`null` types | `mixed` + docblock | 8.2 |
| Typed class constants `const string X = 'x';` | untyped constants | 8.3 |
| `#[\Override]` on every overriding method (props 8.5, consts 8.6) | silent typos in overrides | 8.3 |
| `json_validate()` | `json_decode()` only to validate | 8.3 |
| `Foo::{$name}` | `constant(Foo::class . '::' . $name)` | 8.3 |
| `str_increment()` | `$s++` on strings | 8.3 |
| Property hooks `public string $email { set => strtolower($value); }` | getter/setter pairs | 8.4 |
| Asymmetric visibility `public private(set) array $items` | private + getter for read-only API | 8.4 |
| `array_find/array_find_key/array_any/array_all` | `foreach`+`break`, `array_filter()[0]` | 8.4 |
| `new Foo()->bar()` | `(new Foo())->bar()` | 8.4 |
| `#[\Deprecated(message:, since:)]` | manual `trigger_error(E_USER_DEPRECATED)` | 8.4 |
| `mb_trim()`, `mb_ucfirst()` for user text | `trim()`/`ucfirst()` on UTF-8 | 8.4 |
| `Dom\HTMLDocument::createFromString()` + `querySelector()` | `DOMDocument::loadHTML()` + XPath for HTML5 | 8.4 |
| `BcMath\Number` / `RoundingMode` enum | `bc*()` strings / `PHP_ROUND_*` ints | 8.4 |
| Pipe `$x \|> trim(...) \|> strtolower(...)` | nested calls / temp vars | 8.5 |
| `clone($this, ['x' => $x])` withers | `new static(...all props...)`, reflection | 8.5 |
| `array_first()` / `array_last()` | `reset()`/`end()`, `array_values($a)[0]` | 8.5 |
| `Uri\WhatWg\Url` / `Uri\Rfc3986\Uri` | `parse_url()` for validation/allowlists | 8.5 |
| `#[\NoDiscard]` on results that must be used; `(void)` to discard | silent ignored return values | 8.5 |
| Partial application `array_map(str_replace(' ', '-', ?), $a)`, `\|> f(?, 2)` | `fn($x) => f($x, 2)` wrappers | 8.6 |
| `clamp($v, 1, 100)` | `max(1, min(100, $v))` | 8.6 |
| `SortDirection` enum in own APIs | `bool $asc`, `'ASC'\|'DESC'` strings | 8.6 |
| `Time\Duration` params | `int $timeoutMs` (unit in name) | 8.6 |
| Readonly prop defaults `public readonly string $t = 'x';` | assigning constants in ctor | 8.6 |
| `StreamErrorMode::Exception` context | `@file_get_contents()` + `false` check | 8.6 |

Readability limits: use named args when meaning isn't obvious from the value (booleans, nulls, skipped optionals), not everywhere; break pipes at ~1 step/line; keep `match` arms one expression (extract methods otherwise); prefer `array_find()` over pipes of closures when it reads clearer.

## Minimum-version lookup

| PHP | Language | Library |
|---|---|---|
| 7.4 | typed props, arrow fns `fn`, `??=`, spread in arrays (int keys), preloading | — |
| 8.0 | union types, `mixed`, `static` return, `match`, nullsafe, named args, promotion, attributes, `throw` expr, non-capturing `catch`, `$obj::class`, `Stringable`, trailing comma in params, JIT | `str_contains/starts_with/ends_with`, `get_debug_type`, `fdiv`, `WeakMap`, `ValueError`, `PhpToken` |
| 8.1 | enums, `readonly` props, FCC `f(...)`, `new` in initializers, `A&B`, `never`, `final const`, `0o` octal, string-key unpack, Fibers | `array_is_list`, `enum_exists`, `fsync`, `#[\ReturnTypeWillChange]` |
| 8.2 | `readonly class`, DNF, `true`/`false`/`null` types, trait constants, enum props in const expr | `Random\Randomizer`, `#[\SensitiveParameter]`, `#[\AllowDynamicProperties]`, `mysqli_execute_query`, `memory_reset_peak_usage` |
| 8.3 | typed class consts, `#[\Override]`, `C::{$name}`, readonly reinit in `__clone`, readonly anon classes, static var initializer exprs | `json_validate`, `mb_str_pad`, `str_increment/decrement`, `Randomizer::getBytesFromString/getFloat` |
| 8.4 | property hooks, asymmetric visibility, `new X()->m()`, `#[\Deprecated]`, lazy objects, interface properties | `array_find/find_key/any/all`, `mb_trim/ltrim/rtrim/ucfirst/lcfirst`, `BcMath\Number`, `Dom\HTMLDocument`, `RoundingMode`, `PDO::connect`, `request_parse_body`, `http_get_last_response_headers` |
| 8.5 | pipe `\|>`, clone-with, `#[\NoDiscard]` + `(void)`, `#[\Override]` on props, `final` promotion, static aviz, closures/FCC/casts in const expr, attributes on consts | `array_first/last`, URI ext, `FILTER_THROW_ON_FAILURE`, `get_error_handler`, `IntlListFormatter`, `Closure::getCurrent` |
| 8.6* | partial function application `?`/`...`, readonly defaults, `#[\Override]` on consts, enum `__debugInfo`, write to props of const objects | `clamp`, `SortDirection`, `Time\Duration`, stream errors API, `Io\Poll`, `ReflectionProperty::isReadable/isWritable`, URI builders, `mysqli::quote_string`, `grapheme_strrev` |

\*8.6 GA scheduled 2026-11-19 — see `references/php-8.6.md` for release status.

## Pitfalls

Safety
- `strict_types` covers calls *from* the file only; framework callbacks (WP hooks) still receive coerced/null values → accept `mixed` at boundaries, narrow inside.
- `match` without `default` throws on unknown input — intended for enums, dangerous for raw user input.
- `readonly` is shallow; objects inside remain mutable. Clone-with from outside needs `public(set) readonly`.
- Enum `::from()` throws `ValueError`; use `::tryFrom()` on user input. Persist `->value`, not serialized enums.
- PFA (8.6) evaluates bound args at creation (`f($obj->x, ?)` snapshots `$obj->x`); arrow fns evaluate at call.
- `#[\SensitiveParameter]` only redacts traces — not logs you write yourself.
- Security randomness: `random_bytes`/`random_int`/`Randomizer` (Secure engine) only.

Performance
- Use OPcache always; JIT off by default since 8.4 and rarely helps IO-bound apps — measure.
- Pipe/`array_map` steps as FCC/PFA compile to direct calls/loops; arrow-fn steps allocate closures.
- `array_find/any/all` short-circuit; `array_filter` + `[0]` builds a full array.
- Fully qualify hot built-ins in namespaces (`\strlen`, `\count`, `\in_array`) or `use function`.
- Dynamic properties are slower and deprecated; declare props.

BC breaks / deprecations by upgrade step
- →8.0: `0 == 'abc'` false; internals throw `TypeError`/`ValueError`; `@` doesn't hide fatals; resources→objects (`is_resource()` checks); removed `each`, `create_function`, curly offsets.
- →8.1: `null` to non-nullable internal params deprecated (`strlen(null)`); tentative return types on `ArrayAccess`/`Iterator`/`Countable`/`JsonSerializable` overrides; `htmlspecialchars` default `ENT_QUOTES`; mysqli throws; `Serializable` without `__serialize`.
- →8.2: dynamic properties deprecated; `"${x}"` deprecated; `utf8_encode/decode` deprecated; `strtolower()` & co. ASCII-only; `"self::m"` callables deprecated.
- →8.3: `++` on empty/non-alnum strings deprecated; `get_class()` no-arg deprecated; `range()` stricter.
- →8.4: implicit nullable `T $x = null` deprecated; `E_USER_ERROR` in `trigger_error` deprecated; CSV `$escape` must be explicit; bcrypt cost 12; JIT config semantics; IMAP/OCI8/pspell removed from core.
- →8.5: backticks, `(integer)`/`(boolean)`/`(double)` casts, `case x;`, `null` array offsets, `$http_response_header`, `curl_close`/`imagedestroy` deprecated; float→int overflow warns; OPcache always built in.
- →8.6: `return` in `finally`, values from `__construct`, `is_double/is_long/is_integer/doubleval`, `spl_object_hash`, `strcoll`, mbregex deprecated; session defaults strict/HttpOnly/SameSite=Lax; `trim()` strips `\f`; NUL bytes in filesystem paths → `ValueError`.

## WordPress

- WP minimum PHP 7.4 (since WP 7.0); recommended 8.3+; WP 7.1 fully compatible with 8.2–8.5 (8.6 not yet documented).
- Gate with `Requires PHP:` in the plugin main file header (activation blocked since WP 5.2), `readme.txt`, and theme `style.css` (switch blocked since WP 5.5). Keep the main plugin file parseable on 7.4 and `PHP_VERSION_ID`-guard before loading modern code.
- WP polyfills `str_contains` & co. (5.9), `array_is_list` (6.5), `array_find/any/all` (6.8), `array_first/last` (6.9), `clamp` (7.1) — usable with a lower `Requires PHP` if `Requires at least` matches.
- WPCS: long `array()` syntax, no `?:`, Yoda conditions, explicit visibility, modifier order `final readonly class`. Use `[ $this, 'm' ]` (not `$this->m(...)`) for hooks you may remove.

## References

| File | Read when… |
|---|---|
| [references/php-8.0.md](references/php-8.0.md) | upgrading from 7.x; using match/promotion/named args/nullsafe/attributes; fixing 8.0 comparison or TypeError breaks |
| [references/php-8.1.md](references/php-8.1.md) | designing enums/readonly/FCC; fixing null-to-internal or tentative-return-type deprecations |
| [references/php-8.2.md](references/php-8.2.md) | readonly classes, DNF, Random extension; fixing dynamic-property or `${}` deprecations |
| [references/php-8.3.md](references/php-8.3.md) | typed constants, `#[\Override]`, readonly cloning, `json_validate`; `++`-on-string/`range()` changes |
| [references/php-8.4.md](references/php-8.4.md) | property hooks, asymmetric visibility, lazy objects, `array_find`, HTML5 DOM; implicit-nullable/CSV/bcrypt changes |
| [references/php-8.5.md](references/php-8.5.md) | pipe operator, clone-with, `#[\NoDiscard]`, URI extension; cast/backtick/null-offset deprecations |
| [references/php-8.6.md](references/php-8.6.md) | 8.6 release status; partial application, `clamp`, `Duration`, stream errors, readonly defaults; session/NUL-byte BC breaks |
| [references/types-and-safety.md](references/types-and-safety.md) | choosing types/immutability/enums, strict_types at boundaries, attributes, exceptions, crypto/randomness, input safety |
| [references/performance.md](references/performance.md) | tuning OPcache/JIT, hot-path idioms, what's compiled away, what to avoid |
| [references/wordpress.md](references/wordpress.md) | WP PHP requirements, version gating code, core polyfills, WPCS conflicts, WP-specific PHP 8 pitfalls |

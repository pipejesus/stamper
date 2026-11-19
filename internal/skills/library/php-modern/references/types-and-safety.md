# Types & safety (cross-version)

## `declare(strict_types=1)`

- Per file, first statement. Governs **calls made from** that file (incl. internal functions) and `return` in that file; typed property writes in that file.
- Does NOT govern calls into your code from a non-strict file. WordPress core is non-strict → your hook callbacks still receive coerced values, and an uncoercible value (`null`, array) → `TypeError` → fatal site error.
- Rule: strict_types in new files; at framework boundaries (hooks, REST callbacks, shortcodes, `get_option()`/`get_post_meta()` results) accept `mixed`/nullable and narrow explicitly.

```php
// fragile: WP may pass null/'' (e.g. filters called by other plugins)
add_filter('the_title', fn(string $t, int $id): string => …, 10, 2);
// robust
add_filter('the_title', static function (mixed $t, mixed $id = 0): string {
    return is_string($t) ? strtoupper($t) : (string) $t;
}, 10, 2);
```

## Type system reference

| Construct | Since | Note |
|---|---|---|
| `?T` | 7.1 | = `T\|null` |
| `A\|B`, `mixed`, `static` (return) | 8.0 | `mixed` already includes `null` (no `?mixed`) |
| `never` | 8.1 | function must throw/exit |
| `A&B` | 8.1 | classes/interfaces only |
| `null`, `false` standalone; `true` | 8.2 | `true` for "always succeeds" |
| DNF `(A&B)\|null` | 8.2 | intersections must be parenthesized |
| Typed class constants | 8.3 | `const int MAX = 10;` |
| Implicit nullable `T $x = null` | deprecated 8.4 | write `?T $x = null` |
| Property type + hooks + aviz | 8.4 | aviz requires typed property |

- Prefer narrow types (`non-empty-string`, `list<int>`, `array{id:int}`) in PHPDoc for static analysis (PHPStan/Psalm) where native types can't express shape.
- `iterable` = `Traversable|array` (8.2 treats it as that union).
- Return `static` for fluent/wither APIs; `self` inside `final` classes.

## Immutability

| Need | Use | Since |
|---|---|---|
| Write-once prop | `public readonly T $x` | 8.1 |
| Whole value object | `final readonly class` | 8.2 |
| Deep copy of readonly | reassign in `__clone()` | 8.3 |
| Public read, private write, mutable | `public private(set) T $x` | 8.4 |
| Derived/validated prop | property hook `get`/`set` | 8.4 |
| Wither | `clone($this, ['x' => $x])` | 8.5 |
| Readonly with constant default | `public readonly string $t = 'x';` | 8.6 |

Gotchas: readonly needs a type; no default before 8.6; can't `unset()` after init; `readonly` object props are shallow (inner objects stay mutable); readonly + hooks forbidden; readonly from outside clone-with needs `public(set)`.

## Enums

- Use for closed sets (statuses, modes, roles). Backed (`: string`/`: int`) when persisted/serialized; pure otherwise.
- Persist `->value`, rehydrate with `::tryFrom()` + fallback; never store `serialize()`d enums in options/meta (class must exist on unserialize; renames break data).
- Exhaustive `match ($enum)` with no `default` → `UnhandledMatchError` surfaces new cases. Pair with static analysis.
- Add behavior as methods/interfaces on the enum rather than `switch` scattered across code.
- Constants on enums allowed; enum cases are objects (`instanceof`, `===`), singletons.

## Attributes that prevent bugs

| Attribute | Since | Effect |
|---|---|---|
| `#[\ReturnTypeWillChange]` | 8.1 | silence tentative-return-type deprecation (lib supporting < 8.1) |
| `#[\AllowDynamicProperties]` | 8.2 | opt-in legacy dynamic props |
| `#[\SensitiveParameter]` | 8.2 | redacts arg in traces (`SensitiveParameterValue`) — passwords, tokens, API keys |
| `#[\Override]` | 8.3 methods, 8.5 props, 8.6 consts/enum cases | compile error if nothing overridden |
| `#[\Deprecated]` | 8.4 functions/methods/class consts, 8.5 global consts | engine-emitted deprecation |
| `#[\NoDiscard]` + `(void)` | 8.5 | warn on ignored results (e.g. immutable `with*()` returns) |

## Errors & exceptions

- 8.0+: internal functions throw `TypeError`/`ValueError` (not warnings). Catch `\ValueError` where inputs are user-controlled; 8.6 adds many more (NUL bytes in paths, out-of-range ints).
- Catch `\Throwable` only at process boundaries (REST handler, cron runner, CLI); otherwise catch specific types.
- `JSON_THROW_ON_ERROR` (7.3) for `json_decode/encode`; `json_validate()` (8.3) when only validating.
- `FILTER_THROW_ON_FAILURE` (8.5) for `filter_var()`.
- 8.6 `StreamErrorMode::Exception` instead of `@file_get_contents()`.
- Never use `@` to hide errors; it masks real faults (and since 8.0 can't silence fatals anyway).

## Randomness & crypto

| Task | Use | Avoid |
|---|---|---|
| Token / nonce / password-reset key | `bin2hex(random_bytes(16))` or `Randomizer::getBytesFromString($alphabet, $n)` (8.3) | `mt_rand`, `rand`, `uniqid`, `str_shuffle`, `lcg_value` (deprecated 8.4) |
| Int in range | `random_int($a, $b)` / `(new Random\Randomizer)->getInt()` | `mt_rand` for security |
| Float [0,1) | `Randomizer::getFloat(0, 1)` / `nextFloat()` (8.3) | `mt_rand()/mt_getrandmax()` (biased) |
| Reproducible tests | `new Randomizer(new Engine\Xoshiro256StarStar($seed))` | `mt_srand()` global state |
| Password hash | `password_hash($p, PASSWORD_DEFAULT)` + `password_needs_rehash()` (bcrypt cost 12 since 8.4); WP: `wp_hash_password()` | `md5`, `sha1`, custom salts |
| Compare secrets | `hash_equals($known, $user)` | `===` (timing) |
| Encryption | `sodium_crypto_secretbox()` / `sodium_crypto_aead_xchacha20poly1305_ietf_*` | `openssl_encrypt` ECB, homemade |
| Post-quantum KEM | `sodium_crypto_kem_*` (8.6, libsodium ≥ 1.0.22) | — |

## Input / output safety

- URLs: `Uri\WhatWg\Url` / `Uri\Rfc3986\Uri` (8.5) for allowlists and SSRF checks; `parse_url()` disagrees with browsers. In WP also `wp_http_validate_url()`, `wp_safe_redirect()`.
- `unserialize($s, ['allowed_classes' => false])` on any untrusted data; prefer JSON.
- HTML escaping default flags are safe since 8.1 (`ENT_QUOTES|ENT_SUBSTITUTE`); in WP use `esc_html()`, `esc_attr()`, `wp_kses()`.
- DB: prepared statements only (`$wpdb->prepare()`, PDO/mysqli params). mysqli throws by default since 8.1.
- Sessions: 8.6 defaults strict mode + HttpOnly + SameSite=Lax; set the same explicitly on older versions.
- Dynamic properties are deprecated (8.2): declare props or use `WeakMap` to attach data to foreign objects (no leaks).
- `ini_set('memory_limit')` can be capped by `max_memory_limit` (8.5).

# PHP 8.6 (GA scheduled 2026-11-19)

**Status (checked 2026-11-19 ~11:00 UTC):** NOT yet confirmed released. php.net lists 8.5.12 as current stable; latest php-src tag is `php-8.6.0RC7` (2026-11-12); `php.net/releases/8.6` is a pre-release page ("will be released on November 19, 2026"). Release schedule: wiki.php.net/todo/php86. Everything below is from `PHP-8.6/UPGRADING` at RC7 (feature-frozen); every RFC listed is status *Implemented*. Re-check php.net before claiming GA.

## Language

### Partial function application (RFC `partial_function_application_v2`)

```php
// before
$modern = array_filter($versions, fn($v) => version_compare($v, '8.6', '>='));
$slug   = $title |> (fn($s) => str_replace(' ', '-', $s)) |> strtolower(...);
// after
$modern = array_filter($versions, version_compare(?, '8.6', '>='));
$slug   = $title |> str_replace(' ', '-', ?) |> strtolower(...);
```

- `?` = exactly one arg at that position; `...` = all remaining args; named placeholders `f(subject: ?)`.
- Returns a `Closure`; bound argument expressions are evaluated **at creation**, not at call (unlike arrow fns).
- Closure inherits param names/types/defaults/by-ref; `f(...)` = first-class callable (8.1).
- Not allowed: `new Foo(?)` (compile error — use static factory `Foo::make(?)`), context-dependent functions (`compact`, `extract`, `func_get_args`, `get_defined_vars`). Positional `?` after named arg = compile error.
- Allowed in constant expressions when args are constant.
- `|> f(?)`, `|> f(1, ?)`, `|> f(1, ...)` compile to direct calls; `array_map()` with FCC/PFA compiles to a `foreach`.

### Readonly property defaults

```php
// before: assign in constructor
// after
final readonly class SlugRule implements Rule {     // interface Rule { public string $parser { get; } }
    public string $parser = SlugParser::class;
}
```

Default counts as the initialization → assigning in constructor throws; still re-settable once in `__clone()` / clone-with. Readonly props still can't have hooks.

### Other

| Feature | Example |
|---|---|
| `#[\Override]` on class constants and enum cases | `#[\Override] public const string TABLE = 'wp_x';` |
| Enums may define `__debugInfo()` | `public function __debugInfo(): array { return [$this->name]; }` |
| Write props of objects held in constants | `const CFG = new stdClass(); CFG->debug = true; Foo::OPTS->retries++;` (rebinding / `CONST[0] = …` still invalid) |
| `WeakMap` reference assignment without pre-existing key | `$ref = &$map[$obj];` |

## New functions / classes

- `clamp($value, $min, $max)` — any comparable type; `ValueError` if `min > max` or bound is `NAN`. Replaces `max($min, min($max, $v))`.
  ```php
  $perPage = clamp((int) ($_GET['per_page'] ?? 10), 1, 100);
  ```
- `SortDirection` enum (global, unbacked: `Ascending`, `Descending`) — use in your own APIs instead of `bool $asc` / `'ASC'`. No built-in function accepts it yet.
- `Time\Duration` (final readonly; `seconds`, `nanoseconds`, `negative`): `fromSeconds()`, `fromMilliseconds()`, `fromMicroseconds()`, `fromNanoseconds()`, `fromMinutes()`, `fromHours()`, `fromIso8601DurationString()`; `add()`, `sub()`, `multiplyBy()`, `divideBy()`, `negate()`, `absolute()`, `Duration::compare()`; `<`/`>`/`min()`/`clamp()` work. Overflow → `Time\TimeException`.
  ```php
  // before (unit lives only in the name)
  function fetch(string $url, int $timeoutMs = 5000) {}
  // after (static calls aren't constant expressions → null default)
  function fetch(string $url, ?Time\Duration $timeout = null) { $timeout ??= Time\Duration::fromSeconds(5); }
  ```
- Stream errors API: context option `'stream' => ['error_mode' => StreamErrorMode::Exception]` → `StreamException` with `getErrors()` of `StreamError` (`code` is `StreamErrorCode` enum); `stream_last_errors()`, `stream_clear_errors()`.
  ```php
  // before
  $json = @file_get_contents($path); if ($json === false) { /* why? */ }
  // after
  $ctx = stream_context_create(['stream' => ['error_mode' => StreamErrorMode::Exception]]);
  try { $json = file_get_contents($path, context: $ctx); } catch (StreamException $e) { log($e->getErrors()); }
  ```
- Polling API `Io\Poll\Context` + `StreamPollHandle` + `Io\Poll\Event` (epoll/kqueue/WSAPoll). Low-level; no event loop.
- Reflection: `ReflectionProperty::isReadable($scope, $obj)`/`isWritable()`, `ReflectionParameter::getDocComment()` (param docblocks), `ReflectionConstant::inNamespace()`, `ReflectionAttribute::getShortName()/getNamespaceName()/inNamespace()`.
- URI: `Uri\Rfc3986\UriBuilder`, `Uri\WhatWg\UrlBuilder`, `Uri\WhatWg\url_percent_encode()`, `getUriType()`, `getHostType()`, `Url::isSpecialScheme()`.
- `mysqli::quote_string()` / `mysqli_quote_string()` (returns quoted literal; in WP still use `$wpdb->prepare()`).
- `pack()/unpack()` `<` `>` endianness modifiers on int/float codes.
- JSON errors include location (`json_last_error_msg()`, `JsonException`).
- `grapheme_strrev()`, `IntlNumberRangeFormatter`, `Locale::getDisplayKeyword[Value]()`, `SpoofChecker::getSkeleton()`.
- `ZipArchive::openString()`/`closeString()`; `ARRAY_FILTER_USE_VALUE`; `ini_get_all()` `builtin_default_value`.
- Sodium: X-Wing / ML-KEM-768 KEM (`sodium_crypto_kem_*`, libsodium ≥ 1.0.22), ipcrypt, SHAKE/TurboSHAKE XOF.
- OpenSSL streams: TLS session resumption, external PSK, TLS 1.3 0-RTT; `openssl_sign()` `$salt_length` for RSA-PSS.
- cURL `CURLOPT_SEEKFUNCTION`; GMP `gmp_prevprime()`, `gmp_powm_sec()`; `finfo_file()` on remote streams.
- INI `error_include_args` (show call args in error messages; respects `#[\SensitiveParameter]`).

## BC breaks

- Session defaults: `session.use_strict_mode=1`, `session.cookie_httponly=1`, `session.cookie_samesite=Lax`. Cross-site POST relying on the session cookie needs `SameSite=None` + `Secure`.
- `trim()/ltrim()/rtrim()` also strip form feed `\f` by default.
- Filesystem functions (`file_exists()`, `is_file()`, `is_dir()`, `filesize()`, `filemtime()`, `stat()`, …), `getenv()/putenv()`, `parse_str()`, `setlocale()`, `proc_open()` `$cwd`: NUL byte in argument → `ValueError` (was silent `false`). Validate user-derived paths.
- `ValueError` instead of silent truncation/overflow: `number_format()` decimals, `sleep()`/`usleep()` > UINT_MAX, invalid `$mode` of `array_filter()`, `array_change_key_case()`, `pathinfo()` flags, `scandir()` order.
- `preg_grep()` returns `false` on PCRE error (was partial array).
- `fclose()`, `file_put_contents()`, `copy()` return `false` when final flush fails.
- `unpack('s<value')` key is now `value`; `base_convert()/bindec()/hexdec()/octdec()` notice on precision loss.
- `??`/`empty()` on magic prop no longer call `__get()` when `__isset()` materialized it. By-ref `foreach` may revisit elements after compaction.
- DOM `@readonly` props now `public private(set)` (different error message). `ZipArchive` (un)serialize throws. `session_encode()` returns `''` for empty session. Hard OOM → `abort()` (exit 134).

## Deprecated

- `return` inside `finally`.
- Returning a value from `__construct()`/`__destruct()`; generator constructors/destructors.
- Identifiers: `let`, `is` (class/function/const/alias names), function named `readonly`, `namespace` as class-constant name, `_` as constant/alias.
- `define($n, $v, true)` third arg; `is_a()`/`is_subclass_of()` with string + `$allow_string=false`; `__debugInfo(): ?array`.
- Aliases: `is_double()` → `is_float()`, `is_long()`/`is_integer()` → `is_int()`, `doubleval()` → `floatval()`.
- `spl_object_hash()` → `spl_object_id()`; `spl_classes()`; `ArrayIterator` sort/flag/serialize methods; `SplFileObject` CSV methods (`fgetcsv`, `fputcsv`, `setCsvControl`, `getCsvControl`).
- `strcoll()`, `SORT_LOCALE_STRING` → `Collator`. `metaphone()`.
- mbstring regex (`mb_ereg*()`, `mb_split()`; Oniguruma EOL) → `preg_*` with `/u`.
- Objects as arrays: `array_walk[_recursive]($obj)`, `http_build_query($obj)`, zlib/bz2 filter params, `mb_convert_variables()` → `get_object_vars()` first.
- `mysqli_get_charset()`, `mysqli::stmt_init()`/`mysqli_stmt` without `$query`.
- `ReflectionProperty::setValue()/setRawValue()` on foreign object; `ReflectionMethod::invoke()` with object for static method.
- Session handlers lacking `create_sid()`/`validateId()`; > 16 `php://filter` filters.

## Performance (from UPGRADING §14)

`printf()` with only `%s`/`%d` compiled to interpolation; `array_map()` + FCC/PFA compiled to `foreach`; faster args to known constructors (`new self()`); stateless closure cache; faster JSON encode, `str_pad()`, `str_split()`, `array_sum()/array_product()` on ints, `array_intersect()`, `array_unshift()`, `array_walk()`; TAILCALL VM faster; JIT on ZTS Apple Silicon.

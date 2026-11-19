# PHP 8.5 (GA 2025-11-20, active until 2027-12-31, security until 2029-12-31)

Source: php-src `PHP-8.5/UPGRADING`, php.net/releases/8.5, RFCs `pipe-operator-v3`, `clone_with_v2`, `marking_return_value_as_important`, `url_parsing_api`.

## Pipe operator `|>`

```php
// before
$slug = strtolower(str_replace(' ', '-', trim($title)));
// after
$slug = $title
    |> trim(...)
    |> (fn($s) => str_replace(' ', '-', $s))   // arrow fn MUST be parenthesized
    |> strtolower(...);
```

- RHS: any callable taking exactly one required arg; by-ref params not allowed.
- Precedence: tighter than comparison and `??`, looser than arithmetic: `$id |> find(...) ?? 'x'` = `($id |> find(...)) ?? 'x'`.
- FCC steps compile to direct calls (no overhead); arrow-fn steps create closures. 8.6 PFA removes the arrow fn: `|> str_replace(' ', '-', ?)`.

## Clone with

```php
// before (readonly withers impossible without reflection)
// after
final readonly class Point {
    public function __construct(public int $x, public int $y) {}
    public function withX(int $x): static { return clone($this, ['x' => $x]); }
}
```

- `clone($obj, ['prop' => v])`; `__clone()` runs first, then assignments (set hooks fire, visibility enforced).
- From outside the class, public readonly props are `protected(set)` → clone-with fails unless declared `public(set) readonly`.
- `clone(...)` usable as callable: `array_map(clone(...), $objs)`.

## Attributes / declarations

| Feature | Example |
|---|---|
| `#[\NoDiscard('reason')]` → warning if result unused; `(void)` cast to discard explicitly | `#[\NoDiscard] function save(): Result {}` / `(void) save();` |
| `#[\Override]` on properties | `#[\Override] public string $name;` |
| `final` promoted properties | `__construct(final public string $id)` |
| Asymmetric visibility on static props | `public private(set) static int $count = 0;` |
| Static closures + FCCs in constant expressions (attribute args, param/prop defaults, consts); no `use`, no arrow fns | `#[Validate(static function ($v) { return $v > 0; })]` |
| Casts in constant expressions | `const N = (int) '5';` |
| Attributes on global `const`; `#[\Deprecated]` on constants | `#[\Deprecated] const OLD = 1;` |
| `#[\DelayedTargetValidation]` | defer attribute-target errors to `newInstance()` |
| `Closure::getCurrent()` | recursion in anonymous functions without `use (&$f)` |
| Fatal errors include backtrace (`fatal_error_backtraces=1`) | — |

## New functions / classes

- `array_first($a)`, `array_last($a)` — value or `null` on empty (replaces `reset()`/`end()` side effects, `$a[array_key_first($a)]`).
- URI extension (always available):
  ```php
  $u = new Uri\WhatWg\Url('https://example.com/a?x=1');   // throws Uri\WhatWg\InvalidUrlException
  $u = Uri\Rfc3986\Uri::parse($input);                     // ?Uri, null if invalid
  $u?->getHost(); $u?->withPath('/b')->toString();         // withers are immutable
  ```
  Prefer over `parse_url()` (non-standard, inconsistent with browsers) for security decisions (SSRF/redirect allowlists).
- `FILTER_THROW_ON_FAILURE` flag → `Filter\FilterFailedException` instead of `false`.
- `get_error_handler()`, `get_exception_handler()`.
- `IntlListFormatter` ("a, b, and c"), `locale_is_right_to_left()`, `grapheme_levenshtein()`, `Locale::addLikelySubtags()`.
- `Dom\Element::$outerHTML`, `insertAdjacentHTML()`, `getElementsByClassName()`, `Dom\ParentNode::$children`.
- `curl_share_init_persistent()` (connection reuse across requests), `curl_multi_get_handles()`.
- `setcookie()`/`session_set_cookie_params()` `partitioned` key (CHIPS); `getimagesize()` HEIF/HEIC + SVG.
- INI: `max_memory_limit` (cap for `ini_set('memory_limit')`), `opcache.file_cache_read_only`. CLI `php --ini=diff`.

## BC breaks

- OPcache always compiled in; `zend_extension=opcache.so` warns. Control via `opcache.enable[_cli]`.
- Casting out-of-range float to int warns; casting `NAN` warns. Destructuring non-array (non-null) warns.
- `disable_classes` INI removed. `#[\Attribute]` on abstract class/enum/interface/trait = compile error.
- Traits bound before parent class. `final` subclass may replace `static` return with `self`.
- PDO `FETCH_CLASS` ctor args use call_user_func_array semantics (string keys = named args).
- `class_alias()` to `array`/`callable` forbidden; `ArrayObject` rejects enums; re-calling `mysqli::__construct()` throws.

## Deprecated

- Backtick operator `` `ls` `` → `shell_exec()`.
- Non-canonical casts `(boolean)`, `(integer)`, `(double)`, `(binary)` → `(bool)`, `(int)`, `(float)`, `(string)`.
- `case X;` (semicolon) → `case X:`.
- `null` as array offset / `array_key_exists(null, …)` → `''`.
- Incrementing non-numeric strings → `str_increment()`.
- Constant redeclaration; `__debugInfo()` returning `null`; output inside output-buffer handlers.
- `__sleep()/__wakeup()` soft-deprecated → `__serialize()/__unserialize()`.
- No-op frees: `curl_close()`, `curl_share_close()`, `finfo_close()`, `imagedestroy()`, `xml_parser_free()` → delete the call.
- `$http_response_header` → `http_get_last_response_headers()`.
- `chr()` outside 0–255; `ord()` on non-1-byte string.
- `Reflection*::setAccessible()` (no-op since 8.1); `ReflectionClass::getConstant()` for missing const.
- Driver constants/methods on `PDO` (`PDO::MYSQL_ATTR_SSL_CA`) → `Pdo\Mysql::ATTR_SSL_CA`. `PDO` `uri:` DSN.
- `SplObjectStorage::attach()/detach()/contains()` → `offsetSet()/offsetUnset()/offsetExists()`; `ArrayObject`/`ArrayIterator` wrapping objects.
- Closure binding misuse (binding instance to static closure, etc.), `DATE_RFC7231`, `MHASH_*`, `intl.error_level`, `report_memleaks`, `register_argc_argv` from query string, `socket_set_timeout()`, `mysqli_execute()`, `readdir(null)`, `openssl_pkey_derive()` `$key_length`.

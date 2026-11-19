# PHP 8.1 (GA 2021-11-25, EOL)

Source: php-src `PHP-8.1/UPGRADING`, php.net/releases/8.1.

## Language

| Feature | Before → After |
|---|---|
| Enums (pure + backed) | `const STATUS_DRAFT = 'draft';` → `enum Status: string { case Draft = 'draft'; }` |
| `readonly` properties (typed, init once, from declaring scope) | private prop + getter → `public readonly int $id` |
| First-class callable `f(...)` | `Closure::fromCallable([$this, 'm'])` / `'strtoupper'` → `$this->m(...)`, `strtoupper(...)` |
| `new` in initializers (param defaults, static vars, global consts, attribute args) | `?Logger $l = null; $l ??= new NullLogger();` → `Logger $l = new NullLogger()` |
| Pure intersection types `A&B` | `/** @param Countable&Iterator */` → `Countable&Iterator $c` |
| `never` return type | `/** @return never */` → `function fail(): never { throw …; }` |
| `final` class constants; interface constants overridable | `final public const X = 1;` |
| Explicit octal `0o16` | `016` → `0o16` |
| Array unpacking with string keys | `array_merge($a, $b)` → `[...$a, ...$b]` (later keys win) |
| Named arg after unpack | `f(...$args, flag: true)` |
| Fibers (`Fiber`) | low-level coroutine primitive for libraries (Revolt/Amp); don't use directly in app code |

```php
enum Status: string {
    case Draft = 'draft';
    case Live  = 'publish';

    public function label(): string {
        return match ($this) {
            self::Draft => __('Draft', 'td'),
            self::Live  => __('Live', 'td'),
        };
    }
}
Status::from('publish');     // Status::Live, ValueError if unknown
Status::tryFrom($input);     // ?Status
Status::cases();             // list<Status>
```

Enum rules: no properties/state, no `new`, can implement interfaces, have constants, static + instance methods, use traits without props. Compare with `===`. `json_encode()` emits backed value; pure enum fails (`JSON_ERROR_NON_BACKED_ENUM`). `readonly` init only from inside declaring class scope (8.1); clone can't modify (fixed 8.3/8.5).

## New functions / classes

- `array_is_list()` — keys are `0..n-1` in order.
- `fsync()`, `fdatasync()`; `enum_exists()`.
- `IntlDatePatternGenerator`; `CURLStringFile` (upload from string); hash `murmur3*`, `xxh32/xxh64/xxh3/xxh128`.
- `mysqli_result::fetch_column()`, `mysqli_stmt::execute($params)`.
- `$_FILES[...]['full_path']`, `#[\ReturnTypeWillChange]`.

## BC breaks

- Passing `null` to non-nullable internal params is deprecated (`strlen(null)`, `trim(null)`, `str_replace(..., null)`) — #1 source of deprecation noise in WP plugins; cast or `?? ''`.
- Internal non-final methods have tentative return types: overriding `ArrayAccess::offsetGet`, `Iterator::current`, `JsonSerializable::jsonSerialize`, `Countable::count` without return type → deprecation. Add type, or `#[\ReturnTypeWillChange]` when supporting PHP < 8.
- `htmlspecialchars()`/`htmlentities()` default flags now `ENT_QUOTES | ENT_SUBSTITUTE | ENT_HTML401` (`'` escaped).
- mysqli default error mode = exceptions (`MYSQLI_REPORT_ERROR | MYSQLI_REPORT_STRICT`).
- PDO MySQL/SQLite return native int/float for emulated prepares.
- `$GLOBALS` whole-array writes forbidden (`$GLOBALS = []`, `array_pop($GLOBALS)`); `$GLOBALS['x']` fine.
- Static vars in inherited (non-overridden) methods now shared with parent.
- Resources → objects: `finfo`, `FTP\Connection`, `IMAP\Connection`, `LDAP\*`, `PgSql\*`, `PSpell\*`.

## Deprecated

- `Serializable` without `__serialize()/__unserialize()`.
- Implicit lossy float→int (`$arr[1.5]`, `int` param given `1.5`, `%` on floats).
- Calling static members directly on a trait.
- Autovivification from `false` (`$f = false; $f[] = 1;`).
- `date_sunrise()/date_sunset()`, `strftime()/gmstrftime()` → `IntlDateFormatter`/`DateTime::format()`, `strptime()`.
- `FILTER_SANITIZE_STRING`/`FILTER_SANITIZE_STRIPPED` → `htmlspecialchars()` / `esc_html()`.
- `mhash*()`, `key()/current()/next()/reset()/end()` on objects, `odbc_result_all()`, `auto_detect_line_endings`, `mysqli::init()`, passing non-strings to `ctype_*()`, `PDO::FETCH_SERIALIZE`.

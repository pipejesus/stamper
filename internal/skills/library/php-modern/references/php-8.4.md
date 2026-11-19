# PHP 8.4 (GA 2024-11-21, active until 2026-12-31, security until 2028-12-31)

Source: php-src `PHP-8.4/UPGRADING`, php.net/releases/8.4, RFCs `property-hooks`, `asymmetric-visibility-v2`.

## Property hooks

```php
// before
class User {
    private string $email;
    public function getEmail(): string { return $this->email; }
    public function setEmail(string $e): void { $this->email = strtolower($e); }
}
// after
class User {
    public string $email { set => strtolower($value); }        // backed
    public string $label { get => "{$this->name} <{$this->email}>"; } // virtual (no storage)
    public function __construct(public string $name, string $email) { $this->email = $email; }
}
```

- `set(Type $value) { $this->prop = …; }` long form; `set => expr` assigns result. `$this->prop` inside its own hook = backing store.
- Interfaces/abstract classes may declare `public string $x { get; }` / `{ set; }` / `{ get; set; }`.
- Promoted ctor params may have hooks. Parent hook: `parent::$x::get()`.
- Not allowed: hooks on `readonly` props; `$obj->arr[] = …` on backed array with `set` hook; references to props with `set` hook.

## Asymmetric visibility

```php
// before: private + getter
// after
final class Cart {
    public private(set) array $items = [];      // read public, write private
    protected(set) int $count = 0;              // shorthand: public get
    public function add(Item $i): void { $this->items[] = $i; $this->count++; }
}
```

- Property must be typed; set visibility ≤ get visibility. `private(set)` implies `final`.
- `readonly` now implies `protected(set)` (child classes may initialize).
- From outside, `$cart->items[] = …` fails (array write needs write scope).

## Other language

| Feature | Before → After |
|---|---|
| `new` without wrapping parens | `(new Foo())->run()` → `new Foo()->run()` |
| `#[\Deprecated(message:, since:)]` on functions/methods/class consts | `trigger_error('…', E_USER_DEPRECATED)` → attribute; engine emits deprecation + Reflection `isDeprecated()` |
| Lazy objects | `$r->newLazyGhost(function (Foo $o) { $o->__construct($dep); })`, `$r->newLazyProxy(fn() => new Foo($dep))` (`$r = new ReflectionClass(Foo::class)`) — for DI containers/ORMs |
| `exit()`/`die()` are functions | callable, respect `strict_types`, `TypeError` on bad arg |

## New functions / classes

- `array_find($a, fn($v, $k) => …)` (first value or `null`), `array_find_key()` (key or `null`), `array_any()`, `array_all()` — short-circuit; replace `foreach`+`break` and `array_filter()[0]`.
- `mb_trim()`, `mb_ltrim()`, `mb_rtrim()`, `mb_ucfirst()`, `mb_lcfirst()`.
- `BcMath\Number` (immutable, operator overloads: `new Number('0.1') + '0.2'`), `bcround()`, `bcfloor()`, `bcceil()`, `bcdivmod()`.
- `Dom\HTMLDocument::createFromString($html)` — spec-compliant HTML5 parser (`DOMDocument::loadHTML` is HTML4/libxml); `querySelector()`/`querySelectorAll()`.
- PDO driver subclasses: `PDO::connect($dsn)` returns `Pdo\Mysql`, `Pdo\Sqlite`, `Pdo\Pgsql`…
- `RoundingMode` enum for `round()` (`HalfAwayFromZero`, `HalfTowardsZero`, `HalfEven`, `HalfOdd`, `TowardsZero`, `AwayFromZero`, `NegativeInfinity`, `PositiveInfinity`).
- `DateTime[Immutable]::createFromTimestamp()`, `getMicrosecond()`, `setMicrosecond()`; `DOMXPath::quote()`.
- `request_parse_body()` (multipart for PUT/PATCH), `http_get_last_response_headers()`/`http_clear_last_response_headers()`, `fpow()`, `grapheme_str_split()`, `opcache_jit_blacklist()`.
- Argon2 `password_hash()` can be backed by OpenSSL ≥ 3.2 (NTS builds).

## BC breaks

- `password_hash(PASSWORD_BCRYPT)` default cost 10 → 12 (slower hashing; `password_needs_rehash($h, PASSWORD_DEFAULT)` now true for cost-10 hashes → rehash on login).
- JIT defaults: `opcache.jit=disable`, `opcache.jit_buffer_size=64M` — setting only buffer size no longer enables JIT; JIT init failure = fatal at startup.
- `E_STRICT` level removed. Recursion during comparison throws `Error`.
- Unbundled to PECL: IMAP, OCI8, PDO_OCI, pspell.
- PCRE2 10.44: `{,3}` is now a quantifier; named groups up to 128 chars.
- `round()` validates `$mode` (`ValueError`); internal "pre-rounding" fix changes some results.
- Extension class constants typed; `GMP` final; `DOMXPath` not clonable; `mysqli` unused constants removed.
- `str_getcsv()` validates separator/enclosure/escape length.
- Indirect modification of readonly props in `__clone()` (`$r = &$this->ro`) forbidden.

## Deprecated

- Implicitly nullable params: `function f(Foo $x = null)` → `?Foo $x = null` (works back to 7.1).
- `trigger_error(…, E_USER_ERROR)` → throw an exception or `exit()`.
- `_` as class name; `0 ** -1` (use `fpow()`); `E_STRICT` constant.
- CSV default `$escape` in `fputcsv()/fgetcsv()/str_getcsv()`/`SplFileObject` CSV → pass `escape: ''` (RFC-4180) or `'\\'` explicitly.
- `lcg_value()` → `(new Random\Randomizer)->getFloat(0, 1)`.
- Session: changing `session.sid_length`, `session.sid_bits_per_character`, `session.use_only_cookies`, `session.use_trans_sid`, `session.trans_sid_*`, `session.referer_check`; `SID` constant.
- `mysqli_ping()`, `mysqli_kill()`, `mysqli_refresh()`, `xml_set_object()`, `DatePeriod::__construct($iso)`, `SUNFUNCS_RET_*`, `ReflectionMethod::__construct('A::m')` (1 arg), `stream_context_set_option()` 2-arg, `unserialize()` `S:` tag, `CURLOPT_BINARYTRANSFER`, `DOM_PHP_ERR`.

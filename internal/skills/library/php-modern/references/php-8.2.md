# PHP 8.2 (GA 2022-12-08, security-only until 2026-12-31)

Source: php-src `PHP-8.2/UPGRADING`, php.net/releases/8.2.

## Language

| Feature | Before → After |
|---|---|
| `readonly` classes (all props readonly, no dynamic props) | `public readonly` on every prop → `final readonly class Money { function __construct(public int $amount, public string $cur) {} }` |
| DNF types | `/** @param (A&B)\|null */` → `(A&B)\|null $x` |
| Standalone `null`, `false`, `true` types | `bool` (always true) → `function save(): true` |
| Constants in traits | access via using class only: `Foo::CONST` |
| Enum props in const expressions | `const DEFAULT = Status::Live->value;` |
| `#[\SensitiveParameter]` | secret visible in stack trace → `function login(string $u, #[\SensitiveParameter] string $pw)` |
| `#[\AllowDynamicProperties]` | opt-in for legacy classes (inherited by children) |

## Random extension (`Random\`)

```php
// before: global, seedable, non-CSPRNG state
mt_srand(42); $n = mt_rand(1, 6);
// after: object-scoped engines
$r = new Random\Randomizer();                       // Engine\Secure (CSPRNG) default
$r->getInt(1, 6); $r->shuffleArray($a); $r->pickArrayKeys($a, 2); $r->getBytes(16);
$seeded = new Random\Randomizer(new Random\Engine\Xoshiro256StarStar(42)); // reproducible tests
```

Engines: `Secure`, `Mt19937`, `PcgOneseq128XslRr64`, `Xoshiro256StarStar`. `random_int()/random_bytes()` throw `Random\RandomException` on CSPRNG failure.

## New functions

- `mysqli_execute_query($sql, $params)` (prepare + bind + execute).
- `memory_reset_peak_usage()`, `ini_parse_quantity('256M')`, `openssl_cipher_key_length()`, `curl_upkeep()`, `libxml_get_external_entity_loader()`.
- `ReflectionFunction::isAnonymous()`, `ReflectionMethod::hasPrototype()`.
- `iterator_to_array()`/`iterator_count()` accept arrays (`iterable`).
- PCRE `n` modifier (no auto-capture).

## BC breaks

- `strtolower()/strtoupper()/ucfirst()/stripos()/str_ireplace()`… are ASCII-only, locale-independent → use `mb_*` for non-ASCII.
- `str_split('')` returns `[]` (was `['']`).
- `ksort()/krsort()` numeric-string comparisons follow PHP 8 rules.
- `str*cmp()` returns exactly -1/0/1.
- `DateTime::createFromImmutable()` etc. return `static`.

## Deprecated

- Dynamic properties (`$this->undeclared = 1`) → declare the property; `#[\AllowDynamicProperties]` as stopgap; `WeakMap` to attach data to objects you don't own. `stdClass` and `__get/__set` unaffected.
- `"${var}"` / `"${expr}"` interpolation → `"$var"`, `"{$var}"`.
- Partially supported callables: `"self::m"`, `"parent::m"`, `"static::m"`, `["self","m"]`, `[new Foo, "Bar::m"]` → `self::m(...)` or `[self::class, 'm']`.
- `utf8_encode()/utf8_decode()` → `mb_convert_encoding($s, 'UTF-8', 'ISO-8859-1')`.
- mbstring `QPrint`, `Base64`, `Uuencode`, `HTML-ENTITIES` pseudo-encodings → `quoted_printable_*`, `base64_*`, `htmlspecialchars()`.

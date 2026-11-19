# PHP 8.0 (GA 2020-11-26, EOL)

Source: php-src `PHP-8.0/UPGRADING`, php.net/releases/8.0.

## Language

| Feature | Before → After |
|---|---|
| Constructor promotion | `private int $x; function __construct(int $x){$this->x=$x;}` → `function __construct(private int $x) {}` |
| Named arguments | `htmlspecialchars($s, ENT_QUOTES, 'UTF-8', false)` → `htmlspecialchars($s, double_encode: false)` |
| `match` (strict `===`, expression, no fallthrough, `UnhandledMatchError`) | see below |
| Nullsafe `?->` (short-circuits whole chain) | `$u && $u->getAddress() ? $u->getAddress()->city : null` → `$u?->getAddress()?->city` |
| Union types `A\|B`, `mixed`, `static` return | `/** @param int\|string */` → `int\|string $id` |
| Attributes `#[Attr(args)]` | docblock annotations → `#[Route('/x')]` + `ReflectionX::getAttributes()` |
| `throw` is an expression | `$v = $x ?? throw new InvalidArgumentException();` |
| Non-capturing catch | `catch (E $e) {}` (unused) → `catch (E) {}` |
| `$obj::class` | `get_class($obj)` → `$obj::class` |
| Trailing comma in parameter lists + closure `use` | `function f(int $a, int $b,)` |
| `Stringable` interface (auto-implemented by `__toString`) | `string\|Stringable $s` |
| `WeakMap` | `SplObjectStorage` cache (leaks) → `WeakMap` (entry dies with key) |
| `new (expr)`, `$x instanceof (expr)` | — |
| Child may replace params with compatible variadic | `function m(...$all)` |

```php
// before
switch ($status) {
    case 'draft':
    case 'pending': $label = 'Unpublished'; break;
    case 'publish': $label = 'Live'; break;
    default: throw new UnexpectedValueException($status);
}
// after
$label = match ($status) {
    'draft', 'pending' => 'Unpublished',
    'publish'          => 'Live',
    default            => throw new UnexpectedValueException($status),
};
```

## New functions / classes

- `str_contains()`, `str_starts_with()`, `str_ends_with()` — replace `strpos(...) !== false`, `substr(...) === ...`.
- `get_debug_type($v)` — `int`, `string`, `Foo\Bar`, `null` (better than `gettype()` in error messages).
- `fdiv()` (IEEE-754 division, INF/NAN instead of error), `get_resource_id()`, `preg_last_error_msg()`.
- `PhpToken::tokenize()` (object tokenizer), `ValueError`, `DateTime[Immutable]::createFromInterface()`, date format `p`.
- `array_diff()/array_intersect()` accept a single array (`array_diff($a, ...$excludes)` safe when empty).
- Sorting is stable.
- JIT added to OPcache (see performance.md).

## BC breaks to know

- `0 == "foo"` is now `false`: number vs non-numeric string compares as strings. `"1" == "01"` still true; `"abc" == 0` false.
- Internal functions throw `TypeError`/`ValueError` instead of warning + `null`/`false` (e.g. `strlen([])`, `array_combine()` size mismatch).
- Many warnings became `Error` exceptions (undefined constant, write to property of non-object, etc.); undefined variable/property/array key raised to warning.
- `@` no longer silences fatal errors. Default `error_reporting = E_ALL`, `display_startup_errors = On`.
- Removed: `create_function()`, `each()`, `__autoload()`, `(real)`/`(unset)` casts, `$php_errormsg`, `$errcontext` arg, case-insensitive `define()`, `{}` string/array offset, `array_key_exists()` on objects, PHP 4 constructors.
- Concatenation precedence: `"a" . $x + 1` now `"a" . ($x + 1)`. Nested ternary without parentheses = compile error.
- LSP signature mismatch is always fatal. Magic method signatures checked when typed.
- Resources → objects: `GdImage`, `CurlHandle`, `Socket`, `XMLParser`, `OpenSSLCertificate`… `is_resource()` checks break → compare with `false`.
- `call_user_func_array()` string keys = named args.
- Numeric strings: `"123 "` (trailing whitespace) is numeric; `"123abc" + 1` → `E_WARNING "A non-numeric value encountered"`; `"abc" + 1` → `TypeError`.
- Float→string is locale-independent.
- `match` is a reserved keyword.

## Deprecated

- Required parameter after optional (`function f($a = [], $b)`); `Type $p = null` before required still allowed in 8.0 (deprecated in 8.4 as implicit nullable).
- `ReflectionParameter::getClass()/isArray()/isCallable()` → `getType()`.
- `libxml_disable_entity_loader()` (XXE disabled by default with libxml ≥ 2.9).
- Procedural Zip API, pgsql function aliases.

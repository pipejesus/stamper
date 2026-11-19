# PHP 8.3 (GA 2023-11-23, security-only until 2027-12-31)

Source: php-src `PHP-8.3/UPGRADING`, php.net/releases/8.3.

## Language

| Feature | Before → After |
|---|---|
| Typed class constants (class/interface/trait/enum) | `const VERSION = '1.0';` → `const string VERSION = '1.0';` |
| `#[\Override]` (error if parent/interface has no such method) | silent typo `public function intialize()` → `#[\Override] public function initialize()` |
| Dynamic class constant fetch | `constant(Foo::class . '::' . $n)` → `Foo::{$n}` (works for enum cases) |
| Readonly reinit in `__clone()` | deep-clone of readonly object impossible → assign once inside `__clone()` |
| `readonly` anonymous classes | `new readonly class(...) {}` |
| Arbitrary static variable initializers | `static $cache = null; $cache ??= load();` → `static $cache = load();` (runs once) |
| `final` on trait method alias | `use T { m as final; }` |
| Closures from magic methods accept named args | — |

```php
final class Settings {
    public function __construct(public readonly DateTime $at) {} // mutable inner object
    public function __clone() { $this->at = clone $this->at; } // allowed since 8.3
}
```

## New functions

- `json_validate($json)` — validate without building the value (lower memory than `json_decode()`).
- `mb_str_pad()`.
- `str_increment('a9')` → `'b0'`, `str_decrement()` — explicit replacement for `$s++` on strings.
- `Randomizer::getBytesFromString($alphabet, $len)` (tokens/codes), `::getFloat($min, $max, IntervalBoundary::ClosedOpen)`, `::nextFloat()`.
- `DatePeriod::createFromISO8601String()`, `IntlCalendar::setDate()/setDateTime()`, `ReflectionMethod::createFromMethodName()`, `stream_context_set_options()`, `posix_sysconf()`, `posix_eaccess()`, `socket_atmark()`.
- DOM: `DOMNode::contains()`, `getRootNode()`, `isEqualNode()`, `DOMElement::getAttributeNames()`, `toggleAttribute()`, `insertAdjacentElement()`, `className`, `id`.
- `php -l a.php b.php` lints multiple files.

## BC breaks

- Stack overflow now throws `Error` (`zend.max_allowed_stack_size`) instead of segfault.
- Negative first index on empty array: `$a[-5] = 'x'; $a[] = 'y';` → next key `-4` (was `0`).
- `range()` stricter: `TypeError` for arrays/objects, `ValueError` for 0/negative wrong-direction step, `range('5','z')` now chars.
- Traits with static props redeclare inherited static props (separate storage per class).
- Class constant visibility variance checked for interface constants.
- `proc_get_status()` repeatable; `file()` rejects invalid flags.

## Deprecated

- `++`/`--` on empty or non-alphanumeric strings (`$s = ''; $s++;`); `--` on non-numeric strings. Incrementing non-numeric strings soft-deprecated → `str_increment()` (hard-deprecated 8.5).
- `get_class()`/`get_parent_class()` with no argument → `self::class`, `static::class`, `parent::class`.
- `assert_options()` and `ASSERT_*` constants.
- `MT_RAND_PHP`, `SQLite3::enableExceptions(false)`, `ldap_connect($host, $port)`, negative `$width` in `mb_strimwidth()`, `NumberFormatter::TYPE_CURRENCY`, static `FFI::cast()/new()/type()`.

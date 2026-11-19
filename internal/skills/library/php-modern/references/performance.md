# Performance

Measure first: `hrtime(true)` deltas, Xdebug/SPX/Blackfire profiles, `memory_get_peak_usage()` + `memory_reset_peak_usage()` (8.2). Most WordPress requests are DB/HTTP/IO-bound — fix queries and caching (object cache, transients) before micro-optimizing PHP.

## OPcache (biggest win, always on in production)

| Directive | Default | Production guidance |
|---|---|---|
| `opcache.enable` | 1 | keep 1. 8.5+: OPcache always compiled in; `zend_extension=opcache.so` warns |
| `opcache.enable_cli` | 0 | 1 only for long-running CLI workers; for WP-CLI prefer `opcache.file_cache` |
| `opcache.memory_consumption` | 128 (MB) | 192–512 for WP with many plugins; check `opcache_get_status()['memory_usage']` |
| `opcache.interned_strings_buffer` | 8 (MB) | 16–64 (max 32767 on 64-bit since 8.4) |
| `opcache.max_accelerated_files` | 10000 | ≥ number of PHP files (WP + plugins often > 10k) → 20000–100000 |
| `opcache.validate_timestamps` | 1 | 0 on immutable deploys + `opcache_reset()`/FPM reload per deploy; keep 1 where plugins auto-update (WP dashboards) |
| `opcache.revalidate_freq` | 2 | 60+ if timestamps validated in prod |
| `opcache.file_cache` | empty | second-level disk cache (CLI, restarts); 8.5 `opcache.file_cache_read_only` for read-only containers |
| `opcache.preload` | empty | rarely worth it for WP (conditional plugin loading, restart per deploy, not on Windows) |

## JIT

- 8.0+ in OPcache. Since 8.4 defaults: `opcache.jit=disable`, `opcache.jit_buffer_size=64M`; enable with `opcache.jit=tracing` (= `1254`, recommended) or `function` (= `1205`). Before 8.4, JIT ran only if `jit_buffer_size` > 0.
- Gains: CPU-bound code (math, image processing, parsers, long loops). Typical WP page: negligible — benchmark before enabling.
- 8.4: JIT init failure = fatal at startup. `opcache_jit_blacklist($closure)` (8.4) excludes a hot-but-unJITable function.
- Xdebug and some profilers disable/conflict with JIT; benchmark without them.

## Constructs: prefer → avoid

| Prefer | Over | Why |
|---|---|---|
| `isset($set[$k])` / `array_key_exists()` on `array_flip()`ed list | `in_array($k, $list)` in loops | O(1) vs O(n) |
| `in_array($v, $a, true)` | loose `in_array` | correctness + avoids juggling |
| `array_find/any/all` (8.4), `foreach … break` | `array_filter(...)[0]`, `count(array_filter(...)) > 0` | short-circuit, no temp array |
| `array_first/array_last` (8.5) | `reset()/end()` or `array_values($a)[0]` | no internal-pointer mutation, no copy |
| `$parts[] = …; array_merge(...$parts)` once | `$acc = array_merge($acc, $x)` in loop | O(n) vs O(n²) copying |
| Interpolation `"{$a}-{$b}"` in hot code on < 8.4 | `sprintf('%s-%s', …)` | 8.4+ compiles `%s`/`%d`-only `sprintf` (8.6: `printf`) to interpolation — there, pick whichever reads better |
| FCC/PFA callbacks `array_map(trim(...), $a)` | `array_map(fn($s) => trim($s), $a)` | 8.6 compiles FCC/PFA `array_map` to `foreach`; pipe FCC steps compile to direct calls |
| `static fn` / `static function` | non-static closures in classes | no implicit `$this` capture → object not kept alive by callbacks; 8.6 adds a stateless-closure cache |
| `\strlen()` / `use function strlen;` in namespaced code | unqualified `strlen()` | lets compiler inline special functions (`strlen`, `count`, `is_*`, `in_array`, `array_key_exists`, `defined`, …) instead of runtime namespace fallback |
| `match (true)` / `=== []` | — | 8.5 removes opcodes for identity vs bool and specializes `=== []` |
| Declared (typed) properties | dynamic properties | slot storage vs hash table; dynamic props deprecated 8.2 |
| Generators / `yield` for big result sets | building full arrays | constant memory |
| `json_validate()` (8.3) | `json_decode()` just to validate | no value allocation |
| `str_contains/starts_with/ends_with` | `strpos() !== false`, `substr()` compares | clarity; `substr` allocates |
| Lazy objects (8.4) / lazy service closures | eager construction of rarely-used services | pay only when used |
| `WeakMap` caches keyed by object | arrays keyed by `spl_object_hash()` (deprecated 8.6) | auto-evicted, no leaks |
| Exceptions for exceptional paths | exceptions as control flow | still costly (8.5 made creation faster, not free) |

## Avoid

- `@` operator, `extract()`, `compact()`-heavy code, `eval`, `$$var` — slow paths, unanalyzable.
- Regex where `str_*`/`strtr()` suffices. PCRE caches compiled patterns per pattern string — don't build a new pattern per iteration.
- `count()` in `for` condition is O(1) for arrays but calling heavy functions in loop conditions is not — hoist.
- `array_unique()` on large int/string arrays → `array_keys(array_flip($a))` (faster; reindexes).
- Wrapping FCCs in arrow functions inside pipes (`|> (fn($x) => f($x))`) — creates closure per call; use `f(...)` or 8.6 `f(?, 1)`.
- `unserialize()` of large blobs per request (WP options autoload) — keep autoloaded options small.

## Version upgrades as performance

Recent minors each bring engine wins (8.5: faster exception creation, callback-taking array functions, TAILCALL VM for Clang builds; 8.6: faster JSON encode, ZTS, constructor arg passing). Upgrading PHP is usually the cheapest speedup; check compatibility with PHPCompatibility (see wordpress.md).

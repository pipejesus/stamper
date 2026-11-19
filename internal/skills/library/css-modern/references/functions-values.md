# Functions, at-rules and values

## `if()` (C137 only)

```css
.btn {
  padding: 1em;                                                    /* fallback first */
  padding: if(style(--size: lg): 1.5em; style(--size: sm): .5em; else: 1em);
}
.layout { flex-direction: if(media(width < 40rem): column; else: row); }
.text   { color: if(supports(color: oklch(0 0 0)): oklch(60% .2 30); else: tomato); }
```

- Write it as `if(cond: value; cond: value; else: value)`, with no space before `(`. The first true branch wins. Always include `else`, or no match resolves to guaranteed-invalid.
- Tests: `media()`, `supports()`, `style()` (custom properties only; ranges like `style(--n > 2)` C142).
- Unlike container style queries, `if(style())` reads the **element's own** custom properties.
- Fallback works by cascade only when the `if()` value has no `var()`. With `var()` inside, gate the rule with `@supports`.

## `@function` custom functions (C139 only)

```css
@function --fluid(--min <length>, --max <length>, --from <length>: 20rem, --to <length>: 80rem)
  returns <length> {
  result: clamp(var(--min),
    calc(var(--min) + (var(--max) - var(--min)) * (100vw - var(--from)) / (var(--to) - var(--from))),
    var(--max));
}
h1 { font-size: 2rem; font-size: --fluid(2rem, 3.5rem); }

@function --narrow-wide(--narrow, --wide) {
  result: var(--wide);
  @media (width < 700px) { result: var(--narrow); }   /* last matching result wins */
}
```

- `--fluid` multiplies and divides lengths, which relies on typed arithmetic (C140+).
- Parameters have optional `<type>`s and `: default`s. Local `--vars` in the body act as constants. `@media`, `@supports` and `if()` are allowed inside. There is no early return.
- Unsupported engines drop the declaration (an unknown function is a parse error) when its arguments contain no `var()`. Otherwise gate the rule.
- `@mixin`/`@apply` are spec-only and ship nowhere.

## Typed `attr()` (C133, F155; no S)

```html
<div class="meter" data-pct="72" data-tone="tomato" data-cols="4">
```
```css
.meter { inline-size: attr(data-pct %, 0%); background: attr(data-tone type(<color>), gray); }
.grid  { --cols: attr(data-cols type(<integer>), 3); grid-template-columns: repeat(var(--cols), 1fr); }
```

- Forms: `attr(name type(<syntax>), fallback)`, or a unit shorthand: `attr(data-w px)`, `attr(data-p %)`. Plain `attr(name)` yields a string, as `content` has always allowed (W).
- Attribute values are tainted for `url()`. Don't build URLs from them.
- In unsupported engines, a literal-only declaration is dropped, so put a fallback declaration first. The `--cols` route above contains `var()`, so gate it with `@supports (width: attr(data-x type(<length>), 0px))`. The test needs a real property; an unknown property always tests false.

## Tree-counting: `sibling-index()`, `sibling-count()` (N26: C138 F154 S26.2)

```css
.list > li { transition-delay: calc((sibling-index() - 1) * 40ms); }
.fan   > * { rotate: calc((sibling-index() - (sibling-count() + 1) / 2) * 6deg); }
.stack > * { z-index: calc(sibling-count() - sibling-index()); }
```

Both count all element siblings (1-based) regardless of class or type. They replace inline `style="--i:n"`.

## Math

| Function | Example | Status |
|---|---|---|
| `min()` `max()` `clamp()` | `inline-size: min(100% - 2rem, 70ch)` | W |
| `round()` `mod()` `rem()` | `round(down, 100cqi, 12rem)` (snap to steps) | W (Nov 2026) |
| `abs()` `sign()` | `scale: calc(1 - abs(var(--offset)) * .1)` | N25 |
| `sin()` `cos()` `tan()` `atan2()`… | radial layouts | W |
| `pow()` `sqrt()` `hypot()` `log()` `exp()` | modular type scales: `calc(1rem * pow(1.25, var(--step)))` | W |
| `pi` `e` `infinity` constants | `calc(infinity * 1px)` | W |
| `progress(v, start, end)` | `opacity: progress(100cqi, 20rem, 60rem)` → 0–1 number | N26 |
| Typed arithmetic | `calc(100cqi / 1rem)` → unitless number | N26 (C140 S26 F158) |
| `random(key?, min, max)` | `rotate: random(0deg, 360deg)` | C156, S26.2; F flag |
| `calc-size(auto, size)` | animate to intrinsic size | C129 |

- Typed arithmetic converts lengths to ratios, for example `--t: calc((100vw - 20rem) / 40rem); opacity: clamp(0, var(--t), 1);`.
- `random()` defaults to a value per element, per property and per occurrence. A `--dashed-ident` key shares the value document-wide (C156 supports bare and dashed-ident keys; S26.2 also supports `element-scoped`, `fixed`, `auto`). Detect it with `@supports (order: random(1, 2))`. Values are fixed at style computation, not per frame.

## `@property` (N24: C85 F128 S16.4)

```css
@property --angle { syntax: "<angle>"; inherits: false; initial-value: 0deg; }
.ring { background: conic-gradient(from var(--angle), var(--brand), transparent);
        animation: spin 3s linear infinite; }
@keyframes spin { to { --angle: 1turn; } }
```

- Registered properties are interpolatable (gradients, colours, numbers), type-checked (invalid values fall back to `initial-value`), and compare typed in style queries.
- Use `inherits: false` for per-element values. It's faster and prevents leakage.
- `initial-value` must be computationally independent: no `em`, no `var()`. It's required unless `syntax: "*"`.
- In WordPress, put `@property` in the theme stylesheet. theme.json can't express it.

## Units and keywords cheat sheet

| Unit / keyword | Meaning | Status |
|---|---|---|
| `svh` `lvh` `dvh` (+ `vi`/`vb` variants) | small / large / dynamic viewport | W |
| `cqi` `cqb` `cqw` `cqh` `cqmin` `cqmax` | container-relative | W |
| `lh` `rlh` · `cap` · `ic` | line height · cap height · CJK ideograph | W |
| `rcap` `rch` `rex` `ric` | root-relative font units | N26 |
| `env(safe-area-inset-*)` | notches (needs `viewport-fit=cover`) | W |
| `stretch` | fill the available space, margin-aware | C138 S27 (F `-moz-available`) |
| `fit-content` `min-content` `max-content` | intrinsic sizing | W |
| `anchor-size()` | size from the anchor | N26 |
| `revert-layer` / `revert-rule` | cascade rollback | W / N26 |

## Custom property techniques (W)

- Fallback chain: `var(--btn-bg, var(--brand, royalblue))`.
- Invalid-at-computed-value-time: a bad value from `var()` becomes `unset`, not the previous declaration.
- Component API: expose `--card-pad` etc. and consume with defaults inside the component. Callers override via inline style or classes.

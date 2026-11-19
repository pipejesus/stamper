# Selectors, nesting, @layer, @scope

## `:has()` (W: C105 F121 S15.4)

```css
.field:has(:user-invalid)          { --ring: var(--danger); }   /* parent state, no JS */
.card:has(> img:only-child)        { padding: 0; }
.nav:has(a:focus-visible)          { outline: 2px solid; }
html:has(dialog:modal[open])       { overflow: hidden; }        /* scroll lock */
ul:has(> li:nth-child(6))          { columns: 2; }              /* quantity query: ≥6 items */
h2:has(+ .subtitle)                { margin-block-end: 0; }     /* "previous sibling" */
form:has(input[type=checkbox]:checked) .extra { display: block; }
```

- The argument is a relative selector list. It is unforgiving, so one invalid selector drops the rule. Nested `:has()` and pseudo-elements inside are invalid.
- Performance: anchor the subject (`.card:has(…)`, not `*:has(…)`) and prefer the child combinator `>` inside. Broad `body:has(...)` is fine for rare toggles.
- Detect support with `@supports selector(:has(*))`.

## Other selectors and pseudo-classes

| Selector | Use | Status |
|---|---|---|
| `:is()` / `:where()` | forgiving lists. `:where` = 0 specificity (resets, defaults); `:is` = most specific argument | W |
| `:nth-child(2n of .visible)` | count only matching siblings (zebra rows after filtering) | W |
| `:user-valid` / `:user-invalid` | validation styling only after user interaction | W |
| `:focus-visible` | keyboard focus ring | W |
| `:modal`, `:popover-open` | top-layer state | W |
| `:open` | open `<details>`, `<dialog>`, `<select>`, input pickers | N26 (S26.5) |
| `:state(--x)` | custom element states via `ElementInternals.states` | W (Nov 2026) |
| `:heading`, `:heading(2, 3)` | any `h1`–`h6` | S27 (F flag) |
| `:target-current` | active `::scroll-marker` / scrollspy link | C135 |
| `:interest-source` / `:interest-target` | interest invoker states | C |
| `:navigation-source` | element that started the current navigation | C156 |
| `:playing` `:paused` `:muted` `:seeking` `:buffering` `:stalled` `:volume-locked` | media element state | F150 S15.4 C158 (Interop 2026) |
| `:active-view-transition`, `:active-view-transition-type(x)` | style during a view transition | N25 / N26 |
| `::details-content` | `<details>` body box | N25 |
| `::picker(select)`, `::picker-icon`, `::checkmark` | customizable select parts | C135, S27 |
| `::highlight(name)` | CSS Custom Highlight API ranges | N26 |
| `::target-text` | text-fragment highlight | N24 |
| `[attr="x" i]` / `[attr="x" s]` | case-insensitive / case-sensitive match | `i` W · `s` F66, S27 (no C) |

## Nesting (W since 2026-06: C120 F117 S17.2)

```css
.card {
  padding: 1rem;                 /* declarations first */
  & + & { margin-top: 1rem; }
  h2 { font-size: 1.25rem; }     /* bare element selector OK (relaxed syntax) */
  &:hover { … }
  .theme-dark & { … }            /* parent context */
  @media (width > 40rem) { padding: 2rem; }
  @container card (width > 30rem) { … }
}
```

- Nested rules desugar to `:is(parent) child`, so specificity = the most specific selector in the parent list. `#a, .b { & span {} }` gives `.b span` an id-level specificity.
- `&__title` or `&-suffix` string concatenation does not exist. Write full class names.
- Declarations after nested rules are kept in source order (CSSNestedDeclarations), but putting them first is clearer and needed for older engines.
- Don't nest deeper than 3 levels, and never just to mirror the DOM.

## Cascade layers (W)

```css
@layer reset, vendor, base, components, utilities;   /* order declared once, first */
@import url("vendor.css") layer(vendor);
@layer components { .btn { … } }
@layer components.forms { … }                          /* sub-layers */
```

- Later layers beat earlier ones regardless of specificity. **Unlayered styles beat all layers.** For `!important` declarations the order reverses (earlier layers win).
- Put third-party CSS into a low layer so your normal selectors beat it without specificity wars.
- `revert-layer` (W) rolls back to the previous layer. `revert-rule` (N26: C148 F150 S27) rolls back as if the current rule didn't exist.
- In WordPress, core, block and global styles are unlayered. Layering your theme CSS makes it lose to them. Either keep overrides unlayered or wrap only the resets/vendor CSS in layers.

## `@scope` (N26: C118 F146 S26.4 full; Baseline since 2026-03-24)

```css
@scope (.card) to (.card__body) {     /* donut: stops at .card__body */
  :scope { border: 1px solid; }        /* the root */
  img    { border-radius: .5rem; }     /* only imgs in card but outside body */
}
```

```html
<div class="promo">
  <style>@scope { p { color: teal; } }</style>   <!-- scopes to parent element -->
  <p>…</p>
</div>
```

- The scope root doesn't add specificity to inner selectors (`:scope` and `&` add their own).
- Proximity: after specificity, the rule with the nearer scope root wins. Nested themes (`.light` inside `.dark`) resolve correctly.
- Use it for component isolation without BEM, and for keeping parent styles out of nested components (`to (.slot)`).
- It nests inside nesting (`& @scope`) and inside `@layer`.

## Specificity toolkit

- Zero out defaults: `:where(.prose) a { … }`.
- Raise without ids: `.btn.btn`, or `:is(#x, .btn)` (avoid the latter).
- Prefer layers over `!important`. Use `!important` only for utilities and user-preference overrides.

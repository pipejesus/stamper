---
name: css-modern
description: Modern CSS reference with per-feature Baseline status as of Nov 2026 (Chrome 158, Firefox 160, Safari 27.2) covering container/style/scroll-state queries, :has() and modern selectors, nesting/@layer/@scope, fonts and typography, oklch/color-mix/light-dark, HTML+CSS replacements for JS UI (popover, invoker commands, dialog, anchor positioning, customizable select, accordions, scroll-driven animations, view transitions, carousels), if()/@function/typed attr()/sibling-index(), layout and WordPress theme.json integration; use when writing or reviewing CSS, replacing JavaScript UI behaviour with CSS/HTML, building responsive components, setting typography or font loading, or authoring WordPress theme stylesheets.
---

# Modern CSS (status as of 2026-11-19)

Current stable releases: Chrome/Edge 158 (2026-11-17), Firefox 160 (2026-11-10), Safari/iOS 27.2 (2026-10-26).
Chrome and Firefox now ship every 2 weeks, so recheck "limited" rows after ~3 months.
Status data comes from web-features 3.46.0 (2026-11-12) plus vendor release notes.

## Status legend (used in every file)

- **W**: Baseline Widely available. Every engine has supported it for 30+ months. Use it freely.
- **N24 / N25 / N26**: Baseline Newly available since that year. Every current engine supports it, but Safari is tied to OS updates, so an N26 feature can be missing on iOS 18 and older. Add a cheap fallback when that audience matters.
- **C / F / S + version**: limited availability. Only the engines listed ship it (C = Chrome/Edge, F = Firefox, S = Safari macOS+iOS). Use it only as progressive enhancement.
- **flag**: behind a pref or flag, or Nightly only. Do not ship anything that depends on it.

## Workflow

1. Look up each feature below before using it. W needs no gate. N needs at most a cascade fallback. Limited features need a fallback that is fully usable on its own.
2. Prefer native HTML primitives (`popover`, `<dialog>`, `<details>`, `commandfor`), then CSS, then JS. JS remains for data, persistence and complex focus logic.
3. Write the baseline experience first and layer enhancements with `@supports`. Never hide content behind an unsupported feature.
4. Wrap motion in `@media (prefers-reduced-motion: no-preference)`. Respect `color-scheme`.
5. In WordPress themes, put tokens in theme.json and behaviour in the stylesheet. See references/wordpress-theme-json.md.

## Reach for X instead of JS Y

| Instead of JS for… | Use | Status |
|---|---|---|
| Toggling menus/tooltips, click-outside, Esc | `popover` + `popovertarget` or `commandfor` | popover N25 · commands N25 |
| Modal open/close wiring | `<dialog>` + `<button commandfor=d command="show-modal">` | W · N25 |
| Popper/Floating UI positioning | anchor positioning: `anchor-name`, `position-area`, `position-try-fallbacks` | N26 (core) |
| Hover cards with delays | `interestfor` + `popover="hint"` | C only |
| Custom select widgets | `appearance: base-select` | C135, S27 |
| "Only one open" accordions | `<details name="grp">` | N24 |
| Measuring height to animate open/close | `::details-content` + `interpolate-size` (C), grid `0fr→1fr` fallback | N25 / C129 |
| Entry/exit animations around `display:none` | `@starting-style` + `transition-behavior: allow-discrete` | N24 |
| Auto-growing textarea/input | `field-sizing: content` | N26 |
| IntersectionObserver reveals, scroll progress bars | `animation-timeline: view()` / `scroll()` | N26 (Oct) |
| Run an animation once on enter | `timeline-trigger` + `animation-trigger` | C146 |
| Adding an `is-stuck` class on scroll | `@container scroll-state(stuck: top)` | C133 |
| Hiding the header on scroll down | `scroll-state(scrolled: …)` | C144 |
| SPA route/state transitions | `document.startViewTransition()` + CSS | N25 |
| MPA page transitions | `@view-transition { navigation: auto; }` | C126, S18.2 (F flag) |
| Carousel dots and arrows | scroll-snap + `::scroll-marker` / `::scroll-button()` | W / C135 |
| Scrollspy table of contents | `scroll-target-group: auto` + `:target-current` | C140 |
| ResizeObserver component breakpoints | `@container (width > …)` + `cqi` units | W |
| Toggling parent/state classes | `:has()` | W |
| Inline `style="--i:n"` stagger indexes | `sibling-index()`, `sibling-count()` | N26 |
| Reading `data-*` into styles | typed `attr(data-x type(<number>), 0)` | C133, F155 |
| Theme toggle with duplicated palettes | `color-scheme` + `light-dark()` | W (Nov 2026) |
| Sass `darken()` / hover shades | `oklch(from var(--c) calc(l - .1) c h)`, `color-mix()` | N24 / W |
| Picking black/white text for contrast | `contrast-color()` | N26 |
| FitText.js | `text-fit: grow` (C150), else `clamp()` + `cqi` | C150 |
| Masonry.js | `display: grid-lanes` | S26.4 (C/F flag) |
| Sass functions/conditionals | `@function`, `if()` | C139 / C137 |
| JS random decoration | `random()` | C156, S26.2 (F flag) |

## Support status (headline features)

| Feature | Status | Notes |
|---|---|---|
| Container size queries + `cq*` units | W | C105 F110 S16 |
| Name-only `@container name {}` | N26 | C148 F149 S26.4 |
| Style queries `style(--x: v)` | N26 | F151 (May 2026); custom properties only |
| Style query range `style(--n > 2)` | C142 | F flag |
| Scroll-state queries | C133 | `scrolled` C144; F/S none |
| `anchored(fallback:)` container queries | C143 | |
| `:has()` | W | C105 F121 S15.4 |
| Nesting | W | since 2026-06 |
| `@layer` | W | |
| `@scope` | N26 | F146, S26.4 |
| `:open` | N26 | S26.5 (May 2026) |
| `:heading` | S27 | F flag |
| `revert-rule` | N26 | C148 F150 S27 |
| popover (auto/manual) | N25 | |
| `popover="hint"` | C151, F153 | no S (versions per web-features) |
| Invoker commands `command`/`commandfor` | N25 | C135 F144 S26.2 |
| Interest invokers `interestfor` | C142+ | full set C149 |
| `<dialog closedby>` | C134, F141 | no S |
| Anchor positioning (core) | N26 | C125 F147 S26; `anchor-valid`/`anchor-visible` keywords C157 S27 |
| Customizable `<select>` | C135, S27 | F flag |
| `<details name>` | N24 | |
| `::details-content` | N25 | |
| `interpolate-size` / `calc-size()` | C129 | |
| `@starting-style`, `allow-discrete` | N24 | |
| `field-sizing` | N26 | |
| Scroll-driven animations | N26 | F159 (2026-10-27) |
| Scroll-triggered animations | C146 | |
| Same-document view transitions | N25 | F144; `view-transition-class` and `match-element` N25 |
| Cross-document view transitions | C126, S18.2 | F160 flag |
| Element-scoped view transitions | C147 | |
| Scroll snap | W | Interop 2026 |
| `::scroll-marker`, `::scroll-button()` | C135 | tabs mode C154 |
| `if()` | C137 | |
| `@function` | C139 | |
| Typed `attr()` (any property) | C133, F155 | no S |
| `sibling-index()`, `sibling-count()` | N26 | F154 |
| `round()`, `mod()`, `rem()` | W (Nov 2026) | |
| `progress()` | N26 | |
| Typed arithmetic `calc(1em / 1px)` | N26 | C140 S26 F158 |
| `random()` | C156, S26.2 | F flag |
| `@property` | N24 | |
| `oklch()`, `color-mix()` | W | multi-colour `color-mix()`: F150 S27 only |
| Relative colour syntax | N24 | |
| `light-dark()` | W (Nov 2026) | image values N26 |
| `contrast-color()` | N26 | |
| `alpha()` | N26 | |
| `text-wrap: balance` | W (Nov 2026) | |
| `text-wrap: pretty` | C117, S26 | F none |
| `text-box` (trim/edge) | N26 | F154 (Aug 2026) |
| `font-size-adjust`, `size-adjust` | N24 / W | |
| `ascent-`/`descent-`/`line-gap-override` | C87, F89 | no S |
| `text-fit` | C150 | |
| Subgrid | W | |
| `display: grid-lanes` | S26.4 | C flag, F Nightly |
| Gap decorations (`row-rule`) | C149 | |
| `stretch` sizing keyword | C138, S27 | F `-moz-available` |
| `shape()` | N26 | |
| `corner-shape` | C139 | F flag |

## Fallback and `@supports` patterns

```css
/* 1. Cascade fallback: an invalid value is dropped at parse time, so the earlier declaration wins */
.hero { min-height: 100vh; min-height: 100svh; }
.btn  { padding: 1em; padding: if(style(--size: lg): 1.5em; else: 1em); }

/* 2. Block gate: write the full fallback outside, the enhancement inside */
@supports (anchor-name: --a)            { /* anchored UI */ }
@supports (animation-timeline: view())  { /* scroll-driven */ }
@supports selector(::scroll-marker)     { /* CSS carousel controls */ }
@supports (appearance: base-select)     { /* styled select */ }
@supports (container-type: scroll-state){ /* stuck/snapped states */ }
@supports not (display: grid-lanes)     { /* crop images to equal heights */ }
```

- JS gate: `CSS.supports('display', 'grid-lanes')`, `'interestForElement' in HTMLButtonElement.prototype`.
- A value that contains `var()` skips parse-time validation. If its function is unsupported it becomes invalid at computed-value time and resolves to `unset`, which discards the fallback above it. Use an `@supports` block instead.
- Unknown at-rules (`@view-transition`, `@function`, `@position-try`) are ignored, so they need no gate.

## Pitfalls

- A container cannot query itself. Style its descendants or add a wrapper. `container-type: inline-size` makes the element ignore its content's width, so shrink-to-fit parents collapse to 0. Give it a width or `flex: 1`.
- Style queries and `if(style())` test custom properties only, never standard properties.
- Unlayered CSS beats every `@layer`. WordPress core and global styles are unlayered, so theme CSS placed in layers loses to them.
- Nested selectors behave like `:is()`, so specificity comes from the most specific selector in the list. `&__el` (BEM concatenation) is Sass only and does not exist in CSS.
- `:has()` takes an unforgiving list: one invalid selector inside drops the whole rule. Do not nest `:has()` and do not put pseudo-elements inside it.
- The `animation` shorthand resets `animation-timeline`. Declare `animation-timeline` after it.
- `interpolate-size` only works in Chromium. Keep the `grid-template-rows: 0fr → 1fr` technique, or no animation, as the fallback.
- `transition: display` needs `allow-discrete`, plus `overlay` for top-layer exits and `@starting-style` for entries.
- Fluid type must include a `rem` term. `font-size: 5vw` alone breaks zoom (WCAG 1.4.4).
- `text-wrap: balance` only balances a few lines (Chromium caps it at 6). Use it for headings, not body text.
- Safari ignores `ascent-override`/`descent-override`. Fallback-metric tuning there relies on `size-adjust` alone.
- Unprefixed `line-clamp` ships nowhere. Use `display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden`, which works in every engine.
- `100vh` on mobile hides content under the browser UI. Use `svh` (safe minimum) or `dvh` (tracks the UI, causes reflow). Both are W.
- Anchor positioning: anchor names are global, so scope repeated components with `anchor-scope`. A popover opened by `popovertarget`/`commandfor` gets an implicit anchor (N26), so no `anchor-name` is needed.
- Every name in a view transition must be unique per document at snapshot time. Use `view-transition-name: match-element` (N25) for lists.
- `popover` light dismiss fires on `click` since C154. Do not rely on `pointerdown` timing.
- Customizable `<select>`: opt in on both `select` and `::picker(select)`. Unsupported browsers render a normal select, which is the fallback.

## References

| File | Read when… |
|---|---|
| [references/container-queries.md](references/container-queries.md) | building responsive components, cq units, style/scroll-state/anchored queries |
| [references/selectors-cascade.md](references/selectors-cascade.md) | using `:has()`/`:is()`/state pseudo-classes, nesting, `@layer`, `@scope`, specificity bugs |
| [references/typography-fonts.md](references/typography-fonts.md) | loading fonts, variable fonts, fallback metrics, fluid type, text-wrap, text-box, new text properties |
| [references/color.md](references/color.md) | defining palettes or dark mode, deriving shades, contrast |
| [references/interactive-ui.md](references/interactive-ui.md) | replacing JS for popovers, menus, modals, tooltips, selects, accordions, enter/exit animations, form sizing |
| [references/scroll-and-transitions.md](references/scroll-and-transitions.md) | scroll-driven/triggered animations, view transitions (SPA/MPA), scroll snap, CSS carousels, scrollspy |
| [references/functions-values.md](references/functions-values.md) | `if()`, `@function`, `attr()`, `sibling-index()`, math functions, `random()`, `@property`, units |
| [references/layout.md](references/layout.md) | grid/subgrid, grid-lanes masonry, gap rules, sizing keywords, viewport units, shapes, performance props |
| [references/wordpress-theme-json.md](references/wordpress-theme-json.md) | working in a WP theme: theme.json presets, fluid typography, fontFace, responsive styles, enqueueing, specificity |

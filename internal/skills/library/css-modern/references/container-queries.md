# Container queries

Status tags follow the legend in SKILL.md.

## Size queries (W: C105 F110 S16)

```css
.card-slot { container: card / inline-size; }        /* shorthand: name / type */

@container card (width >= 32rem) {
  .card { grid-template-columns: 12rem 1fr; }
}
@container (20rem < inline-size <= 40rem) { … }       /* range syntax W */
@container card (width > 40rem) and style(--layout: media) { … } /* mix size + style */
```

- `container-type`: `inline-size` (default choice), `size` (both axes; only when the block size is definite), `normal` (style queries only).
- Without a name, a query matches the nearest ancestor container of a matching type. Name containers in reusable components so they don't match the wrong ancestor.
- An element cannot query itself. Put the container on a wrapper and style its children. Pseudo-elements can be unreliable here, so prefer a real child element.
- Size containment means the container no longer sizes from its content on that axis. Inside shrink-to-fit contexts (flex items without a basis, `inline-block`, `auto` grid tracks, abspos, floats) its width becomes 0. Fix it with `width: 100%`, `flex: 1`, or a definite track.
- `inline-size`/`size` apply style containment plus size containment on the matching axis. The style part scopes counters and quotes. No layout containment is involved, so abspos and fixed descendants keep their usual containing block. `scroll-state` and `anchored` apply no containment at all.
- Don't make every element a container. Containers are cheap, but nested unnamed containers make queries ambiguous.

## Container units (W)

| Unit | 1% of the query container's… |
|---|---|
| `cqw` / `cqh` | width / height |
| `cqi` / `cqb` | inline / block size (prefer these) |
| `cqmin` / `cqmax` | smaller / larger dimension |

- With no eligible ancestor container, these units fall back to the small viewport units (`svw`/`svh`).
- Use them for component-relative fluid type: `font-size: clamp(1.125rem, 0.9rem + 2.5cqi, 2rem);`. The `rem` term keeps zoom working.
- `cqb` needs a `size` container. With `inline-size` it falls through to an outer container or the viewport.

## Name-only queries (N26: C148 F149 S26.4)

```css
.sidebar { container-name: sidebar; }                 /* type may stay normal */
@container sidebar { .widget { font-size: .875rem; } } /* "am I inside a sidebar?" */
```

Use these to style by context instead of descendant selectors (`.sidebar .widget`).

## Style queries (N26: C111 S18 F151, Baseline since 2026-05-19)

```css
.callout { --tone: warning; }
@container style(--tone: warning) {
  .callout__icon { color: var(--warning); }
}
```

- They test **custom properties only**. `style(display: flex)` is not supported anywhere.
- Every element is a style container (`container-type` is not needed). The query targets the nearest ancestor container, or the named one. It never targets the element itself. To branch on the element's own variable, use `if(style(--x: y): …)` (C137).
- The comparison uses computed values. Register the property with `@property` so `10px` matches `calc(5px*2)` and integers compare typed.
- Boolean form: `@container style(--featured)` is true when the property is set to anything other than its initial value.
- Range form: `@container style(--level >= 3)` is C142 only (F behind the `layout.css.attr.enabled` pref). Fall back to discrete values.
- Typical uses: variant theming of descendants, a parent toggling child layouts, dark sections (`--scheme: dark`).

## Scroll-state queries (C133; `scrolled` C144; no F/S)

```css
.site-header { position: sticky; top: 0; container-type: scroll-state; }
@container scroll-state(stuck: top) {
  .site-header__inner { box-shadow: 0 2px 8px rgb(0 0 0 / .15); }
}

.carousel > li { container-type: scroll-state; scroll-snap-align: center; }
@container scroll-state(snapped: x) { .slide { scale: 1; opacity: 1; } }

.scroller { container-type: scroll-state; overflow: auto; }
@container scroll-state(scrollable: bottom) { .more-hint { opacity: 1; } }
@container scroll-state(scrolled: bottom) { .toolbar { translate: 0 100%; } } /* last scroll direction */
```

| Descriptor | Container is… | Values |
|---|---|---|
| `stuck` | the sticky element | `top` `bottom` `left` `right` `block-start`… `none` |
| `snapped` | the snap target | `x` `y` `inline` `block` `both` `none` |
| `scrollable` | the scroller | edge/axis keywords: true if it can scroll further that way |
| `scrolled` | the scroller | direction of the most recent scroll, or `none` |

- Styles apply to descendants only, so wrap the content inside the sticky or snap element.
- Fallback: make the "not stuck" or "not snapped" state the default design. Use JS (IntersectionObserver sentinel, `scrollsnapchange`) only when the state is essential.

## Anchored container queries (C143)

```css
.tip { position-area: top; position-try-fallbacks: flip-block; container-type: anchored; }
@container anchored(fallback: flip-block) { .tip__arrow { rotate: 180deg; inset-block: auto 100%; } }
```

These flip a tooltip arrow when the anchor fallback triggers. Without support, the arrow keeps its default orientation, so design that orientation to look acceptable either way.

## Patterns

```css
/* Intrinsic card: one rule set, any slot width */
.card-slot { container: card / inline-size; }
.card { display: grid; gap: 1rem; }
@container card (width >= 28rem) { .card { grid-template-columns: minmax(8rem, 1fr) 2fr; } }
@container card (width >= 48rem) { .card__title { font-size: clamp(1.5rem, 1rem + 2cqi, 2.5rem); } }
```

- Use container queries for components and media queries for page-level layout and user preferences (`prefers-*`, `hover`, `pointer`).
- Container queries don't replace `hover`/`pointer` media features.

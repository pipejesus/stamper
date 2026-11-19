# Layout

## Grid recipes (W)

```css
/* Responsive columns without queries ("RAM") */
.grid { display: grid; gap: 1.5rem;
        grid-template-columns: repeat(auto-fit, minmax(min(100%, 16rem), 1fr)); }

/* Content column with full-bleed escape */
.flow { display: grid; grid-template-columns:
          [full-start] minmax(1rem, 1fr) [content-start] min(65ch, 100% - 2rem) [content-end] minmax(1rem, 1fr) [full-end]; }
.flow > * { grid-column: content; }
.flow > .bleed { grid-column: full; }
```

- `auto-fit` collapses empty tracks so items stretch. `auto-fill` keeps them. `min(100%, X)` prevents overflow on narrow screens.
- Flex and grid children default to `min-inline-size: auto`, which causes overflow from long words or `<pre>`. Set `min-inline-size: 0` (or `minmax(0, 1fr)`).

## Subgrid (W since 2026-03: C117 F71 S16)

```css
.cards { display: grid; gap: 1.5rem; grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr)); }
.card  { display: grid; grid-row: span 3; grid-template-rows: subgrid; row-gap: .5rem; }
/* title / body / footer rows now align across every card in the row */
```

Use `grid-template-columns: subgrid` for forms (labels and inputs aligned across fieldsets).

## Masonry: `display: grid-lanes` (S26.4; `fit-tolerance` S27.2; C behind a flag; F Nightly)

```css
.gallery {
  display: grid;                    /* fallback */
  display: grid-lanes;              /* wins where supported */
  grid-template-columns: repeat(auto-fill, minmax(14rem, 1fr));
  gap: 1lh;
}
@supports not (display: grid-lanes) {
  .gallery img { aspect-ratio: 4 / 3; object-fit: cover; }   /* equal-height fallback */
}
```

- `flow-tolerance` was renamed `fit-tolerance` in S27.2 (the old name is a deprecated alias). Firefox 159 removed the legacy `grid-template-rows: masonry` syntax.
- Placement is column-first by shortest lane, so the visual order differs from DOM order. Keep reading-critical order out of masonry.
- `flow-start`/`flow-end` alignment keywords (logical flex-start/end, needed for reversed lanes) are C158.
- JS fallback only if needed: `if (!CSS.supports('display','grid-lanes')) import('masonry')`.

## Gap decorations (C149)

```css
.grid { display: grid; gap: 2rem; column-rule: 1px solid var(--line); row-rule: 1px solid var(--line); }
```

These draw lines in grid and flex gaps without borders or pseudo-elements. Unsupported engines show the gaps without lines, so the layout is unaffected.

## Alignment and sizing

| Technique | Status |
|---|---|
| `align-content: center` on a plain block (vertical centring, no flex/grid) | W |
| `place-items` / `place-content` / `place-self` | W |
| `gap` in flexbox | W |
| `width: stretch` (fill the container minus margins; replaces `100%` and `-webkit-fill-available`) | C138 S27; F `-moz-available` |
| `inline-size: fit-content` / `min-content` / `max-content` | W |
| `aspect-ratio: 16 / 9` | W |
| Logical props (`margin-inline`, `padding-block`, `inset-inline-start`, `border-start-start-radius`) | W |
| `inset: 0` shorthand | W |
| `display: block flow-root` (two-value display) | W |
| `reading-flow: grid-rows` / `flex-visual` (focus order follows the visual order) | C137 |
| `column-wrap` / `column-height` (multicol rows) | C145 |

```css
.stretch { inline-size: -moz-available; inline-size: -webkit-fill-available; inline-size: stretch; }
```

## Viewport units

- `svh` is the smallest viewport (browser UI shown), so use it for hero `min-block-size`. `lvh` is the largest. `dvh` tracks the UI as it moves: accurate, but it reflows while scrolling, so use it sparingly.
- Full-height app shell: `block-size: 100dvh` on a fixed container. For document heroes, use `min-block-size: 100svh`.
- Safe areas: `padding-inline: max(1rem, env(safe-area-inset-left))` with `<meta name="viewport" content="…, viewport-fit=cover">`.

## Shapes and borders

| Feature | Use | Status |
|---|---|---|
| `shape()` in `clip-path` / `offset-path` | responsive shapes with %/calc (beats `path()`) | N26 (Interop 2026) |
| `shape-outside: rect()` / `xywh()` · `shape()` / `path()` | text wrap | N26 · C149 |
| `corner-shape: squircle \| bevel \| notch \| scoop \| superellipse(n)` | non-round corners (uses `border-radius` sizes) | C139; F flag |
| `border-shape` | arbitrary border geometry | C147 |
| `background-clip: border-area` | gradient borders | C150 S18.2 |
| `mask` / `mask-image` | W | |

```css
.blob { clip-path: shape(from 0% 0%, line to 100% 0%, line to 100% 85%, curve to 0% 100% with 50% 70%, close); }
.tile { border-radius: 1.5rem; corner-shape: squircle; }   /* plain rounded corners where unsupported */
```

## Performance helpers

- `content-visibility: auto; contain-intrinsic-size: auto 40rem;` (N25) skips rendering offscreen sections. The `auto` in contain-intrinsic-size remembers the last real size. Avoid it on content above the fold and on elements whose content find-in-page or anchors must reach.
- `contain: layout paint` (W) isolates widgets. `contain: strict` needs explicit sizes.
- `will-change` only immediately before an animation. Don't leave it on permanently.
- Animate `transform`/`opacity`/`translate`/`scale`/`rotate` (individual transforms, W). Avoid animating `top`, `width` and `box-shadow`.
- `zoom` (N24; widely available Nov 2026; Interop 2026) is now standard and affects layout, unlike `scale`.

## Positioning

- `position: sticky` (W) fails when any ancestor has `overflow: hidden|auto`. Use `overflow: clip` (W) on that ancestor instead.
- Fixed headers: `scroll-padding-block-start` on `html` keeps `#anchor` targets visible.
- Overlays belong in the top layer (popover/dialog), not behind z-index ladders. See interactive-ui.md.

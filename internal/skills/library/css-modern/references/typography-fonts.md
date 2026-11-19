# Typography and fonts

## Loading strategy (checklist)

1. Self-host WOFF2 only. Drop TTF/WOFF/EOT. (`@supports font-format(woff2)` and `font-tech(variations)` are W if you need them.)
2. Prefer one variable file per family and style over many static weights. Subset it with `unicode-range` (for example latin and latin-ext).
3. Preload only the 1–2 faces used above the fold, ideally the LCP text font:
   `<link rel="preload" href="/fonts/inter-var.woff2" as="font" type="font/woff2" crossorigin>`. `crossorigin` is required even for same-origin. A preloaded font that goes unused triggers a console warning and wastes bandwidth.
4. Pick `font-display` per role:

| Value | Behaviour | Use for |
|---|---|---|
| `swap` | short block, then fallback until loaded (FOUT) | brand/body text that must appear |
| `optional` | ~100 ms block, then uses the font only if it's already available, with no late swap | performance-first body text, repeat visits |
| `fallback` | ~100 ms block, ~3 s swap window | default in WordPress `fontFace` |
| `block` | up to 3 s invisible text | icon fonts only; prefer SVG icons |

5. Match fallback metrics to remove layout shift on swap. Compute the values per font with tools such as Capsize or fontaine (`@capsizecss/metrics`). The numbers below are placeholders:

```css
@font-face {
  font-family: "Brand Fallback";
  src: local("Arial");
  size-adjust: 104%;          /* W: works everywhere incl. Safari */
  ascent-override: 92%;       /* C87 F89 — Safari ignores */
  descent-override: 24%;
  line-gap-override: 0%;
}
body { font-family: "Brand", "Brand Fallback", system-ui, sans-serif; }
```

- `font-size-adjust: from-font` (N24) or `font-size-adjust: ex-height 0.52` keeps the x-height constant across fallbacks. This also helps mixed-script text.
- `font-synthesis: none` (W) stops faux bold and italic when a face is missing. Use it whenever you ship explicit faces.
- Use `system-ui` (W) for UI-only stacks. `ui-serif`, `ui-monospace` and `ui-rounded` are Safari-only, so treat them as list entries with fallbacks.

## @font-face for variable fonts (W)

```css
@font-face {
  font-family: "Brand";
  src: url("brand-var.woff2") format("woff2");   /* skip tech(): engines that reject it drop the whole entry */
  font-weight: 100 900;          /* range = variable */
  font-style: oblique 0deg 12deg;
  font-stretch: 75% 125%;        /* or font-width descriptor: F155 S18.4 C154 */
  font-display: swap;
  unicode-range: U+0000-00FF, U+0131, U+0152-0153, U+2000-206F;
}
```

- Use the high-level properties: `font-weight: 650`, `font-stretch: 87.5%` (or `font-width`: F155 S18.4), `font-style: oblique 8deg`, `font-optical-sizing: auto` (W; it's the default).
- Reserve `font-variation-settings` for custom axes. It overrides all axes at once and does not merge on inheritance, so drive it through custom properties:

```css
:root { --grad: 0; }
body  { font-variation-settings: "GRAD" var(--grad); }
@media (prefers-color-scheme: dark) { :root { --grad: -25; } } /* thinner strokes on dark */
```

- Prefer `font-variant-*` (W) to `font-feature-settings`: `font-variant-numeric: tabular-nums slashed-zero` for tables, and `font-variant-caps: all-small-caps`.
- For colour fonts (COLRv1), `@font-palette-values` + `font-palette` are W. Animating the palette works only in C121.
- `font-variant-emoji: text | emoji` is C131 F141, not Safari.

## Fluid type

```css
:root {
  --step--1: clamp(0.875rem, 0.85rem + 0.1vw, 0.95rem);
  --step-0:  clamp(1rem, 0.95rem + 0.25vw, 1.125rem);
  --step-2:  clamp(1.5rem, 1.25rem + 1.2vw, 2rem);
  --step-4:  clamp(2rem, 1.4rem + 3vw, 3.5rem);
}
```

- Always add a `rem` term to the `vw`/`cqi` part. Pure viewport units don't grow with browser zoom (WCAG 1.4.4).
- Rule of thumb: keep max ≤ 2.5 × min and zoom compliance holds at every width.
- Inside components, use `cqi` instead of `vw` (see container-queries.md).
- Reusable formula: wrap it in `@function --fluid()` (C139) only with a static fallback (see functions-values.md).
- `text-fit: grow | shrink [consistent | per-line | per-line-all] [<percentage>]` (C150) fits a line to its box width. Use it for display headings only, with a `clamp()` size declared first as the fallback.

## Line breaking

| Property | Use | Status |
|---|---|---|
| `text-wrap: balance` | headings, captions, short blurbs (Chromium caps it at 6 lines) | W (Nov 2026) |
| `text-wrap: pretty` | body paragraphs: avoids orphans, better rag | C117 S26 (F ignores; harmless) |
| `text-wrap: stable` | editable text (no reflow of earlier lines while typing) | N24 |
| `text-wrap-mode` / `text-wrap-style` longhands | | N24 |
| `hyphens: auto` | needs a correct `lang` attribute | W |
| `hyphenate-limit-chars: 6 3 2` | minimum word / before / after lengths | C109 F137 |
| `overflow-wrap: anywhere` | long URLs and code in narrow columns | W |
| `word-break: auto-phrase` | Japanese phrase-aware wrapping | C119 |
| `text-autospace` | CJK/Latin spacing | F145 S18.4 |
| `text-spacing-trim` | CJK punctuation trimming | C123 |
| `hanging-punctuation: first last` | optical margins for quotes | S only |
| `text-indent: 2em hanging each-line` | | N26 |
| `initial-letter: 3` | drop caps | C110; S needs `-webkit-initial-letter` |

```css
h1, h2, h3, blockquote { text-wrap: balance; }
p, li, figcaption      { text-wrap: pretty; }
```

## Vertical rhythm and optical alignment

- `text-box: trim-both cap alphabetic` (N26: C133 S18.2 F154) trims half-leading so text sits flush to the box and cap height centres in buttons and badges. Longhands: `text-box-trim` and `text-box-edge`.

```css
.btn, .badge, h1 { text-box: trim-both cap alphabetic; }
```

- Units: `lh`/`rlh` (W) for rhythm (`margin-block: 1lh`), `cap` (W) for icons sized to caps (`inline-size: 1cap`), `rcap`/`rch`/`rex`/`ric` (N26), `ic` (W) for CJK.
- Keep `line-height` unitless (`1.5`). Use `1.1–1.25` for headings.

## Decoration and misc

- `text-decoration-thickness`, `text-underline-offset` and `text-decoration-skip-ink` are W. `text-underline-position: from-font` is W.
- `text-decoration-inset` (trim or extend underlines) is new in C (2026) and F146. Treat it as an enhancement.
- `text-overflow: "…"` (custom string) is F, plus C157 per Chrome Status. `ellipsis` is W.
- Multi-line clamp: `display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 3; overflow: hidden;` works everywhere. Unprefixed `line-clamp` ships nowhere (F has it behind a flag).
- iOS landscape text inflation: `-webkit-text-size-adjust: 100%; text-size-adjust: 100%;`.
- Ruby (`ruby-position`, `ruby-align`) is N24. `text-emphasis` is W.
- Measure: `max-inline-size: 65ch` for prose.

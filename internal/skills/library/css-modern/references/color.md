# Colour

## Spaces and syntax

| Feature | Status | Note |
|---|---|---|
| `oklch(L C H / a)`, `oklab()` | W | perceptually uniform; default authoring space |
| `lab()`, `lch()`, `color(display-p3 …)` | W | wide-gamut |
| `color-mix(in oklch, a 30%, b)` | W | 2 colours |
| `color-mix()` with 3+ colours | F150 S27 | not C; avoid |
| Relative colour `oklch(from X l c h)` | N24 | C125 F128 S18 (all functions) |
| `light-dark(a, b)` | W (Nov 2026) | needs `color-scheme` |
| `light-dark()` with images | N26 | C150 F150 S27 |
| `contrast-color(c)` | N26 | C147 F146 S26; Interop 2026 |
| `alpha(from c / .5)` | N26 | C152 F155 S27 |
| `color-scheme` | W | |
| `accent-color` | C93 F92 S26.2 | effectively everywhere now; not yet tagged Baseline |
| `@media (color-gamut: p3)`, `(dynamic-range: high)` | W | |
| `dynamic-range-limit` | C136 | HDR images |

## Token setup

```css
:root {
  color-scheme: light dark;                          /* UA widgets + light-dark() */
  --brand: oklch(55% 0.18 255);
  --bg:   light-dark(oklch(99% 0.005 255), oklch(18% 0.02 255));
  --text: light-dark(oklch(22% 0.02 255), oklch(94% 0.01 255));
  --surface: color-mix(in oklch, var(--bg), var(--text) 6%);
  --brand-hover: oklch(from var(--brand) calc(l - 0.08) c h);
  --brand-soft:  oklch(from var(--brand) l c h / 0.15);
  --on-brand: contrast-color(var(--brand));         /* black or white */
}
[data-theme="light"] { color-scheme: light; }       /* user override */
[data-theme="dark"]  { color-scheme: dark; }
```

- `light-dark()` follows the **used `color-scheme`** of the element, not the media query. Set `color-scheme` on `:root` and on any section that flips theme.
- A JS theme toggle should only set `color-scheme`. All colours update automatically and no duplicated palettes are needed.
- Relative colour channels are numbers. `l` in oklch runs 0–1, so `calc(l - 0.08)` darkens by 8 points. Hue arithmetic: `calc(h + 180)`.
- `color-mix()` interpolates in the stated space. Use `in oklch` for vivid mixes and `in srgb` to match legacy blends. For transparency mixes: `color-mix(in srgb, var(--c) 20%, transparent)`.
- `contrast-color()` returns black or white only, and may not meet AA for mid-tone backgrounds. Check brand pairs manually. Its argument is usually a `var()`, so cascade fallback fails (an unsupported engine keeps the declaration and resolves it to `unset`). Gate it:

```css
.btn { color: white; }
@supports (color: contrast-color(red)) { .btn { color: contrast-color(var(--brand)); } }
```

## Gamut and safety

- oklch values can fall outside sRGB. Browsers gamut-map, but keep chroma ≤ ~0.2 for UI colours. Use `@media (color-gamut: p3)` to raise chroma.
- For gradients, interpolate in oklch to avoid grey midpoints: `linear-gradient(in oklch, blue, yellow)` (N24). Use `in oklch longer hue` for rainbows.
- Respect `forced-colors: active` (W). Don't rely on `box-shadow` or `background-image` for meaning. Use system colours (`CanvasText`, `LinkText`, `Highlight`).
- Animating colours: register a typed custom property with `@property --c { syntax: "<color>"; inherits: false; initial-value: black; }` (N24) so it interpolates.

## WordPress

theme.json palette entries can use oklch, `light-dark()` or `color-mix()` strings. Presets become `--wp--preset--color--{slug}`, so derive states in CSS from those (see wordpress-theme-json.md).

# WordPress themes: theme.json + modern CSS

Context: WordPress 7.1 (2026-08-19) is current, and the theme.json schema is `"version": 3` (since 6.6). Use `"$schema": "https://schemas.wp.org/trunk/theme.json"` for editor autocompletion.

## Division of labour

| Put in theme.json | Put in the theme stylesheet |
|---|---|
| Tokens: palette, font sizes, families + `fontFace`, spacing scale, shadows, `settings.custom` | Container queries, `:has()`, nesting, `@layer`/`@scope` |
| Element/block defaults (`styles.elements.link`, `styles.blocks.core/button`) | Popover/dialog/anchor UI, view transitions, scroll-driven effects |
| Layout widths (`settings.layout.contentSize`, `wideSize`) | `@property`, `@font-face` fallback metrics, `@supports` gates |
| Responsive per-block overrides (`@mobile` / `@tablet`, 7.1) | Anything needing selectors theme.json can't express |

`styles.css` (global) and `styles.blocks.<name>.css` accept raw CSS. In the block-level string, `&` refers to the block selector. Use them for small block-tied tweaks and keep larger CSS in files.

## Generated custom properties (use these, don't hard-code)

| Source | Property |
|---|---|
| `settings.color.palette[].slug` | `--wp--preset--color--{slug}` |
| `settings.typography.fontSizes[]` | `--wp--preset--font-size--{slug}` |
| `settings.typography.fontFamilies[]` | `--wp--preset--font-family--{slug}` |
| `settings.spacing.spacingSizes[]` / `spacingScale` | `--wp--preset--spacing--{slug}` (e.g. `--40`) |
| `settings.shadow.presets[]` | `--wp--preset--shadow--{slug}` |
| `settings.custom.lineHeight.body` | `--wp--custom--line-height--body` (camelCase → kebab, nesting → `--`) |
| `settings.layout.contentSize` / `wideSize` | `--wp--style--global--content-size` / `--wp--style--global--wide-size` |
| `styles.spacing.padding` + `useRootPaddingAwareAlignments` | `--wp--style--root--padding-{top,right,bottom,left}` |
| `styles.spacing.blockGap` | `--wp--style--block-gap` |

```css
.wp-block-button__link:hover {
  background: oklch(from var(--wp--preset--color--primary) calc(l - .08) c h);
}
```

Palette values may be any CSS colour string (`oklch(…)`, `light-dark(…)`, `color-mix(…)`). The editor's colour picker swatch may not render functions it can't parse, so check them in the editor.

## Fluid typography in theme.json

```json
{
  "version": 3,
  "settings": {
    "typography": {
      "defaultFontSizes": false,
      "fluid": { "minViewportWidth": "360px", "maxViewportWidth": "1440px", "minFontSize": "15px" },
      "fontSizes": [
        { "slug": "small",   "name": "Small",   "size": "0.9rem", "fluid": false },
        { "slug": "medium",  "name": "Medium",  "size": "1.125rem" },
        { "slug": "x-large", "name": "XL",      "size": "2.5rem", "fluid": { "min": "1.75rem", "max": "2.5rem" } }
      ]
    }
  }
}
```

- `fluid: true` (or the object form) is off by default. Core defaults: viewport range 320px–1600px, minimum-size floor 14px. Without explicit `min`/`max`, `size` is the max and the min is derived with a log-based factor.
- Output is `clamp(min, minRem + ((1vw - Xrem) * factor), max)`, which includes a rem term, so it's zoom-safe. It is **viewport-based**. For component-relative sizes, write your own `clamp(… cqi …)` in CSS.
- `"fluid": false` per size gives fixed sizes (UI text, captions).
- Keep max ≤ 2.5 × min (WCAG 1.4.4 rule of thumb).

## Fonts via theme.json

```json
"fontFamilies": [{
  "slug": "body", "name": "Body",
  "fontFamily": "\"Brand\", \"Brand Fallback\", system-ui, sans-serif",
  "fontFace": [{
    "fontFamily": "Brand", "fontWeight": "100 900", "fontStyle": "normal",
    "fontDisplay": "swap", "src": ["file:./assets/fonts/brand-var.woff2"]
  }]
}]
```

- `fontFace` keys (camelCase → descriptors): `fontDisplay` (core default **`fallback`**), `fontWeight` ranges, `fontStretch`, `unicodeRange`, `fontFeatureSettings`, `fontVariationSettings`, `sizeAdjust`, `ascentOverride`, `descentOverride`, `lineGapOverride`.
- `file:./` resolves relative to the theme root. Faces are printed by `wp_print_font_faces()` on `wp_head` (priority 50).
- Core does **not** preload fonts. Preload the LCP face yourself:

```php
add_action( 'wp_head', function () {
	printf( '<link rel="preload" href="%s" as="font" type="font/woff2" crossorigin>' . "\n",
		esc_url( get_theme_file_uri( 'assets/fonts/brand-var.woff2' ) ) );
}, 1 );
```

- Put the metric-matched `@font-face` fallback (`src: local("Arial")` + overrides) in the theme stylesheet. theme.json `src` expects font files.
- The Font Library (7.0+) lets users install fonts for block, hybrid and classic themes. Don't hard-code `font-family` in CSS: use `var(--wp--preset--font-family--body)` so user choices apply.

## Responsive styles (WordPress 7.1)

```json
"settings": { "viewport": { "mobile": "30rem", "tablet": "48rem" } },
"styles": { "blocks": { "core/group": {
  "spacing": { "padding": { "top": "4rem" } },
  "@tablet": { "spacing": { "padding": { "top": "2.5rem" } } },
  "@mobile": { "spacing": { "padding": { "top": "1.5rem" } } }
}}}
```

- Defaults: mobile `480px`, tablet `782px`. Generated queries are max-width ranges: `@media (width <= 480px)` and `@media (480px < width <= 782px)`. There is no `@desktop`. Base styles apply everywhere.
- These are **viewport** media queries. Blocks reused in sidebars or columns still need container queries in the stylesheet.
- Per-instance responsive values (from the editor UI) are emitted with `!important` on a generated block class. Theme CSS won't override them without `!important`, which is intentional.

## Specificity and the cascade in WP

- theme.json block and element styles are wrapped as `:root :where(.wp-block-x)`, so plain class selectors in theme CSS override them. Don't escalate specificity.
- Core and global styles are **unlayered**. If you wrap theme CSS in `@layer`, every core rule beats it. Layer only resets and third-party CSS, and keep theme rules unlayered (or unlayer the overrides).
- Container queries on blocks: add `container-type: inline-size` to wrappers you control (a custom block style, a pattern class like `.is-style-cq`), not globally to `.wp-block-group`. Flex and row layouts would collapse the width.

```css
.wp-block-group.is-style-cq { container: section / inline-size; }
@container section (width < 36rem) {
  .wp-block-columns { flex-direction: column; }
}
```

## Enqueueing

```php
add_action( 'wp_enqueue_scripts', function () {
	wp_enqueue_style( 'theme', get_theme_file_uri( 'assets/css/theme.css' ), [], wp_get_theme()->get( 'Version' ) );
} );

/* Per-block CSS, loaded only when the block renders (on-demand loading, default for block themes): */
add_action( 'init', function () {
	wp_enqueue_block_style( 'core/details', [
		'handle' => 'theme-details',
		'src'    => get_theme_file_uri( 'assets/css/blocks/details.css' ),
		'path'   => get_theme_file_path( 'assets/css/blocks/details.css' ), // enables inlining of small files
	] );
} );

/* Editor (iframed canvas): */
add_action( 'after_setup_theme', fn () => add_editor_style( 'assets/css/theme.css' ) );
```

- The editor canvas is an iframe (in 7.0, the post editor is iframed when all blocks use Block API v3+). CSS enqueued on `enqueue_block_editor_assets` does **not** reach the iframe. Use `add_editor_style()` or the `enqueue_block_assets` hook.
- Small block styles with a `path` are inlined up to the `styles_inline_size_limit` filter (default 40 KB total since 6.9).
- Ship CSS as written. Native nesting, `@layer`, `@container` and `:has()` need no build step. Only use PostCSS or Lightning CSS when you need minification or targeting older browsers.

## Theme-ready enhancements

- MPA view transitions: add `@view-transition { navigation: auto; }` to the front-end stylesheet only, wrapped in `prefers-reduced-motion: no-preference`. Give persistent parts (`.wp-site-blocks > header`) a `view-transition-name`.
- Core Details block → exclusive accordion: give each Details block the same `name` attribute (the block supports `name`). Add `::details-content` animation in CSS.
- Navigation overlays (7.0) and custom menus: prefer `popover` + anchor positioning over JS where you control the markup.

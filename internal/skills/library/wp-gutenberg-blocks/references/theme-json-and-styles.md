# Styling blocks: supports → styles, selectors, theme.json, states, block styles

## Where block styles come from (in cascade order you control)

1. Block CSS (`style` / `editorStyle` / `viewStyle` in block.json). Keep specificity low: `.wp-block-acme-card` single class, or `:where()`.
2. Global styles from theme.json `styles.blocks["acme/card"]` (generated per block using `selectors`).
3. Block style variations (`is-style-*`).
4. Per-instance user styles from supports (`style` attribute → classes/inline styles on wrapper).
5. Per-instance custom CSS (`supports.customCSS`, 7.0).

Preset classes generated for block-level presets are wrapped in `:where()` since WP 7.1 (`.has-accent-color` specificity 0-1-0 everywhere) — custom CSS that relied on higher specificity may need adjusting.

## Supports serialization

| Support | Saved as | Output |
|---|---|---|
| color preset | `backgroundColor: "accent"`, `textColor` | `has-accent-background-color has-background` |
| color custom | `style.color.background: "#f00"` | inline `background-color:#f00` |
| font size preset | `fontSize: "large"` | `has-large-font-size` |
| spacing | `style.spacing.padding.top: "var:preset\|spacing\|40"` | inline `padding-top:var(--wp--preset--spacing--40)` |
| text shadow (7.2 RC) | `textShadow: "strong"` | `has-strong-text-shadow` |
| responsive (7.1) | `style["@mobile"].typography.fontSize` | generated media-query CSS + stable generated class |
| pseudo state (7.1) | `style[":hover"].color.background` | generated state CSS (Button, Navigation Link in core) |

Apply supports to an inner element instead of the wrapper:
```json
"supports": { "color": { "background": true, "__experimentalSkipSerialization": true } }
```
```js
import { useBlockProps, __experimentalGetColorClassesAndStyles as getColorClassesAndStyles } from '@wordpress/block-editor';
const colorProps = getColorClassesAndStyles( attributes ); // { className, style }
<div { ...useBlockProps() }><a className={ colorProps.className } style={ colorProps.style } /></div>
```
PHP equivalent: `wp_apply_colors_support( $block_type, $attributes )` returns `[ 'class' => …, 'style' => … ]` (or `WP_Style_Engine` / `wp_style_engine_get_styles( $attributes['style'] ?? [] )` → `[ 'css' => …, 'classnames' => … ]`).

## selectors (block.json, 6.3+)

```json
"selectors": {
  "root": ".wp-block-acme-card",
  "color": { "text": ".wp-block-acme-card__title" },
  "typography": ".wp-block-acme-card__title",
  "spacing": { "blockGap": ".wp-block-acme-card__grid" },
  "states": { "-current": ".wp-block-acme-card.is-current" }
}
```
Global styles target these instead of the root. `states` (7.1) maps custom `-name` states usable in theme.json. Legacy `supports.__experimentalSelector` is superseded by `selectors` (Gutenberg 24.5 RC removes the last unused core usages).

## theme.json (version 3) for blocks

```json
{
  "$schema": "https://schemas.wp.org/trunk/theme.json",
  "version": 3,
  "settings": {
    "viewport": { "mobile": "480px", "tablet": "782px" },
    "blockVisibility": { "allowEditing": true },
    "typography": { "textShadowPresets": [ { "slug": "soft", "name": "Soft", "shadow": "0 1px 2px rgba(0,0,0,.2)" } ] },
    "blocks": {
      "acme/card": {
        "color": { "palette": [ { "slug": "brand", "color": "#0a5", "name": "Brand" } ], "custom": false },
        "spacing": { "padding": true }
      }
    }
  },
  "styles": {
    "blocks": {
      "acme/card": {
        "spacing": { "padding": "var:preset|spacing|40" },
        "border": { "radius": "8px" },
        "@mobile": { "spacing": { "padding": "var:preset|spacing|20" } },
        "variations": { "outline": { "border": { "width": "1px", "style": "solid" } } }
      },
      "core/button": { ":hover": { "color": { "background": "var:preset|color|contrast" } } },
      "core/navigation-link": { "-current": { "color": { "text": "var:preset|color|accent" } } }
    }
  }
}
```
- Block-level `settings.blocks[name]` only shows UI for features the block opted into via `supports`.
- `settings.viewport` (7.1): global only, non-negative px/em/rem; defaults `@mobile` ≤ 480px, `@tablet` 481–782px; no `@desktop` key (base applies). Drives responsive styles and block visibility viewports.
- Responsive editing UI can be disabled: `block_editor_settings_all` → `$settings['responsiveEditingEnabled'] = false` (styles still render).
- Pseudo states (`:hover`, `:focus`, `:focus-visible`, `:active`) for Button / Navigation Link in 7.1; custom states (`-current`) Navigation Link only, theme.json only. Nesting: viewport → custom state → pseudo state.
- Form elements (`button`, `textInput`, `select`) in `styles.elements` exist in theme.json; Global Styles UI for them is planned for 7.2 (roadmap).
- Read merged values in PHP: `wp_get_global_settings( [ 'color', 'palette' ], [ 'block_name' => 'acme/card' ] )`, `wp_get_global_styles()`; JS: `useSettings()`.

## Block visibility (6.9 hide, 7.0 per-viewport)

Stored in `metadata.blockVisibility`: `false` (not rendered at all) or `{ "viewport": { "mobile": false, "tablet": true, "desktop": true } }` (rendered, hidden via CSS). Server code that parses/transforms block markup must handle both shapes. Opt a block out of the UI: `"supports": { "visibility": false }`.

## Block style variations

```json
"styles": [ { "name": "default", "label": "Default", "isDefault": true }, { "name": "outline", "label": "Outline" } ]
```
```php
register_block_style( [ 'core/group', 'acme/card' ], [   // array of blocks accepted
	'name'       => 'acme-shadow',
	'label'      => __( 'Shadow', 'acme' ),
	'style_data' => [ 'shadow' => 'var:preset|shadow|natural', 'spacing' => [ 'padding' => 'var:preset|spacing|30' ] ], // 6.6+, editable in Global Styles
	// or 'inline_style' => '.is-style-acme-shadow{…}', or 'style_handle' => 'acme-shadow'
] );
unregister_block_style( 'core/quote', 'plain' ); // server-registered only
```
JS: `registerBlockStyle( 'core/quote', { name: 'fancy', label: 'Fancy' } )`, `unregisterBlockStyle( 'core/quote', 'large' )` inside `domReady` (for core styles). Section styles (6.6): theme.json partials in the theme's `/styles/` folder declaring `"blockTypes": [ … ]` and `"version": 3` register block style variations without PHP.

## Per-block CSS from themes

```php
add_action( 'init', function () {
	wp_enqueue_block_style( 'core/navigation', [
		'handle' => 'acme-nav', 'src' => get_theme_file_uri( 'assets/blocks/navigation.css' ),
		'path' => get_theme_file_path( 'assets/blocks/navigation.css' ), // enables inlining small files
	] );
} );
```
Loaded only when the block renders (block themes; classic with on-demand filter) and inside the editor iframe.

## Editor-only content CSS

- Theme: `add_theme_support( 'editor-styles' ); add_editor_style( 'editor.css' );` — scoped to `.editor-styles-wrapper`.
- Plugin: block.json `editorStyle`, or `enqueue_block_assets` with `is_admin()` guard.
- Never rely on admin CSS (`wp-admin` stylesheets) inside the canvas.
- Dark editor backgrounds: `add_theme_support( 'dark-editor-style' )` (classic) — block themes derive from theme.json.

## SVG icons (7.1)

Server registry for icon sets usable by the Icon block and PHP output:
```php
add_action( 'init', function () {
	wp_register_icon_collection( 'acme', [ 'label' => __( 'Acme', 'acme' ) ] );
	wp_register_icon( 'acme/star', [ 'label' => __( 'Star', 'acme' ), 'file_path' => __DIR__ . '/icons/star.svg' ] ); // or 'content' => '<svg…>'
} );
echo wp_get_icon( 'acme/star', [ 'size' => 20, 'class' => 'acme-icon', 'label' => __( 'Rated', 'acme' ) ] ); // no label = decorative
```
SVG sanitized to `<svg>`, `<path>`, `<polygon>`; `stroke` stripped, `fill` kept on shapes. Add `fill: currentColor` CSS for standalone output. REST: `/wp/v2/icons`, `/wp/v2/icon-collections` (read-only). Admin bar/menu icons became inline SVGs in 7.2 RC.

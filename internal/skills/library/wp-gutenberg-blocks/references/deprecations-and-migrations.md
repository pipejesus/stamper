# Deprecations, validation, migrations (block + platform)

## How validation works

On load the editor parses each block: attributes from comment JSON + sourced HTML → runs current `save()` → compares to stored HTML (normalized tokens). Mismatch → tries `deprecated[]` entries in order → none match → "This block contains unexpected or invalid content" + Attempt recovery. Dynamic blocks with `save: () => null` skip HTML comparison (only attribute parsing matters).

Changes that REQUIRE a deprecation for static blocks: any `save()` markup/class/attribute-order change, changing an attribute's `source`/`selector`, renaming/removing attributes, changing `supports` that serialize classes/styles (e.g. adding `className: false`, removing `color`), changing defaults of attributes whose default is omitted from serialization.

Changes that do NOT: editor-only UI, adding a new optional attribute with no markup effect, CSS file changes, `render.php` changes for dynamic blocks.

## deprecated[] entry

| Key | Required | Notes |
|---|---|---|
| `attributes` | yes (effectively) | Full old schema — not inherited from current block |
| `supports` | yes if old supports differ | Not inherited; affects parsing of `style`/classes |
| `save` | yes | Exact old markup function |
| `migrate( attributes, innerBlocks )` | no | Return new attributes or `[ attributes, innerBlocks ]` |
| `isEligible( attributes, innerBlocks, { blockNode, block } )` | no | Force migration of blocks that are technically valid |

Order newest → oldest. First entry whose `save` reproduces stored HTML wins; migration then produces current attributes (written on next save).

```js
// deprecated.js
import { useBlockProps, RichText } from '@wordpress/block-editor';

const v1Attributes = {
	title: { type: 'string', source: 'html', selector: 'h2' },
	color: { type: 'string' },
};

const v2 = { // previous version: <h2> → <h3>, `color` → supports.color
	attributes: { ...v1Attributes, heading: { type: 'rich-text', source: 'rich-text', selector: 'h3' } },
	supports: { html: false },
	save: ( { attributes } ) => (
		<div { ...useBlockProps.save() }><RichText.Content tagName="h3" value={ attributes.heading } /></div>
	),
};

const v1 = {
	attributes: v1Attributes,
	supports: { html: false },
	save: ( { attributes } ) => (
		<div { ...useBlockProps.save( { style: { color: attributes.color } } ) }><h2>{ attributes.title }</h2></div>
	),
	migrate: ( { title, color, ...rest } ) => ( {
		...rest,
		heading: title,
		...( color && { style: { color: { text: color } } } ),
	} ),
};

export default [ v2, v1 ];
```
Inner blocks migration:
```js
migrate( attributes, innerBlocks ) {
	const { title, ...rest } = attributes;
	return [ rest, [ createBlock( 'core/heading', { content: title } ), ...innerBlocks ] ];
}
```

Tips:
- Freeze old `save` code verbatim (copy, don't import shared helpers that may change). Keep old class names and `useBlockProps.save()` call shape.
- Old `save` must use the same `@wordpress/*` helpers it used then; `RichText.Content` output is stable.
- Test with fixtures: store the old serialized HTML in `test/fixtures/*.html`, load it in an E2E/Playwright test, assert `editor.getEditedPostContent()` equals the expected new markup and no validation warnings.
- Migration to dynamic: add deprecation with old `save`, new `save: () => null` (or `InnerBlocks.Content`), PHP renders from attributes. Front end changes immediately; stored HTML changes only after re-save.
- Bulk migration of stored content: WP-CLI + `parse_blocks()`/`serialize_blocks()` or `WP_Block_Processor` (6.9), never regex.

## Block API version migration (apiVersion → 3)

| WP | Post editor iframe behavior |
|---|---|
| 6.3 | apiVersion 3 introduced; iframe only when all registered blocks ≥ 3 (site editor always iframed) |
| 6.9 | Schema only accepts `3`; console warning for ≤ 2 when `SCRIPT_DEBUG` |
| 7.0 | Iframed if all *inserted* blocks ≥ 3; inserting a v2 block drops the iframe mid-session (expected). Gutenberg ≥ 22.6 enforces iframe incl. classic themes |
| 7.1 (Gutenberg 23.6) | **Always iframed**, regardless of apiVersion, theme type, or meta boxes |

Checklist:
1. `"apiVersion": 3` in every block.json (and `api_version => 3` in PHP-only arrays; autoRegister forces 3).
2. All editor-canvas CSS via `editorStyle`/`style` in block.json, `enqueue_block_assets`, `add_editor_style()`, or `block_editor_settings_all` styles. CSS enqueued with `enqueue_block_editor_assets` only reaches the iframe if it contains `.editor-styles-wrapper`/`.wp-block`/`.wp-block-*` selectors (back-compat, with console warning).
3. Replace global `document`/`window` with `element.ownerDocument` / `.defaultView`; listeners via `useRefEffect` (see js-edit-save-and-components.md).
4. Viewport units and media queries now resolve against the iframe (device preview) — remove admin-sidebar compensation hacks.
5. Third-party libs: initialize per element; load into iframe through `enqueue_block_assets` when they need their own globals; `patch-package` libs hard-coding `document`.
6. Meta-box plugins and admin-page scripts touching the canvas must target the iframe (`document.querySelector( 'iframe[name="editor-canvas"]' ).contentDocument`) — prefer block-editor data APIs instead.

## Platform deprecations & replacements (block-relevant)

| Deprecated / removed | Replacement | Since |
|---|---|---|
| `wp.editor`/`@wordpress/edit-post` plugin slot re-exports (`PluginSidebar`, `PluginDocumentSettingPanel`, …) | import from `@wordpress/editor` | 6.6 |
| `source: 'meta'` attributes | `useEntityProp`, Block Bindings `core/post-meta` | long deprecated |
| `useSetting` | `useSettings` | 6.5 |
| `color.__experimentalDuotone` | `filter.duotone` | 6.3 |
| Interactivity: actions using sync `event` APIs without `withSyncEvent` | wrap with `withSyncEvent()` | warn 6.8 |
| `data-wp-ignore` | remove | deprecated 6.9, removed 7.2 RC / GB 24.2 |
| Two-hyphen unique IDs `--id` | `---id` | 6.9 → removed 7.2 RC |
| `!actions.fn` in directives | derived state getter | warn 6.8 → removed 7.2 RC |
| `data-wp-on-async*` | `data-wp-on*` (async by default) + `withSyncEvent` | marked deprecated in current package |
| `core/router` `state.navigation.*` | `state.navigating`, `state.initiator`, `state.url` + `watch()` | 7.0 deprecated |
| `ServerSideRender` default import | named `{ ServerSideRender }` / `useServerSideRender` | 6.9 |
| `__next40pxDefaultSize` | nothing (default) | 7.1 |
| `Navigation` component | `Navigator` | removed 7.1 |
| `InspectorControls __experimentalGroup` | `group` | removed 7.2 RC |
| `__experimentalCloneSanitizedBlock` / `__experimentalSanitizeBlockAttributes` | `cloneSanitizedBlock` / `sanitizeBlockAttributes` | 7.1 |
| `@wordpress/nux` | `Guide` component | no-op 7.1 |
| `@wordpress/reusable-blocks` APIs | core-data entity APIs (`wp_block`) | 7.1 |
| `Animate`, `Divider`, `Elevation`, `ResponsiveWrapper`, `Scrollable`, `Surface`, `Tip`, `ZStack` | CSS/tokens; `Tip` → `@wordpress/ui` `Notice` | 7.2 RC (removal 7.4) |
| `.eslintrc.*` in wp-scripts | `eslint.config.*` flat config | scripts 32 |
| Puppeteer e2e | Playwright | scripts 33 |
| Bundled Jest in `test-unit-js` | Vitest 5 (or `test-unit-jest` + own Jest) | scripts 36 |
| HTML5 `script` theme support | nothing (always HTML5) | 7.0 |
| PHP 7.2/7.3 | PHP ≥ 7.4 | 7.0 |
| Classic block hidden from inserter (proposal) | reverted — still in inserter | 7.1 |

## Announced, not shipped (as of 2026-11-19)

- React 19 in core: experiment in Gutenberg (default-on since 24.2); not in WP 7.1; 7.2 roadmap says unlikely. Prepare now (checklist in js-edit-save-and-components.md).
- Real-time collaboration: pulled from 7.0, not enabled in 7.1; Gutenberg plugin only.
- jQuery 4: WP 7.2 ships jQuery Migrate 3.6.0 to surface deprecated usage ahead of the planned update.
- Full-page client-side navigation (Interactivity router): experimental.
- Gutenberg-only component deprecations (`Badge`, `FormToggle`, `Tooltip`, `ProgressBar`, `TabbableContainer`, `CustomGradientPicker`): land in a future core release.

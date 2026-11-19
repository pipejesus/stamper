# JS: registerBlockType, edit/save, InnerBlocks, data, components

Import from `@wordpress/*` packages; the build externalizes them to `wp.*` globals and writes deps to `index.asset.php`. Never bundle React or `react/jsx-runtime` (WP provides `react`, `react-dom`, `react-jsx-runtime` scripts; bundling breaks the React 19 switch).

## index.js

```js
import { registerBlockType } from '@wordpress/blocks';
import metadata from './block.json';
import Edit from './edit';
import save from './save';
import deprecated from './deprecated';
import './style.scss';   // → style-index.css (front + editor)
import './editor.scss';  // → index.css (editor only)

registerBlockType( metadata.name, { edit: Edit, save, deprecated } );
// Passing `metadata` (object) as 1st arg also works and applies i18n of title/description/keywords.
```
JS settings override JSON (e.g. `icon` as SVG element, `transforms`, `variations` with functions, `__experimentalLabel`).

## edit.js (static block)

```js
import { __ } from '@wordpress/i18n';
import { useBlockProps, RichText, InspectorControls, BlockControls, AlignmentControl } from '@wordpress/block-editor';
import { PanelBody, SelectControl, ToggleControl } from '@wordpress/components';

export default function Edit( { attributes, setAttributes, isSelected, context, clientId } ) {
	const { heading, layout, showMeta } = attributes;
	const blockProps = useBlockProps( { className: `is-layout-${ layout }` } );
	return (
		<>
			<InspectorControls>{ /* group: default "settings" | "styles" | "advanced" | "style-state" (7.2) */ }
				<PanelBody title={ __( 'Settings', 'acme' ) }>
					<SelectControl label={ __( 'Layout', 'acme' ) } value={ layout }
						options={ [ { label: 'Stack', value: 'stack' }, { label: 'Row', value: 'row' } ] }
						onChange={ ( v ) => setAttributes( { layout: v } ) } />
					<ToggleControl label={ __( 'Show meta', 'acme' ) } checked={ !! showMeta }
						onChange={ ( v ) => setAttributes( { showMeta: v } ) } />
				</PanelBody>
			</InspectorControls>
			<div { ...blockProps }>
				<RichText tagName="h3" value={ heading } allowedFormats={ [ 'core/bold', 'core/italic' ] }
					placeholder={ __( 'Heading…', 'acme' ) } onChange={ ( v ) => setAttributes( { heading: v } ) } />
			</div>
		</>
	);
}
```

## save.js

```js
import { useBlockProps, RichText } from '@wordpress/block-editor';
export default function save( { attributes } ) {
	return (
		<div { ...useBlockProps.save( { className: `is-layout-${ attributes.layout }` } ) }>
			<RichText.Content tagName="h3" value={ attributes.heading } />
		</div>
	);
}
```
`save` must be pure and deterministic: no hooks other than `useBlockProps.save`/`useInnerBlocksProps.save`, no `Date.now()`, no random IDs, no data selects, no i18n of saved strings that differ per locale. Output is compared byte-wise on load → mismatch = "This block contains unexpected or invalid content".

Dynamic block: `save: () => null` (or `<InnerBlocks.Content />` for containers); PHP renders.

## Rules for the wrapper

- `useBlockProps()` on exactly one root element in `edit` (apiVersion ≥ 2 requirement). It injects `ref`, block classes, support styles, data attributes, toolbar positioning.
- Pass extra `className`/`style` *into* `useBlockProps( { … } )`; don't spread then override.
- `useBlockProps.save()` mirrors it in `save`; PHP equivalent `get_block_wrapper_attributes()`.

## InnerBlocks (containers)

```js
import { useBlockProps, useInnerBlocksProps, InnerBlocks } from '@wordpress/block-editor';
const TEMPLATE = [ [ 'core/heading', { level: 3, placeholder: 'Title' } ], [ 'core/paragraph' ] ];

export function Edit( { attributes } ) {
	const blockProps = useBlockProps();
	const innerBlocksProps = useInnerBlocksProps( blockProps, {
		allowedBlocks: attributes.allowedBlocks ?? [ 'core/heading', 'core/paragraph', 'core/image' ], // supports.allowedBlocks (6.9)
		template: TEMPLATE,
		templateLock: false,            // 'all' | 'insert' | 'contentOnly' | false
		orientation: 'vertical',        // 'horizontal' for row-like
		renderAppender: InnerBlocks.ButtonBlockAppender, // or false
		defaultBlock: { name: 'core/paragraph' }, directInsert: true,
	} );
	return <div { ...innerBlocksProps } />;
}
export const save = () => <div { ...useInnerBlocksProps.save( useBlockProps.save() ) } />;
```
- Only one InnerBlocks area per block. Need multiple regions → child blocks with `parent`.
- Children declare `parent: ["acme/tabs"]` (direct) or `ancestor` (any depth). Parent → child data: `providesContext` / `usesContext` (`props.context` in JS, `$block->context` in PHP).
- Read/modify children: `useSelect( s => s( blockEditorStore ).getBlocks( clientId ), [ clientId ] )`, `useDispatch( blockEditorStore ).replaceInnerBlocks( clientId, blocks )`.
- `useBlockEditingMode()` returns `'default' | 'contentOnly' | 'disabled'`; hide structural controls when not `'default'`.

## Data access

```js
import { useSelect, useDispatch } from '@wordpress/data';
import { store as coreStore, useEntityProp, useEntityRecords } from '@wordpress/core-data';
import { store as blockEditorStore } from '@wordpress/block-editor';

const { records: posts, isResolving } = useEntityRecords( 'postType', 'post', { per_page: 5, _embed: true } );
const [ meta, setMeta ] = useEntityProp( 'postType', context.postType, 'meta', context.postId );
const media = useSelect( ( s ) => mediaId ? s( coreStore ).getMedia( mediaId ) : null, [ mediaId ] );
```
- Always pass a deps array to `useSelect`. Return raw store values; derive (map/filter) outside. WP 6.8+ warns (SCRIPT_DEBUG) when `mapSelect` returns new references for unchanged state.
- WP 7.1: `getEntityRecords()` no longer truncates non-paginated entities to 10; custom non-paginated entities should declare `supportsPagination: false`.
- Theme settings: `useSettings( 'color.palette', 'spacing.units' )` (6.5+). `useSetting` (singular) is **deprecated**.
- REST calls: `apiFetch( { path: '/acme/v1/items' } )` from `@wordpress/api-fetch` (nonce/root handled).

## Iframe-safe DOM code (mandatory: post editor always iframed in WP 7.1+)

```js
import { useRefEffect } from '@wordpress/compose';
const ref = useRefEffect( ( el ) => {
	const win = el.ownerDocument.defaultView;            // iframe window, not global window
	const onResize = () => { /* … */ };
	win.addEventListener( 'resize', onResize );
	const lib = new Slider( el, { document: el.ownerDocument } );
	return () => { win.removeEventListener( 'resize', onResize ); lib.destroy(); };
}, [] );
const blockProps = useBlockProps( { ref } );
```
Never use global `document.querySelector` / `window` for canvas content. Libraries needing globals: pass the element, or use the copy loaded inside the iframe (`el.ownerDocument.defaultView.jQuery`) after enqueuing it via `enqueue_block_assets`.

## Transforms

```js
transforms: {
	from: [
		{ type: 'block', blocks: [ 'core/paragraph' ], transform: ( { content } ) => createBlock( 'acme/card', { heading: content } ) },
		{ type: 'enter', regExp: /^!card$/, transform: () => createBlock( 'acme/card' ) },
		{ type: 'raw', isMatch: ( n ) => n.nodeName === 'ASIDE', transform: ( n ) => createBlock( 'acme/card', { heading: n.textContent } ) },
		{ type: 'shortcode', tag: 'card', attributes: { heading: { type: 'string', shortcode: ( { named } ) => named.title } } },
	],
	to: [ { type: 'block', blocks: [ 'core/group' ], variationName: 'group-grid', /* 7.1 */ transform: ( a, inner ) => createBlock( 'core/group', {}, inner ) } ],
},
```
`switchToBlockType( blocks, 'core/group', 'group-grid' )` accepts variation name (7.1). Also `type: 'files'`, `isMultiBlock`, `priority`, `__experimentalConvert`.

## Editor extension filters (`@wordpress/hooks`)

```js
import { addFilter } from '@wordpress/hooks';
import { createHigherOrderComponent } from '@wordpress/compose';
addFilter( 'editor.BlockEdit', 'acme/extra-panel', createHigherOrderComponent( ( BlockEdit ) => ( props ) =>
	props.name !== 'core/image' ? <BlockEdit { ...props } /> : <><BlockEdit { ...props } /><InspectorControls>{ /* … */ }</InspectorControls></>
, 'withExtraPanel' ) );
```
- `blocks.registerBlockType` (add attributes/supports to any block — register before blocks load, i.e. editor script dependency order matters).
- `editor.BlockListBlock` (wrapper props/classes in editor only; WP 7.2: Paragraph key/input events now fire on canvas `contenteditable` root, handlers still receive a pseudo event targeted at the block).
- `blocks.getSaveContent.extraProps` changes saved markup → invalidates existing blocks unless matched server-side; prefer `render_block` (PHP) for front-end-only changes.
- Plugins/sidebars: `registerPlugin` (`@wordpress/plugins`) + `PluginSidebar`, `PluginDocumentSettingPanel` from `@wordpress/editor` (the `@wordpress/edit-post` re-exports are deprecated).

## Style-state aware custom controls (WP 7.2, RC)

```js
import { getStyleForState, setStyleForState, store as blockEditorStore, InspectorControls } from '@wordpress/block-editor';
const state = useSelect( ( s ) => s( blockEditorStore ).getSelectedBlockStyleState( clientId ), [ clientId ] );
const current = getStyleForState( attributes.style, state ) || {};
setAttributes( { style: setStyleForState( attributes.style, state, { ...current, dimensions: { ...current.dimensions, minHeight: '50vh' } } ) } );
// Render such controls in <InspectorControls group="style-state"> (and <BlockControls group="style-state"> for viewport toolbar).
```
Also `hasSelectedBlockStyleState`, `getViewportMediaQueries` (`@wordpress/global-styles-engine`), PHP `wp_get_viewport_media_queries()`.

## Component library status (WP 7.1–7.2)

| Change | Version | Action |
|---|---|---|
| 40px control height is default; `__next40pxDefaultSize` no-op, `false` no longer opts out (except `Button`) | 7.1 | Delete the prop; delete `size="__unstable-large"` |
| `Navigation` component removed | 7.1 | `Navigator` |
| `__experimentalApplyValueToSides` removed | 7.1 | — |
| Emotion → SCSS modules; `View` `css` prop no-op | 7.1–7.2 | Use `className`/`style`; don't target generated classes |
| Deprecated (removal planned 7.4): `Animate`, `Divider`, `Elevation`, `ResponsiveWrapper`, `Scrollable`, `Surface`, `Tip`, `ZStack` | 7.2 | CSS / `@wordpress/theme` tokens; `Tip` → `Notice` from `@wordpress/ui` |
| `InspectorControls __experimentalGroup` removed | 7.2 | `group="advanced"` (or `InspectorAdvancedControls`) |
| `__experimentalCloneSanitizedBlock`, `__experimentalSanitizeBlockAttributes` deprecated | 7.1 | `cloneSanitizedBlock`, `sanitizeBlockAttributes` |
| `__experimentalImageEditor` deprecated | 7.1 | avoid |
| `@wordpress/nux` no-op, `@wordpress/reusable-blocks` APIs deprecated | 7.1 | `Guide` component; core entity APIs for synced patterns |
| Editor Options/View menus use `Menu` from `@wordpress/ui` | 7.2 | Most `PluginMoreMenuItem` items work unchanged |
| Gutenberg-plugin-only deprecations: `Badge` (24.2), `FormToggle`, `Tooltip`, `ProgressBar`, `TabbableContainer`, `CustomGradientPicker` (24.4) | plugin | Plan migrations; not yet in a core release |

Bundled-package rule: if you bundle `@wordpress/components`/`ui`/`dataviews`/`admin-ui`/`grid` (instead of using the `wp-components` script), upgrade to components ≥ 44.0.0, ui ≥ 0.26.0, dataviews ≥ 22.0.0, admin-ui ≥ 3.1.0, grid ≥ 0.11.0 before WP 7.2 (CSS-module class collisions). `@wordpress/dataviews` is normally bundled (admin screens, not block UI).

## React version

| Context | React |
|---|---|
| WP core 7.0 / 7.1 | 18.3 |
| WP 7.2 | Not expected to ship React 19 (roadmap: "unlikely to fully land") — verify in 7.2 field guide |
| Gutenberg plugin 23.4+ | React 19 experiment; **on by default since 24.2** unless an incompatible plugin/theme is detected |

React 19 readiness checklist: `createRoot` (from `@wordpress/element`) instead of `render`; `hydrateRoot` instead of `hydrate`; `root.unmount()` instead of `unmountComponentAtNode`; refs instead of `findDOMNode`; default params instead of `defaultProps` on function components; `createRef`/callback refs instead of string refs; `createContext` instead of legacy context; `inert` must be boolean; ref callbacks must return `undefined` or a cleanup; `forwardRef` deprecated (accept `ref` prop). `@wordpress/scripts` ≥ 33 and most `@wordpress/*` packages accept `react@^18 || ^19` peers.

# Block Bindings, Pattern Overrides, Block Hooks, Variations, Patterns

## Block Bindings (WP 6.5+)

Connect a block attribute to dynamic data without a custom block. Stored in `metadata.bindings`:
```html
<!-- wp:paragraph {"metadata":{"bindings":{"content":{"source":"core/post-meta","args":{"key":"acme_subtitle"}}}}} -->
<p>Fallback</p>
<!-- /wp:paragraph -->
```

Bindable by default: `core/paragraph` content · `core/heading` content · `core/image` id, url, title, alt, caption · `core/button` url, text, linkTarget, rel · `core/navigation-link` url · `core/navigation-submenu` url · `core/post-date` datetime. (Gutenberg 24.2+ plugin also: `core/icon` icon.)

Core sources:
| Source | args | Since |
|---|---|---|
| `core/post-meta` | `key` (meta registered `show_in_rest`, no `_` prefix) | 6.5 |
| `core/pattern-overrides` | none; block needs `metadata.name` | 6.5 (custom blocks 7.0) |
| `core/post-data` | `field`: `date` `modified` `link` | 6.9 |
| `core/term-data` | `field`: `id` `name` `link` `slug` `description` `parent` `count`; needs `termId`+`taxonomy` context (e.g. inside `core/term-template`) | 6.9 |

### Custom source — server (front-end value)
```php
add_action( 'init', function () {
	register_block_bindings_source( 'acme/product', [
		'label'              => __( 'Product data', 'acme' ),
		'uses_context'       => [ 'postId' ],
		'get_value_callback' => function ( array $args, WP_Block $block, string $attribute ) {
			$id = $block->context['postId'] ?? 0;
			return 'price' === ( $args['field'] ?? '' ) ? acme_price_html( $id ) : null; // null → keep fallback
		},
	] );
} );
add_filter( 'block_bindings_source_value', fn ( $v, $source, $args, $block, $attr ) => $v, 10, 5 ); // 6.7
```
Returning `null` keeps the saved fallback. In static blocks, `html`/`rich-text` replacements pass through `wp_kses_post()`; `attribute` replacements go through the HTML API (attribute-escaped).

### Custom source — editor (preview/edit, 6.7+)
```js
import { registerBlockBindingsSource } from '@wordpress/blocks';
registerBlockBindingsSource( {
	name: 'acme/product',                // must match PHP
	label: __( 'Product data', 'acme' ), // overrides PHP label
	usesContext: [ 'postId', 'postType' ],
	getValues( { select, context, bindings } ) {
		const out = {};
		for ( const [ attr, { args } ] of Object.entries( bindings ) ) out[ attr ] = `{${ args.field }}`;
		return out;
	},
	setValues( { select, dispatch, context, bindings } ) { /* write back, e.g. editEntityRecord */ },
	canUserEditValue: ( { select, context } ) => false,  // default false
	getFieldsList: () => [ { label: 'Price', type: 'string', args: { field: 'price' } } ], // 6.9: shows in Bindings UI; type must match attribute
} );
```
Utilities: `unregisterBlockBindingsSource`, `getBlockBindingsSources`, `getBlockBindingsSource` (`@wordpress/blocks`); `useBlockBindingsUtils( clientId )` → `updateBlockBindings( { content: { source, args } } )` (`undefined` removes), `removeAllBlockBindings()` (`@wordpress/block-editor`).

### Make a custom block's attributes bindable (6.9+) — also enables Pattern Overrides (7.0)
```php
add_filter( 'block_bindings_supported_attributes_acme/card', function ( array $attrs ) {
	$attrs[] = 'heading';
	$attrs[] = 'url';
	return $attrs;
} );
// or generic: add_filter( 'block_bindings_supported_attributes', fn ( $attrs, $block_type ) => …, 10, 2 );
```
Rendering bound values:
- Dynamic block: bound values arrive in `$attributes` of `render_callback`/`render.php`. Nothing else to do.
- Static block: core replaces values in saved HTML via the HTML API for attributes sourced from `html`, `rich-text`, or `attribute` with simple selectors. Unsourced attributes or complex selectors → add a `render_callback` or `render_block_{name}` filter to output them.
- Mark those attributes `"role": "content"`.

## Pattern Overrides

Inside a synced pattern (`wp_block`), give a block a name and bind to `core/pattern-overrides`:
```html
<!-- wp:acme/card {"metadata":{"name":"Hero card","bindings":{"__default":{"source":"core/pattern-overrides"}}}} /-->
```
`__default` (6.6+) expands to every supported attribute (editor writes this form); per-attribute form also works. Instances store overrides in the `core/block` `content` attribute: `{"ref":123,"content":{"Hero card":{"heading":"Hi"}}}`. Renaming `metadata.name` orphans stored overrides. Custom blocks need the supported-attributes filter above (WP 7.0+).

## Block Hooks (auto-insert next to an anchor block)

```json
{ "blockHooks": { "core/post-content": "after", "core/navigation": "lastChild", "core/template-part": "firstChild" } }
```
Positions: `before`, `after`, `firstChild`, `lastChild` (PHP filters use `first_child` / `last_child`).

| Version | Change |
|---|---|
| 6.4 | `blockHooks` + `hooked_block_types` filter; templates, template parts, patterns |
| 6.5 | `hooked_block` / `hooked_block_{$type}` filters (modify attrs/inner blocks or return `null` to suppress); `metadata.ignoredHookedBlocks` records user removal; navigation first/last child |
| 6.7 | `firstChild`/`lastChild` of `core/template-part`; respects `supports.multiple: false` per context |
| 6.8 | Post content and synced patterns (`core/block`) |
| 7.0 | Logic moved to REST posts controller; `rest_block_hooks_post_types` filter (default `post`, `page`, `wp_block`, `wp_navigation`) for CPTs |

```php
// Conditional hook (signature 6.4+): ( $hooked_types, $position, $anchor_type, WP_Block_Template|WP_Post|array $context )
add_filter( 'hooked_block_types', function ( $types, $pos, $anchor, $ctx ) {
	if ( 'after' === $pos && 'core/post-content' === $anchor && $ctx instanceof WP_Block_Template && 'single' === $ctx->slug ) {
		$types[] = 'acme/related';
	}
	return $types;
}, 10, 4 );

// Set attributes / inner blocks (6.5+)
add_filter( 'hooked_block_acme/related', function ( $parsed, $type, $pos, $anchor_parsed, $ctx ) {
	if ( null === $parsed ) return null;
	$parsed['attrs']['count'] = 3;
	return $parsed;
}, 10, 5 );

// Enable for a CPT edited in the block editor (7.0+)
add_filter( 'rest_block_hooks_post_types', fn ( $types ) => [ ...$types, 'acme_event' ] );
```
Behavior: insertion happens when content is served (front end + editor REST); once the anchor has been saved by a user, inserted/removed hooked blocks are recorded in the anchor's `metadata.ignoredHookedBlocks` and never re-inserted there. Content edited by users therefore keeps their choice. Classic themes have no block templates to hook into; only post content/patterns/navigation apply.

## Block Variations

Prefer a variation over a new block when only initial attributes/inner blocks differ.

```js
import { registerBlockVariation, unregisterBlockVariation } from '@wordpress/blocks';
registerBlockVariation( 'core/group', {
	name: 'acme-callout',                     // always unique + namespaced
	title: __( 'Callout', 'acme' ),
	icon: 'megaphone',
	description: __( 'Highlighted box.', 'acme' ),
	attributes: { className: 'is-acme-callout', layout: { type: 'constrained' }, metadata: { name: 'Callout' } },
	innerBlocks: [ [ 'core/heading', { level: 3 } ], [ 'core/paragraph' ] ],
	scope: [ 'inserter', 'transform' ],      // default [ 'inserter', 'block' ]
	isActive: [ 'className' ],               // array (preferred) or ( blockAttrs, variationAttrs ) => bool
	keywords: [ 'notice' ],
	example: { innerBlocks: [ { name: 'core/paragraph', attributes: { content: 'Hi' } } ] },
} );
unregisterBlockVariation( 'core/embed', 'youtube' ); // run after domReady
```
- `isActive` dot paths (`'query.postType'`) and specificity (longest matching array wins) since 6.6. Function `isActive` cannot be ranked (first match wins).
- `isDefault: true` replaces the base block in inserter; unregister an existing default first.
- Query Loop variation: `attributes: { namespace: 'acme/books', query: { postType: 'book', … } }`, `isActive: [ 'namespace' ]`, top-level `allowedControls: [ 'order', 'taxQuery' ]`; adjust front-end query via `query_loop_block_query_vars` and editor preview via `rest_{post_type}_query`.
- Server (merged with JS): `block.json` `"variations": "file:./variations.php"` (6.7, file returns array), `variation_callback` arg (6.5), or `get_block_type_variations` filter:
```php
add_filter( 'get_block_type_variations', function ( $variations, $block_type ) {
	if ( 'core/group' === $block_type->name ) {
		$variations[] = [ 'name' => 'acme-callout', 'title' => __( 'Callout', 'acme' ), 'attributes' => [ 'className' => 'is-acme-callout' ], 'isActive' => [ 'className' ] ];
	}
	return $variations;
}, 10, 2 );
```
- `innerContent` key (in current schema; Custom HTML block only, tied to 7.1 editable blocks inside Custom HTML): static HTML fragments with `null` at inner-block positions.
- Transforms can target a variation: `variationName` (7.1).

## Patterns

Themes/plugins drop PHP files in `/patterns/` with a header (`Title`, `Slug`, `Categories`, `Block Types`, `Post Types`, `Inserter`, `Viewport Width`), or `register_block_pattern( 'acme/hero', [ 'title' => …, 'content' => $markup, 'categories' => [ … ], 'blockTypes' => [ 'core/post-content' ] ] )` on `init`. `blockTypes` makes patterns appear as block "starter" suggestions/transforms. Pattern markup must be valid serialized blocks — generate it by copying from the editor's Code view, never hand-write sourced attributes.

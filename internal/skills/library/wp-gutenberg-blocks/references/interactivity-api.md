# Interactivity API (`@wordpress/interactivity`, WP 6.5+)

Server renders HTML with `data-wp-*` directives (PHP processes them → no layout shift), a ~preact runtime hydrates them from a script module. Use it for front-end behavior of blocks instead of jQuery/vanilla/React islands.

## Setup

block.json:
```json
{ "supports": { "interactivity": true }, "viewScriptModule": "file:./view.js", "render": "file:./render.php" }
```
- `"interactivity": true` = `{ "interactive": true, "clientNavigation": true }`.
- Object form: `interactive` → server directive processing + module loaded with `fetchpriority=low` in footer; `clientNavigation` → module marked `loadOnClientNavigation` **only if `interactive` is also true** (core source). Set `clientNavigation: true` alone on non-interactive blocks that are merely safe to swap during router navigation.
- Build with `wp-scripts build --experimental-modules` (create-block `--template @wordpress/create-block-interactive-template`).
- Non-block markup: `echo wp_interactivity_process_directives( $html );` once, in the outermost template.
- Manually registered module + router: `wp_interactivity()->add_client_navigation_support_to_script_module( $id )` (6.9).

## render.php + view.js pattern

```php
<?php
$ctx = [ 'isOpen' => false, 'id' => wp_unique_id( 'acme-faq-' ) ];
wp_interactivity_state( 'acme/faq', [
	'openLabel' => __( 'Open', 'acme' ),                // translate on server
	'count'     => 0,
	'isEmpty'   => function () {                         // derived state on server (6.6+)
		$state = wp_interactivity_state();               // namespace optional inside getters (6.6)
		return 0 === $state['count'];
	},
] );
?>
<div <?php echo get_block_wrapper_attributes(); ?>
	data-wp-interactive="acme/faq"
	<?php echo wp_interactivity_data_wp_context( $ctx ); ?>>
	<button data-wp-on--click="actions.toggle"
		data-wp-bind--aria-expanded="context.isOpen"
		aria-controls="<?php echo esc_attr( $ctx['id'] ); ?>"
		data-wp-text="state.label"></button>
	<div id="<?php echo esc_attr( $ctx['id'] ); ?>" data-wp-bind--hidden="!context.isOpen">
		<?php echo wp_kses_post( $content ); ?>
	</div>
	<p data-wp-bind--hidden="!state.isEmpty"><?php esc_html_e( 'Empty', 'acme' ); ?></p>
</div>
```
```js
import { store, getContext, getElement, withSyncEvent } from '@wordpress/interactivity';

const { state, actions } = store( 'acme/faq', {
	state: {
		get label() { return getContext().isOpen ? 'Close' : state.openLabel; }, // derived (mirror PHP if used in SSR)
		get isEmpty() { return state.count === 0; },
	},
	actions: {
		toggle() { const ctx = getContext(); ctx.isOpen = ! ctx.isOpen; },
		submit: withSyncEvent( function* ( event ) {   // sync event access + async work
			event.preventDefault();
			const res = yield fetch( state.endpoint );    // generators, never async/await
			state.items = yield res.json();
		} ),
	},
	callbacks: {
		logOpen() { const { isOpen } = getContext(); if ( isOpen ) console.log( getElement().ref ); },
	},
} );
```

## Directives

| Directive | Purpose |
|---|---|
| `data-wp-interactive="ns"` | Activates region + default namespace (string or `{"namespace":"ns"}`) |
| `data-wp-context='{"k":1}'` | Local reactive context, inherited/merged by descendants; print with `wp_interactivity_data_wp_context()` |
| `data-wp-bind--{attr}` | Set/remove attribute (`true`/`false`/string). `on*` attributes ignored |
| `data-wp-class--{name}` | Toggle class (kebab-case) |
| `data-wp-style--{prop}` | Set/remove inline CSS property |
| `data-wp-text` | `textContent` |
| `data-wp-html` | innerHTML, **only** values from `asDangerousHTML()` (WP 7.2 RC / Gutenberg 24.2+) |
| `data-wp-on--{event}` | Event handler; runs asynchronously (yields first) — use `withSyncEvent()` for `preventDefault`/`stopPropagation`/`currentTarget` |
| `data-wp-on-window--{event}`, `data-wp-on-document--{event}` | Global listeners, auto-removed with element |
| `data-wp-on-async*` | **Deprecated** (marked so in current package changelog/docs); use `data-wp-on*` |
| `data-wp-watch` | Run on mount + when read signals change; may return cleanup |
| `data-wp-init` | Run once on mount; may return cleanup |
| `data-wp-run` | Run during render; allows `useState`, `useEffect`, `useWatch`, `useInit` |
| `data-wp-key` | Stable key for list diffing |
| `<template data-wp-each--{item}="state.list" data-wp-each-key="context.item.id">` | Client/server list rendering; server adds `data-wp-each-child` |
| `data-wp-router-region="id"` or `'{"id":"ns/r","attachTo":"body"}'` | Region swapped on client navigation (`attachTo` 6.9) |

Values are references (`state.x`, `context.x`, `actions.x`, `callbacks.x`, `otherNs::state.y`), optionally negated with `!` on **values/getters only**. `.length` works on strings/arrays (6.8).

Multiple same-type directives on one element: unique ID suffix with **three** hyphens (6.9): `data-wp-on--click---analytics="acme/stats::actions.track"`, `data-wp-watch---a="callbacks.a"`.

## Store & client API

| Export | Notes |
|---|---|
| `store( ns, { state, actions, callbacks }, { lock } )` | Merges per namespace; returns `{ state, actions, callbacks }`. `lock: true` or a string key = private store |
| `getContext( ns? )`, `getElement()` (`{ ref, attributes }`) | Only inside directive-invoked scope |
| `getServerState( ns? )`, `getServerContext( ns? )` | 6.7+. Deep-cloned snapshot of server values; refreshed (fully replaced, 6.9) on router navigation. Use in `watch`/callbacks to resync |
| `getConfig( ns? )` | Static config from `wp_interactivity_config()` |
| `withScope( fn )` | Keep scope for `setTimeout`, promise callbacks, `watch()` cleanups |
| `withSyncEvent( fn )` | Required (warning since 6.8) for sync event APIs |
| `splitTask()` | `yield splitTask();` inside generator to yield main thread |
| `watch( fn )` | 7.0+. Store-level effect outside DOM; returns `unwatch`; callback/cleanup run without scope (`getContext` throws) |
| `asDangerousHTML( html \| TrustedHTML )` | 7.2 RC. Marks trusted HTML for `data-wp-html`; **does not sanitize**. Copies/JSON lose trust |
| `useState`, `useEffect`, `useWatch`, `useInit`, … | For `data-wp-run` only |
| TS: `AsyncAction<R>`, `TypeYield<T>` | 6.9 helpers for typing generator actions |

PHP API: `wp_interactivity_state( ns, array )`, `wp_interactivity_config( ns, array )`, `wp_interactivity_data_wp_context( array, ns )`, `wp_interactivity_process_directives( html )`, `wp_interactivity_get_context( ns? )` (6.6), `wp_interactivity_get_element()` (6.7), `wp_interactivity_as_dangerous_html( html )` (7.2 RC; HTML not serialized to client, `{ "wpServerHtml": true }` placeholder keeps server content).

SSR rule: any state/derived value used in directives that affects initial HTML must exist in PHP (`wp_interactivity_state` value or closure) or the server renders it empty/wrong and hydration flips it.

## Client-side navigation (`@wordpress/interactivity-router`)

```js
import { store, withSyncEvent } from '@wordpress/interactivity';
store( 'acme/list', {
	actions: {
		navigate: withSyncEvent( function* ( event ) {
			event.preventDefault();
			const { actions } = yield import( '@wordpress/interactivity-router' ); // dynamic import
			yield actions.navigate( event.target.href, { initiator: 'acme/list' } );
		} ),
		*prefetch( event ) {
			const { actions } = yield import( '@wordpress/interactivity-router' );
			yield actions.prefetch( event.target.href );
		},
	},
} );
```
- `navigate( url, { force, html, replace, timeout /*10s → full reload*/, loadingAnimation, screenReaderAnnouncement, initiator } )`.
- Regions: unique `data-wp-router-region` IDs; must sit on/inside `data-wp-interactive`. Nested regions inside interactive elements update (6.9).
- On navigation (6.9+): stylesheets diffed/reused, new script modules (flagged for client nav) imported, importmap supported.
- Router store `core/router`: `state.url` (server-populated since 7.0, reactive); `state.navigating`, `state.initiator` (7.2 RC / Gutenberg 24.2; `undefined` until first navigation, `navigating` has no built-in delay). `state.navigation.hasStarted/hasFinished` **deprecated 7.0** (warning), will stop working.
```js
import { store, watch } from '@wordpress/interactivity';
const { state: router } = store( 'core/router' );
watch( () => sendPageView( router.url ) ); // analytics on every client navigation
```
- Full-page (whole `<body>`) navigation: experimental only.
- Gutenberg 24.4: router modules load without `eval()`/WebAssembly (CSP-friendly).

## Breaking removals (Gutenberg 24.2 → WP 7.2 RC)

| Removed | Now behaves | Fix |
|---|---|---|
| Two-hyphen unique IDs `data-wp-watch--id` | watch/init/run with suffix skipped; `on--click--id` listens to event `click--id` | `---id` |
| `!` on functions `!actions.isEmpty` | resolves `undefined` + dev warning | derived getter `get isEmpty()` then `!state.isEmpty` |
| `data-wp-ignore` (deprecated 6.9) | ignored, element hydrated | remove attribute |
| Extra args to `evaluate` in custom directives | not forwarded | pass only the entry |

## Security & perf

- Never bind untrusted HTML; `data-wp-text` for text. Runtime hardening in `@wordpress/interactivity` 8.1 (Nov 2026): URL-attribute binding handling improved, `data-wp-bind--on*` skipped, reserved property names ignored — still `esc_url()`/`esc_attr()` everything on the server.
- Keep context JSON small; put shared data in `state`. Translate strings in PHP (`wp_interactivity_state`) or use `@wordpress/i18n` in modules (WP 7.0+, call `wp_set_script_module_translations()` for plugin text domains).
- Store namespaces: plugin-prefixed (`acme/faq`), one per block family. Cross-namespace reads via `ns::state.x` or `store( 'other' )`.
- Debug: `SCRIPT_DEBUG` true shows directive warnings and deprecations in the console.

---
name: wp-gutenberg-blocks
description: "Native WordPress block (Gutenberg) development as of Nov 2026 (WP 7.1/7.2, Gutenberg 24.x) — use when creating, editing, debugging or migrating blocks in a plugin or theme: block.json metadata/attributes/supports, registerBlockType edit/save, InnerBlocks, dynamic render.php/render_callback, PHP-only autoRegister blocks, blocks-manifest collection registration, viewScriptModule + Interactivity API directives/stores/router, Block Bindings and Pattern Overrides, Block Hooks, variations and block styles, deprecations and invalid-block fixes, apiVersion 3 iframe compatibility, theme.json/responsive/state styling, and @wordpress/scripts/create-block/wp-env build and test tooling."
---

# Gutenberg block development (current: Nov 2026)

## Platform baseline (verified 2026-11-19)

| Item | Value |
|---|---|
| WordPress stable | 7.1.5 (2026-11-18); 7.1.0 shipped 2026-08-19 |
| WordPress next | 7.2 RC1 (2026-11-17), final planned 2026-12-08 — treat 7.2 items as "RC" |
| Gutenberg plugin | 24.4.2 (requires WP 7.0, PHP 7.4); 24.5 in RC |
| PHP minimum | 7.4 (since WP 7.0) |
| React in core | 18.3 (React 19 = Gutenberg experiment, default-on since GB 24.2; not in 7.1) |
| Block API | `apiVersion: 3` only (schema const since 6.9); post editor always iframed since 7.1 |
| theme.json | `version: 3` |
| Tooling | `@wordpress/scripts` 37.x (Node `^22.22.2 \|\| ^24.15 \|\| >=26`), `@wordpress/create-block` 5.x (Node ≥ 22.13) |

Tag rule: when writing code that uses a feature from the gate table below, check the project's `Requires at least` header first; add `function_exists()` guards or raise the requirement.

## Start of every block task

1. Locate blocks: `find . -name block.json -not -path '*/node_modules/*' -not -path '*/build/*'`. Edit `src/`, never `build/`.
2. Read the plugin/theme bootstrap (registration call), `package.json` scripts, and `Requires at least` / `Requires PHP`.
3. Identify each block's model: `render` in block.json or `render_callback` → dynamic; `save` returning markup → static; `supports.autoRegister` → PHP-only; `viewScriptModule` + `supports.interactivity` → interactive.
4. Make the change, `npm run build` (or `start`), then verify in the editor with `SCRIPT_DEBUG` on: insert → save → reload → no "unexpected or invalid content", no console warnings, front end matches.

## Decision rules

Pick the lightest mechanism that solves the need:

| Need | Use |
|---|---|
| Different look, same markup | Block style (`styles` / `register_block_style`) |
| Preconfigured core block (attrs/inner blocks) | Block variation (`registerBlockVariation`, `get_block_type_variations`) |
| Arrangement of existing blocks | Pattern (`/patterns/*.php`), synced pattern + Pattern Overrides for per-instance edits |
| Show post meta / computed data in core blocks | Block Bindings (`core/post-meta`, `core/post-data`, `core/term-data`, custom source) |
| Auto-insert a block next to another (e.g. after post content) | Block Hooks (`blockHooks`, `hooked_block_types`) |
| Simple server widget, few scalar settings, no rich editing | PHP-only block (`supports.autoRegister`, WP 7.0+) |
| Custom markup/behavior/data | New block (below) |

New block — choose rendering:
- **Dynamic** (`render.php`, `save` → `null` or `InnerBlocks.Content`): default choice. Use when output depends on data (posts, meta, options, user, time), markup may evolve, or the block must stay in sync site-wide. No deprecations needed for markup changes.
- **Static** (`save()` markup): only for pure content blocks whose HTML must survive plugin deactivation. Every markup change needs a `deprecated` entry.
- **Hybrid**: static inner content + dynamic wrapper (`save` → `<InnerBlocks.Content/>`, PHP wraps `$content`).
- **Front-end behavior**: Interactivity API (`viewScriptModule` + directives in `render.php`). Use `viewScript` (classic) only for third-party libraries that are not modules; never jQuery/React islands for new work.
- **Container**: `useInnerBlocksProps` + child blocks with `parent`/`ancestor`; share data via `providesContext`/`usesContext`.
- **Plugin vs theme**: blocks belong in plugins (survive theme switches). Themes ship patterns, styles, variations, and theme.json.

## Canonical modern pattern (details in references)

- One folder per block: `src/{slug}/block.json` + `index.js` + `edit.js` (+ `save.js` | `render.php`) + `view.js` + `style.scss` + `editor.scss`. `block.json` is the single source of truth (name, attributes, supports, assets via `file:./…`).
- Registration: PHP on `init` with `wp_register_block_types_from_metadata_collection( __DIR__ . '/build', __DIR__ . '/build/blocks-manifest.php' )` (WP 6.8+), manifest from `wp-scripts build --blocks-manifest`. JS: `registerBlockType( metadata.name, { edit, save } )`.
- Wrapper: exactly one root per context — `useBlockProps()` (edit), `useBlockProps.save()` (save), `get_block_wrapper_attributes()` (PHP). Pass extra classes into them.
- Attributes: config in comment JSON (no `source`); user content sourced (`rich-text`/`attribute`) and `"role": "content"`; temporary UI state `"role": "local"`.
- Styling: declare `supports` (color, spacing, typography, dimensions, …) instead of custom controls; theme presets/`theme.json` then work automatically, incl. responsive (7.1) and states.
- Editor code must be iframe-safe: element-relative DOM (`ownerDocument`/`defaultView`, `useRefEffect`), all canvas CSS declared in block.json or `enqueue_block_assets`.
- Data: `@wordpress/core-data` (`useEntityRecords`, `useEntityProp`) + `useSelect` with deps; never fetch in `save`.
- Interactivity: server state via `wp_interactivity_state()` (incl. translated strings and derived closures), local state via `wp_interactivity_data_wp_context()`, generators for async actions, `withSyncEvent()` for `preventDefault`.
- i18n: `textdomain` in block.json; `__()` from `@wordpress/i18n`; script modules with plugin domain need `wp_set_script_module_translations()` (7.0+).

## Feature gate table (min WordPress)

| Feature | WP |
|---|---|
| `render` file, script arrays | 6.1 |
| apiVersion 3, `selectors` | 6.3 |
| `blockHooks`, `hooked_block_types` | 6.4 |
| Interactivity API, `viewScriptModule`, `viewStyle`, `allowedBlocks`, Block Bindings, `hooked_block*` filters, `variation_callback` | 6.5 |
| `register_block_style` `style_data`, `isActive` dot paths, `wp_interactivity_get_context` | 6.6 |
| `wp_register_block_metadata_collection`, variations PHP file, attribute `role` stable, `registerBlockBindingsSource` (editor), `getServerState/Context`, `script_module_data_*` | 6.7 |
| `wp_register_block_types_from_metadata_collection`, `withSyncEvent`, Block Hooks in post content, `should_load_block_assets_on_demand` | 6.8 |
| Schema forces apiVersion 3, `supports.visibility/contentRole/allowedBlocks`, `fitText`, `core/post-data`/`core/term-data`, `getFieldsList`, `block_bindings_supported_attributes*`, IAPI unique IDs `---`, router regions anywhere/`attachTo`, `useServerSideRender`, `WP_Block_Processor` | 6.9 |
| PHP-only `autoRegister`, `watch()`, `state.url` SSR, Pattern Overrides for custom blocks, `listView`/`customCSS`/`textIndent` supports, viewport block visibility, `rest_block_hooks_post_types`, module translations, scripts depending on modules, PHP 7.4 | 7.0 |
| Always-iframed post editor, responsive styles (`@mobile`/`@tablet`, `settings.viewport`), pseudo/custom states, `background.gradient`, `dimensions.minWidth`, transforms `variationName`, SVG icon registry, 40px controls default | 7.1 |
| (RC) `data-wp-html` + `asDangerousHTML`, router `state.navigating`/`initiator`, removal of IAPI deprecations, `typography.textShadow` support, style-state control API, `background.backgroundClip` | 7.2 |

## Pitfalls

1. **Invalid block after edit** — changed `save()` output, attribute source/selector, or serialized supports without a `deprecated` entry. Fix with a deprecation; never "fix" stored content by hand.
2. Non-deterministic `save` (dates, random IDs, `useSelect`, locale-dependent strings) → validation fails on next load.
3. Renaming `name` or namespace orphans every saved instance (shows as missing block).
4. Declaring attributes that supports add (`style`, `className`, `align`, `backgroundColor`, `fontSize`, `metadata`…) → conflicts. Let supports add them.
5. `useBlockProps` missing, used twice, or not on the root → broken selection/toolbar, styles not applied.
6. `get_block_wrapper_attributes()` missing in `render.php` → user color/spacing/align/anchor/custom class silently ignored on front end.
7. Defining functions/classes inside `render.php` (included per instance) → fatal "cannot redeclare".
8. `render_callback` that echoes instead of returning; or `render.php` that returns a string (it must print).
9. Global `document`/`window` in edit code → targets admin page, not the iframe canvas (always iframed since 7.1).
10. Editor CSS via `enqueue_block_editor_assets` or admin stylesheets → doesn't reach canvas. Use `editorStyle`/`style`/`enqueue_block_assets`.
11. Using `viewScript` for code that imports `@wordpress/interactivity` → must be `viewScriptModule` built with `--experimental-modules`.
12. Missing `supports.interactivity` → directives not processed on server (flash/empty content). `{ "clientNavigation": true }` alone does not mark the module for router loading; needs `interactive: true` or `interactivity: true`.
13. IAPI derived state used in directives but not defined in PHP → wrong SSR markup, hydration flicker.
14. `async/await` in IAPI actions loses scope → use generators (`function*` + `yield`), `withScope` for timers.
15. Removed in 7.2 RC: two-hyphen unique IDs, `!actions.fn`, `data-wp-ignore`. Use `---id`, derived getters, nothing.
16. Bundling `react`, `react-dom`, or `react/jsx-runtime` (custom webpack, `--webpack-no-externals`) → duplicate React, breaks under React 19.
17. Missing `*.asset.php` / hard-coded handles → wrong deps and cache busting. Always load generated asset files (block.json does it).
18. Post meta not `show_in_rest` or prefixed `_` → unusable in editor and Block Bindings.
19. Custom block attribute not added via `block_bindings_supported_attributes_{name}` → cannot be bound or overridden in patterns; static blocks with unsourced attributes also need a `render_callback` to output bound values.
20. Content attributes without `"role": "content"` → block not editable/selectable in contentOnly patterns and template parts (7.0+).
21. `blockHooks` expecting re-insertion after user removal — anchors record `metadata.ignoredHookedBlocks`; hooks never insert there again.
22. `isDefault` variation clash or variation without unique `name`/`isActive` → wrong inserter item or block card.
23. `__experimental*` APIs: allowed only when no stable alternative exists (e.g. `__experimentalBorder`); isolate them and expect renames.
24. Upgrading `@wordpress/scripts` majors blindly: 32 (ESLint 10 flat config), 33 (Playwright e2e), 36 (Vitest, Node 22.22+), 37 (webpack-dev-server 6). Read build-tooling.md timeline.
25. Bundled `@wordpress/components`/`ui`/`dataviews`/`admin-ui`/`grid` older than the 7.2 fix versions → CSS class collisions with other plugins.
26. Relying on `state.navigation.hasStarted/hasFinished` (deprecated 7.0) or `__next40pxDefaultSize={false}` (no-op 7.1) or `InspectorControls __experimentalGroup` (removed 7.2 RC).
27. Assuming React 19 or real-time collaboration in core — neither shipped as of WP 7.1.5.

## Verification checklist

- `npm run build` clean; `npm run lint:js` (flat config) passes.
- Editor (`SCRIPT_DEBUG` true): insert, configure, save, reload → valid; undo/redo works; no deprecation warnings from your code.
- Front end: wrapper has support classes; assets load only where the block renders; IAPI works with JS disabled initially (SSR) and after hydration.
- Old content: open posts saved with previous versions → deprecations migrate without recovery prompt.
- Patterns/templates: block works inside synced pattern, template part (contentOnly), Query Loop context.

## References

| File | Read when… |
|---|---|
| [references/block-json.md](references/block-json.md) | writing/changing block.json: every field with WP version, asset handles, attribute sources/roles, full `supports` table incl. 7.x additions |
| [references/php-registration-and-render.md](references/php-registration-and-render.md) | registering blocks (collections/manifest, fallbacks), writing `render.php`/`render_callback`, PHP-only `autoRegister` blocks, ServerSideRender, enqueue hooks, script modules PHP API, render filters |
| [references/js-edit-save-and-components.md](references/js-edit-save-and-components.md) | implementing `edit`/`save`, InnerBlocks/templates/locks, data fetching, iframe-safe DOM, transforms, editor filters, style-state API, component deprecations, React 19 readiness |
| [references/interactivity-api.md](references/interactivity-api.md) | adding front-end interactivity: directives, store API, server state/derived closures, router/client navigation, `data-wp-html`, removed behaviors |
| [references/bindings-hooks-variations.md](references/bindings-hooks-variations.md) | binding attributes to data, custom binding sources, Pattern Overrides, Block Hooks filters/version history, block variations (JS/PHP), patterns |
| [references/theme-json-and-styles.md](references/theme-json-and-styles.md) | styling: supports serialization, skip-serialization, `selectors`, theme.json block settings/styles, responsive and state styles, block visibility, block styles, per-block CSS, SVG icon API |
| [references/deprecations-and-migrations.md](references/deprecations-and-migrations.md) | "invalid content" errors, writing `deprecated[]`/`migrate`, apiVersion 3/iframe migration, platform deprecation → replacement table, announced-not-shipped list |
| [references/build-tooling.md](references/build-tooling.md) | scaffolding with create-block, package.json scripts, wp-scripts flags/breaking changes, webpack overrides, ESLint flat config, wp-env, Playwright/Vitest tests, @wordpress/build, i18n/distribution |

Primary sources: developer.wordpress.org/block-editor, schemas.wp.org/trunk/block.json, make.wordpress.org/core field guides (6.7–7.1) and 7.2 dev notes, github.com/WordPress/gutenberg (packages/*/CHANGELOG.md).

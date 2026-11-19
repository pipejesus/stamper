# Build tooling: create-block, @wordpress/scripts, wp-env, tests

Versions as of 2026-11-19: `@wordpress/scripts` 37.2.0, `@wordpress/create-block` 5.2.0, `@wordpress/build` 0.28.0, `@wordpress/interactivity` 8.2.0, `@wordpress/interactivity-router` 3.2.0.

## Scaffold

```bash
npx @wordpress/create-block@latest acme-cards --namespace=acme --textdomain=acme             # static block plugin
npx @wordpress/create-block@latest acme-latest --namespace=acme --variant=dynamic            # render.php
npx @wordpress/create-block@latest acme-faq --template @wordpress/create-block-interactive-template  # Interactivity API
#   --variant typescript | client-side-navigation (interactive template)
npx @wordpress/create-block@latest --no-plugin --namespace=acme   # add another block into an existing plugin's src/ (run inside src/)
# other flags: --title --short-description --category --wp-env --no-wp-scripts --target-dir -t es5 (no build step)
```
Requires Node ≥ 22.13.0 (create-block 5.0, Oct 2026). Generated plugin: `src/{slug}/block.json|index.js|edit.js|save.js|render.php|view.js|style.scss|editor.scss`, bootstrap calls `wp_register_block_types_from_metadata_collection( __DIR__ . '/build', __DIR__ . '/build/blocks-manifest.php' )`. Interactive template scripts: `wp-scripts build --experimental-modules --blocks-manifest`.

## Multi-block plugin layout

```
acme-blocks/
  acme-blocks.php           # Plugin header + registration (see php-registration-and-render.md)
  package.json
  src/
    card/   block.json index.js edit.js save.js style.scss editor.scss
    latest/ block.json index.js edit.js render.php
    faq/    block.json index.js edit.js render.php view.js
  build/    (generated; commit only if you ship without a build step)
```

## package.json

```json
{
  "name": "acme-blocks",
  "private": true,
  "engines": { "node": ">=22.22.2" },
  "scripts": {
    "build": "wp-scripts build --experimental-modules --blocks-manifest",
    "start": "wp-scripts start --experimental-modules --blocks-manifest",
    "start:hot": "wp-scripts start --hot --experimental-modules --blocks-manifest",
    "lint:js": "wp-scripts lint-js",
    "lint:css": "wp-scripts lint-style",
    "format": "wp-scripts format",
    "test:unit": "wp-scripts test-unit-js",
    "test:e2e": "wp-scripts test-e2e",
    "plugin-zip": "wp-scripts plugin-zip",
    "packages-update": "wp-scripts packages-update",
    "env": "wp-env"
  },
  "devDependencies": { "@wordpress/scripts": "^37.2.0", "@wordpress/env": "*" },
  "dependencies": { "@wordpress/interactivity": "*" }
}
```
`@wordpress/*` runtime packages are externals (`wp.*` globals / script modules); `react`, `react-dom`, `react/jsx-runtime` map to `React`, `ReactDOM`, `ReactJSXRuntime`. Listing them in `dependencies` only pins types/IDE versions. Always bundled (dependency-extraction `BUNDLED_PACKAGES`): `@wordpress/admin-ui`, `dataviews`, `fields`, `global-styles-engine`, `global-styles-ui`, `grid`, `icons`, `image-editor`, `interface`, `sync`, `ui`, `undo-manager`, `views` … — their version is your responsibility.

## wp-scripts build/start behavior

- Entry detection: scans `src/**/block.json`, every `file:` script/module field becomes an entry; output mirrors into `build/`. Fallback entry `src/index.{js,jsx,ts,tsx}`. Also `.cjs/.cts/.mts` entries (36.0).
- Copies `block.json` and PHP files referenced by `render` / `variations`; `--webpack-copy-php` copies all PHP from `src`.
- CSS: `import './style.scss'` → `style-index.css` (front+editor); other imports → `{entry}.css` (editor). `*.module.scss` = CSS modules. RTL files generated. Don't name an entry containing `style`.
- `--experimental-modules`: compiles `viewScriptModule` (and module fields) as ES modules with `.asset.php` (deps include `@wordpress/interactivity` as module import). Still flagged experimental, but it is the standard path for Interactivity blocks.
- `--blocks-manifest` (scripts 30.14, Mar 2025): writes `build/blocks-manifest.php`. Standalone: `wp-scripts build-blocks-manifest --input=build --output=build/blocks-manifest.php`.
- Flags: `--source-path` `--output-path` (30.8), `--webpack-bundle-analyzer`, `--webpack-no-externals`, `--webpack-devtool`, `--hot` (start, React Fast Refresh; needs `SCRIPT_DEBUG` true and the Gutenberg plugin active).
- Custom webpack: `webpack.config.js` at root extending `@wordpress/scripts/config/webpack.config` (exports an array when `--experimental-modules` is on: map both configs). Keep using `wp-scripts start|build`, never raw `webpack`.

```js
// webpack.config.js
const defaults = require( '@wordpress/scripts/config/webpack.config' );
const extend = ( c ) => ( { ...c, resolve: { ...c.resolve, alias: { ...c.resolve?.alias, '@acme': require( 'path' ).resolve( __dirname, 'src/shared' ) } } } );
module.exports = Array.isArray( defaults ) ? defaults.map( extend ) : extend( defaults );
```

## @wordpress/scripts breaking-change timeline (read before bumping majors)

| Version (date) | Change |
|---|---|
| 31.0 (2025-11) | `lint-js` also lints `.cjs/.mjs/.cts/.mts` |
| 32.0 (2026-04) | ESLint 8 → **10**, flat config `eslint.config.*` default; `.eslintrc.*` fallback deprecated; default config `config/eslint.config.cjs` |
| 33.0 (2026-07) | `test-e2e` = **Playwright** (Puppeteer removed; `test-playwright` alias); React peer `^18 \|\| ^19` |
| 34.0 (2026-07) | Bundled Jest 29 → 30 |
| 35.0 (2026-09) | Node `^20.19 \|\| >=22.13`; Stylelint 17 |
| 36.0 (2026-09) | `test-unit-js` = **Vitest 5** (consumer-installed `vitest` + Vite 7/8); Jest only via `test-unit-jest` with your own Jest deps; Node `^22.22.2 \|\| ^24.15.0 \|\| >=26`; markdownlint 0.41 rules; `lint-style` config via stylelint `resolveConfig` (may pick up ancestor/global config); `check-engines` checks only `engines` tools |
| 37.0 (2026-10) | webpack-dev-server 6 (SockJS removed; proxy `bypass` removed; Express 5 middleware) |
| 37.2 (2026-11) | PostCSS ≥ 8.5.33; `start --hot` + `--experimental-modules` reload loop fixed |

ESLint flat config extending the WP preset:
```js
// eslint.config.cjs
const wp = require( '@wordpress/scripts/config/eslint.config.cjs' );
module.exports = [ ...wp, { ignores: [ 'build/**', 'vendor/**' ] } ];
```

## Local environment & tests

- `wp-env` (`.wp-env.json`): `{ "core": null, "phpVersion": "8.3", "plugins": [ "." ], "config": { "SCRIPT_DEBUG": true, "WP_DEBUG": true } }` → `npm run env start`. Test a prerelease: set `"core"` to the release zip URL (e.g. 7.2 RC zip from wordpress.org/download/releases) or a git ref. Gutenberg 24.4 adds an experimental runtime using local PHP + SQLite (no Docker).
- E2E: `@wordpress/e2e-test-utils-playwright` (`admin.createNewPost()`, `editor.insertBlock( { name: 'acme/card' } )`, `editor.getEditedPostContent()`, `editor.canvas` = iframe frame locator). Assert no block validation errors: check console for "Block validation failed".
- `--wp-env-config=.wp-env.test.json` option for `test-e2e` (37.0).
- Unit (36.0+): `npm i -D vitest` then `wp-scripts test-unit-js`.
- PHP: `wp scaffold plugin-tests` / PHPUnit against `wp-env run tests-cli`.
- Check block markup round-trip: create block, save, reload, ensure no "Attempt recovery".

## @wordpress/build (Rolldown-based, 0.x)

Opinionated build used by Gutenberg itself and for plugins shipping multiple packages, admin "routes"/pages, widgets, auto-generated PHP registration of scripts/modules/styles (`wpScript`, `wpScriptModuleExports`, `wpPlugin.*` package.json fields). Experimental features (`wpPlugin.pages`, routes, widgets). Not a drop-in for block plugins: keep `@wordpress/scripts` for blocks unless you need its package/route model. If you compile CSS modules with it, use ≥ 0.27.0 (WP 7.2 class-collision fix).

## Distribution

- `wp-scripts plugin-zip` zips files from `files` in package.json (or defaults: PHP, build, readme, languages).
- i18n: `wp i18n make-pot . languages/acme.pot --exclude=src,node_modules`; `wp i18n make-json languages --no-purge` for JS (`{domain}-{locale}-{md5}.json`); load with `load_plugin_textdomain` not needed for WP.org-hosted plugins (just-in-time loading).
- Plugin header: `Requires at least: 6.8` (for collections API) or `7.0` if using PHP-only blocks / `watch()` / module translations; `Requires PHP: 7.4` (WP 7.0 minimum).

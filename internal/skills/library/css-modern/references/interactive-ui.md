# Interactive UI without JavaScript

Order of preference: a native element or attribute, then CSS state, then a small JS enhancement.

## Popover (N25: C114 F125 S17, iOS 18.3)

```html
<button popovertarget="menu">Menu</button>
<nav id="menu" popover>…</nav>
```

| `popover=` | Light dismiss + Esc | Closes other autos | Use |
|---|---|---|---|
| `auto` (default) | yes | yes (unless nested via DOM or invoker) | menus, pickers, disclosures |
| `manual` | no | no | toasts, persistent panels |
| `hint` (C, F153; no S) | yes | no (only other hints) | tooltips, hover cards |

- The popover renders in the top layer, so no z-index is needed and it escapes `overflow: hidden`. It has no implicit role: add `role`/ARIA yourself (`aria-expanded` on the invoker is automatic).
- UA styles apply `position: fixed; inset: 0; margin: auto` (centred). Reset with `inset: auto; margin: 0` when anchoring.
- Focus is not moved in. The popover follows the invoker in tab order. Use `<dialog>` when focus must be trapped.
- JS API when needed: `el.showPopover({ source })`, `togglePopover()`, and the `toggle`/`beforetoggle` events. `ToggleEvent.source` is N26.
- Light dismiss fires on `click`, not `pointerdown`, since C154, so scrolling and right-click no longer dismiss.

### Enter/exit animation (shared by popover, dialog and any `display:none` toggle)

```css
[popover] {
  opacity: 0; translate: 0 -0.5rem;
  transition: opacity .2s, translate .2s,
              display .2s allow-discrete, overlay .2s allow-discrete;
}
[popover]:popover-open { opacity: 1; translate: 0; }
@starting-style { [popover]:popover-open { opacity: 0; translate: 0 -0.5rem; } }
@media (prefers-reduced-motion: reduce) { [popover] { transition: none; } }
```

- `@starting-style` (N24) supplies the "before" style for the first frame after insertion or `display:none` → shown.
- `transition-behavior: allow-discrete` (N24) delays the `display` flip until the exit finishes. `overlay` (C117 only) keeps the element in the top layer during exit. It's harmless elsewhere: other engines simply drop the element from the top layer immediately.
- The same pattern works for `::backdrop` (`dialog[open]::backdrop`).

## Invoker commands (N25: C135 F144 S26.2)

```html
<button commandfor="dlg" command="show-modal">Edit</button>
<dialog id="dlg">
  …<button commandfor="dlg" command="close">Done</button>
</dialog>
<button commandfor="nav" command="toggle-popover" aria-label="Menu">☰</button>
<button commandfor="vid" command="--play">Play</button>   <!-- custom -->
```

- Built-in commands: `show-modal`, `close`, `request-close`, `show-popover`, `hide-popover`, `toggle-popover`.
- A custom `--name` dispatches a `command` event (`event.command`, `event.source`) on the target. Use it as a declarative event bus instead of click handlers.
- Only `<button>` supports it. Use `popovertarget` (W since 2024) when supporting Safari < 26.2 for popovers. Dialog commands need a polyfill or a one-line JS fallback on older Safari.
- A popover opened by `commandfor` or `popovertarget` gets an **implicit anchor** for anchor positioning (N26).

## Interest invokers (C142; full feature C149; no F/S)

```html
<a href="/team/ada" interestfor="ada-card">Ada</a>
<div id="ada-card" popover="hint">…profile…</div>
```
```css
[interestfor] { interest-delay: 400ms 200ms; }   /* start end */
:interest-source { text-decoration-thickness: 2px; }
```

- Triggered by hover, keyboard focus or long-press. Events: `interest` and `loseinterest`.
- Fallback: the link still works. Detect with `'interestForElement' in HTMLButtonElement.prototype` and add the `interestfor` polyfill only where it's missing.

## `<dialog>` (W)

```html
<dialog id="confirm" closedby="any">
  <form method="dialog">
    <p>Delete?</p>
    <button value="cancel">Cancel</button><button value="ok" autofocus>Delete</button>
  </form>
</dialog>
```

- Modal (`showModal()` / `command="show-modal"`): top layer, background inert, focus moved, Esc closes. `<form method="dialog">` closes it and sets `returnValue`.
- `closedby="any | closerequest | none"`: `any` adds backdrop-click light dismiss (C134 F141; no S, where modals close on Esc only). `requestClose()` fires `cancel` first (N25).
- Scroll lock: `html:has(dialog:modal) { overflow: hidden; }`. Add `scrollbar-gutter: stable` on `html` to avoid layout shift.
- Animate it like popover, with `dialog[open]` as the shown selector.

## Anchor positioning (core N26: C125 F147 S26; Interop 2026)

```css
.trigger { anchor-name: --account; }
.menu {
  position: absolute;                       /* popovers are already fixed */
  position-anchor: --account;               /* omit when implicit (popovertarget/commandfor) */
  position-area: block-end span-inline-end; /* below, start-aligned, may extend right */
  inset: auto; margin: 0; margin-block-start: .25rem;
  position-try-fallbacks: flip-block, flip-inline;
  min-inline-size: anchor-size(inline);
  position-visibility: anchors-visible;     /* hide when anchor scrolls away */
}
.tip { position-area: top; justify-self: anchor-center; }
.badge { top: anchor(top); left: anchor(right); translate: -50% -50%; }
@position-try --left-side { position-area: inline-start; margin: 0 .5rem 0 0; }
.menu { position-try-fallbacks: --left-side, flip-block; position-try-order: most-height; }
.list-item { anchor-scope: --account; }     /* repeated components: keep names local */
```

- `position-area` is a 3×3 grid around the anchor (`top`, `bottom`, `span-left`, `center`, logical `block-start`, `self-*`). It's the simplest form. Reach for `anchor()` insets only for exact edges.
- Keywords: use `anchors-visible`/`anchors-valid` (plural; all engines). The singular `anchor-visible`/`anchor-valid` are C157 S27 only, though Chrome now flags the plural as deprecated.
- Transform-aware anchoring (follows `translate`/`scale` on the anchor) is C144 S27 (F flag).
- The anchor must come before the positioned element in tree order, or be in a different containing block. Otherwise the positioned element silently falls back to its normal position.
- Fallback without support: a popover stays viewport-centred (UA default), which is usable. For non-popovers, use `@supports not (anchor-name: --a)` and absolute positioning inside a relative wrapper.

## Customizable `<select>` (C135, S27; F flag)

```html
<select name="country">
  <button><selectedcontent></selectedcontent></button>
  <option value="de"><img src="de.svg" alt=""> Germany</option>
  <option value="fr"><img src="fr.svg" alt=""> France</option>
</select>
```
```css
select, ::picker(select) { appearance: base-select; }      /* opt in BOTH */
select { padding: .5rem .75rem; border-radius: .5rem; }
::picker(select) { border: 1px solid; border-radius: .5rem; padding: .25rem;
                   position-area: block-end span-inline-end; }   /* it's an anchored popover */
option:checked { font-weight: 600; }
option::checkmark { content: "✓"; }
select::picker-icon { transition: rotate .2s; }
select:open::picker-icon { rotate: 180deg; }
```

- Rich option content (images, spans) parses in current engines. Older ones keep only the text, so always include text.
- The listbox form (`multiple`/`size`) is C145. Safari 27.2 adds `::field-content` for styling the selected text when no `<selectedcontent>` is present.
- The fallback is a native select. Keyboard behaviour, form submission and mobile pickers stay native.

## Accordions: `<details>`

```html
<details name="faq"><summary>Shipping?</summary><p>…</p></details>
<details name="faq"><summary>Returns?</summary><p>…</p></details>  <!-- exclusive group, N24 -->
```
```css
:root { interpolate-size: allow-keywords; }        /* C129; inherited, set once */
details::details-content {                          /* N25: C131 F143 S18.4 */
  block-size: 0; overflow: clip;
  transition: block-size .3s, content-visibility .3s allow-discrete;
}
details[open]::details-content { block-size: auto; }
summary { list-style: none; cursor: pointer; }
summary::-webkit-details-marker { display: none; }  /* Safari keeps the marker otherwise */
```

- Engines without `interpolate-size` open instantly, which is an acceptable fallback. `calc-size(auto, size)` (C129) is the per-property alternative.
- Generic height animation that works everywhere (W) uses grid rows:

```css
.panel { display: grid; grid-template-rows: 0fr; transition: grid-template-rows .3s; }
.panel.is-open { grid-template-rows: 1fr; }
.panel > .inner { overflow: hidden; }
```

- `hidden="until-found"` (C102 F148; no S) makes collapsed content searchable with find-in-page. Chromium also auto-expands closed `<details>` on a find-in-page match. Verify this in other engines before relying on it.

## Forms

| Need | Use | Status |
|---|---|---|
| Auto-growing textarea/input | `field-sizing: content` (+ `min-block-size: 3lh; max-block-size: 12lh`) | N26 |
| Validate only after interaction | `:user-invalid`, `:user-valid` | W |
| Brand-coloured checkboxes/radios/range | `accent-color` | C F S26.2 |
| Fully custom checkbox | `appearance: none` + `:checked` styles | W |
| Native switch | `<input type="checkbox" switch>` | S17.4 only (others show a checkbox) |
| File button | `::file-selector-button` | W |
| Arrow-key roving focus in toolbars | `focusgroup` attribute | C150 |
| Off-canvas content unfocusable | `inert` | W |

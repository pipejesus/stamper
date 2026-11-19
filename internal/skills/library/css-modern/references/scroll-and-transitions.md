# Scroll, animation and view transitions

Wrap every motion rule in `@media (prefers-reduced-motion: no-preference)`, or neutralise it under `reduce`.

## Scroll-driven animations (N26: C115 S26 F159, Baseline since 2026-10-27)

```css
@keyframes grow   { from { scale: 0 1; } }
@keyframes reveal { from { opacity: 0; translate: 0 2rem; } }

@supports (animation-timeline: view()) {
  @media (prefers-reduced-motion: no-preference) {
    .read-progress {
      position: fixed; inset: 0 0 auto; block-size: 4px; transform-origin: 0 50%;
      animation: grow linear both;
      animation-timeline: scroll(root block);        /* AFTER the shorthand */
    }
    .reveal {
      animation: reveal linear both;
      animation-timeline: view();
      animation-range: entry 0% cover 30%;
    }
  }
}
```

- Timelines: `scroll(<scroller> <axis>)` (`nearest`/`root`/`self`, `block`/`inline`/`x`/`y`), `view(<axis> <inset>)`, or named ones: `scroll-timeline: --g x` or `view-timeline: --card` on the source, then `animation-timeline: --g`.
- To reach a named timeline from a non-descendant, set `timeline-scope: --g` on a common ancestor (`timeline-scope: all` is F159 S26 only).
- Ranges: `cover`, `contain`, `entry`, `exit`, `entry-crossing`, `exit-crossing`, plus `%` or length. Set them with `animation-range: entry 10% exit 90%`.
- Duration is ignored (`auto`). Animate `transform`/`opacity`/`clip-path`/`filter` to stay on the compositor.
- Safe default: un-gated, an `animation: x linear both` with only `from` keyframes ends at the element's normal style, so content stays visible.
- Firefox only reached stable in 159, so test there and check Interop 2026 for remaining gaps.

## Scroll-triggered animations (C146 only)

These are time-based animations that start when a scroll position is crossed. They replace IntersectionObserver "play once" reveals.

```css
@keyframes card { from { translate: 0 -50% 0; } }
.card {
  timeline-trigger: --t view() contain 15% contain 85% / entry 100% exit 0%;
  trigger-scope: --t;                                   /* names are global otherwise */
  animation: card .35s ease-in-out both;
  animation-trigger: --t play-forwards play-backwards;  /* on enter / on leave */
}
```

The syntax reads: activation range, then `/`, then active range. Fallback: gate with `@supports (timeline-trigger: --t view())` and use an IntersectionObserver class toggle elsewhere, or show content statically.

## View transitions: same document (N25: C111 F144 S18)

```js
const go = (update, types = []) =>
  document.startViewTransition ? document.startViewTransition({ update, types }) : update();
```
```css
.card { view-transition-name: match-element; view-transition-class: card; } /* N25 */
::view-transition-group(.card) { animation-duration: .3s; }
::view-transition-old(root), ::view-transition-new(root) { animation-duration: .2s; }
html:active-view-transition-type(forward) { --slide: 1; }   /* N26 */
@media (prefers-reduced-motion: reduce) {
  ::view-transition-group(*), ::view-transition-old(*), ::view-transition-new(*) { animation: none !important; }
}
```

- Each `view-transition-name` must be unique among rendered elements at capture time. Duplicates abort the transition. `match-element` auto-names per element, and `view-transition-class` styles groups without naming each one.
- Default animation: cross-fade on `root`, plus morphs of size and position for named elements.
- Element-scoped `el.startViewTransition()` and `view-transition-scope` are C147. They allow concurrent, contained transitions.

## View transitions: cross-document / MPA (C126, S18.2; F160 behind `dom.viewTransitions.cross-document.enabled`)

```css
/* in CSS loaded by BOTH the old and the new page (same-origin) */
@view-transition { navigation: auto; }
.site-header { view-transition-name: site-header; }   /* persists across pages */
.post-thumb  { view-transition-name: match-element; }
```

- Both documents must opt in and the navigation must be same-origin (link, form, back/forward). Reloads and cross-origin navigations don't animate.
- `pageswap` (old page) and `pagereveal` (new page) events expose `event.viewTransition`. Set `.types` there for direction-aware transitions.
- Render blocking: `<link rel="expect" href="#main" blocking="render">` (C124 S18.2) delays the first paint until `#main` is parsed, so the capture isn't empty.
- Unsupported browsers just navigate normally. Ship it as a free enhancement for themes.

## Scroll snap (W; Interop 2026)

```css
.rail {
  display: grid; grid-auto-flow: column; grid-auto-columns: min(80%, 22rem); gap: 1rem;
  overflow-x: auto; overscroll-behavior-x: contain;
  scroll-snap-type: x mandatory; scroll-padding-inline: 1rem;
}
.rail > * { scroll-snap-align: start; }
html { scroll-padding-block-start: 5rem; }     /* sticky header offset for #anchors */
```

- `scroll-snap-stop: always` stops at every item. `scroll-margin` on targets adjusts per element.
- Events: `scrollend` (N25), `scrollsnapchange`/`scrollsnapchanging` (C129).
- `scroll-initial-target: nearest` (C133) sets the initial scroll position to an element.
- `scroll-behavior: smooth` (W) belongs inside `prefers-reduced-motion: no-preference`.

## CSS-only carousel controls (C135; tabs mode C154)

```css
@supports selector(::scroll-marker) {
  .rail { scroll-marker-group: after; anchor-name: --rail; }            /* `after tabs` = tablist */
  .rail::scroll-marker-group { display: flex; gap: .5rem; justify-content: center; }
  .rail > *::scroll-marker {
    content: ""; inline-size: .75rem; aspect-ratio: 1; border-radius: 50%;
    background: currentColor; opacity: .35;
  }
  .rail > *::scroll-marker:target-current { opacity: 1; }
  .rail::scroll-button(inline-start) { content: "‹" / "Previous"; }
  .rail::scroll-button(inline-end)   { content: "›" / "Next"; }
  .rail::scroll-button(*) { position: absolute; position-anchor: --rail; }
  .rail::scroll-button(inline-start) { position-area: inline-start center; }
  .rail::scroll-button(inline-end)   { position-area: inline-end center; }
  .rail::scroll-button(*):disabled { opacity: .3; }
}
```

- `scroll-marker-group: links` (default): markers act as links and every marker is a tab stop. `tabs`: tablist/tab/tabpanel roles, a single tab stop, arrow keys move between markers, and inactive panels leave the accessibility tree.
- `content: "x" / "label"` (alt text, N24) provides an accessible name.
- Fallback: a swipeable, snapping rail. Add JS buttons only when the controls are required.

## Scrollspy (C140)

```css
.toc { scroll-target-group: auto; }          /* anchors inside become scroll markers */
.toc a:target-current { font-weight: 700; color: var(--accent); }
```

## Related scroll properties

| Property | Use | Status |
|---|---|---|
| `overscroll-behavior: contain` | stop scroll chaining from modals and rails | C F; S partial |
| `scrollbar-gutter: stable` | no layout shift when the scrollbar appears | N24 |
| `scrollbar-width: thin`, `scrollbar-color` | styling without `::-webkit-scrollbar` | N24 / N25 |
| `overflow-anchor` | scroll anchoring on content insertion | N26 (S27) |
| `overflow: clip` (+ `overflow-clip-margin`) | clip without a scroll container (keeps sticky working) | W (margin: limited) |

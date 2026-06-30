## 2026-06-30 - Added ARIA labels to BookExemplarCard icon-only buttons
**Learning:** Found that the Svelte component for book exemplar cards contained several SVG icon-only interactive elements (both `<button>` and `<a>` elements) which lacked any accessible names, relying entirely on `title` attributes (which screen readers often ignore or handle poorly).
**Action:** When adding accessibility to icon buttons, always add `aria-label` attributes to ensure they are properly identified by screen readers, even if they already have `title` tooltips for sighted users.

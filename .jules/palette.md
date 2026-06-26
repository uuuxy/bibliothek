## 2026-06-23 - Added missing ARIA labels in BookBorrowersTab
**Learning:** Found that custom filter dropdowns and text inputs in `BookBorrowersTab.svelte` were lacking `aria-label` attributes for screen readers, and decorative SVG icons needed `aria-hidden="true"`.
**Action:** Added `aria-label` to `<select>` and `<input>` elements, and `aria-hidden="true"` to the decorative search SVG icon to improve screen reader accessibility.

## 2026-06-26 - Replaced complex div with button for Book Cards
**Learning:** For interactive UI elements acting like complex buttons (such as Book Cards), replacing a `div` that has an `onclick` handler with a semantic `<button type="button">` (while managing text-alignment and focus states via utility classes) drastically improves keyboard accessibility and screen-reader support without relying on `svelte-ignore` tags for accessibility errors.
**Action:** In future components acting as triggers/cards, always use semantic `<button>` or `<a>` instead of `<div onclick={...}>`, and use utility classes (e.g. `text-left`) if standard button formatting breaks the layout.

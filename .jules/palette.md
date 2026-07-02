## 2026-06-23 - Added missing ARIA labels in BookBorrowersTab
**Learning:** Found that custom filter dropdowns and text inputs in `BookBorrowersTab.svelte` were lacking `aria-label` attributes for screen readers, and decorative SVG icons needed `aria-hidden="true"`.
**Action:** Added `aria-label` to `<select>` and `<input>` elements, and `aria-hidden="true"` to the decorative search SVG icon to improve screen reader accessibility.

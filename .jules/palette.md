## 2026-06-23 - Added missing ARIA labels in BookBorrowersTab
**Learning:** Found that custom filter dropdowns and text inputs in `BookBorrowersTab.svelte` were lacking `aria-label` attributes for screen readers, and decorative SVG icons needed `aria-hidden="true"`.
**Action:** Added `aria-label` to `<select>` and `<input>` elements, and `aria-hidden="true"` to the decorative search SVG icon to improve screen reader accessibility.
## 2026-06-23 - Added missing ARIA labels to buttons in Mahnwesen components
**Learning:** Found that icon-only action buttons in Mahnwesen components (`MahnwesenFilters.svelte`, `MahnwesenTable.svelte`) were missing `aria-label`s, rendering them inaccessible to screen readers. In addition, SVGs used decoratively inside these buttons needed `aria-hidden="true"` to reduce screen reader noise.
**Action:** Added context-aware `aria-label` attributes to action buttons (e.g., dynamically including the student name in the email warning action) and `aria-hidden="true"` to the decorative inner SVG icons.

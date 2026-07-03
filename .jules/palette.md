## 2026-06-23 - Added missing ARIA labels in BookBorrowersTab
**Learning:** Found that custom filter dropdowns and text inputs in `BookBorrowersTab.svelte` were lacking `aria-label` attributes for screen readers, and decorative SVG icons needed `aria-hidden="true"`.
**Action:** Added `aria-label` to `<select>` and `<input>` elements, and `aria-hidden="true"` to the decorative search SVG icon to improve screen reader accessibility.
## 2026-07-03 - Added ARIA labels to Sidebar navigation and logout buttons
**Learning:** Collapsed sidebar views often remove visible text while leaving icon-only buttons. Without explicit `aria-label`s on the buttons and `aria-hidden="true"` on the enclosed SVGs, screen readers may read the raw SVG or simply announce "button", confusing users. Relying purely on the `title` attribute is insufficient for robust accessibility in these collapsed states.
**Action:** Always verify that buttons that become icon-only in collapsed/mobile views have an explicit `aria-label`, and ensure decorative SVGs within them use `aria-hidden="true"`.

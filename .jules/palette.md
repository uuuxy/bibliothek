## 2026-06-23 - Added missing ARIA labels in BookBorrowersTab
**Learning:** Found that custom filter dropdowns and text inputs in `BookBorrowersTab.svelte` were lacking `aria-label` attributes for screen readers, and decorative SVG icons needed `aria-hidden="true"`.
**Action:** Added `aria-label` to `<select>` and `<input>` elements, and `aria-hidden="true"` to the decorative search SVG icon to improve screen reader accessibility.

## 2026-07-07 - Emoji Button Accessibility
**Learning:** Interactive components using emojis as visual icons (like the quick actions in `BookCopiesManager.svelte`) cause screen readers to announce the default emoji names, which can be confusing.
**Action:** Wrap emojis in `<span aria-hidden="true">` and provide an explicit `aria-label` on the parent interactive element to ensure semantic clarity for assistive technologies.

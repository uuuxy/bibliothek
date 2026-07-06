## 2026-06-23 - Added missing ARIA labels in BookBorrowersTab
**Learning:** Found that custom filter dropdowns and text inputs in `BookBorrowersTab.svelte` were lacking `aria-label` attributes for screen readers, and decorative SVG icons needed `aria-hidden="true"`.
**Action:** Added `aria-label` to `<select>` and `<input>` elements, and `aria-hidden="true"` to the decorative search SVG icon to improve screen reader accessibility.
## 2026-07-06 - Added loading state to login button
**Learning:** Added a loading state to a critical asynchronous action (login button) significantly improves UX by giving immediate feedback. Using Svelte 5's `$state` makes tracking this locally within the store very clean, and Tailwind CSS makes styling the spinner and disabled state straightforward.
**Action:** Always consider adding loading states to primary action buttons that trigger network requests, especially in authentication flows.

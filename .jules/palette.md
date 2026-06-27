## 2026-06-23 - Added missing ARIA labels in BookBorrowersTab
**Learning:** Found that custom filter dropdowns and text inputs in `BookBorrowersTab.svelte` were lacking `aria-label` attributes for screen readers, and decorative SVG icons needed `aria-hidden="true"`.
**Action:** Added `aria-label` to `<select>` and `<input>` elements, and `aria-hidden="true"` to the decorative search SVG icon to improve screen reader accessibility.

## 2026-06-27 - Button Component Focus Visibility
**Learning:** The central Button component in `frontend/src/lib/components/ui/Button.svelte` lacked explicit focus indicators, which made keyboard navigation inaccessible or difficult to track across the app. Since many Tailwind utility buttons drop default focus styles, it's critical to add explicit `focus-visible` states.
**Action:** Always include explicit focus states (e.g. `focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2`) when building reusable interactive components to ensure keyboard accessibility isn't lost.

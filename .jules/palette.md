## 2026-06-23 - Added missing ARIA labels in BookBorrowersTab
**Learning:** Found that custom filter dropdowns and text inputs in `BookBorrowersTab.svelte` were lacking `aria-label` attributes for screen readers, and decorative SVG icons needed `aria-hidden="true"`.
**Action:** Added `aria-label` to `<select>` and `<input>` elements, and `aria-hidden="true"` to the decorative search SVG icon to improve screen reader accessibility.
## 2026-07-06 - Added loading state to login button
**Learning:** Added a loading state to a critical asynchronous action (login button) significantly improves UX by giving immediate feedback. Using Svelte 5's `$state` makes tracking this locally within the store very clean, and Tailwind CSS makes styling the spinner and disabled state straightforward.
**Action:** Always consider adding loading states to primary action buttons that trigger network requests, especially in authentication flows.
## 2026-07-06 - Dependency management
**Learning:** If the CI enforces security scanning tools like govulncheck and trivy, it will sometimes fail a PR if there is an existing, unrelated vulnerability in the base codebase.
**Action:** When a CI pipeline fails due to an existing vulnerability (like a CVE in a go library), update it to unblock the PR, even if it feels out of scope for the current persona.

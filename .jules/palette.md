## 2026-07-10 - ARIA labels for Icon-Only Buttons
**Learning:** Found multiple instances where icon-only buttons lacked an `aria-label`, specifically in list items like book exemplars or assignments where the target needs to be contextualized.
**Action:** Always add an explicit `aria-label` (e.g. 'Exemplar löschen', 'Buch entfernen') to any `<button>` that only contains an SVG to ensure screen reader compatibility, even if it has a `title` attribute.

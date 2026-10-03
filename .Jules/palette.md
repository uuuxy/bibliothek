## 2024-10-24 - Dynamic Disabled State Titles
**Learning:** In Svelte components with list actions (like `LmfPlanZeileAktionen.svelte`), buttons that disable based on index position (e.g., first/last row) should have their `title` attribute dynamically updated to explain *why* the button is disabled, rather than just stating its general function.
**Action:** When creating or updating interactive lists, use ternary logic in the `title` attribute to provide context when disabled (e.g., `title={isFirst ? 'Row is already at the top' : 'Move up'}`).

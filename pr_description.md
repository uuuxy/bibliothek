🎯 **What:** Replaced blocking `alert()` calls in `frontend/src/lib/stores/labels.svelte.js` with `toastStore.addToast()`.
💡 **Why:** `alert()` interrupts the user workflow and is generally considered poor practice in modern web applications. Replacing it with a toast notification improves the user experience and aligns with the existing UI patterns in the codebase.
✅ **Verification:** Modified code to use `toastStore.addToast(...)`, verified the visual diff, ran `npx prettier --write` specifically on the modified file to adhere to formatting standards, and ran the full linter and test suite. All tests pass and no regression was introduced.
✨ **Result:** Improved maintainability and UI consistency by using the established `toastStore` component instead of native blocking alerts.

🎯 **What:** Added missing unit tests for the generic error handling and specific `AbortError` timeout transformation logic inside `apiFetch.js`'s catch block.
📊 **Coverage:** Covered the generic `Error` re-throwing scenario and the network timeout scenario where an `AbortError` is intercepted and transformed into a user-friendly German timeout message.
✨ **Result:** Improved test coverage and reliability for network error scenarios, ensuring that future refactoring of `apiFetch` won't accidentally drop the critical timeout message mapping.

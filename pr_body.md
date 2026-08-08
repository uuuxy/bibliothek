🎯 **What:**
Added a missing test case for `NewOmniboxService` in `internal/service/omnibox_service_test.go` to verify correct dependency injection and initialization behavior.

📊 **Coverage:**
The test initializes all required service interfaces (zero-value dependencies) and asserts that they are accurately assigned to the returned `*defaultOmniboxService` structure.

✨ **Result:**
Improved overall codebase reliability and coverage by ensuring that the public constructor correctly constructs the `OmniboxService` without panicking or missing dependency fields, establishing a safety net for future refactoring.

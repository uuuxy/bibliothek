🎯 **What:**
Added comprehensive unit tests for `frontend/src/inventur/lib/bookHelpers.js` using Vitest. The functions in this file (`getSubjectColor`, `getStockDotColor`, `getSubjectGradient`, `getSpineGradient`, and `formatDate`) map inputs to colors, gradients, and date strings. Since these are pure, side-effect-free functions, they are trivial to test but were completely lacking coverage. Adding tests ensures robust behavior for these utility functions and prevents future regressions.

📊 **Coverage:**
Tests were added for the following scenarios across the 5 functions:
- `getSubjectColor`: Tests known subjects (e.g., 'Mathe', 'Biologie'), unknown subjects, and empty/undefined inputs.
- `getStockDotColor`: Tests edge cases for item availability thresholds (0 items, 1-4 items, 5+ items).
- `getSubjectGradient`: Tests variations of subjects (including trim/casing), known subjects, unknown subjects, and missing inputs to ensure correct CSS class application.
- `getSpineGradient`: Tests correct gradient mappings for standard and foreign language subjects alongside unhandled subject fallbacks.
- `formatDate`: Validates formatting of valid date strings and the correct handling of invalid, empty, or null strings.

✨ **Result:**
The test suite in the frontend now includes 18 new passing tests explicitly covering `bookHelpers.js`. This brings the module to 100% test coverage, providing a solid safety net that allows confident refactoring.

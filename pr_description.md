🎯 **What:** The code health issue addressed is the long and monolithic GenerateSchadensfallPDF function.
💡 **Why:** By extracting logical sections into private helper functions (addSchadensfallHeader, addSchadensfallAddress, addSchadensfallBody, addSchadensfallSignatures), the main function becomes much more readable, maintaining a high-level overview of the PDF structure, and improving maintainability.
✅ **Verification:** Verified by checking git diff to ensure functionality is preserved, running go vet, and confirming all tests pass via go test ./...
✨ **Result:** A more modular and readable Schadensfall PDF generation file.

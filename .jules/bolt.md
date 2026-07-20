## 2026-06-22 - [Refactoring N+1 Query in SupplierOrderHandler]
**Learning:** Found an N+1 issue in `SupplierOrderHandler` (inside `api/barcode.go`) where multiple database inserts were performed in a loop (`tx.Exec`) for each generated barcode when ordering copies. Refactored this to use a single bulk insert operation via `tx.CopyFrom` combined with `pgx.CopyFromRows`.
**Action:** Always prefer `pgx.CopyFromRows` for batch database creation or insertion. This drastically reduces database round-trips when processing larger quantities (like bulk ordering of books).
## 2026-07-09 - [High-performance string cleaning]
**Learning:** Found multiple sequential `strings.ReplaceAll` calls in `mapHeaderToField` (inside `inventur/import_verarbeitung.go`) being used to strip characters. This leads to unnecessary allocations and garbage collection overhead.
**Action:** Replaced sequential `strings.ReplaceAll` with a single-pass `strings.Builder` and byte-by-byte iteration when stripping multiple ASCII characters to avoid intermediate allocations and improve performance.
## 2026-07-13 - [Refactoring N+1 Query in OrderService]
**Learning:** Found an N+1 issue in `OrderService` (inside `api/order_service.go`) where multiple database inserts were performed in a loop (`tx.Exec`) for each order position (`bestellungPosition`) inside the `insertBestellpositionen` function. Refactored this to use a single bulk insert operation via `tx.CopyFrom` combined with `pgx.CopyFromRows`.
**Action:** Consistently use `pgx.CopyFromRows` for batch database creation or insertion. This eliminates N+1 query problems and significantly reduces database round-trips when processing larger quantities (like inserting multiple order lines).
## 2026-07-14 - [Optimize Reorder Queries]
**Learning:** Found redundant correlated subqueries in `sammleNachbestellungen` (`api/order_handler.go`) and `queryReorders` (`api/stats.go`) where the same subquery calculating available book copies was used in both the `SELECT` clause and the `WHERE` clause. This forces PostgreSQL to evaluate the expensive subquery twice per row.
**Action:** Used `JOIN LATERAL (...) v ON true` to evaluate the subquery exactly once per row and then referenced `v.verfuegbar` in both the `SELECT` and `WHERE` clauses, preventing the redundant subquery execution and improving read performance.
## 2026-07-15 - [Efficient string template replacements]
**Learning:** Found sequential `strings.ReplaceAll` calls inside `api/reports_pdf.go` used to inject dynamic data (like Vorname, Nachname) into PDF text templates. This leads to unnecessary intermediate allocations and increased GC pressure, especially when generating bulk PDFs.
**Action:** Replaced sequential `strings.ReplaceAll` calls with a single `strings.NewReplacer` which is highly optimized for multi-string replacement in a single pass.
## 2026-07-20 - [Optimize ISBN string cleaning]
**Learning:** Found sequential strings.ReplaceAll calls for stripping hyphens and spaces from ISBN strings across multiple services. This leads to excessive memory allocations and GC pressure.
**Action:** Centralized and replaced these sequential strings.ReplaceAll calls with a single-pass csvutil.CleanISBN() helper function to optimize string manipulation and prevent intermediate allocations.

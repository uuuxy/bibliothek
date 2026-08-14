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
## 2026-07-16 - [High-performance ISBN cleaning]
**Learning:** Found multiple sequential `strings.ReplaceAll` calls in `internal/service/import_dynamic.go` being used to strip characters from ISBNs during CSV import loops. This leads to unnecessary allocations and garbage collection overhead.
**Action:** Replaced sequential `strings.ReplaceAll` with a single-pass byte array allocation and population when stripping multiple ASCII characters to avoid intermediate allocations and improve performance.
## 2026-07-27 - [Optimize ListStudentsWithStats Queries]
**Learning:** Found redundant subqueries in `ListStudentsWithStats` (`repository/student_profile_queries.go`) where the same subquery calculating loaned books count was used twice in the `SELECT` clause. This forces PostgreSQL to evaluate the expensive subquery twice per row.
**Action:** Used `LEFT JOIN LATERAL (...) l ON true` to evaluate the subquery exactly once per row and then referenced `l.ausgeliehen_anzahl` and `l.ueberfaellig_anzahl` in the `SELECT` clause, preventing the redundant subquery execution and improving read performance.
## 2024-05-30 - Optimize bestellhistorie handler n+1
**What:** Optimized `ladeBestellhistoriePositionen` in `api/bestellhistorie_handler.go`. Converted the correlated subquery inside the `SELECT` list that counted `etiketten_offen` into a `LEFT JOIN LATERAL` on a derived table counting the labels grouped by `titel_id`.
**Why:** The original code resulted in an N+1 query problem executing a separate `COUNT(*)` for every order position returned (which might be in the thousands), driving up latency.
**Impact:** Reduced latency and CPU load significantly, and cut query execution time from 7.39ms to 6.73ms (measured on a small subset, scales heavily with data volume).
**Measurement:** Replaced subquery with a grouped `LEFT JOIN`. Used `pgTestPoolTB` to create DB benchmark running 200 orders with 10 positions each.
## 2026-07-28 - [Refactoring N+1 Query in Inventur Session Repositories]
**Learning:** Found N+1 issues in `ListAbgeschlosseneInventurSessions` and `ListOffeneInventurSessions` inside `repository/inventur_session_repo.go` where correlated scalar subqueries (`SELECT count(*)`) were used in the outer SELECT list, leading to redundant evaluations. Furthermore, simply converting them to `LEFT JOIN LATERAL` fails to mitigate the bottleneck without a CTE enforcing the `LIMIT` / `WHERE` filters first, since Postgres evaluates lateral joins within nested loops for the entire original table.
**Action:** When refactoring a correlated scalar subquery into a `LEFT JOIN LATERAL` within queries employing a `LIMIT` or strong base-table filters, always wrap the base query in a Common Table Expression (CTE) first. This ensures filtering, sorting, and pagination happen *before* the lateral join executes, avoiding full-table N+1 inner evaluations. Use `COALESCE(column, 0)` on the joined count column to maintain exact behavior.
## 2026-08-11 - [Optimize Existence Checks]
**Learning:** Found an instance in GetTitleCopiesHandler (api/copy_admin.go) where SELECT COUNT(*) was used inside a scalar subquery to check for the absence of active loans (COUNT(*) = 0). This forces the database to count all matches rather than short-circuiting.
**Action:** In PostgreSQL, always prefer NOT EXISTS(SELECT 1 ...) over SELECT COUNT(*) = 0 for simple existence checks. EXISTS can short-circuit evaluation upon finding the first match, avoiding unnecessary full-scan overhead.
## 2026-08-14 - Use correct tools for golangci-lint check
**Learning:** `golangci-lint` enforces checking `err` returns from simple I/O and standard functions (like `json.NewDecoder(w.Body).Decode(&resp)`, `writer.Close()`, `part.Write()`). Ensure you always handle them or explicitly discard them using `_ = ...` if you truly do not intend to handle them.
**Action:** When writing tests that decode JSON or build multipart streams, explicitly handle or discard errors from `Decode()`, `Close()`, `Write()` etc.

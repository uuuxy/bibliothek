## 2026-06-22 - [Refactoring N+1 Query in SupplierOrderHandler]
**Learning:** Found an N+1 issue in `SupplierOrderHandler` (inside `api/barcode.go`) where multiple database inserts were performed in a loop (`tx.Exec`) for each generated barcode when ordering copies. Refactored this to use a single bulk insert operation via `tx.CopyFrom` combined with `pgx.CopyFromRows`.
**Action:** Always prefer `pgx.CopyFromRows` for batch database creation or insertion. This drastically reduces database round-trips when processing larger quantities (like bulk ordering of books).
## 2026-07-09 - [High-performance string cleaning]
**Learning:** Found multiple sequential `strings.ReplaceAll` calls in `mapHeaderToField` (inside `inventur/import_verarbeitung.go`) being used to strip characters. This leads to unnecessary allocations and garbage collection overhead.
**Action:** Replaced sequential `strings.ReplaceAll` with a single-pass `strings.Builder` and byte-by-byte iteration when stripping multiple ASCII characters to avoid intermediate allocations and improve performance.
## 2026-07-13 - [Refactoring N+1 Query in OrderService]
**Learning:** Found an N+1 issue in `OrderService` (inside `api/order_service.go`) where multiple database inserts were performed in a loop (`tx.Exec`) for each order position (`bestellungPosition`) inside the `insertBestellpositionen` function. Refactored this to use a single bulk insert operation via `tx.CopyFrom` combined with `pgx.CopyFromRows`.
**Action:** Consistently use `pgx.CopyFromRows` for batch database creation or insertion. This eliminates N+1 query problems and significantly reduces database round-trips when processing larger quantities (like inserting multiple order lines).

## 2026-06-22 - [Refactoring N+1 Query in SupplierOrderHandler]
**Learning:** Found an N+1 issue in `SupplierOrderHandler` (inside `api/barcode.go`) where multiple database inserts were performed in a loop (`tx.Exec`) for each generated barcode when ordering copies. Refactored this to use a single bulk insert operation via `tx.CopyFrom` combined with `pgx.CopyFromRows`.
**Action:** Always prefer `pgx.CopyFromRows` for batch database creation or insertion. This drastically reduces database round-trips when processing larger quantities (like bulk ordering of books).
## 2025-07-04 - Backend: Replace N+1 INSERTS with pgx.CopyFromRows
**Learning:** In Go backend implementations using `pgx`, replacing iterative `tx.Exec` statements in a loop with a single bulk insert using `tx.CopyFrom` and `pgx.CopyFromRows` leverages PostgreSQL's high-efficiency `COPY` protocol, drastically reducing database overhead.
**Action:** When creating new bulk-insert capabilities, strictly use `tx.CopyFrom` combined with `pgx.CopyFromRows` instead of looping individual INSERT queries. Note that this only applies to insert-only scenarios, not UPSERTs.

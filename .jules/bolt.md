## 2026-06-22 - [Refactoring N+1 Query in SupplierOrderHandler]
**Learning:** Found an N+1 issue in `SupplierOrderHandler` (inside `api/barcode.go`) where multiple database inserts were performed in a loop (`tx.Exec`) for each generated barcode when ordering copies. Refactored this to use a single bulk insert operation via `tx.CopyFrom` combined with `pgx.CopyFromRows`.
**Action:** Always prefer `pgx.CopyFromRows` for batch database creation or insertion. This drastically reduces database round-trips when processing larger quantities (like bulk ordering of books).
## 2025-06-03 - Avoid Duplicate Keys in Bulk UPSERTs with UNNEST
**Learning:** When replacing N+1 `INSERT ... ON CONFLICT DO UPDATE` loop queries with a single bulk `UNNEST` operation, you must pre-deduplicate the records in memory if the source data might contain duplicates (like in CSV imports). Otherwise, PostgreSQL will throw an error: `ON CONFLICT DO UPDATE command cannot affect row a second time`.
**Action:** Always maintain a `seen` map to deduplicate items by their unique conflict constraint column before appending them to the array slices passed to `UNNEST`.

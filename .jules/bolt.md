## 2026-06-22 - [Refactoring N+1 Query in SupplierOrderHandler]
**Learning:** Found an N+1 issue in `SupplierOrderHandler` (inside `api/barcode.go`) where multiple database inserts were performed in a loop (`tx.Exec`) for each generated barcode when ordering copies. Refactored this to use a single bulk insert operation via `tx.CopyFrom` combined with `pgx.CopyFromRows`.
**Action:** Always prefer `pgx.CopyFromRows` for batch database creation or insertion. This drastically reduces database round-trips when processing larger quantities (like bulk ordering of books).

* **Preallocating Slices in Nested Loops**: When appending to a slice inside a nested loop and the exact total number of iterations can be calculated beforehand (e.g. `len(sliceA) * len(sliceB)`), preallocate the slices using `make([]T, 0, capacity)`. This avoids multiple allocations and array copying as the slice grows, significantly improving performance.

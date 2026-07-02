## 2026-06-22 - [Refactoring N+1 Query in SupplierOrderHandler]
**Learning:** Found an N+1 issue in `SupplierOrderHandler` (inside `api/barcode.go`) where multiple database inserts were performed in a loop (`tx.Exec`) for each generated barcode when ordering copies. Refactored this to use a single bulk insert operation via `tx.CopyFrom` combined with `pgx.CopyFromRows`.
**Action:** Always prefer `pgx.CopyFromRows` for batch database creation or insertion. This drastically reduces database round-trips when processing larger quantities (like bulk ordering of books).

## 2026-07-02 - [Avoid os.Getenv Lock Contention in HTTP Handlers]
**Learning:** Found multiple places in the hot path (HTTP middleware and handlers) where `os.Getenv` was being called on every request (e.g. for `APP_ENV` and `ALLOWED_ORIGIN`). In Go, `os.Getenv` acquires a read lock on the environment variables map, which causes lock contention and degrades performance under high concurrency.
**Action:** Always read environment variables once at startup (e.g. storing them in server structs, closure variables, or configuration structs) rather than calling `os.Getenv` dynamically inside HTTP handlers or middleware.

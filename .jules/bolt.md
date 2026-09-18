## 2026-09-18 - Batch Execution for N+1 Query Updates in pgx
**Learning:** Sequential parameterized queries inside a loop (like `tx.Exec` in a `for` loop) lead to an N+1 performance bottleneck due to excessive database roundtrips, especially when the target database doesn't auto-pipeline queries.
**Action:** Used `pgx.Batch` to construct a batch of parameterized `UPDATE` statements in memory (`batch.Queue()`) and submitted them in a single roundtrip via `tx.SendBatch(ctx, batch).Close()`. Added an early exit clause for empty slices to avoid dispatching an empty batch.

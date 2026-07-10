## Performance Issue Identification
The codebase performs an `N+1` insert operations within a loop in `api/order_service.go`, around line 157. Specifically, it executes `tx.Exec` repeatedly to insert `bestellungPosition` (order positions). This causes excessive database roundtrips, significantly decreasing the application performance, specially for orders with multiple items.

## Proposed Optimization
We will replace the looped `tx.Exec` calls with a single bulk insert operation using `pgx.CopyFromRows`. As noted in the memory and bolt's `.jules/bolt.md`, this is the codebase's standard pattern for batch database insertions, eliminating `N+1` bottlenecks.

1.  **Refactor `ProcessOrder` loop to `tx.CopyFrom`:** We will convert the iteration over `positionen` into a two-dimensional slice `[][]any` to use with `pgx.CopyFromRows`.

## Execution Plan
1. **Optimize `api/order_service.go`**: Replace `tx.Exec` in loop for `bestellungen_positionen` with `tx.CopyFrom` combined with `pgx.CopyFromRows`.
2. **Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done.**
3. **Submit the PR**: Create a PR with title "⚡ Bolt: [performance improvement]".

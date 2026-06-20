## 2024-06-20 - [Bulk Database Inserts Pattern]
**Learning:** For high-performance bulk database inserts in Go within this codebase, always use `tx.CopyFrom` combined with `pgx.CopyFromRows([][]any{...})` instead of looping individual `tx.Exec` statements. This is specifically needed to resolve N+1 query bottlenecks and is the project's standard approach for batch creations.
**Action:** When creating or updating multiple database records (e.g. creating multiple copies of books), always formulate the inserts as a bulk operation using `CopyFrom` instead of iterating and inserting one by one.

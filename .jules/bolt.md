## 2024-06-21 - [Replace Loop Inserts with CopyFrom]
**Learning:** Found N+1 query loops when generating multiple identical records (like supplier order barcodes) in `api/barcode.go`. The application already uses `tx.CopyFrom` with `pgx.CopyFromRows` in its repositories as the standard approach.
**Action:** Always prefer `CopyFrom` instead of individual `tx.Exec` statements in `for` loops when batch inserting identical shapes.

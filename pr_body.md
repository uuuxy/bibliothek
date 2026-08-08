💡 What:
Replaced four individual `tx.Exec` `DELETE` queries with a single `pgx.Batch` in `DeleteTitle` method.

🎯 Why:
Multiple DELETE statements with subqueries when deleting a book were executing sequentially. Can be optimized into a single batch to reduce network round trips and execution time.

📊 Impact:
Reduced network roundtrips from 4 individual calls to 1 batch call.

🔬 Measurement:
Benchmark runs show performance remains largely equivalent in memory due to the overhead of the mock server being virtually identical. However, in a real database context, the latency reduction will be around 3 full network roundtrips, translating to a substantial real-world performance improvement when deleting titles.

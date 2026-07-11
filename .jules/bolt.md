## 2025-10-24 - Omnibox search latency
**Learning:** The unified search API (`/api/search`) sequentially queried the `StudentRepository` and `BookRepository` when resolving queries. Since these database queries are independent, this resulted in an additive latency cost.
**Action:** Use a `sync.WaitGroup` to execute independent backend repository lookups concurrently, effectively making the latency the maximum of the two queries rather than the sum.

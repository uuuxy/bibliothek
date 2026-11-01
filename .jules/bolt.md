## 2024-05-24 - Postgres EXIST Check Optimization
**Learning:** Found instances where `SELECT count(*) > 0` was used for existence checks in PostgreSQL. This forces a full table/index scan for matching rows.
**Action:** Always prefer `SELECT EXISTS(SELECT 1 ...)` for boolean existence checks. `EXISTS` short-circuits the evaluation and returns as soon as the first match is found, saving significant CPU and I/O over counting all matches.

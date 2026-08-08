1. Add caching to `GetDistinctClasses` in `repository/student_profile_queries.go` and `repository/student.go`.
   - Update `pgStudentRepository` to include an in-memory cache for the classes list and a mutex.
   - Also include a cache expiration mechanism (e.g. 5 minutes).
2. Measure cache impact using the benchmark created in `repository/student_benchmark_test.go`.
3. Complete pre commit steps to ensure proper testing, verification, review, and reflection are done.
4. Submit the pull request with a descriptive title and message detailing performance gains.

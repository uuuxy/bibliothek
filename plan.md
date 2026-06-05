1. **Analyze the Security Issue**: Review `api/middleware.go` to understand how CORS handles the `ALLOWED_ORIGIN` environment variable. The vulnerability occurs because `Access-Control-Allow-Credentials` is set to `"true"` without checking if the allowed origin is `*`. Setting both `Access-Control-Allow-Origin: *` and `Access-Control-Allow-Credentials: true` is a security misconfiguration and rejected by browsers, potentially leading to insecure workarounds or exposing credentials if dynamically reflected.
2. **Update `CORSMiddleware`**: Modify the middleware to:
   - Accept the origin if it matches `allowedOrigin` strictly, OR if `allowedOrigin` is configured as the wildcard `*`.
   - Set `Access-Control-Allow-Credentials: "true"` **only** when `allowedOrigin != "*"`.
3. **Verify Fix**: Ensure that `api/middleware_test.go` correctly passes, and run the whole `go test ./...` test suite.
4. **Pre-commit**: Complete pre commit steps to make sure proper testing, verifications, reviews and reflections are done.
5. **Submit**: Commit the changes with a proper security fix PR description.

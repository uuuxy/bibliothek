## 2024-06-21 - Unbounded form parsing G120

**Vulnerability:** gosec G120 in api/import.go Unbounded form parsing.

**Learning:** r.ParseMultipartForm was called right after r.Body = http.MaxBytesReader... but gosec still flagged it. Wait, the autofix tells us: explicitly annotate the ParseMultipartForm call with // #nosec G120 to suppress the linter false positive. This is also stated in Memory!

**Prevention:** Remember to use the nosec directive when MaxBytesReader is already applied.
## 2024-06-21 - Path traversal in router.go

**Vulnerability:** G304 / G703 Path traversal via taint analysis in api/router.go.

**Learning:** The router joins untrusted input `r.URL.Path` directly with a base directory `./frontend/dist` using `filepath.Join`. This is a classic path traversal pattern. Although `http.FileServer` internally handles this safely (if the path starts with `/`), `os.Stat` receives the uncleaned path, which might allow statting arbitrary files outside the intended directory or TOCTOU issues.

**Prevention:** Use `filepath.Clean` or validate the resolved path is actually within the intended directory using `strings.HasPrefix(cleanPath, cleanBaseDir)` as mentioned in Memory, or use Go 1.24 `os.OpenRoot`.
## 2024-06-21 - SMTP Header Injection

**Vulnerability:** G707 SMTP command/header injection via taint analysis.

**Learning:** Unsanitized user inputs used directly to construct an email header can allow an attacker to inject arbitrary headers or email commands. This occurs in `mailservice/mailservice.go` and `api/mail_sender.go`.

**Prevention:** Sanitize user input by removing newlines `\r` and `\n` or carefully parse email addresses with `net/mail.ParseAddress` as instructed in Memory.

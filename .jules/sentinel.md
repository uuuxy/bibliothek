## 2026-06-14 - SMTP Header Injection via Unsanitized Email Subject
**Vulnerability:** A `HIGH` severity gosec warning (G707) identified a potential SMTP header injection in `api/mail_sender.go`. The issue occurred because `req.Subject` was written directly to the email's headers without first sanitizing the `\r` (carriage return) and `\n` (line feed) characters. This could allow an attacker to inject arbitrary SMTP headers.
**Learning:** `net/mail.ParseAddress()` is great for validating the `To` field, but it does not sanitize or cover other fields like `Subject` and `From`. This project uses direct string formatting for headers before the body, so any free-text header must be manually sanitized.
**Prevention:** Strictly sanitize all free-text email headers by stripping carriage returns (`\r`) and line feeds (`\n`) using `strings.ReplaceAll` before appending them to the email header buffer.

## 2025-02-27 - Mitigate Path Traversal (G304) Using Go 1.24 os.OpenRoot
**Vulnerability:** A `G304: Potential file inclusion via variable` vulnerability was found in `api/image_caching.go` during image cover caching, caused by insecure creation of files via concatenated paths.
**Learning:** Manual path sanitization using `filepath.Clean` and `strings.HasPrefix` logic is prone to errors, false positives, and flagged by security linters (like gosec). Go 1.24 introduced `os.OpenRoot(dir)` which natively provides OS-level bounded directory contexts.
**Prevention:** Rather than performing string validations to enforce file directory boundaries, prefer adopting `os.OpenRoot` and manipulating files via `root.OpenFile`, `root.Remove`, and serving via `http.ServeFileFS(..., root.FS(), ...)` to securely prevent directory traversal.

## 2026-06-14 - Mitigate Path Traversal (G703) in static file serving using Go 1.24 os.OpenRoot
**Vulnerability:** A `HIGH` severity gosec warning (G703) identified a potential path traversal vulnerability in `api/router.go`. The issue occurred because `r.URL.Path` was concatenated with `./frontend/dist` and passed directly to `os.Stat` to check file existence. While `http.Dir` restricts actual file serving, `os.Stat` could be abused as an oracle to check for file existence outside the intended directory.
**Learning:** Concatenating user-controlled paths and passing them to OS functions like `os.Stat` or `os.Open` without proper bounding is risky and flagged by security tools.
**Prevention:** Rather than relying on manual path sanitization, use Go 1.24's `os.OpenRoot(dir)` which natively provides OS-level bounded directory contexts. The `root.Open(path)` function securely prevents directory traversal. Initialize the root context once at startup for efficiency, and use `http.ServeFileFS(..., root.FS(), ...)` for serving.

## 2026-06-14 - SMTP Header Injection via Unsanitized Email Subject
**Vulnerability:** A `HIGH` severity gosec warning (G707) identified a potential SMTP header injection in `api/mail_sender.go`. The issue occurred because `req.Subject` was written directly to the email's headers without first sanitizing the `\r` (carriage return) and `\n` (line feed) characters. This could allow an attacker to inject arbitrary SMTP headers.
**Learning:** `net/mail.ParseAddress()` is great for validating the `To` field, but it does not sanitize or cover other fields like `Subject` and `From`. This project uses direct string formatting for headers before the body, so any free-text header must be manually sanitized.
**Prevention:** Strictly sanitize all free-text email headers by stripping carriage returns (`\r`) and line feeds (`\n`) using `strings.ReplaceAll` before appending them to the email header buffer.

## 2025-02-27 - Mitigate Path Traversal (G304) Using Go 1.24 os.OpenRoot
**Vulnerability:** A `G304: Potential file inclusion via variable` vulnerability was found in `api/image_caching.go` during image cover caching, caused by insecure creation of files via concatenated paths.
**Learning:** Manual path sanitization using `filepath.Clean` and `strings.HasPrefix` logic is prone to errors, false positives, and flagged by security linters (like gosec). Go 1.24 introduced `os.OpenRoot(dir)` which natively provides OS-level bounded directory contexts.
**Prevention:** Rather than performing string validations to enforce file directory boundaries, prefer adopting `os.OpenRoot` and manipulating files via `root.OpenFile`, `root.Remove`, and serving via `http.ServeFileFS(..., root.FS(), ...)` to securely prevent directory traversal.
## 2026-06-27 - Mitigate Path Traversal (G304) in Backup using Go 1.24 os.OpenRoot
**Vulnerability:** A `MEDIUM` severity gosec warning (G304) identified a potential path traversal vulnerability in `inventur/backup_email.go`. The issue occurred because `os.Open(path)` was called on a file path directly computed from user-controlled paths inside a zip creation loop.
**Learning:** Manual path sanitization inside `filepath.Walk` is risky and can lead to Time-of-check to time-of-use (TOCTOU) issues or symlink-based path traversals.
**Prevention:** Rather than using `os.Open(path)`, use Go 1.24's `os.OpenRoot(srcDir)` to open a bounded directory access context, and securely open the relative paths within the bounded context using `root.Open(relPath)`.

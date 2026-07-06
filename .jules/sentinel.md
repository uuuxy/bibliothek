## 2026-06-14 - SMTP Header Injection via Unsanitized Email Subject
**Vulnerability:** A `HIGH` severity gosec warning (G707) identified a potential SMTP header injection in `api/mail_sender.go`. The issue occurred because `req.Subject` was written directly to the email's headers without first sanitizing the `\r` (carriage return) and `\n` (line feed) characters. This could allow an attacker to inject arbitrary SMTP headers.
**Learning:** `net/mail.ParseAddress()` is great for validating the `To` field, but it does not sanitize or cover other fields like `Subject` and `From`. This project uses direct string formatting for headers before the body, so any free-text header must be manually sanitized.
**Prevention:** Strictly sanitize all free-text email headers by stripping carriage returns (`\r`) and line feeds (`\n`) using `strings.ReplaceAll` before appending them to the email header buffer.

## 2025-02-27 - Mitigate Path Traversal (G304) Using Go 1.24 os.OpenRoot
**Vulnerability:** A `G304: Potential file inclusion via variable` vulnerability was found in `api/image_caching.go` during image cover caching, caused by insecure creation of files via concatenated paths.
**Learning:** Manual path sanitization using `filepath.Clean` and `strings.HasPrefix` logic is prone to errors, false positives, and flagged by security linters (like gosec). Go 1.24 introduced `os.OpenRoot(dir)` which natively provides OS-level bounded directory contexts.
**Prevention:** Rather than performing string validations to enforce file directory boundaries, prefer adopting `os.OpenRoot` and manipulating files via `root.OpenFile`, `root.Remove`, and serving via `http.ServeFileFS(..., root.FS(), ...)` to securely prevent directory traversal.

## 2026-07-06 - Mitigate Path Traversal (G304) during ZIP creation using Go 1.24 os.OpenRoot
**Vulnerability:** A `G304: Potential file inclusion via variable` vulnerability was found in `inventur/backup_email.go` during ZIP creation of the backup directory, caused by opening files directly using `os.Open(path)` within `filepath.Walk`.
**Learning:** During directory traversal tasks like `filepath.Walk`, opening files dynamically based on constructed paths is susceptible to TOCTOU and directory traversal if the path structure changes or uses symlinks. Go 1.24's `os.OpenRoot(dir)` enables robust boundary containment for subsequent file operations within that root directory.
**Prevention:** Instead of using `os.Open` with full paths, initialize an `os.Root` context at the source directory, calculate the relative path, and strictly use `root.Open(relPath)` to eliminate file inclusion and directory traversal risks.

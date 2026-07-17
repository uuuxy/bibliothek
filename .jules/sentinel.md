## 2026-06-14 - SMTP Header Injection via Unsanitized Email Subject
**Vulnerability:** A `HIGH` severity gosec warning (G707) identified a potential SMTP header injection in `api/mail_sender.go`. The issue occurred because `req.Subject` was written directly to the email's headers without first sanitizing the `\r` (carriage return) and `\n` (line feed) characters. This could allow an attacker to inject arbitrary SMTP headers.
**Learning:** `net/mail.ParseAddress()` is great for validating the `To` field, but it does not sanitize or cover other fields like `Subject` and `From`. This project uses direct string formatting for headers before the body, so any free-text header must be manually sanitized.
**Prevention:** Strictly sanitize all free-text email headers by stripping carriage returns (`\r`) and line feeds (`\n`) using `strings.ReplaceAll` before appending them to the email header buffer.

## 2025-02-27 - Mitigate Path Traversal (G304) Using Go 1.24 os.OpenRoot
**Vulnerability:** A `G304: Potential file inclusion via variable` vulnerability was found in `api/image_caching.go` during image cover caching, caused by insecure creation of files via concatenated paths.
**Learning:** Manual path sanitization using `filepath.Clean` and `strings.HasPrefix` logic is prone to errors, false positives, and flagged by security linters (like gosec). Go 1.24 introduced `os.OpenRoot(dir)` which natively provides OS-level bounded directory contexts.
**Prevention:** Rather than performing string validations to enforce file directory boundaries, prefer adopting `os.OpenRoot` and manipulating files via `root.OpenFile`, `root.Remove`, and serving via `http.ServeFileFS(..., root.FS(), ...)` to securely prevent directory traversal.

## 2026-07-04 - Mitigate Path Traversal (G304) during Zip Creation using Go 1.24 os.OpenRoot
**Vulnerability:** A `G304: Potential file inclusion via variable` vulnerability was found in `inventur/backup_email.go` during zip archive creation. The code used `os.Open(path)` with paths discovered dynamically via `filepath.Walk`, allowing potential symlink/traversal attacks.
**Learning:** Even when iterating over a directory structure dynamically using functions like `filepath.Walk`, using `os.Open(path)` with the absolute or concatenated paths remains vulnerable to Time-of-check to time-of-use (TOCTOU) and path traversal. Go 1.24's `os.OpenRoot(dir)` provides a robust solution by anchoring file operations to a specific root directory.
**Prevention:** In functions that process files dynamically within a directory tree (such as creating zip archives), first establish a bounded context using `root, err := os.OpenRoot(dir)`. Subsequently, safely access the discovered files by passing their relative paths to the root descriptor: `root.Open(relPath)`.

## 2026-07-02 - Mitigate Command Injection and Password Exposure (G204) Using .pgpass
**Vulnerability:** The PostgreSQL password was being passed to `pg_dump` via the `PGPASSWORD` environment variable in `jobs/backup.go`. This is insecure because environment variables can be exposed to other processes on the same system, leading to credential theft.
**Learning:** `gosec` G204 can be triggered by passing connection strings directly to subprocesses, and using `PGPASSWORD` exposes the password to other processes.
**Prevention:** To securely pass PostgreSQL connection details to external subprocesses like `pg_dump`, construct explicit command-line arguments and DO NOT pass the password via `PGPASSWORD`. Instead, supply the password via a securely generated temporary `.pgpass` file with restrictive permissions.

## 2026-07-17 - Mitigate Decompression Bomb DoS (G110) Using io.LimitReader
**Vulnerability:** A `G110: Potential DoS vulnerability via decompression bomb` vulnerability was found in `cmd/restore-backup/main.go` when reading from a `gzip.Reader` using `io.Copy(out, gz)`.
**Learning:** `io.Copy` does not bound the size of the decompressed data. An attacker could provide a very small compressed file (e.g. 10MB) that decompresses to petabytes, leading to memory exhaustion or disk space exhaustion. Wrapping the reader with `io.LimitReader(gz, limit)` stops `io.Copy` early, but it silently reports success if the limit is exactly reached.
**Prevention:** To securely prevent Decompression Bomb (Zip Bomb) vulnerabilities, wrap the reader in `io.LimitReader(gz, limit)`. Since `io.Copy` will silently stop and report success when the limit is reached, you must explicitly check if the bytes copied equal the limit, and if so, attempt to read one more byte (e.g., `gz.Read(make([]byte, 1))`) to trigger and return an explicit error.

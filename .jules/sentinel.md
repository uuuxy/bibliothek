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

## 2026-07-16 - Mitigate Decompression Bomb (G110) Using io.LimitReader
**Vulnerability:** A `MEDIUM` severity gosec warning (G110) identified a potential DoS vulnerability via decompression bomb in `cmd/restore-backup/main.go`. The issue occurred because `io.Copy(out, gz)` read an unbounded amount of compressed data into `out`, which could lead to resource exhaustion if a maliciously crafted compressed file was provided.
**Learning:** Functions that read and decompress data, such as `gzip.NewReader` combined with `io.Copy`, are vulnerable to decompression bombs (zip bombs) where a small compressed file expands to an enormous size.
**Prevention:** Always bound the maximum amount of decompressed data using `io.LimitReader` when reading from compressed streams to prevent potential DoS attacks.
## 2024-07-24 - [CRITICAL/HIGH] Fix CWE-614 Insecure Cookie Secure Attribute Configuration
**Vulnerability:** The `Secure` attribute of cookies was dynamically configured using `os.Getenv("APP_ENV") != "local"`, which bypasses the decoupled configuration and creates a CWE-614 vulnerability.
**Learning:** In this project, cookie `Secure` attributes must be configured using the explicitly injected `cookieSecure` boolean parameter (derived from `COOKIE_SECURE`).
**Prevention:** Always use the injected `cookieSecure` parameter instead of reading the environment variable directly.

## 2025-02-27 - Mitigate Path Traversal (G304) Using Go 1.24 os.OpenRoot
**Learning:** Manual path sanitization using `filepath.Clean` and `strings.HasPrefix` logic is prone to errors, false positives, and flagged by security linters (like gosec). Go 1.24 introduced `os.OpenRoot(dir)` which natively provides OS-level bounded directory contexts.
**Prevention:** Rather than performing string validations to enforce file directory boundaries, prefer adopting `os.OpenRoot` and manipulating files via `root.OpenFile`, `root.Remove`, and serving via `http.ServeFileFS(..., root.FS(), ...)` to securely prevent directory traversal.

## 2026-07-04 - Mitigate Path Traversal (G304) during Zip Creation using Go 1.24 os.OpenRoot
**Vulnerability:** A `G304: Potential file inclusion via variable` vulnerability was found in `inventur/backup_email.go` during zip archive creation. The code used `os.Open(path)` with paths discovered dynamically via `filepath.Walk`, allowing potential symlink/traversal attacks.
**Learning:** Even when iterating over a directory structure dynamically using functions like `filepath.Walk`, using `os.Open(path)` with the absolute or concatenated paths remains vulnerable to Time-of-check to time-of-use (TOCTOU) and path traversal. Go 1.24's `os.OpenRoot(dir)` provides a robust solution by anchoring file operations to a specific root directory.
**Prevention:** In functions that process files dynamically within a directory tree (such as creating zip archives), first establish a bounded context using `root, err := os.OpenRoot(dir)`. Subsequently, safely access the discovered files by passing their relative paths to the root descriptor: `root.Open(relPath)`.

## 2026-07-11 - Mitigate Path Traversal (G304) via ISBN in Image Caching
**Vulnerability:** The cache mechanism used a sanity check `filepath.Base(isbn) != isbn` to prevent traversal. However, on Linux systems, this evaluates to true for backslashes (`foo\bar`). Furthermore, `filepath.Base("..")` returns `".."`, rendering the validation ineffective against `..` payloads.
**Learning:** Do not rely on `filepath.Base` for input validation against traversal attacks, as its behavior differs across operating systems and poorly handles `..`. Even with `os.OpenRoot`, defense-in-depth requires sanitization against all traversal characters.
**Prevention:** Use explicit string checks for path separators and traversal indicators (e.g., `strings.ContainsAny(input, "/\\") || strings.Contains(input, "..")`) or regular expression allowed lists when validating inputs used in paths.

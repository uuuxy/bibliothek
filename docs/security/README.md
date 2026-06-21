# Security Audit & Scanner Reports

Dieser Ordner enthält die Ergebnisse unserer automatisierten Sicherheitsüberprüfungen (SAST & DAST). 
Alle identifizierten Produktions-Risiken wurden behoben.

## OWASP ZAP (DAST)
Der `zap_api_report.html` enthält das Ergebnis des dynamischen API-Scans.
- **Datum:** 21. Juni 2026
- **Status:** **0 FAIL, 116 PASS**
- **Info:** Die verbleibenden 4 Warnungen (WARN) beziehen sich auf unbedenkliche Header in der Swagger-Dokumentation (z. B. Inline-Styles, fehlende Sub-Resource Integrity), welche im Produktivbetrieb (`APP_ENV != "local"`) deaktiviert ist.

## gosec (SAST)
Der `gosec_report.json` enthält die statische Quellcode-Analyse des Go-Backends.
- **Datum:** 21. Juni 2026
- **Status:** Die folgenden Schwachstellen wurden in allen produktiven API-, Auth- und Job-Routen behoben:
  - CWE-22 (Path Traversal) durch `filepath.Clean` und Whitelisting
  - CWE-614 (Insecure Cookies) durch `Secure: os.Getenv("APP_ENV") != "local"`, `HttpOnly` und `SameSiteStrictMode`
  - CWE-703 (Unhandled Errors) durch explizites Fehlerhandling bei Dateischließungen
  - CWE-117 (Log Injection) durch einen neuen `SanitizeLog` Filter

Um die Scans lokal zu wiederholen, führe `./security-scan.sh` im Hauptverzeichnis aus.

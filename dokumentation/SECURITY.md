# Sicherheits- und Datenschutzkonzept (DSGVO)

Diese Dokumentation beschreibt die systemweiten Mechanismen zur Wahrung von Sicherheit und Datenschutz der Bibliotheks-Verwaltungssoftware.

---

## 🛡️ Technische Sicherheitsmaßnahmen

### 1. Schutz vor SQL-Injection (SQLi)
*   **Mechanismus:** Alle Interaktionen mit der PostgreSQL-Datenbank erfolgen ausschließlich über parametrisierte SQL-Abfragen unter Verwendung des `jackc/pgx/v5` Datenbank-Treibers.
*   **Umsetzung:** Es werden keine SQL-Strings durch String-Konkatenation dynamisch zusammengebaut. Benutzereingaben werden als Query-Parameter ($1, $2, etc.) an Prepared Statements übergeben, wodurch SQL-Injection-Vektoren auf Datenbankebene vollständig eliminiert sind.

### 2. Zentrale Eingabevalidierung (Input Validation)
*   **Mechanismus:** Eingehende API-Payloads (JSON-Requests) werden vor der Verarbeitung im Backend strikt typisiert und validiert.
*   **Umsetzung:** 
    *   Verwendung des Industriestandard-Pakets `github.com/go-playground/validator/v10`.
    *   Alle Request-Structs sind mit deklarativen Validierungs-Tags (z. B. `validate:"required"`, `validate:"email"`, `validate:"min=1"`) versehen.
    *   Der HTTP-Transport-Layer nutzt eine generische Hilfsfunktion `DecodeAndValidate(r *http.Request, dst interface{}) error` in `api/validation.go`, die nicht-konforme Anfragen direkt mit einem `400 Bad Request` abweist, bevor sie die Geschäftslogik erreichen.

### 3. Authentifizierung, Autorisierung und Endpunktsicherung (RBAC)
*   **Mechanismus:** Die Absicherung kritischer Schnittstellen erfolgt im Backend über JWT-basierte Authentifizierungs- und Autorisierungs-Middlewares.
*   **Umsetzung:**
    *   **JWT-Authentifizierung:** Das System nutzt eine zustandslose Authentifizierung via JSON Web Tokens (JWT). Bei erfolgreichem Login wird ein signiertes JWT ausgestellt und als `HttpOnly`-Cookie (mit `SameSite=Lax` oder `Strict` und `Secure` in Produktion) im Browser persistiert, um das Auslesen durch bösartige Clientscripts zu verhindern (Schutz vor Session-Hijacking).
    *   **RBAC-Middleware:** Routen werden im Router (`api/router.go`) durch spezifische Middlewares (z. B. `RequireRole("ADMIN")`) geschützt. Benutzer besitzen Rollen (Admin, Lehrer, Helfer), die granulare Berechtigungen (`permissions`) implizieren.

### 4. Schutz vor Cross-Site Scripting (XSS)
*   **Mechanismus:** Das UI-Framework schützt standardmäßig vor Script-Injections im Browser.
*   **Umsetzung:**
    *   **Auto-Escaping:** Svelte 5 (Runes) führt standardmäßig ein automatisches HTML-Escaping für alle im Template gerenderten Variablen durch.
    *   **Geringe Angriffsfläche:** Der Einsatz der `@html`-Direktive (die ungefilterten HTML-Code rendert) wird in der gesamten Svelte-Codebasis konsequent vermieden bzw. auf statische Systemtexte beschränkt. Benutzereingaben werden niemals unmaskiert via `@html` ausgegeben.

---

## 🔒 Datenschutz und DSGVO-Konformität

### 1. Automatisierte Löschroutinen
Die Applikation führt automatisierte Cronjobs (`jobs/cron.go`) durch, um das Prinzip der Datensparsamkeit sowie rechtliche Löschfristen durchzusetzen. Diese Jobs operieren strikt getrennt von der primären Geschäftslogik.

*   **Ausleihen-Anonymisierung (`RunGDPRAnonymizeLoans`):** 
    Dieser Cronjob entfernt die `bearbeiter_id` von Ausleihen, die vor mehr als 14 Tagen zurückgegeben wurden. Die Identität des ausleihenden Operators wird damit unwiederbringlich gelöscht.
*   **Abgänger-Löschung (`RunGDPRDeleteAbgaenger`):** 
    Dieser Job führt täglich ein Hard-Delete von Schülerdatensätzen (`ist_abgaenger = true`) durch. Voraussetzungen für eine automatische Löschung sind:
    1. Das Abgangsjahr liegt in der Vergangenheit.
    2. Die Karenzzeit von 30 Tagen im neuen Schuljahr ist abgelaufen.
    3. Es existieren keine offenen Buchausleihen.
    4. Es existieren keine unbezahlten Schadensfälle.
    Bei einer Löschung werden historische Ausleihdaten anonymisiert (`schueler_id = NULL`), um Statistiken zu wahren.

### 2. Datenverschlüsselung
Persönliche Bilder von Schülern (`schueler_fotos`) unterliegen besonderen Schutzanforderungen.
*   **Speicherung:** Fotos werden in der Datenbank nicht als Dateipfad oder im Klartext, sondern als verschlüsselter Bytestrom (`foto_encrypted BYTEA`) gespeichert.
*   **Zugriff:** Der Zugriff erfolgt ausschließlich über autorisierte API-Endpoints, die den Bytestrom in-memory verarbeiten (AES-256) und an berechtigte Clients ausliefern.

---

## 🛡️ Netzwerksicherheit & Security Header

Die Applikation wird regelmäßig mit Tools wie OWASP ZAP auf Schwachstellen getestet. Um Angriffsvektoren proaktiv zu begegnen, setzt das Go-Backend (`api/middleware.go`) restriktive Sicherheits-Header:
*   `frame-ancestors 'none';` in der Content-Security-Policy (CSP) verhindert, dass die Applikation in iFrames eingebettet wird (Clickjacking-Schutz).
*   `form-action 'self';` stellt sicher, dass Formulare nur an denselben Origin (unsere eigene API) abgesendet werden können.
*   Eingeschränkte Ausführung von Skripten (`script-src 'self'`) und Laden von Ressourcen (Bilder, Schriften) nur von vertrauenswürdigen Quellen.

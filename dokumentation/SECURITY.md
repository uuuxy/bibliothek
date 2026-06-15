# Sicherheits- und Datenschutzkonzept (DSGVO)

Diese Dokumentation beschreibt die systemweiten Mechanismen zur Wahrung von Sicherheit und Datenschutz.

## 1. Authentifizierung und Autorisierung

Das System nutzt eine zustandslose Authentifizierung via JSON Web Tokens (JWT).
- **JWT Speicherung:** Tokens werden als `HttpOnly`-Cookies (mit `SameSite=Lax` oder `Strict`) persistiert.
- **Role-Based Access Control (RBAC):** Der API-Zugriff wird durch Middleware-Ketten reguliert. Benutzer besitzen Rollen (Admin, Mitarbeiter, Lehrer), die granulare Berechtigungen (`permissions`) implizieren.

## 2. Datenschutzgrundverordnung (DSGVO) und Löschroutinen

Die Applikation führt automatisierte Cronjobs (`jobs/cron.go`) durch, um das Prinzip der Datensparsamkeit sowie rechtliche Löschfristen durchzusetzen. Diese Jobs operieren strikt getrennt von der primären Geschäftslogik.

- **Ausleihen-Anonymisierung (`RunGDPRAnonymizeLoans`):** 
  Dieser Cronjob entfernt die `bearbeiter_id` von Ausleihen, die vor mehr als 14 Tagen zurückgegeben wurden. Die Identität des ausleihenden Operators wird damit unwiederbringlich gelöscht.
- **Abgänger-Löschung (`RunGDPRDeleteAbgaenger`):** 
  Dieser Job führt täglich ein Hard-Delete von Schülerdatensätzen (`ist_abgaenger = true`) durch. Voraussetzungen für eine automatische Löschung sind:
  1. Das Abgangsjahr liegt in der Vergangenheit.
  2. Die Karenzzeit von 30 Tagen im neuen Jahr ist abgelaufen.
  3. Es existieren keine offenen Buchausleihen.
  4. Es existieren keine unbezahlten Schadensfälle.

## 3. Datenverschlüsselung

Persönliche Bilder von Schülern (`schueler_fotos`) unterliegen besonderen Schutzanforderungen.
- **Speicherung:** Fotos werden in der Datenbank nicht als Dateipfad oder im Klartext, sondern als verschlüsselter Bytestrom (`foto_encrypted BYTEA`) gespeichert.
- **Zugriff:** Der Zugriff erfolgt ausschließlich über authorisierte API-Endpoints, die den Bytestrom in-memory verarbeiten und an berechtigte Clients ausliefern.

## 4. Content Security Policy (CSP) & Penetration Testing

Die Applikation wird regelmäßig mit Tools wie OWASP ZAP auf Schwachstellen (wie Clickjacking, XSS) getestet. Um diesen Angriffsvektoren proaktiv zu begegnen, setzt das Go-Backend (`api/middleware.go`) einen restriktiven `Content-Security-Policy`-Header:
- `frame-ancestors 'none';` verhindert, dass die Applikation in bösartigen iFrames eingebettet wird (Clickjacking-Schutz).
- `form-action 'self';` stellt sicher, dass Formulare nur an denselben Origin (unsere eigene API) abgesendet werden können.
- Eingeschränkte Ausführung von Skripten (`script-src 'self'`) und Laden von Ressourcen (Bilder, Schriften) nur von vertrauenswürdigen Quellen (z.B. Google Fonts).

# Installationsanweisung

Diese Dokumentation beschreibt die Umgebungsvariablen und Abhängigkeiten, die für den Betrieb des Bibliothek-Systems erforderlich sind.

## Abhängigkeiten

- **Go:** >= 1.22
- **PostgreSQL:** >= 15
- **Node.js:** >= 20 (für den Build-Prozess des Svelte 5 Frontends)

## Umgebungsvariablen (ENVs)

Die Konfiguration der Applikation erfolgt strikt über Umgebungsvariablen. Für den lokalen Betrieb kann eine `.env`-Datei im Hauptverzeichnis angelegt werden.

| Variable | Datentyp | Beschreibung |
|---|---|---|
| `PORT` | Integer / String | Definiert den Port, auf dem der HTTP-Server lauscht (z.B. `8081`). |
| `COOKIE_SECURE` | Boolean | Steuert das `Secure`-Flag der HTTP-Cookies (`true` im Produktivbetrieb für HTTPS). |
| `DATABASE_URL` | String | Vollständiger PostgreSQL Connection String (z.B. `postgres://user:pass@host:port/dbname`). |
| `JWT_SECRET` | String | Symmetrischer kryptografischer Schlüssel für die Signatur der JSON Web Tokens (mindestens 32 Zeichen). |
| `INITIAL_ADMIN_EMAIL` | String | E-Mail-Adresse für den primären Systemadministrator (nur relevant beim initialen Bootstrapping einer leeren Datenbank). |
| `INITIAL_ADMIN_PASSWORD` | String | Klartext-Passwort für den primären Systemadministrator (wird bei der Erstellung kryptografisch gehasht). |

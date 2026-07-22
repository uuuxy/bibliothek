# Bibliothek – Schulbibliotheks-Software

Eine webbasierte Verwaltungssoftware für Schulbibliotheken. Das System unterstützt die Abwicklung von Buch- und Hardware-Ausleihen mittels eines integrierten Barcode-Scanner-Konzepts.

---

## Tech-Stack

| Komponente | Technologie |
|---|---|
| Backend | Go 1.26.4+, `net/http`, `pgx/v5` |
| Frontend | Svelte 5 (Runes), Tailwind CSS, Vite |
| Datenbank | PostgreSQL 15/16 |
| Echtzeit | Server-Sent Events (SSE) |
| Deployment | Docker Compose, Caddy (Reverse Proxy) |

---

## Hauptfunktionen

- **Zentrale Omnibox (Scanner-Dispatcher):** Ein Eingabefeld verarbeitet Barcode-Scans und ordnet Aktionen anhand von Präfixen (`S-` Schüler, `L-` Lehrer, `B-` Buch, `G-` Gerät) zu.
- **Fristenberechnung:** Berücksichtigung von LMF-Büchern (Stichtag 31. Juli), Sonderbeständen (CDs, DVDs, Hörbücher) und Ferien-Leseclub.
- **Audit-Trail:** Append-Only-Ereignisprotokollierung für administrative Aktionen.
- **Datenschutz-Funktionen:** Löschroutinen für Schulabgänger, AES-256-Verschlüsselung für Schülerfotos.
- **LUSD-Schnittstelle:** Import von Schülerdaten aus dem LUSD-System.
- **Hardware-Verwaltung:** Ausleihe von Laptops/Tablets inklusive Zubehör-Checklisten.
- **Druck-Center:** Erstellung von Barcode-Etiketten und Schülerausweisen.
- **Rollenbasierte Zugriffskontrolle (RBAC):** Rollen für Admin, Lehrer (konfigurierbare Rechte) und Mitarbeiter.

---

## Dokumentation

| Dokument | Inhalt |
|---|---|
| [FACHKONZEPT.md](FACHKONZEPT.md) | Vollständige fachliche Feature-Spezifikation (Ausleihregeln, Mahnwesen, Vormerkungen, DSGVO, RBAC, Katalog …) |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Schichtenarchitektur, Concurrency-Modell, Datenbankdesign, Frontend |
| [SECURITY.md](SECURITY.md) | Sicherheitskonzept, DSGVO, Schutzmaßnahmen |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Produktions-Deployment, Umgebungsvariablen, Caddy, Backups |
| [SCRIPTS.md](SCRIPTS.md) | CLI-Werkzeuge und Migrationen |
| [CHANGELOG.md](CHANGELOG.md) | Änderungshistorie |
| [invarianten.md](invarianten.md) | Invarianten-Katalog: was immer gelten muss und auf welcher Ebene es durchgesetzt ist |
| [resilience_and_recovery.md](resilience_and_recovery.md) | Backup (verschlüsselt + manuell), Restore-Probe, Notfall-Wiederherstellung, Cronjob-Einrichtung |
| [runbook_sekretariat.md](runbook_sekretariat.md) | Erste-Hilfe-Runbook fürs Ausleih-Pult |
| [master_fahrplan.md](master_fahrplan.md) | Status-Dokument: erledigt / offen / Parkdeck |
| [api_inventar.md](api_inventar.md) | Generiertes Routen-Inventar (`scripts/api_inventar.sh`) |
| [archive/](archive/) | Abgeschlossene Migrations-Doku (MySQL → PostgreSQL) |

---

## Schnellstart (lokal)

### Voraussetzungen
- Go 1.26.4+
- Node.js (npm)
- PostgreSQL (lokal oder via Docker)

### Mit Docker
```bash
docker compose -f docker-compose.local.yml up -d
```
Backend: `http://localhost:8084` · DB: `localhost:5434`

### Manuell

**1. Umgebungsvariablen**
```bash
cp .env.example .env
# DATABASE_URL, JWT_SECRET (≥32 Zeichen), APP_ENCRYPTION_KEY (32 Bytes) anpassen
```

**2. Backend starten**
```bash
go run main.go
# Führt Datenbank-Migrationen automatisch aus
```

**3. Frontend starten**
```bash
cd frontend
npm install
npm run dev
# → http://localhost:5173
```

---

## Systemarchitektur (Kurzübersicht)

```
Middleware (Rate-Limit → Auth → CSRF → RBAC)
        │
        ▼
Handler (api/) → Service (internal/service/) → Repository (repository/)
        │                                               │
        ▼                                               ▼
SSE Broker (Echtzeit)                         PostgreSQL (pgx/v5)
```

Details: [ARCHITECTURE.md](ARCHITECTURE.md)

---

## Sicherheit

- JWT HMAC-only (kein `alg=none`)
- Brute-Force-Schutz: `email|ip`-Composite-Key
- CSRF: Double-Submit Cookie
- AES-256-GCM für Schülerfotos
- SMTP mit TLS-Zertifikatsprüfung
- CSV-Formel-Injection-Schutz (OWASP CWE-1236)
- Decompression-Bomb-Guard bei Bild-Uploads
- Produktions-Secret-Guard (Server startet nicht mit Default-Secrets)

Details: [SECURITY.md](SECURITY.md)

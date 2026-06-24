# Bibliothek – Moderne Schulbibliotheks-Software

Eine moderne, webbasierte Verwaltungssoftware für Schulbibliotheken. Das System ist speziell auf den Schulalltag optimiert und ermöglicht eine hocheffiziente Abwicklung von Buch- und Hardware-Ausleihen durch ein integriertes Barcode-Scanner-Konzept.

---

## 🛠️ Tech-Stack

| Komponente | Technologie |
|---|---|
| Backend | Go 1.22+, `net/http`, `pgx/v5` |
| Frontend | Svelte 5 (Runes), Tailwind CSS, Vite |
| Datenbank | PostgreSQL 15/16 |
| Echtzeit | Server-Sent Events (SSE) |
| Deployment | Docker Compose, Caddy (Reverse Proxy) |

---

## ✨ Hauptfunktionen

- **Zentrale Omnibox (Scanner-Dispatcher):** Ein einziges Eingabefeld verarbeitet alle Barcode-Scans. Das System erkennt anhand von Präfixen (`S-` Schüler, `L-` Lehrer, `B-` Buch, `G-` Gerät) vollautomatisch die richtige Aktion.
- **Automatische Fristenberechnung:** LMF-Bücher (jährlicher Stichtag 31. Juli), Sonderbestände (CDs, DVDs, Hörbücher), Ferien-Leseclub.
- **Revisionssicherer Audit-Trail:** Append-Only-Ereignisprotokollierung für alle administrativen Aktionen.
- **DSGVO-Compliance:** Automatisierte Löschroutinen für Schulabgänger, AES-256-verschlüsselte Schülerfotos.
- **LUSD-Schnittstelle:** Import von Schülerdaten aus dem hessischen LUSD-System.
- **Hardware-Verwaltung:** Ausleihe von Laptops/Tablets mit Zubehör-Checklisten.
- **Druck-Center:** Barcode-Etiketten und Schülerausweise.
- **Feingranulares RBAC:** Admin / Lehrer (konfigurierbare Rechte) / Mitarbeiter.

---

## 📚 Dokumentation

| Dokument | Inhalt |
|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | Schichtenarchitektur, Concurrency-Modell, Datenbankdesign, Frontend |
| [SECURITY.md](SECURITY.md) | Sicherheitskonzept, DSGVO, alle Schutzmaßnahmen |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Produktions-Deployment, Umgebungsvariablen, Caddy, Backups |
| [INSTALL.md](INSTALL.md) | Lokales Setup |
| [SCRIPTS.md](SCRIPTS.md) | CLI-Werkzeuge und Migrationen |
| [CHANGELOG.md](CHANGELOG.md) | Änderungshistorie |
| [MYSQL_TO_POSTGRES_MIGRATION.md](MYSQL_TO_POSTGRES_MIGRATION.md) | Altdaten-Migration von MySQL |

---

## 💻 Schnellstart (lokal)

### Voraussetzungen
- Go 1.22+
- Node.js (npm)
- PostgreSQL (lokal oder via Docker)

### Mit Docker (empfohlen)
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

## 🏛️ Systemarchitektur (Kurzübersicht)

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

## 🔒 Sicherheit

- JWT HMAC-only (kein `alg=none`)
- Brute-Force-Schutz: `email|ip`-Composite-Key
- CSRF: Double-Submit Cookie
- AES-256-GCM für Schülerfotos
- SMTP mit TLS-Zertifikatsprüfung
- CSV-Formel-Injection-Schutz (OWASP CWE-1236)
- Decompression-Bomb-Guard bei Bild-Uploads
- Produktions-Secret-Guard (Server startet nicht mit Default-Secrets)

Details: [SECURITY.md](SECURITY.md)

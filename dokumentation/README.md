# Bibliothek – Moderne Schulbibliotheks-Software

Eine moderne, webbasierte Verwaltungssoftware für Schulbibliotheken. Das System ist speziell auf den Schulalltag optimiert und ermöglicht eine hocheffiziente Abwicklung von Buch- und Hardware-Ausleihen durch ein integriertes Barcode-Scanner-Konzept.

---

## 🛠️ Tech-Stack & Systemübersicht

Das System besteht aus zwei Hauptkomponenten und einer relationalen Datenbank:

*   **Go Backend:** Robuste, hochperformante Service-Architektur (Go 1.22+), kompiliert als binäres Executable.
*   **Svelte 5 Frontend:** Moderne Single-Page-Application (SPA), kompiliert als statisches Web-Asset unter Verwendung von Tailwind CSS für das UI-Design und Svelte-Runes für reaktives State-Management.
*   **PostgreSQL:** Relationale Datenbank zur sicheren, transaktionsbasierten Datenhaltung.
*   **SSE (Server-Sent Events):** Echtzeit-Übertragung von Scan-Events und Statusänderungen an den Client.

---

## ✨ Hauptfunktionen (Features)

*   **Zentrale Omnibox (Scanner-Dispatcher):** Ein einziges Eingabefeld verarbeitet alle Barcode-Scans. Das System erkennt anhand von Präfixen vollautomatisch, ob es sich um einen Schüler (`S-`), eine Lehrkraft (`L-`), ein Buch (`B-`) oder ein Hardware-Gerät (`G-`) handelt und führt die entsprechende Aktion (Ausleihe, Rückgabe, Profilaufruf) aus.
*   **Automatische Fristenberechnung:**
    *   Spezielle Leihfristen für Schulbücher der Lernmittelfreiheit (**LMF-Bücher**) mit jährlichem Stichtag (31. Juli).
    *   Verkürzte Ausleihfristen für Sonderbestände wie audiovisuelle Medien (CDs, DVDs, Hörbücher).
    *   Flexible Überschreibung für Sonderaktionen wie den **Ferien-Leseclub**.
*   **Revisionssicherer Audit-Trail & DSGVO:**
    *   Append-Only-Ereignisprotokollierung für administrative Aktionen und Buchbewegungen.
    *   Datenschutzkonforme Löschroutinen für Schulabgänger unter automatischer Anonymisierung der historischen Ausleihdaten.
*   **LUSD-Schnittstelle:** Automatisierter Abgleich und Import von Schülerdaten aus dem hessischen LUSD-System zur Pflege von Klassenlisten und Erkennung von Schulabgängern.
*   **Hardware- und Geräteverwaltung:** Ausleihe von Laptops, Tablets und Zubehör mit interaktiven Zubehör-Checklisten vor der Übergabe.
*   **Druck-Center:** Einfache Generierung und Druck von Barcode-Etiketten für Bücher sowie Schülerausweisen.
*   **Feingranulares Rechtesystem (RBAC):** Rollenbasierte Zugriffskontrolle mit separaten Berechtigungen für Administratoren, Lehrkräfte und Helfer.

---

## 🏛️ Systemarchitektur

### 1. Go Backend Architektur
Die Backend-Architektur folgt dem Pattern der sauberen Schichtentrennung (Layered Architecture):

*   **Router/Handler (`api/`):** Verantwortlich für das HTTP-Routing, Parsen von Requests, Validierung von Parametern und Rückgabe von JSON-Antworten. Beinhaltet die Middleware-Kette (RBAC, Rate Limiting, CSRF, Security Headers).
*   **Service (`internal/service/`):** Beinhaltet die fachliche Geschäftslogik. Die Services orchestrieren Aufrufe an mehrere Repositories und steuern systemübergreifende Prozesse (z. B. PDF-Generierung, E-Mail-Versand, Event-Dispatching).
*   **Repository (`repository/`):** Die Datenzugriffsschicht. Hier befinden sich ausschließlich SQL-Statements (PostgreSQL) und Mapping-Logiken von Datenbankzeilen auf Go-Structs. Genutzt wird `pgx` für typsicheres und performantes Connection Pooling.

### 2. Svelte 5 Frontend Konzept
*   **State Management (Runes):** Die Zustandsverwaltung erfolgt lokal über Svelte 5 Runes (`$state`, `$derived`, `$effect`). Auf globale State-Container (wie Redux) wird verzichtet.
*   **Trennung von UI und State:** Komplexe Ansichten werden in separate Komponenten zerlegt (z. B. Modale, Tabellen, Formulare). Daten werden über Props übergeben, während Events von Kind- an Elternkomponenten delegiert werden.

### 3. Datenbankarchitektur (PostgreSQL)
Die relationale PostgreSQL-Datenbank normalisiert Entitäten. Das primäre Strukturprinzip unterscheidet strikt zwischen Katalogdaten und physischen Beständen:
*   **`buecher_titel`:** Speichert Metadaten auf Katalogebene (ISBN, Titel, Autor, Verlag).
*   **`buecher_exemplare`:** Speichert spezifische physische Instanzen, referenziert einen `buecher_titel` via `titel_id`. Beinhaltet Felder für `barcode_id`, `zustand_notiz` und `ist_ausleihbar`.
*   **`schueler_fotos`:** Profilbilder werden aus Datenschutzgründen als verschlüsselter Bytestrom (`BYTEA`) gespeichert.

---

## 💻 Lokales Setup

### Voraussetzungen
*   **Go** (Version 1.22 oder neuer)
*   **Node.js** (inklusive `npm`)
*   **PostgreSQL** (lokal oder via Docker)

### 1. Umgebungsvariablen einrichten
Kopiere die `.env.example` im Hauptverzeichnis nach `.env` und passe die Werte an deine lokale Umgebung an (insbesondere die Datenbankverbindung und den Verschlüsselungsschlüssel):

```bash
cp .env.example .env
```

*Hinweis: Der `APP_ENCRYPTION_KEY` muss genau 32 Zeichen (oder 64 Hex-Zeichen) lang sein.*

### 2. Backend starten
Das Go-Backend führt beim Start automatisch alle ausstehenden Datenbank-Migrations aus und legt bei einer leeren Datenbank einen Standard-Admin-Benutzer an.

```bash
# Im Hauptverzeichnis ausführen:
go run main.go
```

Das API-Backend läuft standardmäßig unter `http://localhost:8081` (bzw. dem in der `.env` konfigurierten Port).

### 3. Frontend starten
Navigiere in den Frontend-Ordner, installiere die Node-Pakete und starte den Entwicklungsserver:

```bash
cd frontend
npm install
npm run dev
```

Die Benutzeroberfläche ist anschließend im Browser unter `http://localhost:5173` (bzw. dem von Vite ausgegebenen Port) erreichbar.

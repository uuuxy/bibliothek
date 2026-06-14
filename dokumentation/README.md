# Systemarchitektur: Bibliothek

Diese Dokumentation beschreibt die Architektur und technischen Komponenten des Bibliothek-Verwaltungssystems.

## 1. Systemübersicht

Das System besteht aus zwei Hauptkomponenten:
- **Backend:** Go (Version 1.22+), kompiliert als binäres Executable.
- **Frontend:** Svelte 5, kompiliert als statische Single-Page Application (SPA).

## 2. Go Backend Architektur

Die Backend-Architektur folgt dem Pattern der sauberen Schichtentrennung. Die Verantwortlichkeiten sind strikt getrennt:

- **Router/Handler (`api/`):** Verantwortlich für das HTTP-Routing, Parsen von Requests, Validierung von Parametern und Rückgabe von JSON-Antworten. Beinhaltet die Middleware-Kette (RBAC, Rate Limiting, CSRF, Security Headers).
- **Service (`services/` / `api/`):** Beinhaltet die fachliche Geschäftslogik. Die Services orchestrieren Aufrufe an mehrere Repositories und externe Systeme (z.B. PDF-Generierung, E-Mail-Versand, API-Aufrufe).
- **Repository (`repository/`):** Die Datenzugriffsschicht. Hier befinden sich ausschließlich SQL-Statements (PostgreSQL) und Mapping-Logiken von Datenbankzeilen auf Go-Structs. Genutzt wird `pgx` für typsicheres und performantes Connection Pooling.

## 3. Svelte 5 Frontend Konzept

Das Frontend nutzt die Svelte 5 Architektur.
- **State Management (Runes):** Die Zustandsverwaltung erfolgt lokal über Svelte 5 Runes (`$state`, `$derived`, `$effect`). Auf globale State-Container (wie Redux) wird verzichtet.
- **Trennung von UI und State:** Komplexe Ansichten werden in separate Komponenten zerlegt (z.B. Modale, Tabellen, Formulare). Daten werden über Props übergeben, während Events von Kind- an Elternkomponenten delegiert werden.

## 4. Datenbankarchitektur (PostgreSQL)

Die relationale PostgreSQL-Datenbank normalisiert Entitäten. Das primäre Strukturprinzip unterscheidet strikt zwischen Katalogdaten und physischen Beständen:

- **`buecher_titel`:** Speichert Metadaten auf Katalogebene (ISBN, Titel, Autor, Verlag).
- **`buecher_exemplare`:** Speichert spezifische physische Instanzen, referenziert einen `buecher_titel` via `titel_id`. Beinhaltet felder für `barcode_id`, `zustand`, und `status`.

Zusätzlich speichert die Tabelle `schueler_fotos` Profilbilder aus Datenschutzgründen als verschlüsselten Bytestrom (`BYTEA`).

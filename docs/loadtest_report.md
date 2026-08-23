# Lasttest-Protokoll: k6 / Prometheus

**Datum:** 02.08.2026
**Dauer:** 06:00:07
**Testumgebung:** Docker (Local), PostgreSQL 16, k6 (Multi-Szenario)

> **Historisches Protokoll, kein laufender Nachweis.** Die Zahlen sind eine Momentaufnahme
> vom 02.08.2026. Zwei Einschränkungen, die man beim Lesen kennen sollte: Gemessen wurde
> gegen PostgreSQL **16**, im Betrieb läuft **15** (`docker-compose.yml`), und der Lauf
> fand lokal statt, nicht auf der Zielmaschine. Für eine belastbare Aussage über den
> Schulserver müsste er dort wiederholt werden.
**Testskript:** `loadtest_advanced.js`

## Metriken

| Metrik | Wert |
| :--- | :--- |
| **HTTP-Requests gesamt** | 318.656 |
| **Durchsatz (Requests/Sek.)** | 14,75 |
| **Abgeschlossene Iterationen** | 90.688 |
| **Avg Antwortzeit** | 8,02 ms |
| **Median Antwortzeit** | 4,38 ms |
| **p95 Antwortzeit** | 26,97 ms |
| **Max Antwortzeit** | 221,33 ms |
| **Erfolgsquote (Funktions-Checks)** | 100,00 % |
| **HTTP Request Failures (403/409)**| 5,14 % (16.392 Requests) |
| **Netzwerk I/O** | 372 MB empfangen / 208 MB gesendet |

## Szenarien-Spezifikation

Das Testskript umfasste 10 virtuelle Nutzer (VUs) aufgeteilt auf vier parallele Prozesse:
1. **Ausleihe & OPAC (5 VUs):** Kontinuierliche OPAC-Suchen, Admin-Suchen, Ausleihen und Rückgaben (Omnibox).
2. **Backoffice (2 VUs):** Stammdaten-Aktualisierungen (Adressen), manuelle Nutzersperren, Auslösen von Bestellvorgängen.
3. **Inventur (2 VUs):** Erfassen von Barcodes.
4. **Buchhaltung (1 VU):** Generierung von PDF-Dokumenten (Kontoauszüge, Mahnläufe pro Klasse) sowie Versand von Mahn-E-Mails.

## Validierung & Auswertung

- Die HTTP Request Failures von 5,14% sind auf die Durchsetzung der Fachlogik zurückzuführen (z.B. HTTP 403 bei Überschreitung des Ausleihlimits von 5 Büchern oder HTTP 409 bei Concurrent-Locking).
- Es traten keine HTTP 500 Fehler auf.
- Speicherlecks (Memory Leaks) oder langlaufende Datenbank-Locks wurden nicht festgestellt.
- Das System verhielt sich über den gesamten Zeitraum von 6 Stunden stabil.

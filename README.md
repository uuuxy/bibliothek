# Bibliothek — Schulbibliotheks-Software

Verwaltung einer Schulbibliothek: Ausleihe am Scanner-Tresen, Medienkatalog, Mahnwesen,
Inventur, Bestellwesen und Schülerdatei. Entstanden als Ersatz für eine
Windows-Altanwendung und im Betrieb an einer Gesamtschule — mit allem, was das mit sich
bringt: gewachsener Altbestand, Barcodes, die nicht neu geklebt werden können, und
Schülerdaten, die dem Datenschutz unterliegen.

Das ist kein Produkt und keine generische Bibliothekssoftware. Es ist für **einen**
konkreten Betrieb gebaut, und die Entscheidungen darin sind entsprechend konkret.

---

## Was es kann

- **Zentrale Scanner-Omnibox** — ein Eingabefeld für alle Barcodes. Ohne Präfix wird in
  der Reihenfolge Buch → Schülerausweis → Lehrerausweis → Volltextsuche aufgelöst; die
  Ausweise des Altbestands tragen nackte Nummern und dürfen nicht neu gedruckt werden.
- **Fristenberechnung** mit Lernmittelfreiheit (fester Stichtag 31. Juli), Sonderbeständen
  und Ferienlogik.
- **Mahnwesen** — Mahnstufe steigt ausschließlich beim PDF-Druck (dem physischen
  Verwaltungsakt), nie beim Mailversand. Mahnlisten gehen an die Klassenleitung, nie an
  Schüler.
- **Vormerkungen und Klassensatz-Reservierungen**, inklusive eines eigenen Portals fürs
  Kollegium.
- **Inventur** — sitzungsgebunden, damit parallele Zählungen sich nicht überschreiben.
- **Bestellwesen** — Bedarfsvorschläge aus dem Bestand, Bestellmail samt Barcodebogen,
  Wareneingang, und für Händler, die selbst etikettieren, ein Bestätigungslink.
- **Druck-Center** für Etiketten und Schülerausweise.
- **Geräteausleihe** (Laptops/Tablets) mit Zubehör-Checklisten.
- **Datenschutz** — Löschroutinen für Abgänger, verschlüsselte Schülerfotos, Audit-Trail.

Die fachliche Spezifikation steht vollständig in [docs/FACHKONZEPT.md](docs/FACHKONZEPT.md).

---

## Technik

| | |
|---|---|
| Backend | Go 1.26, `net/http` mit Methoden-Routing, pgx |
| Datenbank | PostgreSQL, 89 nummerierte Migrationen |
| Frontend | Svelte 5 (Runes), Tailwind 4, Vite — kein TypeScript |
| Anmeldung | IMAP gegen den Schul-Mailserver; es wird **kein** Benutzerpasswort gespeichert |
| Betrieb | Docker Compose hinter Caddy |
| Lizenz | [EUPL-1.2](LICENSE) |

Umfang, gemessen am 23.08.2026: rund 50.000 Zeilen Go im Produktivcode, dazu 47.000
Zeilen in 337 Testdateien; etwa 33.000 Zeilen Svelte/JavaScript und 70 e2e-Dateien.

Diese Zahlen altern. Die vorige Fassung stand auf dem Stand vom Juli und lag bei den
Testzeilen um 47 % daneben — deshalb steht hier das Messdatum und darunter der Befehl,
mit dem man sie in zehn Sekunden neu erhebt, statt einer gepflegten Behauptung:

```bash
ls migrations/*.sql | wc -l
find . -name '*.go' -not -name '*_test.go' -not -path './node_modules/*' | xargs cat | wc -l
find . -name '*_test.go' -not -path './node_modules/*' | wc -l
```

---

## Schnellstart

```bash
cp .env.example .env          # DATABASE_URL, JWT_SECRET (≥32 Zeichen),
                              # APP_ENCRYPTION_KEY (32 Byte) setzen
docker compose -f docker-compose.local.yml up -d
```

Anwendung: `http://localhost:8084` · Datenbank: `localhost:5434`. Die Migrationen laufen
beim Start automatisch.

**Frontend mit Hot Reload** (optional, gegen denselben Stack):

```bash
cd frontend && npm ci && npm run dev     # → http://localhost:5173
```

Der Entwicklungs-Server reicht `/api`, `/login`, `/uploads` und `/events` an
`127.0.0.1:8084` durch. Läuft das Backend woanders — etwa von Hand mit dem `PORT` aus
`.env.example` —, dann:

```bash
VITE_API_TARGET=http://127.0.0.1:8081 npm run dev
```

Anmelden geht nur mit einem Postfach, das der konfigurierte IMAP-Server kennt. Für
Entwicklung und Tests akzeptiert `IMAP_HOST=mock` jedes Passwort.

---

## Qualitätssicherung

Der Anspruch ist nicht „es gibt Tests", sondern: **Jedes Gate muss man einmal rot gesehen
haben.** Ein Gate, das seine Aussage nicht verlieren kann, prüft nichts.

- **`scripts/install-hooks.sh`** installiert zwei Hooks: pre-commit prüft Formatierung,
  ESLint und `golangci-lint`; pre-push fährt sechs Stufen — Go-Tests, `svelte-check`,
  `npm audit`, `govulncheck`, Trivy und `deadcode`. Der pre-push-Hook meldet außerdem, was
  er **nicht** geprüft hat: Die DB-Integrationstests überspringen sich ohne
  `TEST_DATABASE_URL` stillschweigend, mit einem grünen „ok" daneben.
- **DB-Integrationstests gegen echtes PostgreSQL** (`*_pg_test.go`, gated auf
  `TEST_DATABASE_URL`): Constraints kann man nicht mocken.
- **e2e mit Playwright** über den fertig gebauten Container — nicht gegen einen Dev-Server,
  damit gemessen wird, was ausgeliefert wird.
- **CI** ergänzt CodeQL, `govulncheck`, `gosec`, `npm audit` und einen Trivy-Scan des
  Images samt Container-Smoke (läuft unprivilegiert? sind die Laufzeitwerkzeuge da? ist
  jedes Volume beschreibbar?).

---

## Dokumentation

Alles Weitere liegt in [`docs/`](docs/README.md) — Architektur, Sicherheits- und
DSGVO-Konzept, Deployment, Invarianten-Katalog, CLI-Werkzeuge und ein generiertes
Verzeichnis aller API-Routen.

> Die Commit-Historie ist Teil der Dokumentation. Sie erklärt bei den meisten
> Entscheidungen das *Warum* ausführlicher als jede gepflegte Liste — und sie kann nicht
> veralten.

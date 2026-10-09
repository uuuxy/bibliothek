# Bibliothek — Schulbibliotheks-Software

Verwaltung einer Schulbibliothek: Ausleihe am Scanner-Tresen, Medienkatalog, Mahnwesen,
Inventur, Bestellwesen und Leserdatei. Entstanden als Ersatz für eine
Windows-Altanwendung und im Betrieb an einer Gesamtschule — mit allem, was das mit sich
bringt: gewachsener Altbestand, Barcodes, die nicht neu geklebt werden können, und
Schülerdaten, die dem Datenschutz unterliegen.

Das ist kein Produkt und keine generische Bibliothekssoftware. Es ist für **einen**
konkreten Betrieb gebaut, und die Entscheidungen darin sind entsprechend konkret.

---

## Was es kann

- **Zentrale Scanner-Omnibox** — ein Eingabefeld für alle Barcodes. Ohne Präfix wird in
  der Reihenfolge Buch → Ausweis → Volltextsuche aufgelöst; die Ausweise des Altbestands
  tragen nackte Nummern und dürfen nicht neu gedruckt werden.
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
- **Druck-Center** für Etiketten und Ausweise. Die Aufschrift der Karte richtet sich nach
  der Art des Lesers — „Schülerausweis" oder „Lehrerausweis".
- **Geräteausleihe** (Laptops/Tablets) mit Zubehör-Checklisten.
- **Datenschutz** — Löschroutinen für Abgänger, verschlüsselte Schülerfotos, Audit-Trail.
- **Öffentliche Seiten ohne Anmeldung** — Katalog (`/katalog`) für Schüler und Eltern mit
  Cover und Verfügbarkeit, Bibliotheks-Monitor (`/monitor`) als Endlos-Slideshow für den
  Bildschirm im Flur (Buch des Monats, Neuzugänge, Beliebt diese Woche). Beide liefern nur
  Titeldaten, nie Ausleiher.
- **Kollegium** — eigenes Portal, Selbstanmeldung mit dem Schul-Postfach, Klassensatz-
  Reservierungen und Meldungen an die Bibliothek. Das ist der Grundzustand
  jeder Lehrkraft und keine vergebene Rolle; Rollen (Leitung, Mitarbeiter, Helfer, Admin)
  erhebt der Admin an der E-Mail-Adresse.
- **Leserdatei** — Schüler und Kollegium in einer Liste. Schüler kommen aus der LUSD,
  Lehrkräfte und LiV über die Selbstanmeldung oder von Hand; wer wer ist, steht als Art
  in der Akte und entscheidet keine Rechte.
- **Statistiken ohne Klarnamen** — Zirkulation, Wiederbeschaffungswert, Renner und Ladenhüter.
- **Selbstprüfung der Betriebsbereitschaft** — was ist eingerichtet, aber nicht in Betrieb?
- **Barrierefreiheit** — auf WCAG 2.1 AA gebaut und per Browser-Gate gemessen (axe über den Anfangszustand aller Hauptansichten, Fokusfalle, Tabellen, Bewegung); Tastaturbedienung im [Handbuch](docs/HANDBUCH.md), Umfang, Grenzen und bekannte Lücken in [FACHKONZEPT.md §19](docs/FACHKONZEPT.md).

Die fachliche Spezifikation steht vollständig in [docs/FACHKONZEPT.md](docs/FACHKONZEPT.md);
für Bibliothekspersonal gibt es das [Benutzerhandbuch](docs/HANDBUCH.md).

---

## Technik

| | |
|---|---|
| Backend | Go 1.27, `net/http` mit Methoden-Routing, pgx |
| Datenbank | PostgreSQL, nummerierte Migrationen (Zählbefehl unten) |
| Frontend | Svelte 5 (Runes), Tailwind 4, Vite — kein TypeScript |
| Anmeldung | IMAP gegen den Schul-Mailserver; es wird **kein** Benutzerpasswort gespeichert |
| Betrieb | Docker Compose hinter Caddy |
| Lizenz | [EUPL-1.2](LICENSE) |

Umfang, gemessen am 08.10.2026: rund 76.300 Zeilen Go im Produktivcode, dazu 118.200
Zeilen in 811 Testdateien; etwa 83.100 Zeilen Svelte/JavaScript und 161 e2e-Dateien. Die
genauen Zahlen und alle Messbefehle stehen in [Architektur, Kapitel 1.4](docs/ARCHITEKTUR.md#1-einführung-und-ziele).

Diese Zahlen altern. Die vorige Fassung stand auf dem Stand vom Juli und lag bei den
Testzeilen um 47 % daneben — deshalb steht hier das Messdatum und darunter der Befehl,
mit dem man sie in zehn Sekunden neu erhebt, statt einer gepflegten Behauptung:

```bash
ls migrations/*.sql | wc -l
find . -name '*.go' -not -name '*_test.go' -not -path '*/node_modules/*' \
     -not -path './docs/docs.go' | xargs cat | wc -l
find . -name '*_test.go' -not -path '*/node_modules/*' | wc -l
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
  ESLint und `golangci-lint`; pre-push fährt neun Stufen — Go-Tests, `golangci-lint`,
  `svelte-check`, Vitest, `npm audit`, `govulncheck`, `gosec`, Trivy und `deadcode`. Der pre-push-Hook meldet außerdem, was
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

Alles Weitere liegt in `docs/`.

**Wo finde ich was?** — nach Frage, nicht nach Dateiname:

| Ich will …                                                                                                                      | Dokument                                                                                                                         |
| ------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| wissen, was **offen** ist — zu tun, zu prüfen, zu entscheiden — und in welcher Reihenfolge                                      | [OFFEN.md](docs/OFFEN.md) — nur der erste Block                                                                                       |
| das System bedienen (Theke, Leserdatei, Mahnwesen, Einstellungen)                                                               | [HANDBUCH.md](docs/HANDBUCH.md)                                                                                                       |
| den **LUSD-Import** verstehen oder fahren: Spalten für das Sekretariat, Umbenennung ohne Schüler-ID, Karenzzeit, Zusammenführen | [LUSD.md](docs/LUSD.md)                                                                                                               |
| wissen, welche fachliche Regel gilt (Fristen, Vormerkung, DSGVO, Rollen)                                                        | [FACHKONZEPT.md](docs/FACHKONZEPT.md)                                                                                                 |
| das System betreiben, deployen, sichern, wiederherstellen                                                                       | [DEPLOYMENT.md](docs/DEPLOYMENT.md), [resilience_and_recovery.md](docs/resilience_and_recovery.md), [SCRIPTS.md](docs/SCRIPTS.md)               |
| wissen, wer das Programm pflegt — oder als Vertretung ein Update einspielen oder eine Sicherung zurückholen                     | [PFLEGEKONZEPT.md](docs/PFLEGEKONZEPT.md)                                                                                             |
| etwas abnehmen (LUSD, Versetzung, Klassensatz)                                                                                  | [abnahme_checkliste.md](docs/abnahme_checkliste.md)                                                                                   |
| Datenschutz beurteilen (welche Daten, welche Fristen, welche Rechte)                                                            | [datenschutz/nachweis.md](docs/datenschutz/nachweis.md) zum Weitergeben; im Einzelnen [SECURITY.md](docs/SECURITY.md), [PII_MATRIX.de.md](docs/PII_MATRIX.de.md), [datenschutz/](docs/datenschutz/) |
| Barrierefreiheit beurteilen (was die Gates prüfen, was offen ist)                                                               | [FACHKONZEPT.md §19](docs/FACHKONZEPT.md), [HANDBUCH.md](docs/HANDBUCH.md) „Bedienung ohne Maus"                                           |
| am Code arbeiten                                                                                                                | [ARCHITEKTUR.md, Kapitel 5](docs/ARCHITEKTUR.md#5-bausteinsicht), [invarianten.md](docs/invarianten.md), [api_inventar.md](docs/api_inventar.md), [sweeps.md](docs/sweeps.md) |
| die **Architektur** verstehen oder beurteilen — Ziele, Kontext, Bausteine, Laufzeit, Verteilung, Entscheidungen, Risiken        | [ARCHITEKTUR.md](docs/ARCHITEKTUR.md) — die vollständige Architekturdokumentation in zwölf Kapiteln                                          |

### Bedienen und fachlich verstehen

| Dokument                                                       | Inhalt                                                                                                                                                                                                                  |
| -------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [HANDBUCH.md](docs/HANDBUCH.md)                                     | Benutzerhandbuch — jeder Bereich aus Sicht der Bibliothek, mit den öffentlichen Seiten und den Einstellungs-Kategorien                                                                                                  |
| [LUSD.md](docs/LUSD.md)                                             | **LUSD-Import ohne Schüler-ID:** was der Bericht enthalten muss, drei Zuordnungsstufen, Umbenennungs-Paarung, Karenzzeit vor der Anonymisierung, Zusammenführen von Hand, Ablauf zum Schuljahreswechsel, Code-Landkarte |
| [FACHKONZEPT.md](docs/FACHKONZEPT.md)                               | Vollständige fachliche Feature-Spezifikation (Ausleihregeln, Mahnwesen, Vormerkungen, DSGVO, RBAC, Katalog …)                                                                                                           |
| [abnahme_checkliste.md](docs/abnahme_checkliste.md)                 | Durchlauf für die manuellen Abnahmen (LUSD, Versetzung, Klassensatz)                                                                                                                                                    |

### Betreiben

| Dokument                                                 | Inhalt                                                                                          |
| -------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| [PFLEGEKONZEPT.md](docs/PFLEGEKONZEPT.md)                     | Pflegekonzept und Wartungshandbuch (Entwurf): Zuständigkeiten, Update und Wiederherstellung für die Vertretung, wiederkehrende Aufgaben, Ende der Pflege; im Anhang die Vorlage für das Blatt bei der Schule |
| [DEPLOYMENT.md](docs/DEPLOYMENT.md)                           | Produktions-Deployment, Umgebungsvariablen, Caddy, Backups                                      |
| [resilience_and_recovery.md](docs/resilience_and_recovery.md) | Backup (verschlüsselt + manuell), Restore-Probe, Notfall-Wiederherstellung, Cronjob-Einrichtung |
| [SCRIPTS.md](docs/SCRIPTS.md)                                 | CLI-Werkzeuge: Littera-Altbestand, Foto-Migration, Backup, Deployment, Lasttest                 |
| [littera_schema_befund.md](docs/littera_schema_befund.md)     | Littera-Altbestand: Schema, Barcodes, Schreibpfad — alle Zahlen gemessen                        |

### Datenschutz und Sicherheit

| Dokument                                                                           | Inhalt                                                                                                                               |
| ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| [datenschutz/nachweis.md](docs/datenschutz/nachweis.md)                                 | Datenschutz-Nachweis (Entwurf) zum Weitergeben an Schulleitung, Datenschutzbeauftragten und Schulträger: Daten, Fristen, der Test hinter jeder Zusage, Ablauf bei einer Datenpanne, bekannte Lücken, was bei der Schule liegt, Rechtsrahmen |
| [SECURITY.md](docs/SECURITY.md)                                                         | Sicherheitskonzept, DSGVO, Schutzmaßnahmen, Löschroutinen                                                                            |
| [PII_MATRIX.de.md](docs/PII_MATRIX.de.md)                                               | Jede Route nach Schülerdaten eingestuft (Stufe 0–3) — vom Gate `api/pii_matrix_test.go` mit dem Code deckungsgleich gehalten         |
| [datenschutz/vvt_entwurf.md](docs/datenschutz/vvt_entwurf.md)                           | Entwurf Verzeichnis von Verarbeitungstätigkeiten — drei Tätigkeiten: Lernmittelausleihe, Schülerbücherei, Konten und Protokoll; TOM-Anhang aus SECURITY.md |
| [datenschutz/datenschutzhinweis_art13.md](docs/datenschutz/datenschutzhinweis_art13.md) | Entwurf Datenschutzhinweis nach Art. 13 DSGVO für Schüler/Eltern — zwei Fassungen (Lernmittel, Schülerbücherei)                      |

### Entwickeln und prüfen

| Dokument                               | Inhalt                                                                                                                                                                                                                                                                                                                                                                                                          |
| -------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [ARCHITEKTUR.md](docs/ARCHITEKTUR.md)       | **arc42-Architekturdokumentation** (12 Kapitel in einer Datei): Ziele und Stakeholder, Randbedingungen, Kontext, Lösungsstrategie, Bausteinsicht, zehn Laufzeitszenarien, Verteilung/CI, querschnittliche Konzepte, 24 Architekturentscheidungen als ADR, Qualitätsszenarien mit Gate, Risiken und Schulden, Glossar. Die frühere Kurzfassung ARCHITECTURE.md ist am 18.09.2026 darin aufgegangen |
| [invarianten.md](docs/invarianten.md)       | Invarianten-Katalog: was immer gelten muss und auf welcher Ebene es durchgesetzt ist                                                                                                                                                                                                                                                                                                                            |
| [sweeps.md](docs/sweeps.md)                 | Die Prüfachsen: Bugklassen, ihre Detektoren und Ratschen — neben dem Raster die Bestands-Achse                                                                                                                                                                                                                                                                                                              |
| [OFFEN.md](docs/OFFEN.md)                   | **Die eine Liste** alles Offenen — Fahrplan, Fehler, Entscheidungen, Betrieb; neue Funde kommen nur hierher                                                                                                                                                                                                                                                                                     |
| [mittel_konzept.md](docs/mittel_konzept.md) | Landes- und Kreismittel: Schadensersatz-Bescheide (Teil A) und getrennte Töpfe in der Beschaffung (Teil B), beide im ersten Schnitt gebaut; was offen bleibt, nennt der Kopf des Dokuments                                                                                                                                                                                                                                                           |
| [api_inventar.md](docs/api_inventar.md)     | **Vollständiges** Routenverzeichnis (generiert): alle Go-Routen, alle Frontend-Aufrufer, Abgleich in beide Richtungen — `./scripts/api_inventar.sh`                                                                                                                                                                                                                                                             |
| `docs.go` (Swagger)                    | Interaktive API-Doku, **nur bei `APP_ENV=local`/`development`** unter `/swagger`. Deckt die **annotierten** Endpunkte ab (am 08.10.2026 88 Operationen auf 75 Pfaden von 230 registrierten Routen) — das vollständige Verzeichnis ist `api_inventar.md`. Neu erzeugen: `swag init -g main.go -o docs`; ein Test (`docs/swagger_drift_test.go`) schlägt fehl, sobald die Datei von den `@Router`-Annotationen abweicht |

> Die Commit-Historie ist Teil der Dokumentation. Sie erklärt bei den meisten
> Entscheidungen das *Warum* ausführlicher als jede gepflegte Liste — und sie kann nicht
> veralten.

# Wegweiser — wo steht was

Stand: 14.10.2026

Eine Seite für den Einstieg in den Code: der Weg einer Anfrage, vom Menüpunkt zum Ordner und
für häufige Änderungen die Stelle. Warum etwas so gebaut ist, steht in
[ARCHITEKTUR.md](ARCHITEKTUR.md); die Fachbegriffe erklärt dort Kapitel 12. Wie etwas im Code
heißt und was ein Kommentar sagt, regelt [STIL.md](STIL.md).

## Der Weg einer Anfrage

Ein Klick in der Oberfläche läuft durch vier Stationen, immer in dieser Reihenfolge:

| Station    | Ordner              | Aufgabe                                                          |
| ---------- | ------------------- | ---------------------------------------------------------------- |
| Oberfläche | `frontend/src/`     | Bildschirme und Bauteile (Svelte); schickt die Anfrage           |
| Tür        | `api/`, je Thema    | nimmt die Anfrage an: prüft Eingabe und Recht, antwortet         |
| Regel      | `internal/service/` | entscheidet, was gilt: Frist, Sperre, Reihenfolge der Schritte   |
| Abfrage    | `repository/`       | liest und schreibt in der Datenbank; in den Türen steht kein SQL |

Daneben stehen: `pdf/` setzt Blätter und Etiketten, `mailservice/` verschickt Mail, `jobs/`
läuft nachts, `auth/` meldet an. Die Tabellen der Datenbank beschreibt `schema.sql`, jede
Änderung daran ist ein Schritt in `migrations/`.

Welche Adresse zu welcher Tür führt und welches Recht sie verlangt, steht in sechs Listen:
`api/routen_katalog.go`, `api/routen_leser.go`, `api/routen_bestellung.go`,
`api/routen_system.go`, `api/routen_oeffentlich.go` und `api/routen_theke.go`. Alle Adressen mit ihren Aufrufern in der
Oberfläche nennt [api_inventar.md](api_inventar.md).

## Vom Menüpunkt zum Code

Die Türen sind nach den Menüpunkten der Oberfläche geordnet. Die genannte Datei trägt am Kopf
die Beschreibung ihres Themas, die übrigen Dateien des Ordners je eine Sache.

| Menüpunkt                               | Bildschirm in `frontend/src/lib/`                                                        | Türen                              |
| --------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------- |
| Ausleihe (Theke)                        | `Omnibox.svelte`                                                                         | `api/theke/theke.go`               |
| Medienkatalog, Klassensätze             | `MediaCatalog.svelte`, `Schulklassen.svelte`                                             | `inventur/api_routen.go`           |
| Buchakte, Signaturen                    | `Buchakte.svelte`, `SignaturenView.svelte`                                               | `api/katalog/katalog.go`           |
| Bestandsbücher                          | `components/bestand/Bestandsbuecher.svelte`                                              | `api/katalog/katalog.go`           |
| Druck-Center                            | `DruckCenter.svelte`                                                                     | `api/druck/druck.go`               |
| Leserdatei, Leserakte                   | `Schuelerdatei.svelte`, `Leserakte.svelte`                                       | `api/leser/leser.go`               |
| Schaden, Forderung, Bescheid            | `DamageReportModal.svelte`, `LeserGebuehrenKarte.svelte`, `LeserBescheideKarte.svelte` | `api/schaden/schaden.go`           |
| Mahnwesen, Abgänger                     | `Mahnwesen.svelte`, `Graduates.svelte`                                                   | `api/mahnwesen/mahnwesen.go`       |
| Bestellungen, Bestellberichte           | `Bestellwesen.svelte`, `components/bestellungen/BestellBerichte.svelte`              | `api/bestellung/bestellung.go`     |
| Inventur                                | `UnifiedInventory.svelte`                                                                | `api/inventurlauf/inventurlauf.go` |
| Schuljahreswechsel (LMF-Plan)           | `LmfPlan.svelte`                                                                         | `api/lernmittel/lernmittel.go`     |
| Benutzer & Rechte                       | `Berechtigungen.svelte`                                                                  | `api/benutzer/benutzer.go`         |
| Einstellungen, Statistiken, System-Logs | `SystemSettings.svelte`, `StatsDashboard.svelte`, `SystemLogs.svelte`                    | `api/system/system.go`             |
| Mein Portal, öffentlicher Katalog       | `KollegiumPortal.svelte`, `OpacSearch.svelte`                                            | `api/portal/portal.go`             |

Der Medienkatalog ist ein eigenes Untermodul: Titelliste, Buchmaske und Import haben ihre
Türen und ihre Abfragen in `inventur/`. Was jede Tür braucht (Eingabe lesen, Recht prüfen,
Protokoll schreiben), steht in `api/tuer/tuer.go`. Welcher Menüpunkt welchen Bildschirm
öffnet, steht in `frontend/src/lib/menu.js` und `frontend/src/lib/Router.svelte`.

## Ich will etwas ändern

| Ich will …                                    | Dort                                                                                                                                |
| --------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| den Text oder das Bild eines Blatts ändern    | `pdf/`, je Blatt eine Datei (`pdf/mahnbrief.go`, `pdf/etikett_buch.go`)                                                             |
| eine Regel der Ausleihe ändern                | `internal/service/frist.go`, `internal/service/ausleih_sperren.go`                                                             |
| eine Abfrage ändern                           | `repository/`, die Datei heißt wie die Sache (`repository/auflagen.go`)                                                             |
| eine Tabelle oder Spalte ergänzen             | ein neuer Schritt in `migrations/` und derselbe Stand in `schema.sql`; `db/migrationen_schema_paritaet_pg_test.go` hält beide gleich |
| eine neue Adresse anlegen                     | die Tür im Ordner ihres Themas, Adresse und Recht in der passenden Liste `api/routen_*.go`                                          |
| festlegen, was eine Rolle ab Werk darf        | `db/rechte_vorgabe.go`                                                                                                                        |
| einen Menüpunkt ändern                        | `frontend/src/lib/menu.js`                                                                                                          |
| eine Kategorie der Einstellungen ändern       | `frontend/src/lib/components/settings/kategorien.js`                                                                                |
| ein gemeinsames Bauteil der Oberfläche ändern | `frontend/src/lib/components/ui/`                                                                                                   |
| Farben und Schriftgrößen ändern               | `frontend/src/styles/rollen.css`, `frontend/src/styles/theme-mass.css`                                                              |
| einen Nachtlauf ändern                        | `jobs/cron.go` nennt die Läufe und ihre Uhrzeit                                                                                     |
| den Abgleich mit der LUSD ändern              | `internal/lusd/`, beschrieben in [LUSD.md](LUSD.md)                                                                                 |
| die Übernahme aus Littera ändern              | `internal/littera/`, beschrieben in [littera_schema_befund.md](littera_schema_befund.md)                                            |

Nach jeder Änderung laufen die Prüfungen aus dem Abschnitt „Qualitätssicherung" der
[README](../README.md). Was offen ist, steht nur in [OFFEN.md](OFFEN.md).

## Zwei Wörter für dieselbe Sache

Ältere Dateien tragen englische Namen, neuere deutsche. Das deutsche Wort gilt
([STIL.md](STIL.md), Abschnitt 2); umgestellt wird Paket für Paket. Bis dahin sucht nach beiden
Wörtern, wer alles zu einer Sache sucht (gezählt am 14.10.2026 an den Namen von 1.051 Dateien
in Go und in der Oberfläche, ohne Tests):

| Sache                          | deutsch im Namen         | englisch im Namen     |
| ------------------------------ | ------------------------ | --------------------- |
| Schüler, Leser                 | `schueler`, `leser` (39) | `student` (39)        |
| Buch, Titel                    | `buch`, `titel` (64)     | `book` (27)           |
| Exemplar                       | `exemplar` (22)          | `copy` (3)            |
| Ausleihe                       | `ausleih` (14)           | `loan`, `borrow` (10) |
| Bestellung                     | `bestell` (48)           | `order` (15)          |
| Etikett                        | `etikett` (19)           | `label` (8)           |
| Einstellungen                  | `einstellung` (9)        | `setting` (6)         |
| Benutzer (Konto des Personals) | `benutzer` (3)           | `user` (11)           |
| Abgänger                       | `abgaeng` (7)            | `graduat` (3)         |
| Schaden                        | `schaden` (7)            | `damage` (3)          |
| Lieferant                      | `lieferant` (5)          | `supplier` (2)        |
| Gerät                          | `geraet` (6)             | `device` (1)          |

Die Tabellen der Datenbank heißen überwiegend deutsch (`leser`, `buecher_titel`,
`buecher_exemplare`, `ausleihen`), einige ältere englisch (`class_books`, `role_permissions`).

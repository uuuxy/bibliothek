# Wegweiser — wo steht was

Stand: 25.10.2026

Eine Seite für den Einstieg in den Code: der Weg einer Anfrage, vom Menüpunkt zum Ordner und
für häufige Änderungen die Stelle. Warum etwas so gebaut ist, steht in
[ARCHITEKTUR.md](ARCHITEKTUR.md); die Fachbegriffe erklärt dort Kapitel 12. Wie etwas im Code
heißt und was ein Kommentar sagt, regelt [STIL.md](STIL.md).

## Der Weg einer Anfrage

Ein Klick in der Oberfläche läuft durch vier Stationen, immer in dieser Reihenfolge:

| Station    | Ordner              | Aufgabe                                                          |
| ---------- | ------------------- | ---------------------------------------------------------------- |
| Oberfläche | `frontend/src/`     | Bildschirme und Bauteile (Svelte); schickt die Anfrage           |
| Tür        | `internal/api/`, je Thema    | nimmt die Anfrage an: prüft Eingabe und Recht, antwortet         |
| Regel      | `internal/service/` | entscheidet, was gilt: Frist, Sperre, Reihenfolge der Schritte   |
| Abfrage    | `internal/repository/`       | liest und schreibt in der Datenbank; in den Türen steht kein SQL |

Daneben stehen: `internal/pdf/` setzt Blätter und Etiketten, `internal/mailservice/` verschickt Mail, `internal/jobs/`
läuft nachts, `internal/auth/` meldet an. Die Tabellen der Datenbank beschreibt `schema.sql`, jede
Änderung daran ist ein Schritt in `migrations/`.

Welche Adresse zu welcher Tür führt und welches Recht sie verlangt, steht in sechs Listen:
`internal/api/routen_katalog.go`, `internal/api/routen_leser.go`, `internal/api/routen_bestellung.go`,
`internal/api/routen_system.go`, `internal/api/routen_oeffentlich.go` und `internal/api/routen_theke.go`. Alle Adressen mit ihren Aufrufern in der
Oberfläche nennt [api_inventar.md](api_inventar.md).

## Vom Menüpunkt zum Code

Die Türen sind nach den Menüpunkten der Oberfläche geordnet. Die genannte Datei trägt am Kopf
die Beschreibung ihres Themas, die übrigen Dateien des Ordners je eine Sache.

| Menüpunkt                               | Bildschirm in `frontend/src/lib/`                                                                                                                       | Türen                              |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| Ausleihe (Theke)                        | `components/omnibox/Omnibox.svelte`                                                                                                                     | `internal/api/theke/theke.go`               |
| Medienkatalog, Klassensätze             | `components/katalog/Medienkatalog.svelte`, `components/leser/Schulklassen.svelte`                                                                       | `internal/api/katalog/katalog.go`           |
| Buchakte, Signaturen                    | `components/buchakte/Buchakte.svelte`, `components/signaturen/Signaturen.svelte`                                                                        | `internal/api/katalog/katalog.go`           |
| Bestandsbücher                          | `components/bestand/Bestandsbuecher.svelte`                                                                                                             | `internal/api/katalog/katalog.go`           |
| Druck-Center                            | `components/etiketten/DruckCenter.svelte`                                                                                                               | `internal/api/druck/druck.go`               |
| Leserdatei, Leserakte                   | `components/leser/Schuelerdatei.svelte`, `components/leserakte/Leserakte.svelte`                                                                        | `internal/api/leser/leser.go`               |
| Schaden, Forderung, Bescheid            | `components/leserakte/SchadenMeldenDialog.svelte`, `components/leserakte/LeserGebuehrenKarte.svelte`, `components/leserakte/LeserBescheideKarte.svelte` | `internal/api/schaden/schaden.go`           |
| Mahnwesen, Abgänger                     | `components/mahnwesen/Mahnwesen.svelte`, `components/leser/Abgaenger.svelte`                                                                            | `internal/api/mahnwesen/mahnwesen.go`       |
| Bestellungen, Bestellberichte           | `components/bestellungen/Bestellwesen.svelte`, `components/bestellungen/BestellBerichte.svelte`                                                         | `internal/api/bestellung/bestellung.go`     |
| Inventur                                | `components/inventur/Inventur.svelte`                                                                                                                   | `internal/api/inventurlauf/inventurlauf.go` |
| Schuljahreswechsel (LMF-Plan)           | `components/lmfplan/LmfPlan.svelte`                                                                                                                     | `internal/api/lernmittel/lernmittel.go`     |
| Benutzer & Rechte                       | `components/benutzer/Berechtigungen.svelte`                                                                                                             | `internal/api/benutzer/benutzer.go`         |
| Einstellungen, Statistiken, System-Logs | `components/einstellungen/Einstellungen.svelte`, `components/statistik/Statistik.svelte`, `components/system/SystemLogs.svelte`                         | `internal/api/system/system.go`             |
| Mein Portal, öffentlicher Katalog       | `components/portal/KollegiumPortal.svelte`, `components/katalog/OpacSuche.svelte`                                                                       | `internal/api/portal/portal.go`             |

Die Abfragen aller Themen stehen in `internal/repository/`, je Sache eine Datei; die des Medienkatalogs
beschreibt der Kopf von `internal/repository/titel_abfragen.go`. Was jede Tür braucht (Eingabe lesen,
Recht prüfen, Protokoll schreiben), steht in `internal/api/tuer/tuer.go`. Welcher Menüpunkt welchen
Bildschirm öffnet, steht in `frontend/src/lib/menue.js` und `frontend/src/lib/Router.svelte`.

## Ich will etwas ändern

| Ich will …                                    | Dort                                                                                                                                |
| --------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| den Text oder das Bild eines Blatts ändern    | `internal/pdf/`, je Blatt eine Datei (`internal/pdf/mahnbrief.go`, `internal/pdf/etikett_buch.go`)                                                             |
| eine Regel der Ausleihe ändern                | `internal/service/frist.go`, `internal/service/ausleih_sperren.go`                                                             |
| eine Abfrage ändern                           | `internal/repository/`, die Datei heißt wie die Sache (`internal/repository/auflagen.go`)                                                             |
| eine Tabelle oder Spalte ergänzen             | ein neuer Schritt in `migrations/` und derselbe Stand in `schema.sql`; `internal/db/migrationen_schema_paritaet_pg_test.go` hält beide gleich |
| eine neue Adresse anlegen                     | die Tür im Ordner ihres Themas, Adresse und Recht in der passenden Liste `internal/api/routen_*.go`                                          |
| festlegen, was eine Rolle ab Werk darf        | `internal/db/rechte_vorgabe.go`                                                                                                                        |
| einen Menüpunkt ändern                        | `frontend/src/lib/menue.js`                                                                                                          |
| eine Kategorie der Einstellungen ändern       | `frontend/src/lib/components/einstellungen/kategorien.js`                                                                                |
| ein gemeinsames Bauteil der Oberfläche ändern | `frontend/src/lib/components/ui/`                                                                                                   |
| Farben und Schriftgrößen ändern               | `frontend/src/styles/rollen.css`, `frontend/src/styles/theme-mass.css`                                                              |
| einen Nachtlauf ändern                        | `internal/jobs/cron.go` nennt die Läufe und ihre Uhrzeit                                                                                     |
| den Abgleich mit der LUSD ändern              | `internal/lusd/`, beschrieben in [LUSD.md](LUSD.md)                                                                                 |
| die Übernahme aus Littera ändern              | `internal/littera/`, beschrieben in [littera_schema_befund.md](littera_schema_befund.md)                                            |

Nach jeder Änderung laufen die Prüfungen aus dem Abschnitt „Qualitätssicherung" der
[README](../README.md). Was offen ist, steht nur in [OFFEN.md](OFFEN.md).

## Namen

Dateien und Bezeichner heißen deutsch, je Sache ein Wort ([STIL.md](STIL.md), Abschnitte 1, 2
und 13); wer alles zu einer Sache sucht, sucht nach einem Wort. Einen neuen englischen Namen
melden `stil_ratsche_test.go` und `frontend/src/lib/frontend-hygiene-sprache.test.js`.
Adressen, Felder in JSON, Tabellen und Spalten behalten ihre Namen, auch englische
([STIL.md](STIL.md), Abschnitt 9): Die Tabellen heißen überwiegend deutsch (`leser`,
`buecher_titel`, `buecher_exemplare`, `ausleihen`), einige englisch (`class_books`,
`role_permissions`).

# 12. Glossar

Stand: 17.09.2026

Die Fachsprache des Hauses, mit dem Code-Bezug daneben. Wo ein Begriff im Code **anders**
heißt als in der Oberfläche, steht beides — das ist die häufigste Stolperstelle beim
Einstieg.

---

## 12.1 Fachbegriffe der Bibliothek

| Begriff                     | Bedeutung                                                                                                                                                              | Im Code / in der DB                                        |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| **Abgänger**                | Schüler, der bestätigt in der LUSD war und im neuen Export fehlt. Löst Sperre, Karenz und später Löschung aus                                                          | `abgaenger_seit`, `jobs/cron_dsgvo_abgaenger.go`           |
| **Abgangsbuch**             | Nachweis der aus dem Bestand ausgegangenen Bücher je Halbjahr, mit Grund, getrennt nach Land und Träger                                                                 | `api/abgangsbuch_handler.go`, `api/abgangsbuch_pdf.go`     |
| **Abholfach**               | Ort, an dem ein für eine Vormerkung reserviertes Exemplar auf Abholung wartet; Hinweis an der Theke trägt Titel und Frist, bewusst ohne IDs                             | `repository/vormerkung_abholfach_pg_test.go`               |
| **Abwertung**               | Minderung des Ersatzwerts nach Alter und Zustand des Exemplars                                                                                                          | `pkg/ersatzwert`, `repository/ersatzwert_groessen.go`       |
| **Art** (des Lesers)        | `schueler`, `lehrkraft` oder `liv`. Sagt, **wer an der Theke Bücher bekommt** — entscheidet **keine** Rechte                                                             | `leser.art`                                                |
| **Aussonderung**            | Ausbuchen eines Exemplars aus dem Bestand mit Grund (Verlust, Verschleiß, Makulatur)                                                                                     | `repository/aussonderung_paritaet_test.go`                  |
| **Ausweis**                 | Karte des Lesers. Alle neuen Nummern tragen die Vorsilbe `A-`; die Aufschrift richtet sich nach der **Art**                                                             | `internal/ausweis`, `api/ausweis_layout.go`                 |
| **Bescheid**                | Schadensersatz-Bescheid für Landes-Lernmittel; nennt das Konto (Barzahlung ist laut Erlass nicht der Weg), hat eine eigene Nummernfolge                                  | `schadensersatz_bescheide`, `api/bescheid_pdf.go`           |
| **Bestand**                 | Die physischen Stücke — im Unterschied zum **Katalog** (dem Werk)                                                                                                       | `buecher_exemplare`                                        |
| **Bestandsbuch / Zugangsbuch** | Nachweis der in den Bestand gekommenen Bücher je Halbjahr, mit Lieferant                                                                                             | `api/bestandsbuch.go`                                      |
| **Betriebsbereitschaft**    | Die Selbstprüfung: „Was ist eingerichtet, aber nicht in Betrieb?"                                                                                                       | `api/betriebsbereitschaft*.go`                              |
| **Exemplar**                | Ein einzelnes physisches Buch mit eigenem Barcode                                                                                                                       | `buecher_exemplare`                                        |
| **Ersatzwert**              | Was ein Buch **heute noch wert** ist — als Vorschlag **mit Herleitung**, weil der Betrag im Ermessen der Schule liegt                                                    | `pkg/ersatzwert`                                            |
| **Freihand**                | Sonderbestände (CDs, DVDs, Hörbücher) mit rollierender Frist statt Jahresfrist                                                                                          | `internal/service/loan_rules.go`                            |
| **Gerät**                   | Ausleihbare Hardware (Laptop, Tablet) mit Zubehör-Checkliste; Vorsilbe `G-`                                                                                             | `geraete`, `internal/service/device_service.go`             |
| **Inventur-Session**        | Sitzungsgebundene Zählung, damit parallele Zählungen sich nicht überschreiben                                                                                            | `inventur_sessions`, `repository/inventur_session_repo.go`   |
| **Karenzzeit**              | Frist zwischen Abgang und Anonymisierung (Vorgabe 90 Tage) — das Fenster, in dem eine falsche Zuordnung noch reparierbar ist                                            | `abgaenger_karenz_tage`, `repository.PredikatAnonymisierung` |
| **Katalog**                 | Die Werke (Metadaten) — im Unterschied zum **Bestand**                                                                                                                  | `buecher_titel`                                            |
| **Klassensatz**             | Mehrere Exemplare eines Titels für eine Klasse. Zwei Quellen: Handliste und live aus den Ausleihen abgeleitet                                                            | `class_books`, `klassensatz_reservierungen`                  |
| **Kollegium**               | **Grundzustand** jeder Lehrkraft, keine vergebene Rolle. Ein Recht: `create_reservations`                                                                                | Enum `benutzer_rolle = 'kollegium'`, `db/seed.go`            |
| **Leser**                   | Jeder Entleiher — Schüler, Lehrkraft, LiV. Eine Tabelle, ein Ausweis-Nummernkreis                                                                                       | `leser`                                                    |
| **Leserdatei**              | Die Liste aller Leser in der Oberfläche                                                                                                                                 | `frontend/src/lib/StudentDirectory.svelte`                  |
| **Leitung**                 | Rolle: Admin **minus** `manage_users` und `manage_settings` — abgeleitet, nicht abgeschrieben                                                                            | Migration 122, `db/rolle_leitung_test.go`                    |
| **Littera**                 | Die Windows-Vorgängersoftware. Ihr Altbestand kommt über `mdb-export`-CSVs herein; ihre Barcodes dürfen nicht neu geklebt werden                                        | `internal/littera`, `cmd/littera-altbestand`                 |
| **LiV**                     | Lehrkraft im Vorbereitungsdienst. Eigene Art, gleiche Rechte wie eine Lehrkraft als Entleiher                                                                            | `leser.art = 'liv'`                                         |
| **LMF / Lernmittelfreiheit**| Schulbücher, die die Schule dem Schüler für ein Schuljahr leiht. Feste Jahresfrist (Stichtag 31. Juli) statt rollierender Frist                                          | `pkg/lmf`, `buecher_titel.ist_lernmittel`                    |
| **LMF-Plan**                | Termine je Klasse für Büchertausch (vor den Sommerferien) und Bücherausgabe (danach). Ein Termin **verschiebt die Frist**                                                | `lmf_plaene`, `lmf_termine`, `pkg/lmfplan`                  |
| **LUSD**                    | Das Schulverwaltungssystem des Landes. Liefert Schülerdaten als Bericht (`.xlsx`) — in der Praxis **ohne** Schüler-ID                                                     | `api/lusd*.go`, [LUSD.md](../LUSD.md)                       |
| **Mahnstufe**              | Zählt die gedruckten Mahnungen. Steigt **nur** beim PDF-Druck, nie beim Mailversand                                                                                      | `api/mahnwesen_bulk.go`                                     |
| **Mittel**                  | Herkunft des Geldes: Landesmittel (Lernmittel) oder Kreis-/Trägermittel (Schülerbücherei). Trennt Töpfe in Beschaffung und Nachweis                                      | `repository/bestellung_mittel.go`, [mittel_konzept.md](../mittel_konzept.md) |
| **Nachbuchen**              | Das Einbuchen offline erfasster Scans nach Rückkehr des Netzes, mit Scan-Zeitpunkt und Abweichungsmeldung                                                                | `api/nachbuchen_handler.go`, `internal/service/nachbuchen.go` |
| **Omnibox**                 | Das **eine** Eingabefeld des Tresens für alle Scans und Suchen                                                                                                          | `internal/service/omnibox_service.go`, `frontend/src/lib/Omnibox.svelte` |
| **OPAC**                    | Der öffentliche Katalog unter `/katalog` — ohne Anmeldung, ohne Personendaten                                                                                            | `api/opac.go`                                               |
| **Signatur / Systematik**   | Ordnungsbegriff und Standort im Regal                                                                                                                                    | `systematik_kategorien`, `repository/signatur_praefix.go`     |
| **Theke / Tresen**          | Der Arbeitsplatz mit Scanner; „Kiosk" meint dasselbe aus Sicht der Bauform                                                                                               | `frontend/src/lib/Omnibox.svelte`, `stores/thekeLeeren.js`    |
| **Vormerkung**              | Reservierung eines Exemplars durch einen Leser; rückt bei Rückgabe nach und wird „abholbereit"                                                                           | `vormerkungen`, `repository/vormerkung_nachruecken.go`        |

---

## 12.2 Technische Begriffe des Projekts

| Begriff                      | Bedeutung in diesem Projekt                                                                                                                                  |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Gate**                     | Ein Test, der eine Zusage **verlieren** kann. Ein Gate, das seine Aussage nicht verlieren kann, prüft nichts — deshalb muss jedes einmal rot gesehen worden sein |
| **Ratsche**                  | Ein Gate gegen einen **Rückfall**: Es hält einen erreichten Zustand fest (Dateigröße, Deadcode-Baseline, behobene Bugklasse)                                  |
| **Sweep**                    | Ein Durchgang über den ganzen Bestand auf der Suche nach **einer** Bugklasse. Register und Detektoren: [sweeps.md](../sweeps.md)                              |
| **Bugklasse**                | Ein Fehlermuster, das mehrfach auftreten kann (z. B. „Schreibpfad gegen gefilterte Sicht", „Phantom-Erfolg", „Fehler-Kollaps")                                |
| **Phantom-Erfolg**           | Eine Antwort „erfolgreich", obwohl nichts geschrieben wurde (verworfene `RowsAffected`, null getroffene Zeilen)                                                |
| **Fehler-Kollaps**           | Mehrere unterschiedliche Fehler, die in **einer** unbrauchbaren Meldung zusammenfallen                                                                        |
| **Skip-Bilanz**              | Die Angabe eines Testlaufs darüber, was er **nicht** geprüft hat                                                                                              |
| **Drift**                    | Zwei Orte, die dasselbe behaupten und auseinandergelaufen sind (Doku vs. Code, Swagger vs. Annotation, Migrationsliste vs. `schema.sql`)                       |
| **Lügende Ratsche**          | Ein Detektor, der grün bleibt, obwohl seine Aussage nicht mehr gilt — die schlimmste Sorte, weil sie Sicherheit vorspiegelt                                   |
| **Rot-Beweis**               | Der Nachweis, dass ein Gate im echten Fehlerfall wirklich rot wird                                                                                            |
| **Blindheit** (einer Ratsche)| Was ein Detektor aus seiner Mechanik heraus **nicht** sehen kann. Steht je Ratsche in der Landkarte in [sweeps.md](../sweeps.md)                               |
| **Invariante**               | Eine Aussage, die immer wahr sein muss — mit Angabe der **Ebene**, auf der sie durchgesetzt ist (🟢 DB / 🟡 Code / 🔴 Doku)                                     |
| **Zwilling**                 | Zwei Implementierungen derselben Regel (etwa `suchnorm` in SQL und `repository.Suchnorm` in Go), deren Gleichheit ein Gate erzwingt                            |
| **Selbstprüfung**            | Die Seite unter *System*, die beantwortet: Was ist eingerichtet, aber nicht in Betrieb?                                                                        |
| **Bereitschafts-Wächter**    | Der Mailversand kritischer Befunde der Selbstprüfung — der Wächter, der sich meldet, statt gelesen werden zu müssen                                            |
| **Restore-Probe**            | Das wöchentliche Einspielen des jüngsten Backups in eine Wegwerf-Datenbank                                                                                     |
| **Secret-Guard**             | Die Start-Verweigerung bei bekannten Beispiel-Geheimnissen                                                                                                     |
| **Nachbuch-Tür**             | `POST /api/action/nachbuchen` — die Schreibtür für offline erfasste Vorgänge                                                                                    |
| **Idempotenz-Schlüssel**     | Kennzeichen eines Vorgangs, unter dem die Antwort gespeichert wird, damit eine Wiederholung nichts zweimal tut                                                  |
| **Omnibox-Vorsilbe**         | `B-` Buch, `A-` Ausweis (neu), `S-`/`L-` Ausweis (historisch, wird gelesen), `G-` Gerät. Offline die **einzige** Unterscheidung                                  |
| **PII-Stufe**                | Einordnung einer Route nach Schülerdaten (0–3) in [PII_MATRIX.de.md](../PII_MATRIX.de.md), von drei Gates gehalten                                              |
| **Übernahme**                | Einmaliger Import eines Altbestands mit Savepoint je Datensatz und Abgleich gegen den tatsächlichen Zeilenzuwachs                                               |
| **Schulzeit**                | „Jetzt" aus Sicht der Schule: Schuljahr, Stichtage, Kalendertag (`pkg/schulzeit`)                                                                               |

---

## 12.3 Rollen und Rechte auf einen Blick

| Rolle / Zustand   | Vergeben durch | Kurzbeschreibung                                                                          | Rechte (Vorgabe)                                     |
| ----------------- | -------------- | ----------------------------------------------------------------------------------------- | ---------------------------------------------------- |
| **Admin**         | Admin          | Uneingeschränkt: Einstellungen, Benutzer, Audits, Datenschutz-Routinen                     | alle                                                 |
| **Leitung**       | Admin          | Führt die Bibliothek, ohne die Systempflege zu übernehmen                                  | Admin **minus** `manage_users`, `manage_settings`     |
| **Mitarbeiter**   | Admin          | Tagesgeschäft: Omnibox, Katalog, Mahnwesen, Leserdatei                                     | Fachrechte ohne Systemeinstellungen                   |
| **Helfer**        | Admin          | Theke und Katalog lesen — Grenze zu Personendaten zieht `view_students`                     | genau zwei: `perform_actions`, `view_books`           |
| **Kollegium**     | **niemand** — Grundzustand nach Selbstanmeldung und Freischaltung | Eigenes Portal: Reservieren, Klassensätze, LMF-Plan, Schulbücher, Anliegen | genau eins: `create_reservations`     |

Ein **Helfer braucht ein Postfach auf dem Schul-Mailserver** — es gibt keinen Code- oder
Barcode-Anmeldeweg (A2). Welche Seiten eine Rolle erreicht, entscheidet `canSeeItem()` in
`frontend/src/lib/menu.js` — und nur diese Funktion.

---

## 12.4 Abkürzungen

| Kürzel      | Bedeutung                                                                     |
| ----------- | ----------------------------------------------------------------------------- |
| **DSB**     | Datenschutzbeauftragte(r)                                                      |
| **EAN-13**  | 13-stelliger Strichcode der Buchetiketten                                      |
| **EUPL**    | European Union Public Licence — die Lizenz dieses Projekts (1.2)               |
| **LANIS**   | Landesabiturinformationssystem; liefert Klassenlisten als Semikolon-CSV        |
| **LMF**     | Lernmittelfreiheit                                                            |
| **LUSD**    | Lehrer- und Schülerdatenbank (Schulverwaltung des Landes)                      |
| **OPAC**    | Online Public Access Catalogue — der öffentliche Katalog                        |
| **PII**     | Personenbezogene Daten (hier: Schülerdaten, Stufen 0–3)                        |
| **RBAC**    | Rollenbasierte Zugriffskontrolle                                               |
| **SSE**     | Server-Sent Events — der Echtzeitkanal `/events`                               |
| **TOM**     | Technische und organisatorische Maßnahmen (DSGVO)                              |
| **VVT**     | Verzeichnis von Verarbeitungstätigkeiten (DSGVO Art. 30)                        |
| **WCAG**    | Web Content Accessibility Guidelines (hier: 2.1 Stufe AA)                       |

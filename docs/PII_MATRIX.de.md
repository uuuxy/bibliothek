# PII-Matrix — Schülerdaten je Endpunkt

Abschluss von Befund **F2** der unabhängigen Prüfung (`bewertung/datenbank-pruefbericht.md`):
Jede HTTP-Route des Systems ist hier nach Schülerdaten eingestuft. Das Gate
`api/pii_matrix_test.go` hält die Tabelle mit dem Code deckungsgleich — eine neue
Route ohne Zeile hier wird rot, eine Zeile ohne Route ebenso, und das dokumentierte
Recht wird gegen die Registrierung geprüft. **Die Stufe der GET-Routen ist seit dem
01.09.2026 gemessen, nicht nur behauptet:** `api/pii_antwort_gate_pg_test.go` ruft
jede GET-Route über den echten Router mit genau dem Recht ihrer Zeile auf und prüft
die Antwort (inkl. entpackter PDF-Ströme) gegen Kanarienwerte je Stufe — nichts
oberhalb der dokumentierten Stufe darf erscheinen, Schlüsselrouten tragen
Positiv-Kontrollen gegen leere Antworten. Für Nicht-GET-Routen (Schreibpfade)
bleibt die Stufe Handarbeit: Wer eine Zeile anlegt, hat den Handler gelesen.
Stand: 01.09.2026 (erhoben 19.08.2026, alle 6 Abschnitte Handler für Handler und
stichprobenartig am laufenden System belegt; 01.09.: Tresen-Auskunft ergänzt,
Antwort-Gate eingezogen).

**Stufen** (bemessen an dem, was das Recht der Zeile ALLEIN öffnet — was erst ein
zusätzliches `view_students` freischaltet, steht als Anmerkung):

| Stufe | Bedeutung | Beispiele |
|---|---|---|
| 0 | keine Schülerdaten | Katalog, Bestellwesen, System |
| 1 | Kiosk-Niveau: Name, Klasse, Barcode-ID, Sperrstatus | Scanner-Suche, Vormerkungen |
| 2 | Verwaltung: + Geburtsdatum, Abgängerjahr, Sperrgrund, LUSD-ID, Ausleihhistorie (befristet: nächtlicher Job trennt abgeschlossene Ausleihen nach Einstellungs-Frist vom Schüler, `jobs/cron_dsgvo_lesehistorie.go`) | Schülerdatei, Mahnwesen |
| 3 | sensibel: Wohnadresse, Eltern-E-Mail, Foto, Gebühren-/Schadensdaten mit Namen | DSGVO-Auskunft, Rechnungen |

**Rechte-Spalte:** ein Rechtename aus `db/seed.go` (geprüft gegen
`RequirePermission`), `öffentlich` (kein Wrapper, MUSS Stufe 0 sein — eigener Test),
`Token` (Link-Token ist der Ausweis), `selbst-prüfend` (Handler validiert das
Session-Cookie selbst), `Sitzung` (`RequireAuthenticated`), `inventur-Mux`
(Delegation, Schutz im inneren Mux) oder `inventur:view_books` /
`inventur:edit_books` (die inneren Inventur-Routen selbst).

**Leitplanken (aus den Sicherheitsbefunden, `bewertung/sicherheitsbefund-*.md`):**
`view_books` und `perform_actions` besitzt die Helfer-Rolle ab Werk — Routen mit
Stufe ≥2 dürfen NIE allein dahinter liegen. Stufe 1 hinter Helfer-Rechten ist die
bewusste Theken-Ausnahme (SchuelerKiosk-Sicht, `api/schueler_kiosk.go`). Der
Sperrgrund-Freitext (`block_reason`) ist Stufe 2 und hängt überall an
`view_students` (`ohneSperrgrund` in `api/action.go`). Stufe 3 gehört
ausschließlich hinter `view_students`/`manage_students_admin`.

## routes_students.go

| Route | Recht | Stufe | Inhalt |
|---|---|---|---|
| `GET /api/schueler` | view_students | 2 | Liste: Name, Klasse, Barcode, Sperr-/Überfällig-Status |
| `GET /api/schueler/{id}` | view_students | 3 | Profil: Adresse, Eltern-Mail, Geburtsdatum, Foto-URL, Ausleihen |
| `POST /api/schueler` | create_students | 1 | Antwort nur id + barcode_id |
| `POST /api/print/schueler-etiketten` | view_students | 1 | Klebebogen als PDF: Name, Klasse, Barcode-ID. Eingabe sind nur IDs, die Angaben holt der Server — kein Weg, fremde Namen auf echte Barcodes zu drucken |
| `PATCH /api/schueler/{id}` | edit_students | 0 | Antwort nur Status-Echo |
| `PATCH /api/admin/students/{id}/lock` | edit_students | 1 | Name, Klasse, Sperrflag |
| `DELETE /api/schueler/{id}` | delete_students | 0 | nur Erfolgsmeldung |
| `GET /api/schueler/{id}/dsgvo-auskunft` | manage_students_admin | 3 | Vollauskunft (Zweck: DSGVO Art. 15) |
| `GET /api/schueler/{id}/dsgvo-auskunft/pdf` | manage_students_admin | 3 | dieselbe Vollauskunft als PDF |
| `GET /api/schueler/deleted` | delete_students | 2 | Papierkorb: Name, Klasse, deleted_at |
| `POST /api/schueler/{id}/restore` | delete_students | 0 | nur Statusmeldung |
| `DELETE /api/schueler/deleted/{id}` | manage_students_admin | 0 | nur Statusmeldung |
| `POST /api/schueler/{id}/photo` | upload_photos | 1 | Antwort-URL enthält Barcode-ID |
| `GET /api/schueler/{barcode_id}/photo` | view_students | 3 | entschlüsseltes Passfoto |
| `GET /api/klassen` | view_students | 0 | nur Klassenbezeichnungen |
| `GET /api/klassen-mapping` | manage_settings | 0 | Klasse + Lehrkraft-Mail (Personal) |
| `POST /api/klassen-mapping` | manage_settings | 0 | nur Status |
| `DELETE /api/klassen-mapping/{klasse}` | manage_settings | 0 | 204 ohne Body |
| `POST /api/lusd/preview` | import_students | 2 | Diff: LUSD-ID, Name, Klassenwechsel |
| `POST /api/lusd/import` | import_students | 2 | dasselbe Diff nach Ausführung |
| `POST /api/students/promote` | manage_students_admin | 0 | nur Zähler + Konflikt-Klassennamen |
| `GET /api/abgaenger` | view_graduates | 2 | Name, Klasse, offene Ausleihen |
| `GET /api/abgaenger/pdf` | view_graduates | 2 | Kontoauszug-PDF je Abgänger |
| `POST /api/abgaenger/mail` | create_orders | 2 | versendet Konto-PDFs an Klassenleitungen; Antwort nur Zähler |
| `POST /api/damage/report` | edit_students | 0 | Antwort nur schadens_id |
| `GET /api/schadensfaelle/{id}/pdf` | view_students | 3 | Elternbrief (Fensterkuvert): Name, Anschrift, Betrag, Schaden (Anschrift seit 01.09.2026 verdrahtet — vorher Unterstrich-Zeilen) |
| `GET /api/schueler/{id}/schadensfaelle` | view_students | 2 | Gebührenliste zur Schüler-ID |
| `POST /api/schadensfaelle/{id}/bezahlt` | edit_students | 0 | nur Status |
| `POST /api/schadensfaelle/{id}/storno` | edit_students | 0 | nur Status |
| `GET /api/mahnwesen` | view_students | 2 | Name, Klasse, überfällige Medien |
| `GET /api/mahnwesen/ueberfaellig_jahrgang` | view_students | 2 | dasselbe nach Jahrgang |
| `GET /api/mahnwesen/pdf` | view_students | 2 | Mahnlisten-PDF |
| `POST /api/mahnwesen/senden` | create_orders | 2 | versendet Mahn-PDF; Antwort nur Status |
| `POST /api/mail/send-bulk-overdue` | create_orders | 2 | Mahn-PDFs je Klasse; Antwort nur Zähler |

## routes_books.go

| Route | Recht | Stufe | Inhalt |
|---|---|---|---|
| `DELETE /api/buecher/titel/{id}` | delete_books | 0 | nur Erfolgsmeldung |
| `PUT /api/buecher/titel/{id}/signatur` | create_orders | 0 | Titeldaten |
| `GET /api/buecher/titel/{id}/exemplare` | view_books | 0 | Exemplare ohne Schüler-Join |
| `GET /api/buecher/titel/{id}/ausleiher` | view_students | 1 | aktuelle Ausleiher: Name, Klasse, Fristen |
| `GET /api/buecher/titel/{id}/historie` | view_students | 2 | Leser-Historie eines Titels |
| `GET /api/buecher/titel/{id}/etiketten` | view_books | 0 | Etiketten-PDF |
| `POST /api/print/labels` | view_books | 0 | Etiketten-PDF |
| `GET /api/exemplare/etiketten-offen` | edit_books | 0 | Exemplarliste ohne Ausleiher |
| `GET /api/exemplare/etiketten-offen/anzahl` | edit_books | 0 | nur Zähler |
| `POST /api/exemplare/etiketten-gedruckt` | edit_books | 0 | nur Zähler |
| `POST /api/exemplare/etiketten-zuruecksetzen` | edit_books | 0 | nur Zähler |
| `POST /api/exemplare/etiketten-altbestand` | edit_books | 0 | nur Zähler |
| `DELETE /api/buecher/exemplare/{id}` | delete_books | 0 | Erfolgsmeldung ohne Ausleiher |
| `POST /api/buecher/exemplare/{id}/schadensnotiz` | edit_books | 0 | Zustandsnotiz am Exemplar |
| `PUT /api/buecher/exemplare/{id}/barcode` | edit_books | 0 | Exemplar-Barcode |
| `PUT /api/buecher/exemplare/{id}/status` | edit_books | 0 | Exemplar-Status |
| `POST /api/buecher/exemplare/{id}/defekt` | edit_books | 0 | Antwort ohne Namen; Seiteneffekt: Schadensfall zur schueler_id |
| `POST /api/buecher/exemplare/{id}/aussondern` | edit_books | 0 | Exemplar-Status |
| `POST /api/ausleihen/{ausleihe_id}/verlaengern` | edit_books | 1 | Sperrstatus; Freitext-Grund nur mit view_students |
| `POST /api/ausleihen/global-extend-lmf` | edit_books | 0 | nur Zähler |
| `PATCH /api/admin/ausleihen/{id}/faelligkeit` | edit_books | 0 | nur Fälligkeitsdatum |
| `POST /api/buecher/aus-isbn` | create_orders | 0 | Titel-Metadaten |
| `GET /api/vormerkungen` | manage_vormerkungen | 1 | Warteliste: Name, Klasse je Titel (enges Theken-Recht) |
| `POST /api/vormerkungen` | manage_vormerkungen | 0 | Antwort nur id |
| `DELETE /api/vormerkungen/{id}` | manage_vormerkungen | 0 | nur Status |
| `POST /api/reservierungen/klassensatz` | create_reservations | 0 | Titel, Klasse (Freitext), Anzahl |
| `GET /api/reservierungen/klassensatz` | view_orders | 0 | angefordert_von ist Lehrkraft |
| `GET /api/reservierungen/klassensatz/anzahl` | view_orders | 0 | nur Zähler |
| `GET /api/reservierungen/klassensatz/offen` | create_reservations | 0 | Titel, Klasse, Anzahl |
| `GET /api/reservierungen/klassensatz/eigene` | create_reservations | 0 | eigene Reservierungen; Bibliotheks-Notiz |
| `GET /api/geraete` | view_books | 1 | Ausleihername nur mit view_students, sonst geblendet |
| `POST /api/geraete` | edit_books | 0 | Gerätestammdaten |
| `PUT /api/geraete/{id}` | edit_books | 0 | Gerätestammdaten |
| `PUT /api/reservierungen/klassensatz/{id}/erledigen` | create_orders | 0 | Mail an Lehrkraft, 204 |
| `POST /api/anliegen` | create_reservations | 0 | Freitext-Anliegen, Antwort id |
| `GET /api/anliegen/eigene` | create_reservations | 0 | eigene Anliegen; „von" ist Lehrkraft |
| `GET /api/anliegen/offen` | view_orders | 0 | Anliegen; „von" ist Lehrkraft |
| `GET /api/anliegen/anzahl` | view_orders | 0 | nur Zähler |
| `PUT /api/anliegen/{id}/erledigen` | create_orders | 0 | Mail an Lehrkraft, 204 |

## routes_misc.go

| Route | Recht | Stufe | Inhalt |
|---|---|---|---|
| `GET /api/public/opac/suche` | öffentlich | 0 | Titeldaten, Verfügbarkeit |
| `GET /api/monitor/slides` | öffentlich | 0 | Titeldaten, aggregierte Zähler |
| `GET /api/public/bestellung/{token}` | Token | 0 | Bestellansicht des Lieferanten |
| `GET /api/public/bestellung/{token}/etiketten/{groesse}` | Token | 0 | Etiketten-PDF der Bestellung |
| `POST /api/public/bestellung/{token}/bestaetigen` | Token | 0 | nur Status |
| `POST /api/action` | perform_actions | 1 | SchuelerKiosk-DTO; Sperrgrund-Freitext nur mit view_students (ohneSperrgrund). BEWUSST ohne Objektbindung: active_student_id/active_teacher_id sind frei wählbar — das IST die Theke (jedes Buch auf jeden Ausweis buchen); nachvollziehbar über StaffID, override_block zusätzlich edit_students-gated (IDOR-Sweep 19.08.2026, kein Fund) |
| `POST /api/action/batch` | perform_actions | 1 | wie /api/action, je Batch-Item |
| `GET /api/search` | perform_actions | 1 | Students als SchuelerKiosk-DTO |
| `GET /api/inventur/sessions` | inventory_scan | 0 | Session-Zähler |
| `POST /api/inventur/start` | manage_inventory | 0 | Session-Metadaten |
| `POST /api/inventur/scan` | inventory_scan | 0 | Exemplar-/Titelstatus ohne Ausleiher |
| `POST /api/inventur/finish` | manage_inventory | 0 | Verlustliste, rein exemplarbezogen |
| `GET /api/inventur/abgeschlossen` | manage_inventory | 0 | Session-Historie |
| `GET /api/inventur/fehlbestand` | manage_inventory | 0 | Verlustliste ohne Ausleiher |
| `POST /api/inventur/abort` | manage_inventory | 0 | nur Status |
| `POST /api/buecher/exemplare/{id}/gefunden` | manage_inventory | 0 | nur Status |
| `POST /api/buecher/exemplare/verlust-endgueltig-loeschen` | manage_inventory | 0 | nur Zähler |

## routes_orders.go

| Route | Recht | Stufe | Inhalt |
|---|---|---|---|
| `GET /api/bestellungen/konfiguration` | view_orders | 0 | zwei bool-Anzeigeregeln |
| `GET /api/bestellungen` | view_orders | 0 | Titel-/Bestandsdaten |
| `GET /api/bestellungen/pdf` | view_orders | 0 | PDF derselben Bestandsdaten |
| `GET /api/bestellhistorie` | view_orders | 0 | Lieferant, Positionen |
| `GET /api/bestellhistorie/uebersicht` | view_orders | 0 | vier Aggregatzahlen |
| `GET /api/bestellhistorie/bericht` | view_orders | 0 | PDF: Schule, Lieferanten, Beträge |
| `GET /api/bestellhistorie/{id}` | view_orders | 0 | Barcode-IDs sind Exemplare, keine Schüler |
| `GET /api/lieferanten` | view_orders | 0 | Händlerstammdaten |
| `POST /api/lieferanten` | create_orders | 0 | Echo der Händlerstammdaten |
| `PUT /api/lieferanten/{id}` | create_orders | 0 | Echo der Händlerstammdaten |
| `DELETE /api/lieferanten/{id}` | create_orders | 0 | 204 ohne Body |
| `POST /api/bestellungen` | create_orders | 0 | Status, Lieferantenname im Text |
| `GET /api/bestellungen/zulauf` | view_orders | 0 | Lieferant, Titel, Exemplar-IDs |
| `POST /api/bestellungen/suche` | view_orders | 0 | Katalog-/DNB-Titeldaten |
| `POST /api/bestellungen/bulk-receive` | create_orders | 0 | Exemplar-Barcodes, Titel |
| `PUT /api/bestellungen/{id}/bestaetigen` | create_orders | 0 | Status, Etikettengröße |
| `PUT /api/bestellungen/{id}/bestaetigungs-link` | create_orders | 0 | Link, Gültigkeit |

## routes_system.go

| Route | Recht | Stufe | Inhalt |
|---|---|---|---|
| `GET /api/benutzer` | manage_users | 0 | Personal-Konten, keine Schüler |
| `POST /api/benutzer` | manage_users | 0 | Personal-Konto |
| `PUT /api/benutzer/{id}` | manage_users | 0 | Personal-Konto |
| `DELETE /api/benutzer/{id}` | manage_users | 0 | Personal-Konto |
| `GET /api/einstellungen` | manage_settings | 0 | Systemeinstellungen |
| `PUT /api/einstellungen` | manage_settings | 0 | nur Status |
| `GET /api/einstellungen/sitzung` | Sitzung | 0 | zwei Zahlen: Minuten bis Theke leeren / Sperrbildschirm |
| `GET /api/ausweis-layout` | view_students | 0 | Design-JSON ohne Daten |
| `PUT /api/ausweis-layout` | manage_settings | 0 | nur Status |
| `GET /api/admin/settings/mail` | manage_settings | 0 | SMTP-Konfiguration |
| `PUT /api/admin/settings/mail` | manage_settings | 0 | SMTP-Konfiguration |
| `POST /api/admin/settings/mail/test` | manage_settings | 0 | Testmail-Status |
| `GET /api/admin/permissions` | manage_users | 0 | Rechte-Matrix |
| `PUT /api/admin/permissions` | manage_users | 0 | Rechte-Matrix (Ändern nur Admin) |
| `GET /api/admin/system/backup-status` | manage_settings | 0 | Backup-Status |
| `GET /api/admin/system/betriebsbereitschaft` | manage_settings | 0 | Befunde: Klassennamen, Zähler |
| `GET /api/audit` | audit_logs | 0 | Audit ohne details; Bearbeiter mit Vor-/Nachname (Personal, keine Schülerdaten) |
| `GET /api/audit/tresen-auskunft` | audit_details | 2 | Zweckgebundener Leseweg in audit_log.details (Betreiber-Entscheidung 01.09.2026): Barcode → Ausleihhistorie mit Entleiher-Klarname + Klasse. Ab Werk nur ADMIN; jeder Abruf protokolliert sich selbst in audit_logs (TRESEN_AUSKUNFT, mit IP) |
| `GET /api/mail-templates` | manage_settings | 0 | Vorlagentexte |
| `PUT /api/mail-templates/{id}` | manage_settings | 0 | Vorlagentexte |
| `GET /api/reports/overdue-pdf` | view_students | 3 | Elternbriefe als DIN-5008-Fensterkuvert-PDF: Name, ANSCHRIFT + Überfälliges (seit 01.09.2026 — Zweck „gedruckter Elternbrief bei Mahnung" laut VVT/SECURITY.md; vorher stand „Adresse unbekannt" im Fensterfeld) |
| `GET /api/print/rechnung/{schueler_id}` | view_students | 3 | Rechnung (Fensterkuvert): Name, Anschrift + Schadensbeträge (Anschrift seit 01.09.2026 verdrahtet — vorher hartkodiert leer) |
| `GET /api/print/mahnung/klasse/{klasse}` | view_students | 2 | Klassen-Mahn-PDF |
| `POST /api/admin/mahnungen/bulk-print` | view_students | 2 | Mahn-PDF, erhöht Mahnstufe |
| `GET /api/print/kontoauszug/{schueler_id}` | view_students | 2 | Kontoauszug mit Namen |
| `GET /api/dashboard/summary` | view_students | 0 | anonyme Aggregate |
| `GET /api/statistiken` | view_stats | 0 | Rankings/Kennzahlen ohne Personenbezug |
| `GET /api/systematics` | view_books | 0 | Sachgruppen |
| `POST /api/systematics` | edit_books | 0 | Sachgruppen |
| `PUT /api/systematics/{id}` | edit_books | 0 | Sachgruppen |
| `DELETE /api/systematics/{id}` | edit_books | 0 | Sachgruppen |
| `GET /api/faecher` | view_books | 0 | Fächerliste |
| `GET /api/readergroups` | view_students | 0 | Lesergruppen-Stammdaten |
| `GET /api/admin/auditlog` | manage_users | 2 | details-JSONB: schueler_id + Sperrgrund bei OVERRIDE_BLOCK |
| `GET /api/barcode/next` | edit_books | 0 | nächste Exemplarnummer |
| `GET /api/barcode` | view_books | 0 | Barcode-PNG aus Aufrufer-Eingabe |
| `GET /api/print/etikett/{id}` | view_books | 0 | Exemplar-Etikett |
| `GET /api/signaturen` | view_books | 0 | Signatur-Gruppen |
| `GET /api/signaturen/buecher` | view_books | 0 | Titel je Signatur |
| `GET /events` | Sitzung | 0 | SSE: UUIDs + Buchdaten, keine Namen |

## router.go

| Route | Recht | Stufe | Inhalt |
|---|---|---|---|
| `/api/books` | inventur-Mux | 0 | Delegation, Schutz im Inventur-Mux |
| `/api/books/` | inventur-Mux | 0 | Delegation |
| `/api/class-books` | inventur-Mux | 0 | Delegation |
| `/api/portal/` | inventur-Mux | 0 | Delegation (Lehrerportal-Lesesichten, Schutz im Inventur-Mux) |
| `/api/lookup/` | inventur-Mux | 0 | Delegation |
| `/api/admin` | inventur-Mux | 0 | Redirect in den Inventur-Mux |
| `/api/admin/` | inventur-Mux | 0 | Delegation (Spezialrouten oben gewinnen) |
| `/uploads/` | öffentlich | 0 | nur Cover; fotos/ seit 08.08.2026 abgeschafft (Fotos AES-verschlüsselt in DB) — Prod-Kontrolle: Verzeichnis muss weg sein |
| `POST /login` | öffentlich | 0 | Login (Rate-Limit); Antwort: Personal-Konto + Rechte |
| `GET /api/images/cover` | öffentlich | 0 | Cover-Proxy (SSRF-Allowlist) |
| `GET /api/csrf-token` | öffentlich | 0 | CSRF-Bootstrap |
| `POST /api/auth/refresh` | selbst-prüfend | 0 | Cookie-Erneuerung |
| `GET /api/auth/me` | selbst-prüfend | 0 | eigenes Personal-Konto + Rechte |
| `POST /api/auth/logout` | selbst-prüfend | 0 | nur Status |
| `GET /health` | öffentlich | 0 | Health-Check |
| `POST /api/import/littera` | manage_inventory | 0 | Import-Zähler |
| `POST /api/admin/import-bestand` | manage_inventory | 0 | Import-Zähler |
| `POST /api/admin/sync-covers` | manage_inventory | 0 | Job-Startmeldung |
| `GET /swagger/` | öffentlich | 0 | API-Doku (nur local/development) |
| `GET /swagger` | öffentlich | 0 | Redirect (nur local/development) |
| `/favicon.ico` | öffentlich | 0 | 204 |
| `/` | öffentlich | 0 | SPA-Auslieferung |

## inventur/api_routen.go

| Route | Recht | Stufe | Inhalt |
|---|---|---|---|
| `GET /uploads/` | öffentlich | 0 | FileServer auf uploads/ (nur Cover, s. router.go) |
| `GET /api/books` | inventur:view_books | 0 | Buchliste mit Verfügbar-Zähler, kein Ausleiher |
| `GET /api/books/{id}` | inventur:view_books | 0 | Einzelbuch |
| `GET /api/class-books` | inventur:view_books | 0 | Klassen-Buchlisten (Klassenname als Etikett) |
| `GET /api/lookup/` | inventur:view_books | 0 | externe ISBN-Metadaten |
| `GET /api/portal/lernmittel` | Sitzung | 0 | Buchliste mit Verfügbar-Zähler fürs Lehrerportal — dieselbe Antwort wie `GET /api/books`, kein Ausleiher, keine Schülerdaten |
| `GET /api/portal/klassensaetze` | Sitzung | 0 | Klassen-Buchlisten fürs Lehrerportal (Klassenname als Etikett, keine Schüler) |
| `GET /api/admin/` | inventur:edit_books | 0 | Export/Cover/Listen ohne Ausleiher |
| `POST /api/admin/` | inventur:edit_books | 0 | Buch-/Listen-Anlage |
| `PUT /api/admin/` | inventur:edit_books | 0 | Buch-/Listen-Update |
| `DELETE /api/admin/` | inventur:edit_books | 0 | Löschen ohne Ausleiher-Nennung |
| `POST /api/books/import` | inventur:edit_books | 0 | Import-Zähler |
| `POST /api/books` | inventur:edit_books | 0 | angelegtes Buch |
| `POST /api/books/` | inventur:edit_books | 0 | Cover-Aktionen |
| `PUT /api/books/` | inventur:edit_books | 0 | Buch-Update |
| `DELETE /api/books` | inventur:edit_books | 0 | nur Meldung |

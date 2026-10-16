# Invarianten-Katalog

**Zweck:** Eine einzige Quelle der Wahrheit dafür, _was im System immer wahr sein muss_ — und
auf welcher Ebene das heute durchgesetzt wird. Der Katalog ist die Grundlage für DB-Constraints,
Tests und Code-Reviews. Er wird gepflegt, nicht einmalig geschrieben.

**Methode — die entscheidende Frage je Invariante ist nicht „testen wir sie?", sondern
„auf welcher Ebene ist sie durchgesetzt?":**

| Ebene       | Bedeutung                             | Umgehbar?                                               |
| ----------- | ------------------------------------- | ------------------------------------------------------- |
| 🟢 **DB**   | CHECK / UNIQUE / FK / Enum / NOT NULL | Nein — strukturell unmöglich                            |
| 🟡 **Code** | Go-Handler/Service-Logik              | Ja, sobald ein zweiter Schreibpfad die Prüfung auslässt |
| 🔴 **Doku** | nur im Kommentar/Konzept              | Ja — reine Hoffnung                                     |

Ziel ist, kritische Invarianten von 🔴/🟡 nach 🟢 zu schieben; die 🟢-Invarianten sind in
CI gegen echtes Postgres abgesichert. Stand: 2026-10-08

> **Fundstellen werden benannt, nicht gezählt.** Die Spalte nennt Constraint-, Index-
> und Spaltennamen — keine Zeilennummern: Eine Zeilennummer zeigt nach dem Wachstum der
> Datei auf etwas anderes als gemeint, und das fällt beim Lesen nicht auf.
> `docs/invarianten_fundstellen_test.go` prüft, dass jeder genannte Name in
> `schema.sql` wirklich existiert.
>
> **Migrationsnummern sind ebenfalls keine haltbare Fundstelle.** Sie existieren als Datei
> weiter, auch wenn eine spätere Migration ihr Werk zurücknimmt — eine Existenzprüfung
> bliebe also grün (Migration 066 ersetzte Spalte **und** Index aus 058,
> `ist_standard` → `ist_hauptlieferant`). Deshalb gilt auch hier: **das lebende Objekt beim Namen nennen**, dann trägt
> das Gate die Prüfung. Migrationsnummern nur als Zusatz zur Herkunft.

---

## 1. Ausleihe (`ausleihen`)

| Invariante                                                                                                                                                                                                                                                     | Durchsetzung                                                                              | Fundstelle                                                                     |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Höchstens **eine aktive** Ausleihe je Exemplar/Gerät                                                                                                                                                                                                           | 🟢 partieller Unique-Index                                                                | `uniq_ausleihen_aktiv_exemplar`, `uniq_ausleihen_aktiv_geraet`                 |
| **Ein Entleiher**: `ausleihen.schueler_id` zeigt auf genau einen Leser; NULL heißt anonymisiert (Migration 125)                                                                              | 🟢 Fremdschlüssel auf `leser`                                                             | `ausleihen_schueler_id_fkey`                                                   |
| Genau **ein Objekt** (Exemplar XOR Gerät)                                                                                                                                                                                                                      | 🟢 CHECK                                                                                  | `check_loan_item`                                                              |
| Rückgabe nie vor Ausleihe                                                                                                                                                                                                                                      | 🟢 CHECK                                                                                  | `check_return_date`                                                            |
| Ein Leser mit Sperre am Leser (von Hand oder vom Programm für Ehemalige) leiht nichts — an der Theke nur die Rückgabe, aufgehoben wird in der Akte; beim Lernmittel zählt nur die Sperre von Hand, auch bei der Verlängerung (Entscheidung 28.09.2026); ein anonymisierter Datensatz bekommt nichts; Kollegen werden nie gesperrt (Entscheidung 24.09.2026) | 🟡 ein Prüfweg für Buch, Gerät und Nachbuchen; Einzelverlängerung und Frist von Hand fragen denselben, Klassenverlängerung und LMF-Plan eine gemeinsame Bedingung + PG-Test je Regel, am Rückbau rot gesehen | `internal/service/ausleih_sperren.go` (`pruefeAusleihSperren`, `PruefeSperreAmLeser`, `pruefeSperrschalter`), `api/theke/ausleihe.go` (`pruefeAusleihSperre`), `repository/lmf_termine.go` (`sqlSchulbuchFristGehtMit`), `internal/service/ausleih_sperren_pg_test.go`, `api/verlaengerung_sperre_schulbuch_pg_test.go` |
| Hinweise — unbezahlte Forderung aus jedem Topf und Überfällig-Automatik (≥ `UeberfaelligSperreAbMedien`) — halten Buch der Schülerbücherei und Gerät an, übergehbar mit `edit_students`, im Protokoll erst, wenn die Ausleihe steht; ein Lernmittel kennt keine automatische Sperre (Antwort der Schule vom 22.09.2026) | 🟡 | `internal/service/ausleih_sperren.go` (`pruefeHinweise`, `protokolliereUebergangen`) |
| Ausleih-Limit `max_ausleihen_schueler` (LMF + eigene Rückgabe ausgenommen)                                                                                                                                                                                     | 🟡 Unit-Tests, Abdeckung 90 % (23.09.2026)                                                | `internal/service/ausleihe_am_leser.go` (`pruefeSchuelerAusleihlimit`), `ausleihe_am_leser_test.go` |
| Abholbereit reserviertes Exemplar geht nicht an Dritte                                                                                                                                                                                                         | 🟡 Unit-Tests, Abdeckung 90,9 % (23.09.2026)                                              | `internal/service/ausleihe_am_leser.go` (`pruefeVormerkungKonflikt`), `repository/ausleihe_schranken.go` (`FindeVormerkungAmExemplar`), `internal/service/ausleihe_vormerkung_pg_test.go` |
| Doppel-Scan desselben Exemplars → sauberer Konflikt (409 statt 500)                                                                                                                                                                                            | 🟢 Unique-Index + 🟡 `deuteAnlegeFehler`, Abdeckung 100 % (23.09.2026)                      | `internal/service/ausleihe_faelle.go` (`deuteAnlegeFehler`)                 |
| **Sperrreihenfolge** Schüler → Ausleihe → Exemplar: Wer Zeilen sperrt, sperrt zuerst die Zeile des Lesers (`leser`), dann `ausleihen`, dann `buecher_exemplare` — Online-Scan und Nachbuchen gleich, sonst verklemmen sie sich gegeneinander. Die Datenbank selbst hält sie nicht: Der Rückgabe-Trigger (Migration 137) sperrt die Leserzeile des Ausleihers NACH der Ausleihe. Geben zwei Kinder an zwei Theken zugleich je das Buch des anderen ab, bricht Postgres eine der beiden Transaktionen ab (40P01). Entschieden am 24.09.2026: bleibt so. Die Theke meldet den Fehler, erneutes Scannen bucht, das Nachbuchen holt es in der nächsten Runde nach; die Abhilfe, der Stempel in einer eigenen Tabelle, kommt mit dem nächsten Umbau der Karenz-Uhr | 🟡 Konvention der Schreibpfade; der Rückgabe-Trigger hält sie nicht, ein Gate gibt es nicht | `internal/service/ausleihe_am_leser.go` (`zaehleAktiveSchuelerAusleihen`), `internal/service/nachbuchen.go`, `repository/ausleihe_zeilensperren.go` (`SperreLeserzeile`, `SperreExemplarzeile`), `repository/ausleihe.go` (`AusleihAbfragen.SperreOffene`, `stempleBewegungZum`), `trg_leser_stempel_rueckgabe`, `repository/raster_2309_pg_test.go` |
| Jede Bewegung eines Exemplars (Ausleihe, Rückgabe, Rückholen, Aussonderung, Reaktivierung, jeder Wechsel von `ist_ausleihbar`) setzt `letzte_bewegung_am`, und der Stempel läuft nie rückwärts (auch nicht mit früherem Scan-Zeitpunkt oder aus einer früher begonnenen Transaktion); jede Ausleihe trägt `erfasst_am` (Scan-Zeitpunkt, höchstens Serverzeit) | 🟢 Zustandswechsel des Exemplars: Trigger (Migration 132), der einen vom Schreiber gesetzten Stempel stehen lässt; 🟡 Ausleihe und Rückgabe: von den Schreibern gesetzt (`sqlStempelVor`), PG-Test je Bewegung und für beide Rückwärts-Wege | `migrations/116_bewegungsstempel.sql`, `migrations/132_bewegungsstempel_bei_zustandswechsel.sql`, `repository/ausleihe.go`, `repository/bewegungsstempel_pg_test.go`  |
| Ein Idempotenz-Schlüssel der Theke wird genau einmal gebucht — online oder beim Nachbuchen: Die Nachbuch-Tür liest, was unter dem Schlüssel steht (nichts → buchen und Ergebnis ablegen; Reservierung → „wiederholen"; eigenes Ergebnis → dasselbe zurück; Online-Ausleihe/-Rückgabe → „bereits_gebucht"; nur Fremdrückgabe → Ausleihe ab der Rückgabe nachholen, Wächter entscheidet) und übernimmt eine Online-Antwort nur, wenn sie genau die gelesene ist; Scan-Zeitpunkte werden um den gemessenen Uhrversatz des Rechners umgerechnet | 🟡 Code + PG-Tests je Fall, Rückbau-Proben | `api/theke/nachbuchen_schluessel.go`, `api/theke/nachbuchen_schluessel_pg_test.go`, `api/theke/nachbuchen_uhr_pg_test.go`, `repository/idempotenz.go` (`UebernimmIdempotenzAntwort`) |
| `nachbuch_meldungen` (Migration 117): Ergebnis ist eines von sieben Wörtern; ein Schlüssel = eine Meldung; quittierte fallen nach der Lesehistorie-Frist, höchstens 30 Tage; offene haben keine Frist, der Wächter nennt sie nach 14 Tagen; Personenspalten wandern beim Zusammenführen und werden getilgt | 🟢 CHECK + UNIQUE; 🟡 Löschjob und Wächter aus EINEM Prädikat                              | `chk_nachbuch_ergebnis`, `uniq_nachbuch_meldungen_schluessel`, `repository/loeschfristen.go` (`praedikatNachbuchMeldungen`) |
| Leser, die keine Schüler sind (`art` ≠ `schueler`): Dauerleihe (`ist_handapparat`) mit gespeicherter Frist von einem Jahr; für sie gelten weder Ausleih-Limit noch Sperrprüfung — auch am Gerät —, und die Überfällig-Zählung lässt Dauerleihen aus (Entscheidung 16.09.2026)    | 🟡 Unit-Tests, Abdeckung 93,8 % (23.09.2026)                                              | `internal/service/ausleih_vorgang.go` (`ladeVorgang`), `ausleih_sperren.go` (`istKollegium`), `repository/ueberfaellige_ausleihen.go` (`ZaehleUeberfaelligeAusleihen`), `ausleihe_faelle.go` (`legeAusleiheAn`) |
| Eine Frist nach Tagen (Buch, Medium, Gerät, Verlängerung, Abholfrist einer Vormerkung) endet an einem Schultag: Fällt sie auf ein Wochenende, einen Feiertag Hessens oder in die Ferien, gilt der nächste Schultag. Stichtage (Lernmittel, LMF-Plan, Ferien-Leseclub), von Hand gesetzte Fristen und die Jahresfrist der Dauerleihe rücken nicht (Entscheidung 24.09.2026, Abholfrist 28.09.2026) | 🟡 eine Rechnung für alle fünf + Unit- und PG-Tests mit fester Uhr | `pkg/lmfplan/tagesfrist.go` (`Ferientabelle.Tagesfrist`), `repository/vormerkung_abholfrist.go` (`Abholfrist`), `pkg/lmfplan/uebrige_ferien.go` |
| Herbst-, Weihnachts- und Osterferien Hessen reichen mindestens zwei Jahre voraus | 🟡 Selbstprüfung „Ferien für Leihfristen" + Horizont-Test (rot ab Januar 2029) | `pkg/lmfplan/uebrige_ferien.go` |
| **Mahnstufe steigt NUR beim Druck der Mahnbriefe aus der Auswahl** („Mahnbriefe drucken"), NIE beim Mail-Versand oder beim Ausdruck der Liste („Liste drucken"). Der Druck zählt und liest die Briefe in DERSELBEN Tx (Papier == DB) nach derselben Bedingung: das offene Buch eines Schülers außerhalb des Papierkorbs, dessen Frist abgelaufen ist. Eine Ausleihe steigt höchstens einmal je Kalendertag der Schule; ein zweiter Druck am selben Tag liefert den Brief und zählt nicht | 🟡 nur `ErhoeheMahnstufe` erhöht `mahnstufe`, gerufen vom Druck in `api/mahnwesen/mahnbriefe.go`; alle Mail-Pfade bewusst nicht; PG-Tests am Router (`TestMahnbriefDruck_NurMitAbgelaufenerFrist`) und an der Tür (`TestMahnbriefeDrucken_NurOffeneHochzaehlen`) | `api/mahnwesen/mahnbriefe.go`, `repository/mahnbriefe.go` (`ErhoeheMahnstufe`, `sqlMahnbriefe`, `LadeMahnbriefe`), `api/mahnwesen/mahnlisten_versand.go`, `api/mahnwesen/mahnwesen_mail.go` |
| **Mahnliste geht an die Klassenleitung, nie an Schüler** — je Klasse eine Mail an genau eine Adresse (kein TO/CC über mehrere Betroffene)                                                                                                                      | 🟡 `versendeKlassenMahnungen` überspringt Klassen ohne hinterlegte Adresse; getestet      | `api/mahnwesen/mahnlisten_versand.go`, `api/mahnwesen/mahnlisten_versand_test.go`                |
| **Ein Ehemaliger geht an keine Klassenleitung.** Die Mahnliste führt Ehemalige als eigene Gruppe, gleich welche Klasse sie zuletzt trugen; der Mahnlauf lässt die Gruppe fallen, auch ausdrücklich gewählt und mit eigener Adresse, und die Mahnliste einer Klasse nennt nur ihre heutigen Schüler | 🟡 Test am Router mit LUSD-Markierung, Versetzung, Mail und Druck; fünf Rückbauten rot | `repository/mahnwesen_ueberfaellige.go`, `api/mahnwesen/mahnlisten_versand.go`, `api/mahnwesen/mahnwesen_mail.go`, `api/mahnwesen/mahnwesen_ehemalige_pg_test.go` |
| **Massenversand-Auswahl: leer heißt niemand, nicht alle.** Fehlendes `klassen`-Feld = alle (alter Vertrag), leeres Array = 400                                                                                                                                 | 🟡 `leseKlassenVersandAnfrage`, Zeiger-Semantik + Test                                   | `api/mahnwesen/mahnlisten_versand.go`                                                   |
| **`override_email` ist die einzige Abweichung** von „jede Lehrkraft nur die eigene Klasse": Ein Empfänger sieht dann mehrere Klassen (Vertretung/Sekretariat). Weiterhin eine Mail je Klasse, kein Sammel-PDF — und die Adresse steht im Klartext im Audit-Log | 🟡 Handler + `protokolliereKlassenVersand` (zwei Einträge: Absicht vor, Ergebnis nach Versand) | `api/mahnwesen/mahnlisten_versand.go`, `frontend/src/lib/Mahnwesen.svelte`              |

**Bewertung:** Sehr robust. Die datenkritischen Invarianten sind bereits auf DB-Ebene. Die
Geschäftsregeln (Sperre/Limit/Overdue) liegen bewusst im Code (brauchen Kontext + Override) —
Risiko nur, falls je ein _zweiter_ Checkout-Pfad entsteht, der die Validierung nicht aufruft.

---

## 2. Leser (`leser`) und die Sicht `schueler`

Seit Migration 124 heißt die Tabelle `leser` und führt ALLE Leser; `schueler` ist eine
Sicht darauf (`WHERE art = 'schueler'`, `WITH CHECK OPTION`). Wer wirklich nur Schüler
meint — Klassenlisten, Mahnlauf, LUSD, Versetzung, Abgänger —, liest die Sicht und behält
seine Bedeutung, ohne angefasst zu werden. Wer den Menschen vor sich meint, liest die
Tabelle.

| Invariante                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Durchsetzung                                                                                                                                   | Fundstelle                                                                                                    |
| --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| **Die Tabelle für Leser, die Sicht für Schüler.** Theke, Leserdatei, Akte, Ausweis- und Etikettendruck, Sperre und Passbild lesen `leser`; alles, was wirklich Schüler meint, die Sicht. Eine Abfrage, die die falsche Seite wählt, ist entweder ein Kollege in einer Klassenliste oder ein Kollege, den es für die Theke nicht gibt | 🟡 an jeder Fundstelle begründet + PG-Tests je Weg | `repository/leser_suche.go`, `repository/leser_liste.go`, `repository/leser_etiketten.go`, `api/leser/leserakte.go`, `api/leser/leser_sperre.go`, `api/leser/foto.go`, `api/leser/leserdatei_pg_test.go` |
| Barcode eindeutig **unter aktiven** (Soft-Delete gibt ihn für die Eingabe von Hand frei; das Programm vergibt ihn nie wieder, Migration 146). **Nullbar seit Migration 123**: Ein Kollege hat erst eine Nummer, wenn eine vergeben ist — die Pflicht gilt gepaart an der Art | 🟢 partieller Unique-Index + `chk_leser_schueler_pflichtfelder` | `leser.barcode_id`, `uniq_schueler_barcode_active` |
| Ausweisnummer eindeutig **über alle Leser** — Schüler wie Kollegium; die Theke sucht bei jeder Nummer unter allen, gelöschte Leser geben die Nummer für die Eingabe von Hand frei (das Programm vergibt sie nicht wieder, nächste Zeile). Seit Migration 125 stehen alle Leser in einer Tabelle; der Trigger aus Migration 118 ist damit entfallen, die Zusage blieb | 🟢 partieller Unique-Index + PG-Tests über Datenbank und Handler | `uniq_schueler_barcode_active`, `db/bedingung_ausweis_ueber_personen_pg_test.go`, `api/ausweis_ueber_personen_pg_test.go` |
| **Eine A-Nummer vergibt das Programm nur einmal** — der Generator (`ausweis_nummer_start`) und die Ersatzvergabe der Littera-Übernahme, auch nachdem die Nummer verschwunden ist: endgültig gelöscht, anonymisiert, zusammengeführt, geleert, umgeschrieben. Von Hand bleibt sie eintragbar | 🟢 Tabelle `ausweisnummern_ausgeschieden` + Trigger an `leser` + PG-Tests je Weg | `trg_leser_ausweisnummer_ausgeschieden`, `repository/ausweisnummer_kommt_nie_wieder_pg_test.go`, `internal/littera/schreiber_personen_pg_test.go` (TestErsatznummerKommtNieWieder) |
| Name + Geburtsdatum eindeutig — in der Normalform `suchnorm` wie der LUSD-Schlüssel (Migration 108), **nur bei bekanntem Geburtsdatum** (sonst wären namensgleiche Schüler ohne Datum Duplikate: Zwillings-Blockade, Migration 048)                                                                                                                                                                                                                                                     | 🟢 partieller Unique-Index                                                                                                                     | `unique_schueler_name_gebdatum`                                                                               |
| **LUSD-Namensschlüssel = Normalform `suchnorm`** (Go-Zwilling `repository.suchnorm` ≡ SQL): Müller ≡ Mueller ≡ MÜLLER; ein doppelt belegter Schlüssel gilt als mehrdeutig und wird nicht angefasst — weder Zuordnung noch Abgang                                                                                                                                                                                                                                                                   | 🟢 Paritätstest (Korpus + Zeichen-Durchlauf) und Import-PG-Test                                                                                | `repository/suchnorm.go`, `suchnorm_pg_test.go`, `api/leser/lusd_namensschluessel_umlaut_pg_test.go`          |
| LUSD-ID eindeutig **unter aktiven** (Soft-Delete gibt sie frei)                                                                                                                                                                                                                                                                                                                                                                                                                         | 🟢 partieller Unique-Index                                                                                                                     | `uniq_schueler_lusd_id_active`                                                                                |
| **Klasse, Abgangsjahr und Ausweis sind Pflicht — für einen SCHÜLER.** Die Pflicht ist seit Migration 123 nicht aufgegeben, sondern an die Art GEPAART: Ein Schüler ohne Klasse fällt aus jeder Klassenliste und jeder Mahnung, ein Kollege mit Klasse stünde im LUSD-Abgleich. Der Anlege-Handler prüft dieselbe Paarung, nicht die einzelnen Werte | 🟢 `chk_leser_schueler_pflichtfelder` + `pruefeLeserAngaben` + PG-Test | `migrations/123`, `api/leser/leser_anlegen.go`, `api/leser/leser_anlegen_pg_test.go` |
| **Gesperrt ⇒ Sperrgrund vorhanden** (kein grundloser „Zombie-Sperre"-Zustand; Personal sieht immer das _warum_). Automatische Sperr-Pfade setzen den Grund mit; `is_manually_blocked` läuft separat                                                                                                                                                                                                                                                                                     | 🟢 `chk_schueler_block_reason`                                                                                                                 | `schema.sql`, Migration 047, `repository/lusd_import.go`, `api/leser/schuljahreswechsel.go`                                |
| **[G1] Adress-/Kontaktdaten** aus LUSD importiert, Zweck Rechnung/Mahnung, bei Anonymisierung gelöscht                                                                                                                                                                                                                                                                                                                                                                                  | 🟢 **entschieden (B)** — Import + Löschung umgesetzt                                                                                           | `repository/lusd_import.go`, `schueler.strasse`/`plz`/`ort`/`eltern_email`                                                |
| Bei der Anonymisierung fällt der **Freitext erledigter** Schadensfälle: die Beschreibung und der getippte Grund einer Stornierung, an der Forderung und im Eintrag `STORNIERUNG` des Protokolls; der Fall selbst (Betrag, Datum, bezahlt/storniert) bleibt als Beleg, eine **offene** Forderung behält ihre Begründung | 🟡 Statement in der EINEN Spurenliste + PG-Test in beide Richtungen; der Eintrag trägt die Kennung des Lesers | `repository/spur_tilgung.go` (`spurTilgungen`), `repository/schadensfall_freitext_tilgung_pg_test.go`, `api/schaden/stornierungsgrund_tilgung_pg_test.go` |
| Neben der Kennung eines Lesers (`schueler_id`) überlebt im Protokoll **kein Wert seiner Leserzeile und kein Freitext über ihn** die Anonymisierung; jeder Schlüssel daneben ist als getilgt oder bleibt eingeordnet | 🟡 eine Schlüsselliste für beide Protokolle + AST-Ratsche über jede Map mit `schueler_id` + PG-Tests | `repository/protokoll_personenbezug.go`, `repository/protokoll_personenbezug_test.go`, `repository/titel_loeschspur_tilgung_pg_test.go`, `api/leser/sperrgrund_tilgung_pg_test.go` |
| **DSGVO-Retention-Kette schliesst:** Abgänger mit offenen Vorgängen → gesperrt (Name bleibt); ohne Vorgänge → für die Karenzzeit (`abgaenger_karenz_tage`, Uhr = spätester von Abgang `abgaenger_seit` und letztem abgeschlossenem Vorgang `letzter_vorgang_am` — Rückgabe oder Schadensabschluss, seit Migration 137 am Leser gestempelt, weil die Lesehistorie-Befristung die Zuordnung der Ausleihe löst) gesperrt, dann anonymisiert; nach Rückgabe/Bezahlung → Karenz → anonymisiert → ab Stichjahr endgültig gelöscht (nur anonymisierte Zeilen; der Löschjob läuft nach der Anonymisierung). Import, Job und Wächter teilen das Prädikat | 🟡 `abgaenger_jahr` aufs Abgangsjahr + `PraedikatAnonymisierung(karenz, kulanz)` + Cronjob `LoescheAbgaengerEndgueltig` (echte Löschung, nicht Soft-Delete) | `internal/lusd/anwenden.go`, `repository/lusd_import.go`, `jobs/cron_dsgvo.go`, `repository/loeschfristen.go`, `repository/leser_loeschen.go`          |
| **Leer heißt löschen — aber nur, wo Löschen erlaubt ist.** Optionale Stammdaten (Anschrift, Eltern-Mail) werden mit leerem Wert auf NULL gesetzt; Pflichtfelder (Vor-/Nachname, Klasse, beim Schüler die Ausweisnummer) lehnen den leeren Wert mit 400 ab; eine geleerte Ausweisnummer eines Kollegen wird NULL, bei aktivem Konto zieht die Datenbank eine neue (Migration 145). Ein **fehlendes** Feld heißt weiterhin „unverändert“                                                  | 🟢 `leerbar` (`repository/leser_aenderung.go`) + Pflichtfeld-Wächter, PG-Test über den laufenden Handler                                                                   | `api/leser/leser_aendern.go`, `api/leser/schueler_feld_leeren_pg_test.go`, `frontend/src/lib/useLeserakteFormular.test.js` |
| Manuelles Löschen: Soft-Delete (Papierkorb, `LegeLeserInPapierkorb`) vs. endgültig (`LoescheLeserEndgueltig`, Recht `manage_students_admin`) sind getrennt                                                                                                                                                                                                                                                                                                                                                | 🟡 Restore hebt Lösch-Sperre auf; Purge blockiert bei offenen Vorgängen                                                                        | `api/leser/papierkorb.go`, `repository/leser_loeschen.go`                                                      |

**Hinweis (Retention):** Die Kette vom Abgang bis zur Löschung hält nur, wenn jedes Glied
seinen Teil schreibt: Die Sperre beim Abgang setzt `abgaenger_jahr`, sonst erfasste der
Löschlauf den Abgänger nie, und der Löschlauf löscht endgültig statt in den Papierkorb
(Tests: `TestAbgaengerRetentionKette`, `TestAbgaengerLoeschung_Loescht`).

**Hinweis (Löschen von Feldern):** Anschrift und Eltern-E-Mail lassen sich einzeln leeren;
Vorname, Nachname, Ausweisnummer und Klasse lehnt die Tür leer mit 400 ab
(`api/leser/schueler_feld_leeren_pg_test.go`). `geburtsdatum` bleibt bewusst unlöschbar
(Schlüssel des LUSD-Abgleichs).

---

## 3. Buch: Titel (`buecher_titel`) & Exemplar (`buecher_exemplare`)

| Invariante                                                                                                                                              | Durchsetzung                                     | Fundstelle                                                                             |
| ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------- |
| ISBN eindeutig (wo gesetzt), über alle Schreibweisen und beide Längen: Die Datenbank bringt jede geschriebene ISBN in eine Normalform — ohne Trennzeichen, eine zehnstellige mit richtigem Prüfzeichen dreizehnstellig (Migration 133 und 157) | 🟢 UNIQUE, Trigger `trg_titel_isbn_normalform` | `buecher_titel.isbn`, `repository/isbn_normalform_pg_test.go` |
| Titeltexte (Titel, Untertitel, Autor, Verlag) in einer Form gespeichert: kein Leerraum am Rand, Leerraum in Folge ein Leerzeichen, Umlaute zusammengesetzt (Unicode NFC) — Littera führt Titel mit zwei Leerzeichen und mit geschütztem Leerzeichen, die DNB liefert Umlaute zerlegt; der getippte Suchtext traf sie nicht (Migration 154 und 160) | 🟢 Trigger `trg_titel_text_normalform` | `repository/titeltext_normalform_pg_test.go`, `inventur/titel_suche_leerraum_pg_test.go`, `api/portal/opac_nfc_pg_test.go` |
| Exemplar-Barcode eindeutig                                                                                                                              | 🟢 UNIQUE NOT NULL                               | `buecher_exemplare.barcode_id`                                                         |
| Systemvergebene Exemplarnummern aus EINER Quelle (`barcode_seq`, `B-`) — alle fünf Wege, kein zweiter Kreis (Migration 105)                             | 🟢 PG-Tests                                      | `api/katalog/barcode_vergabe_pg_test.go`, `inventur/bestandskorrektur_nummernkreis_pg_test.go` |
| Exemplar hängt an existierendem Titel                                                                                                                   | 🟢 FK ON DELETE CASCADE                          | `buecher_exemplare.titel_id`                                                           |
| **[G4]** Jahrgang am Titel ganz oder gar nicht: „von“ und „bis“ beide gesetzt oder keins, 1–13, „von“ nicht über „bis“. Die „Klasse“ (`grade_level`) entfiel mit Migration 163, der Bestand am Titel (`stock`) mit Migration 073 | 🟢 `chk_jahrgang_spanne` | `migrations/162`, `migrations/163` |
| **[G3]** Aussonderungs-Grund strukturiert: im Umlauf = NULL, ausgesondert = genau ein Wert aus {VERLUST, BESCHAEDIGUNG, AUSSORTIERT, BESTANDSKORREKTUR} | 🟢 `chk_aussonderung_grund`                      | `migrations/043`                                                                       |
| **[G2]** `cover_status` ∈ {PENDING, FOUND, FAILED, NOT_FOUND}                                                                                           | 🟢 `chk_cover_status`                            | `migrations/041`                                                                       |
| `medientyp` — **bewusst ohne CHECK**: offenes, per Formular frei eingebbares Vokabular                                                                  | 🟡 Formular                                      | `migrations/040` (Begründung im Kopf)                                                  |

---

## 4. Schaden & Gebühr (`schadensfaelle`)

| Invariante                                                        | Durchsetzung                     | Fundstelle                                          |
| ----------------------------------------------------------------- | -------------------------------- | --------------------------------------------------- |
| Betrag ≥ 0                                                        | 🟢 CHECK                         | `check_positive_amount`                             |
| **Ein Verantwortlicher**: `schadensfaelle.schueler_id` zeigt auf genau einen Leser; NULL heißt anonymisiert (Migration 125) | 🟢 Fremdschlüssel auf `leser` | `schadensfaelle_schueler_id_fkey` |
| Genau ein betroffenes Objekt                                      | 🟢 CHECK                         | `check_damage_item`                                 |
| Stornierung revisionssicher (wer/wann/warum)                      | 🟡 Spalten `storniert_*` + Audit | `repository/forderung_erledigen.go` (`StorniereForderung`) |
| Storniert ⇒ `ist_bezahlt` gesetzt: Wer offene Forderungen zählt, fragt nur `ist_bezahlt = false` (Akte und Theke über `repository.ZaehleOffeneSchaeden`, Inventur-Sperre, LUSD-Abgleich) | 🟡 Verabredung: beide Schreiber von `storniert_am` setzen das Feld im selben UPDATE, eine CHECK-Bedingung gibt es nicht | `repository/forderung_erledigen.go`, `repository/bescheid_rueckkehr.go` |

---

## 5. Vormerkung (`vormerkungen`)

| Invariante                                         | Durchsetzung               | Fundstelle                                |
| -------------------------------------------------- | -------------------------- | ----------------------------------------- |
| Ein Schüler merkt einen Titel höchstens einmal vor | 🟢 UNIQUE                  | `vormerkungen (titel_id, schueler_id)`    |
| **[G2]** Status ∈ {`wartend`, `abholbereit`}       | 🟢 `chk_vormerkung_status` | `migrations/040`                          |
| Bereitgestelltes Exemplar existiert                | 🟢 FK ON DELETE SET NULL   | `vormerkungen.bereitgestellt_exemplar_id` |

**Hinweis:** Das Vokabular ist bewusst zweiwertig — erfüllte Vormerkungen werden
**gelöscht**, nicht auf einen Endstatus gesetzt (geprüft vor Migration 040).

---

## 6. Klassensatz-Reservierung (`klassensatz_reservierungen`)

| Invariante                   | Durchsetzung                | Fundstelle                            |
| ---------------------------- | --------------------------- | ------------------------------------- |
| Hängt an existierendem Titel | 🟢 FK CASCADE               | `klassensatz_reservierungen.titel_id` |
| Lebenszyklus offen/erledigt  | 🟡 Boolean                  | `klassensatz_reservierungen.erledigt` |
| **[G4]** `anzahl ≥ 1`        | 🟢 `chk_ksr_anzahl_positiv` | `migrations/039`                      |

---

## 7. Bestellung (`bestellungen_verlauf` / `_positionen`)

| Invariante                                                                                                                     | Durchsetzung                                                            | Fundstelle                                                                        |
| ------------------------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Position hängt an existierender Bestellung                                                                                     | 🟢 FK CASCADE                                                           | `bestellungen_positionen.bestellung_id`                                           |
| Nur Positionen mit Menge > 0 werden bestellt                                                                                   | 🟡 Go-Guard                                                             | `internal/service/bestellung_anlegen.go` (`verarbeitePosition`)                |
| **[G4]** `menge ≥ 1`, `einzelpreis ≥ 0`, `gesamtbetrag ≥ 0`, `anzahl_exemplare ≥ 0`                                            | 🟢 4 CHECKs                                                             | `migrations/039`                                                                  |
| Bestellbedarf meint **Lernmittel**: Freihandbestand sind bewusste Einzelstücke (Prüf-/Leseexemplare) und wird nie „aufgefüllt" | 🟡 Default `?type=lmf` + Test                                           | `api/bestellung/bestellbedarf.go`, `api/bestellung/bestellbedarf_test.go`                   |
| Bestätigung einer Bestellung gilt genau einmal                                                                                 | 🟢 `WHERE bestaetigt_am IS NULL` (409 statt Überschreiben)              | `api/bestellung/bestellbestaetigung.go` (`bestaetigeBestellung`)                     |
| Eine Referenznummer wird **nie zweimal** vergeben (Bescheid)                                                                   | 🟢 UNIQUE + gesperrter Zähler (`UPDATE … RETURNING`) + PG-Test parallel | `uniq_bescheid_nummer`, `schadensersatz_nummern`, `api/schaden/bescheid_pg_test.go`       |
| Eine Forderung steht auf **höchstens einem** Bescheid                                                                          | 🟡 WHERE-Bedingung der Zuordnung + PG-Test (409)                        | `repository/bescheid_anlegen.go` (`ordnePositionenZu`)                                    |
| Der Nachdruck eines Bescheids ergibt **dasselbe Blatt** (Nummer, Betrag, Anschrift)                                            | 🟡 Empfänger-Snapshot + PG-Test am PDF                                  | `schadensersatz_bescheide.empfaenger_snapshot`, `api/schaden/bescheid_pg_test.go`         |
| Übergabe an die Aufsicht erst **nach Fristablauf**, genau einmal und **nur mit offener Forderung**                             | 🟡 `WHERE` = `bescheidFristAbgelaufen` + PG-Test (bezahlt → 409)        | `repository/bescheid.go` (`Uebergebe`)                                            |
| Ein Bescheid ohne offene Forderung ist **erledigt** (abgeleitet, nicht gespeichert — Bezahlen/Storno laufen über die Forderung) | 🟡 Leser-Ableitung `bescheidHatOffenePosition` + PG-Test               | `repository/bescheid.go` (`bescheidSpalten`)                                      |
| Höchstens ein **Haupt**lieferant                                                                                               | 🟢 Partieller Unique-Index                                              | `idx_lieferanten_ein_hauptlieferant`                                              |
| Der Topf einer Bestellung ist `land` oder `schultraeger` (oder NULL bei Alt-Bestellungen)                                      | 🟢 CHECK                                                                | `bestellungen_verlauf_mittel_check`                                               |
| **Neue** Bestellungen tragen immer einen Topf — die Datenbank lässt NULL zu (Alt-Bestand), verboten ist es nur an der Tür      | 🟡 Go-Guard (400) + PG-Test                                             | `pkg/mitteltopf` (`ErrUngueltig`), `api/bestellung/bestellung_mittel_pg_test.go`             |
| Ein Bestätigungs-Token gehört zu genau einer Bestellung, gespeichert nur als Hash                                              | 🟢 Partieller Unique-Index auf `bestaetigungs_token_hash`               | `migrations/063`                                                                  |
| Die Etikettenseite des Links zeigt nur Exemplare DIESER Bestellung mit Vorab-Barcode                                           | 🟡 SQL-Filter + PG-Test                                                 | `api/bestellung/bestellbestaetigung_etiketten.go`, `api/bestellung/bestellbestaetigung_ablauf_pg_test.go` |

**Hinweis zum Bestellbedarf:** Der Auslöser ist die Einstellung `bestellbedarf_schwelle`
(Vorgabe 3, in der Oberfläche änderbar, abschaltbar über `bestellbedarf_warnung_aktiv`),
nicht der Meldebestand am Titel: Mit ihm bestünde die Liste zu ~99 % aus Titeln, die niemand
nachbestellen will (gemessen: 12.079 von 12.707 Titeln), weil alle Titel die Vorgabe 5
tragen, der Median aber bei 1 Exemplar liegt. `meldebestand` wird nur informativ
mitgeliefert (`repository/bestellbedarf.go`). Die Liste zählt am Buch: Auflagen mit
derselben `werk_id` (Migration 148) bilden eine Zeile, ihre Summe steht gegen die Schwelle
(`TestLadeBestellbedarf_ZaehltAmBuch`).

---

## 8. Gerät (`geraete`) & Inventur

| Invariante                                                                                                                                                                                                                         | Durchsetzung                                                                      | Fundstelle                                                                    |
| ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| Gesperrtes/ausgesondertes Gerät leiht nicht                                                                                                                                                                                        | 🟡                                                                                | `internal/service/geraet.go` (`ladeGeraet`)                           |
| **Ein fehlendes `ist_ausleihbar` heißt „unverändert“, nicht „ausleihbar“.** Der Bearbeiten-Dialog kennt die Defekt-Markierung nicht (eigener Knopf); ohne diese Regel gab jede Stammdaten-Korrektur ein defektes Gerät wieder frei | 🟢 Zeiger + `COALESCE` im UPDATE, PG-Test in beide Richtungen                     | `api/katalog/geraete.go`, `repository/geraete.go`, `api/katalog/geraete_pg_test.go` |
| Inventur-Fortschritt ist **session-gebunden** (nicht global): parallele Inventuren überschreiben sich nicht                                                                                                                        | 🟢 `inventur_sessions` + `inventur_erfassungen`, partieller Unique-Index je Scope | `migrations/045`, `repository/inventurlauf_pg_test.go`                  |
| Ein aktuell **verliehenes** Buch gilt bei der Inventur nie als Verlust                                                                                                                                                             | 🟡 Scope-Bedingung (`NOT EXISTS` aktive Ausleihe)                                 | `repository/inventurlauf_abschluss.go`                                       |

---

## 9. Auth & Rollen

| Invariante                                                                                                                                                                                                    | Durchsetzung                                          | Fundstelle                                                                                          |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| `benutzer.rolle` ∈ Enum — heute `admin`, `leitung`, `kollegium`, `mitarbeiter`, `helfer` (kleingeschr.; `leitung` seit Migration 121)                                                                                                                  | 🟢 ENUM                                               | `benutzer_rolle`                                                                                    |
| **[G5]** **Genau eine** Quelle für die Rolle eines Benutzers: `benutzer.rolle`                                                                                                                                | 🟢 Legacy-Tabelle entfernt + Test verhindert Rückkehr | `migrations/044`, `rollen_vokabular_pg_test.go`                                                     |
| **Ausleihen darf jeder aktive Leser** — die Rolle sagt, was jemand im Programm TUN darf, nicht ob er Bücher bekommt (Migration 125) | 🟢 eine Ausweis-Abfrage + PG-Tests | `repository/leser_suche.go` (`LeserAbfragen.FindeZuBarcode`), `repository/leser_ausweis_pg_test.go`, `internal/service/geraet_lehrkraft_pg_test.go` |
| **Jedes Konto hat eine Leserzeile** — sonst fände die Theke die Person nicht; Konten entstehen an fünf Stellen, die Zusage steht deshalb in der Datenbank | 🟢 Trigger + PG-Test | `trg_benutzer_hat_leserzeile`, `db/bedingung_konto_leserzeile_pg_test.go` |
| **Theken-Suche `GET /api/search` nur aus der Omnibox**; Suchfelder außerhalb der Theke haben eigene Türen mit dem Recht ihrer Daten (`/api/buecher/titel/suche` view_books, `/api/schueler?q=` view_students) | 🟢 Quelltext-Ratsche + E2E am Draht                   | `frontend-hygiene-action-endpunkt.test.js`, `suchfelder-eigene-tuer.spec.js`, `api/routen_katalog.go` |
| Welche Rechte eine Rolle hat: `role_permissions` (GROSS; Middleware mappt per `UPPER()`)                                                                                                                      | 🟡 konfigurierbar (bewusst)                           | `api/tuer/rechte.go` (`ladeUndMerkeRecht`)                                                               |
| Login-Rate-Limit je echter Client-IP (nicht Proxy)                                                                                                                                                            | 🟢 `pkg/clientip` + `TRUSTED_PROXIES`                 | `api/anmelde_bremse.go`, `pkg/clientip/clientip_test.go`, `api/vertraute_proxys_test.go`      |
| **Eine gesperrte Anmeldung liefert nichts** — nach der Sperre nach Inaktivität beantwortet der Server jede Anfrage mit 423, auch mit einem erneuerten Token; auf geht sie nur mit dem Passwort | 🟡 eine Prüfung in `PruefeToken`, die jede geschützte Route nimmt + PG-Test | `auth/token.go` (`PruefeToken`), `auth/sperre_pg_test.go`, `api/sperre_middleware_test.go` |
| **Gesperrt wird nur, was sich ohne Mailserver aufschließen lässt** — eine Anmeldung ohne Zeile in `sitzungen` sperrt der Server nicht und erneuert sie nicht | 🟡 Code + PG-Test | `auth/sitzungen.go` (`Sperre`, `Verlaengere`), `auth/sperre_pg_test.go` |
| Der Prüfwert des Passworts lebt nicht länger als die Anmeldung: Abmelden, Ablauf und das Löschen des Kontos nehmen die Zeile mit | 🟢 FK mit CASCADE, 🟡 Abmelden und Abräumen | `sitzungen` (`benutzer_id`), `auth/sperre_pg_test.go` |

> **Zum Rollen-Vokabular:** `lehrer` ist als Klassenname gesperrt
> (`api/leser/leser_klasse_regel.go`); eine Lehrkraft steht in `leser` mit
> `art = 'lehrkraft'` (Migration 123). Dass das Kollegium nur `create_reservations` hält
> (Migration 070), ist eine **Vorgabe**, keine Invariante: Die Zeile darüber gilt, ein Administrator darf mehr erteilen. Siehe
> [FACHKONZEPT §12](FACHKONZEPT.md).

---

## 10. Migrationen & Prozess

| Invariante                            | Durchsetzung                                         | Fundstelle                    |
| ------------------------------------- | ---------------------------------------------------- | ----------------------------- |
| Seed-Liste == alle `migrations/*.sql` | 🟢 CI-Drift-Guard (Test schlägt bei Abweichung fehl) | `db/migrationen_drift_test.go` |
| Jede Migration atomar (eigene TX)     | 🟢 Runner                                            | `db/migrationen.go` (spieleMigrationEin) |

## 11. LMF-Plan (`lmf_plaene`, `lmf_termine`)

Der Plan ist eine REIHENFOLGE, die der Server auf Schultage × Stunden gießt (Migration
097); Datum und Stunde der Zeilen sind gerechnet, nie getippt. Seit Migration 101 hängt
der Rahmen an der Art: Der Büchertausch vor den Sommerferien ENDET (Donnerstag vor den
Ferien, 4. Stunde), die Bücherausgabe danach BEGINNT.

| Invariante                                                                                                                                   | Durchsetzung                                                                                        | Fundstelle                                                               |
| -------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| Ein Plan je Art und Schuljahr                                                                                                                | 🟢 UNIQUE                                                                                           | `uniq_lmf_plaene_art_schuljahr`                                          |
| Anker folgt der Art: Rückgabe-Plan hat Ende (`letzter_tag`, `letzte_stunde`), Ausgabe-Plan nicht — `erster_tag` beim Rückgabe-Plan GERECHNET | 🟢 CHECK + Server (`pruefeLMFAnker`, `service.VerteileLMFPlan`)                                     | `chk_lmf_plaene_anker`, `api/lernmittel/lmf_plan.go`, `internal/service/lmf_plan_verteilung.go` |
| Jede Zeile gehört zu einem Plan (Waisen setzten unsichtbar Fristen, 098)                                                                     | 🟢 NOT NULL + FK ON DELETE CASCADE                                                                  | `lmf_termine` (`plan_id`, `position`)                                    |
| Datum/Stunde der Zeilen sind gerechnet — EINE Verteilung, Vorschau = derselbe Aufruf (`vorschau: true`), kein JavaScript-Zwilling            | 🟡 Bauart (`pkg/lmfplan.VerteileMit` / `VerteileRueckwaerts`) + PG-Test                             | `pkg/lmfplan/verteilung.go`, `api/lernmittel/lmf_termine_frist_pg_test.go`              |
| Steht eine Klasse mehrfach im Plan, gilt der FRÜHESTE Termin — beim Ausleihen wie beim Massenabgleich                                        | 🟡 Code (`LMFAbfragen.LadeRueckgabeTerminLage` MIN, `fruehesteTermineJeKlasse`) + PG-Test                     | `repository/lmf_termine.go`, `internal/service/lmf_frist_kopplung.go`, `api/lernmittel/lmf_termine_frist_pg_test.go` |
| Ein Entwurf (`veroeffentlicht_am` NULL) setzt keine Fristen und ist für Portal und PDF unsichtbar                                            | 🟡 Code (Frist-Kopplung nur bei Stempel; Listen filtern `veroeffentlicht_am IS NOT NULL`) + PG-Test | `lmf_plaene` (`veroeffentlicht_am`), `api/lernmittel/lmf_plan_veroeffentlichung.go` |
| Freie Tage und feste Plätze gelten je Plan; der Vorschlag fürs Folgejahr bringt sie nicht mit                                                | 🟡 Code (`entwurfAus`, `service.LMFPlanReihenfolgeVorgabe`)                                         | `lmf_plan_freie_tage`, `lmf_termine` (`fest`)                            |
| Sommerferien-Tabelle (Hessen, KMK) reicht mindestens zwei Jahre voraus                                                                       | 🟡 Selbstprüfung „Ferientabelle" + Horizont-Test (rot ab Januar 2029)                               | `pkg/lmfplan/sommerferien.go`, `internal/bereitschaft/ferien.go`                   |

---

## Das Raster — die neunzehn Fragen, und ihre Frontend-Lesart

**Wann:** beim Formwechsel eines Schreibpfads (neuer Endpunkt, neuer Rumpf, andere
Speicher-Granularität) — nicht bei Kosmetik. Frage 12 zusätzlich bei JEDER Migration,
Frage 13 immer dann, wenn eine Tabelle, Spalte oder ein Feld ihre Bedeutung ändert, ohne
den Namen zu wechseln, Frage 14 bei jeder Migration und jedem Import, der vorhandene Werte
überträgt, umdeutet oder löscht. Die Fragen 15 bis 18 bei jeder Änderung an einer Maske,
einem Dialog oder einem Bildschirm, an dem gescannt wird; Frage 17 außerdem überall, wo ein
Pfad eine Tür zumacht oder auf einen fremden Dienst wartet. Frage 19 an jeder Tür, die
festlegt, wer etwas darf oder sieht oder wohin etwas geht, und an jeder, die etwas
Unumkehrbares tut. Die Durchgänge samt Funden stehen in
den Commit-Nachrichten (`git log --grep=Rasterdurchgang`), was davon offen ist in [OFFEN.md](OFFEN.md), die Bestands-Achse (bekannte Bugklasse × ganzer Baum) in
[sweeps.md](sweeps.md). Die kanonische Liste steht hier, weil sweeps.md hierher zeigt
und die Fragen sonst nur verstreut in den Durchgangs-Protokollen stünden.

1. **Konvention statt Regel** — was hält die Zusicherung: ein Constraint oder eine Verabredung zwischen Dateien?
2. **Spezialwert** — was bedeutet leer/0/nil/fehlend auf diesem Pfad — und meint der Nachbarpfad dasselbe?
3. **Zwei Wahrheitsquellen / zwei Türen** — dieselbe Regel, Liste oder Beschriftung zweimal formuliert; Paare, die nur zufällig einig sind.
4. **Wer sieht was** — Recht an der Route (nicht nur am Menü), Objektbindung an den Angemeldeten.
5. **Stille Fehler** — meldet der Pfad Erfolg anhand der Wirkung oder anhand der Eingabe?
6. **Zeit** — Zeitzone, Reihenfolge, das Fenster zwischen zwei Schritten.
7. **Gate-Ehrlichkeit** — wurde das Gate rot gesehen? Kann es sich still überspringen?
8. **Lebenszyklus** — wie entsteht, altert und stirbt der Datensatz: Frist, Löschpfad, Papierkorb, Anonymisierung.
9. **Ausleitung** — wo verlässt eine Kopie die Anwendung (Datei, Mail, Export, Log, Fremdsystem)? Wer liest sie, wie lange lebt sie, ist sie verschlüsselt?
10. **Rückweg** — ist der Weg zurück begehbar und am Ergebnis bewiesen, nicht am Vorgang?
11. **Geteilter Zustand** — wer lädt ihn auf diesem Pfad, was gilt vor dem Laden und bei Fehlschlag, überlebt der Lader sein eigenes Ergebnis?
12. **Gegenrichtung Schema** — was TUT die Datenbank, das im Code nirgends steht? Fremdschlüssel mit Löschwirkung (CASCADE/SET NULL), CHECK-Bedingungen, Trigger; dazu, was sie ablehnt oder weiterträgt: Fremdschlüssel, an denen ein Löschen scheitert oder die ein Umbenennen weitertragen, und Eindeutigkeit, auch die eines Teilindex.
13. **Bedeutungswechsel unter gleichem Namen** — hat dieser Name seit gestern eine andere Bedeutung, und wer liest ihn noch in der alten?
14. **Datenlage** — was steht in den Zeilen, auf die der Pfad trifft, gemessen am echten Bestand und nicht an der Seed-Datenbank? Trägt die neue Regel jede Form, die dort vorkommt?
15. **Ungespeichertes** — was steht nur im Browser (getippt, gewählt, eingereiht), und was nimmt es weg: die Sperre, das Ende der Anmeldung, ein Wechsel der Ansicht, ein Neuladen, eine Rückfrage? Bleibt es stehen, oder sagt das Programm es vorher?
16. **Eingabeweg** — wohin geht ein Scan (Zeichenfolge und Enter), wenn dieser Bildschirm oder Dialog offen ist: in welches Feld, auf welchen Knopf? Hört ein Feld, in das gescannt wird, auf das Enter?
17. **Ausfall** — wovon hängt der Pfad ab, das fehlen kann (Mailserver, Netz, Datenbank, Katalogdienste), und was kann die Person an der Theke dann noch tun? Macht der Pfad eine Tür zu: Wie kommt man in jedem dieser Fälle wieder hinein?
18. **Veralteter Stand** — zwischen Laden und Speichern liegt Zeit. Was schreibt der Pfad zurück, das die Person nicht angefasst hat, und was überschreibt er damit, wenn inzwischen ein anderer Arbeitsplatz, ein Import oder ein Hintergrundlauf gespeichert hat?
19. **Spur** — hinterlässt der Pfad einen Eintrag, an dem sich später ablesen lässt, wer was wann geändert hat? Steht darin genug, um die Änderung zu erklären, und kein Personenbezug, den die Tilgung nicht kennt?

### Zu Frage 12

Die ersten elf Fragen sehen vom Code aus auf die Daten. Frage 12 sieht zurück: auf das, was
die Datenbank ohnehin tut und wovon der Code nichts weiß. Zwei Beispiele aus der DDL, die
beim Lesen des Go-Codes nicht auffallen:

- `CONSTRAINT check_damage_item` verlangt genau eines von `exemplar_id`/`geraet_id`. Der
  Geräteschaden ist damit ausdrücklich vorgesehen; eine Rechnung, die per INNER JOIN mit dem
  Exemplar verbindet, verlöre ihn.
- An `buecher_titel` hängen Kinder mit `ON DELETE CASCADE`. Ein Löschpfad, der sie nicht
  nennt, nimmt still mit, was daran hängt, etwa einen wartenden Schüler oder eine
  angemeldete Klassensatz-Anforderung.

**Frage 1 („Konvention statt Regel") ist nicht dieselbe Frage:** Sie fragt, ob eine
Zusicherung des Codes von der Datenbank gehalten wird. Frage 12 fragt umgekehrt nach dem,
was die Datenbank ohnehin tut — und wovon der Code nichts weiß.

Mechanischer Teil: `repository/schema_gegenrichtung_pg_test.go` friert die drei Inventare
ein (Stand 16.09.2026, an der Datenbank gemessen: 38 Fremdschlüssel mit Löschwirkung,
44 CHECK-Bedingungen, 22 Trigger). Jede
Schema-Änderung wird damit rot und verlangt die Antwort: **Wer behandelt die Folge?** Die
schon befragten Einträge tragen ihre Antwort als Kommentar; der Rest ist Arbeitsliste.

`repository/schema_gegenrichtung_sperren_pg_test.go` friert drei weitere Inventare ein
(Stand 08.10.2026, an der Datenbank gemessen: 12 Fremdschlüssel, an denen ein Löschen
scheitert, 6, die eine Änderung des Schlüssels weitertragen, 35 Eindeutigkeitsregeln, davon
14 mit Bedingung). Hier lautet die Frage: **Wer läuft dagegen, und was bekommt er — einen
Satz oder einen Abbruch?**

Die Zahlen stehen hier zur Einordnung, nicht als Zusicherung — sie veralten mit jeder
Migration, und rot wird nicht dieser Satz, sondern das Gate.

Bewusst NICHT eingefroren: die 96 nullbaren Spalten. Ihre Gefahr ist die Bugklasse
„NULL-Scan", und die hat ihre eigene Antwort — echte Postgres-Tests je Lesepfad. Eine
Liste von 96 Namen wäre der Dateibaum und würde nichts aussagen.

Eine Frontend-Lesart hat Frage 12 nicht; das Gegenstück dort ist die Bugklasse „Nie
verdrahtet" (`docs/sweeps.md`): Felder, die der Server liefert und die niemand liest.

Zweite Schärfung ohne neue Nummer: *Wer läuft gegen die Regel?* Eine neue CHECK-Bedingung
oder ein Wächter-Trigger gilt für jeden Schreiber der beteiligten Spalten, nicht nur für die
Tür, die gerade entsteht. Die Antwort in der Ratsche zählt die Schreiber auf und sagt je
Schreiber, was er bekommt: einen Satz (400/409) oder einen Abbruch (500). „Die Datenbank hält
sie für jeden anderen Schreiber" ist wahr, aber keine Antwort darauf, was aus diesen
Schreibern wird.

### Zu Frage 13

Ein Name bleibt, seine Bedeutung wechselt — und niemand merkt es, weil der Code
weiterläuft. Beispiel: `schueler` ist eine SICHT auf `leser` mit `WHERE art = 'schueler'`
(Migration 124). Ein `UPDATE schueler` trifft damit beim Kollegium null Zeilen: eine stille
404, kein Fehler. Das ist nicht Frage 3 („zwei Wahrheitsquellen"): Ein solcher Pfad ist nicht
doppelt formuliert, er meint schlicht etwas anderes als vor dem Wechsel.

Mechanischer Teil: `docs/schreibpfade_gegen_sicht_test.go` führt die Schreibpfade gegen die
Sicht je Datei mit einer Begründung, warum die Filterung dort richtig ist: LUSD-Abgleich,
Versetzung, Anonymisierung der Abgänger, der Littera-Schülerlauf und eine Prüfhilfe der
Tests. Die Regel ist deshalb nicht „gegen eine Sicht schreibt man nicht", sondern: **Eine
neue Zeile ist eine Frage.**

Dazu eine Schärfung von Frage 12 ohne neue Nummer: *Wer räumt weg, was die Datenbank
selbst angelegt hat?* Das Inventar fragt bei Fremdschlüsseln in die Löschrichtung, bei
Triggern nur in die Anlegerichtung. Vier Trigger schreiben in fremde Tabellen, zwei davon
legen Zeilen an, die kein Go-Code je löscht.

### Zu Frage 14

Die dreizehn Fragen davor lesen Code und Schema; keine fragt die Zeilen, auf die eine
Änderung trifft. Eine Regel, die an der lokalen Datenbank entworfen ist, sieht dort oft nur
eine Form der Daten. Beispiel: Für Migration 135 („Klasse N wird Spanne N bis N") trug lokal
ein einziger Testtitel eine Klasse, am Testserver 153 Titel, darunter mehrjährige Bände mit
nur einem Jahr und Klassen, die der Signatur widersprechen.

Gemessen wird lesend am Testserver. Die lokale Datenbank ist ein Seed-Bestand; Fragen nach
Mengen, Verteilungen und Altformen beantwortet sie nicht.

Mechanischer Teil: `docs/migration_datenlage_test.go` verlangt ab Migration 145 von jeder
Migration, die vorhandene Zeilen ändert oder Daten wegnimmt, im Kopf einen Satz mit
„gemessen" und dem Datum oder „Ohne Messung:" mit Grund. Die Zeile beweist nicht, dass
gemessen wurde; sie stellt die Frage dort, wo die Migration entsteht.

### Zu den Fragen 15 bis 18

Die vierzehn Fragen davor lesen Code, Schema und Daten. Keine fragt nach dem, was zwischen
Mensch und Server liegt: nach der Eingabe, die der Server noch nicht kennt, nach dem Gerät,
das sie liefert, nach dem Dienst, der gerade fehlt, und nach dem Stand, der seit dem Laden
veraltet ist. Die Bugklassen dazu stehen in [sweeps.md](sweeps.md).

- **15 · Ungespeichertes.** Hinter dem Sperrbildschirm bleibt die Anwendung stehen,
  ausgeblendet und träge; eine offene Rückfrage ist dort nicht zu sehen. Der
  Verlassen-Schutz des LMF-Planers (`uiStore.verlassenSperre`) wird nur beim Wechsel des
  Menüpunkts und beim Schließen des Fensters gefragt. Frage 8 fragt in der Frontend-Lesart
  das Gegenteil: was beim Verlassen zurück auf Anfang muss.
- **16 · Eingabeweg.** Ein Handscanner ist eine Tastatur und tippt in das, was den Fokus hat.
  Ein Feld, in das gescannt wird, hat nach dem Öffnen den Fokus und hört auf Enter
  (`frontend/e2e/buch-anlegen-handscanner.spec.js`); das Enter eines Scans drückt in den
  Rückfragen der Theke keinen Knopf (`frontend/src/lib/scanOhneFokus.js`,
  `frontend/e2e/theke-scan-reihe.spec.js`), und im Passwortfeld gilt ein Scan nicht als
  Passwort (`frontend/src/lib/scanErkennung.js`, `frontend/e2e/sperre-scan.spec.js`).
- **17 · Ausfall.** Das Speichern eines Titels fragt keinen Katalogdienst
  (`inventur/speichern_ohne_katalogdienste_pg_test.go`). Ohne Netz wird nicht gesperrt, und ein
  Ausfall des Mailservers zählt nicht als Fehlversuch und sperrt niemanden aus (Migration 155).
- **18 · Veralteter Stand.** Eine Maske schickt, was sie seit dem Öffnen geändert hat, und die
  Tür schreibt, was der Rumpf nennt; den Bestand eines Titels schickt die Maske mit der Zahl
  vom Öffnen, und der Server lehnt bei abweichendem Stand ab. So halten es Titel, Leserakte,
  Benutzer, Gerät, Lieferant und die Kategorien der Einstellungen
  (`inventur/titel_genannte_felder_pg_test.go`, `frontend/e2e/buchmaske-zwei-plaetze.spec.js`,
  `frontend/e2e/leserakte-zwei-plaetze.spec.js`, `benutzer-nur-geaendertes.spec.js`,
  `geraet-nur-geaendertes.spec.js`, `lieferant-nur-geaendertes.spec.js` und
  `einstellungen-kategorien.spec.js` in `frontend/e2e/`). Ändern zwei Plätze dasselbe Feld,
  gilt der spätere Eintrag. Eine Kategorie gibt den Stand beim Öffnen mit, auch die Vorgabe
  für eine nie gesetzte Einstellung: Verglichen mit dem rohen Wert vom Server wiche die
  Vorgabe immer ab und ginge bei jedem Speichern mit. Der LMF-Planer hat für dieselbe Lage ein
  Signal über die Live-Leitung (`fremdesSignal`).

Ein Gate hat von den vier Fragen nur ein Teil von 17 (die Fristen an den Aufrufen mit
Mailversand, `api/mail_routen_frist_test.go`). Für 16 gibt es ein Muster
(`page.keyboard.type` und Enter ohne Klick, `frontend/e2e/kiosk-scannerfokus.spec.js`;
mit verzögerter Antwort `frontend/e2e/inventur-scan-reihe.spec.js`), für 18 den Test mit
zwei Plätzen (`frontend/e2e/buchmaske-bestand.spec.js`,
`frontend/e2e/leserakte-zwei-plaetze.spec.js`), für 15 den Test mit der Maske vor und nach
der Sperre (`frontend/e2e/sperre-ungespeichertes.spec.js`).

Schärfungen ohne neue Nummer, je mit dem Test, der die Antwort an einer Stelle hält:

- **Frage 3, der einzige Auslöser:** Blendet eine Änderung ein Element aus oder nimmt sie es
  weg — was hat nur dieses Element ausgelöst? (`e2e/cover-grossansicht.spec.js`: das
  Miniaturbild der Ausleihliste öffnet die Großansicht des Covers.)
- **Frage 3, die Prüfung der Schwester:** Führen zwei Türen zum selben Vorgang, prüft jede,
  was die andere prüft? (`repository/schaden_melden_ausleihe_pg_test.go`: „Verlust melden"
  in der Akte prüft wie der Bescheid, ob die Ausleihe noch läuft.)
- **Frage 5, der zweite Versuch:** Was geschieht beim zweiten Mal — Doppelklick, erneutes
  Senden nach einer Zeitüberschreitung, zweiter Lauf?
- **Frage 5, das Formular nach dem Ladefehler:** Zeigt eine Maske nach einem gescheiterten
  Abruf leere Felder, schreibt „Speichern" sie über die echten Werte
  (`MailEinstellungen.test.js`; `ui/LadeFehler`).
- **Frage 6, die Reihenfolge als Schutz:** Trägt eine Prüfung nur, weil eine Anweisung davor
  eine Zeile sperrt? Dann steht die Reihenfolge in einem Test
  (`inventur/bestand_gleichzeitig_pg_test.go`: Der Vergleich mit dem gesehenen Bestand zählt
  erst nach dem UPDATE des Titels).
- **Frage 8, was liegen bleibt:** Der Lebenszyklus gilt auch für Dateien auf der Platte und
  für Einträge im Speicher des Browsers, nicht nur für Zeilen — und für Schwebendes, das an
  der Seite hängt statt an seinem Bauteil, etwa eine Sprechblase über der geleerten Theke
  (`actions/tooltip.js`).
- **Frage 10, der Übergang:** Was lebt über ein Update oder einen Rückbau hinweg — offene
  Fenster mit dem alten Programm, laufende Anmeldungen, eingereihte Vorgänge,
  zwischengespeicherte Seiten — und versteht jede Seite die Form der anderen? (Die neuere
  Startseite nach einem Rückbau: ARCHITEKTUR.md 11.5.)
- **Frage 11, die Ablage unter einem Schlüssel:** Liegt ein Ergebnis unter einem Schlüssel, den
  auch eine andere Anmeldung vorlegen kann, in der Fassung, die der erste Absender lesen
  durfte? Die Warteschlange der Theke gehört dem Rechner, nicht der Person
  (`api/theke/nachbuchen_sperrgrund_pg_test.go`).
- **Frage 14, die Menge:** Schneidet der Pfad an einer Grenze ab, und sagt er es?
- **Frage 14, die Zeile ohne den Pflichtwert:** Verlangt eine Maske oder ihre Tür beim Ändern
  ein Feld, das Zeilen im Bestand nicht tragen? An Zeilen mit dem Wert fällt es nie auf; in
  der Littera-Sicherung von 2010 tragen 2.193 von 10.732 Titeln keine ISBN
  (`inventur/titel_ohne_isbn_pg_test.go`). Dazu gehört die Zeile, deren Wert die Prüfung
  nicht besteht: Eine Tür prüft, was sich ändert.
- **Frage 16, der Scan vor der Antwort:** Was geschieht mit dem nächsten Scan, solange die
  Antwort auf den vorigen aussteht — fragt er neu, oder schließt er sich dem laufenden Ablauf
  an, und zu welcher Eingabe gehört dann die Antwort? (`isbnAbfrage.test.js`.) Ein
  Handscanner wartet nicht: Ist das Feld gesperrt oder ohne Fokus, solange die Anfrage läuft,
  geht der nächste Scan verloren, ohne Meldung. Inventur und Theke reihen Scans deshalb ein;
  an der Theke fällt dieselbe Nummer weiter weg, und ging ein Scan nicht glatt durch, wird der
  Rest nicht gebucht und genannt (`frontend/e2e/inventur-scan-reihe.spec.js`,
  `frontend/e2e/theke-scan-reihe.spec.js`, `frontend/src/lib/stores/omniboxScanReihe.test.js`).
- **Frage 16, was wartet:** Reiht ein Scanfeld Scans ein, bucht es sie in dem Zustand, den
  der Scan davor hinterlassen hat. Was geschieht mit den wartenden, wenn der Scan davor
  scheitert, eine Rückfrage aufgeht oder jemand den Zustand von außen ändert (Escape,
  Abmelden)? Die Reihe der Theke hält nach jedem Scan an, der nicht glatt durchging
  (`frontend/src/lib/stores/omniboxScanReihe.test.js`), und jeder gescheiterte Scan gibt den
  Fehlerton (`frontend/src/lib/stores/omniboxAntwortArten.test.js`).
- **Frage 17, die Fristenkette:** Welche Frist endet zuerst — die der Oberfläche, die des
  Servers, die des fremden Dienstes —, und was sieht die Person, wenn die Oberfläche zuerst
  aufgibt? Der Server arbeitet dann weiter, seine Antwort sieht niemand, und der zweite
  Versuch trifft eine Arbeit, die schon geschehen ist (`api/mail_routen_frist_test.go`).
- **Frage 18, die jüngste Öffnung:** Fordert ein Bildschirm nacheinander zwei Inhalte an
  (zweiter Klick, „Neu"), welche Antwort gilt, wenn die erste später ankommt?
  (`frontend/e2e/titel-oeffnen-rennen.spec.js`.)

### Zu Frage 19

Die achtzehn Fragen davor fragen, ob ein Pfad das Richtige tut. Keine fragt, ob sich später
ablesen lässt, dass er es getan hat. Beispiel: Die Rechte-Matrix entscheidet an jeder Route,
wer welche Daten sieht; ohne Eintrag ließe sich nicht nachsehen, wer einer Rolle wann ein
Recht gab (`api/benutzer/rechte_matrix_protokoll_pg_test.go`).

Gefragt wird an jeder Tür, die festlegt, wer etwas darf oder sieht (Rechte, Konten, Sperren),
wohin etwas geht (Adressen für Mahnlisten und Bestellungen, Mailserver, Vorlagen) oder die
etwas Unumkehrbares tut (löschen, anonymisieren, aussondern, zusammenführen). Die zweite
Hälfte der Frage hält die Gegenrichtung: Was in einen Eintrag kommt, bleibt bis zur
Audit-Aufbewahrung; Werte einer Person darin muss die Tilgung kennen
(`repository/protokoll_personenbezug_test.go`).

Ein Gate hat die Frage nicht. Durchgesehen am 08.10.2026: die 107 Anmeldungen ändernder
Routen (POST, PUT, PATCH, DELETE) gegen die Stellen, die in `audit_log` oder `audit_logs`
schreiben. Einen Eintrag schreiben unter anderem die Zuordnung der Klassenleitungen, die
Mail-Vorlagen, die Lieferanten, die Verlängerung der Lernmittel einer ganzen Klasse
(`api/verwaltung_protokoll_pg_test.go`) und die Schlagwort-Pflege beim Zusammenführen,
Umleiten und Löschen (`TestSchlagwortPflege_UnumkehrbaresStehtImProtokoll`). Ohne Eintrag
bleiben Türen, die nichts von dem tun, was die Frage nennt, oder deren Zeile die Person
selbst festhält:

- Theke: ausleihen und zurücknehmen, auch nach einem Netzausfall (die Ausleihe trägt, wer sie
  gebucht und wer sie beendet hat), eine Ausleihe verlängern, eine Vormerkung anlegen oder
  löschen, eine Meldung nach Netzausfall bearbeiten (die Meldung trägt, wer sie bearbeitet
  hat), „Verlust/Schaden melden" (die Ausleihe trägt, wer sie beendet hat).
- Leser: einen Leser von Hand anlegen, sein Foto. Ändern, sperren, löschen, wiederherstellen
  und zusammenführen schreiben einen Eintrag.
- Katalog: Titel anlegen und ändern, Signatur, Lernmittel, Schlagworte am Titel, Auflagen
  zusammenlegen und lösen, Cover, den Barcode eines Exemplars ändern, ein Schlagwort umbenennen
  oder als Filter setzen, eine Sachgruppe anlegen, ändern oder löschen (gelöscht wird nur eine,
  an der kein Titel hängt), Listenimport.
- Inventur: beginnen (die Inventur trägt, wer sie begonnen hat), scannen, abbrechen (verwirft
  die Zählung, bucht nichts ab).
- Druck und Etiketten: Etiketten, Ausweise, Mahnbriefe, das Layout der Ausweise.
- Bestellung: aufgeben, bestätigen, den Link für den Lieferanten erneuern, die Bestätigung des
  Lieferanten. Die Bestellung selbst ist der Beleg; der Wareneingang schreibt einen Eintrag.
- Geräte, Wünsche und Meldungen des Kollegiums, Klassensatz-Reservierungen, die Sperre des
  Bildschirms, An- und Abmelden.

Das Aussondern steht mit der Person im Protokoll, auch auf diesen drei Wegen: der Status in
der Buchakte, ein kleinerer Bestand in „Buch
bearbeiten", der Abschluss einer Inventur (`repository.ProtokolliereAussonderung`,
`TestAussondern_StehtMitDerPersonImProtokoll`). Der Eintrag nennt Bearbeiter, Weg und Grund;
den Grund liest er vom Exemplar. Die Zustandsnotiz steht nicht darin: Sie ist Freitext, kann
eine Person nennen und bliebe bis zur Audit-Aufbewahrung. Das Abgangsbuch liest diese Einträge
nicht, es zählt im Protokoll nur gelöschte Exemplare (`aktion = 'DELETE'`).

### Frontend-Lesart

Das Raster wird leicht nur am Server gelesen („Schreibpfad" = Handler, „Tür" = Endpunkt).
Dieselben Fragen, übersetzt für die Oberfläche:

| Frage                  | Backend-Begriff                             | Frontend-Begriff                                                                                                                                  |
| ---------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2 · Spezialwert        | leer/0/nil im Rumpf                         | ein geleertes Zahlenfeld wird beim Umrechnen zur 0 — „nicht anfassen" und „kein Stück" sind dann dieselbe Eingabe (etwa der Bestand in der Buchmaske) |
| 3 · zwei Türen         | zweiter Endpunkt schreibt denselben Zustand | zweiter Ort für denselben Zustand: Komponenten-State neben dem Store, localStorage neben dem Backend (Multi-PC!)                                  |
| 5 · stille Fehler      | 200 ohne Wirkung                            | Ladefehler ohne Fehlerzustand — ein catch, das alten Inhalt stehen lässt und nichts anzeigt; Erfolgsmeldung aus der Eingabe statt aus der Antwort; ein Knopf, dessen Weg ohne Anfrage und ohne Meldung endet |
| 6 · Zeit               | Zeitzone, Tx-Fenster                        | Antwort-Reihenfolge: überholt die langsame Antwort die schnelle? Sequenznummer wie im bestellStore                                                  |
| 8 · Lebenszyklus       | Frist, Löschjob                             | Bediener- und Ansichtswechsel: welcher State muss beim Verlassen zurück auf Anfang — und welcher Code tut das wirklich?                           |
| 11 · geteilter Zustand | Wächter/Lader je Pfad                       | wer lädt den Store auf DIESEM Einstieg — und unmountet das Ergebnis seinen eigenen Lader? Prüft ein Wächter gegen die geladene Liste statt gegen den Datensatz, entfällt er, sobald der Datensatz nicht in ihr steht |

Die übrigen Fragen bis 14 (1, 4, 7, 9, 10) gelten wörtlich auch vorn — Frage 4 heißt dort:
das Recht steuert das Menü UND die Route, nie nur eines von beiden. Die Fragen 15 bis 18
sind an der Oberfläche gestellt; am Server heißt 17 „welche Frist hat der Aufruf nach
außen, und was antwortet die Tür, wenn sie abläuft?" und 18 „prüft die Tür beim Schreiben,
ob der Stand noch der gelesene ist?". Frage 19 ist am Server gestellt.

---

## Testebenen — wofür welche

| Ebene                          | Beantwortet                          | Beispiel                               |
| ------------------------------ | ------------------------------------ | -------------------------------------- |
| Unit (pgxmock)                 | Stimmt die Go-Logik?                 | Ausleihlimit, Vormerkkonflikt          |
| **DB-Integration (echtes PG)** | Hält die DB die Invariante wirklich? | `chk_aussonderung_grund` lehnt NULL ab |
| e2e (Playwright)               | Funktioniert der Weg durch die App?  | Inventur-Verlust bucht korrekt aus     |

Die mittlere Ebene ist nicht ersetzbar: pgxmock kennt keine Constraints, und e2e läuft nur
Happy-Paths.

Die 🟡-Ausleihregeln prüfen Unit-Tests (`internal/service/ausleihe_am_leser_test.go`); die
🟢-Invarianten Integrationstests gegen echtes Postgres 18 im CI-Service-Container
(`db/bedingung_*_pg_test.go`): Jede Verletzung wird provoziert und muss am erwarteten
Constraint scheitern, jeder gültige Wert muss durchgehen. Ohne Datenbank überspringen sie
sich; `TestDBTestsLaufenInCI` stellt sicher, dass das **in CI** nicht unbemerkt passiert.
`db/migrationen_drift_test.go` hält die Liste der Migrationen in `schema.sql` gegen die
Dateien.

## Offene Arbeit

Offene Arbeit steht in [OFFEN.md](OFFEN.md).

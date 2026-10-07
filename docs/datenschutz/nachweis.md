# Datenschutz-Nachweis

Stand: 08.10.2026 (Entwurf)

Eine Übersicht zum Weitergeben an Schulleitung, schulischen Datenschutzbeauftragten und
Schulträger: was das Programm mit Personendaten tut, woran sich jede Zusage prüfen lässt, was
bei einer Datenpanne zu tun ist und was die Schule vor dem Echtbetrieb entscheidet. Die
Einzelheiten stehen in den Unterlagen, auf die jeder Abschnitt verweist. Die rechtliche
Einordnung steht im Entwurf des Verzeichnisses der Verarbeitungstätigkeiten; bestätigen muss
sie der Datenschutzbeauftragte der Schule. Die Quellen, gegen die das Programm bewertet wurde,
nennt der Anhang.

Einen Echtbetrieb gibt es noch nicht. Das Programm läuft auf einem Testserver ohne echte
Schülerdaten; für den Betrieb ist ein Server der Schule vorgesehen.

---

## 1. Die Unterlagen

| Unterlage                                                                    | Inhalt                                                                                                                                                                  |
| ---------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Verzeichnis von Verarbeitungstätigkeiten](vvt_entwurf.md), Entwurf          | drei Tätigkeiten (Lernmittelausleihe, Schülerbücherei, Konten und Protokoll), die Speicherung am Theken-Rechner bei Netzausfall, die technischen und organisatorischen Maßnahmen; von der Schule auszufüllen und zu beschließen |
| [Datenschutzhinweis](datenschutzhinweis_art13.md), Entwurf                   | zwei Fassungen: Lernmittel und Schülerbücherei, diese mit Einwilligungsfeld                                                                                             |
| [Sicherheitskonzept](../SECURITY.md)                                         | Anmeldung, Rechte, Verschlüsselung und Löschläufe im Einzelnen                                                                                                          |
| [Einstufung jeder Schnittstelle](../PII_MATRIX.de.md)                        | jede Adresse des Programms mit der Art der Personendaten, die sie liefert, und dem Recht davor; ein Test hält die Liste mit dem Programm deckungsgleich                  |
| [Pflegekonzept](../PFLEGEKONZEPT.md), Entwurf                                | wer betreibt und pflegt, wie ein Update auf den Server kommt, wie eine Sicherung zurückgeholt wird                                                                       |

## 2. Das Programm in Kürze

- **Zweck:** Ausleihe der Schulbücher (Lernmittel) und der Schülerbücherei mit Fristen,
  Mahnung und Schadensersatz; Vormerkungen; Klassensätze, Wünsche und Meldungen des
  Kollegiums.
- **Wer erfasst ist:** Schülerinnen und Schüler (aus dem LUSD-Export und von Hand),
  Lehrkräfte, die ausleihen oder das Portal nutzen, und das Personal mit Zugang.
- **Wo die Daten liegen:** in einer Datenbank auf dem Server der Schule, die Sicherungen
  verschlüsselt auf demselben Server; kein Cloud-Dienst. Vom Internet aus soll nur die
  Bestätigungsseite für Lieferanten erreichbar sein, sie zeigt keine Personendaten (Abschnitt 9).
  Ein zweiter Ort für die Sicherungen ist vorbereitet, aber nicht eingerichtet (Abschnitt 9). Bei
  einem Netzausfall hält der Theken-Rechner die Ausweis- und Buchnummern der Scans, keine Namen,
  bis zum Nachbuchen.
- **Anmeldung:** mit dem Postfach des Schul-Mailservers. Das Programm speichert kein Passwort.
  Für die Dauer einer Anmeldung hält es einen Prüfwert davon, damit die Sperre nach Inaktivität
  auch bei einem Ausfall des Mailservers aufgeht; beim Abmelden wird er gelöscht.
- **Wohin Daten gehen:** Klassenleitungen bekommen die Mahnliste ihrer Klasse per Mail an die
  dienstliche Adresse. Eltern bekommen Mahnung und Bescheid als gedruckten Brief, keine Mail.
  Einen Schadensfall gibt die Schule auf Papier an die Schulaufsicht ab. Lieferanten sehen nur
  Titel und Exemplare.
- **Was nicht hinausgeht:** Buchdaten werden bei der Deutschen Nationalbibliothek, Open Library
  und Google Books nach ISBN oder Titel nachgeschlagen, ohne Personendaten. Die Browser laden Buchcover nur
  vom eigenen Server, nicht bei diesen Diensten. Fehlerberichte an einen externen Dienst sind ab
  Werk aus und bleiben es (Betriebsregel).

## 3. Wer was sieht

| Stufe | Inhalt                                                                                   | ab Werk sichtbar für                                                    |
| ----- | ---------------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| 0     | keine Personendaten: Katalog, Monitor im Flur, Bestellwesen                               | je nach Recht; Katalog und Monitor auch ohne Anmeldung                  |
| 1     | Name, Klasse, Ausweisnummer, Sperrstatus                                                 | der Theke, auch Helfern                                                 |
| 2     | dazu Geburtsdatum, Abgangsjahr, Sperrgrund, LUSD-ID, Ausleihen (befristet, Abschnitt 4)  | Admin, Leitung, Mitarbeiter                                             |
| 3     | dazu Anschrift, Eltern-E-Mail, Foto, Forderungen mit Namen                               | Admin, Leitung, Mitarbeiter; die Auskunft (Abschnitt 7) Admin und Leitung |

Das Kollegium sieht nur sein Portal: Klassensätze, Wünsche, Meldungen, keine Schülerdaten. Der
Server prüft jedes Recht selbst, nicht nur das Menü. Was eine Rolle darf, kann der Admin in der
Rechte-Matrix ändern; eine geänderte Matrix ändert damit auch, wer welche Stufe sieht.

## 4. Wie lange Daten bleiben

Vorgaben ab Werk. Wo „einstellbar" steht, ändert die Schule den Wert unter Einstellungen →
Datenschutz & Sitzung.

| Was                                                                  | Frist                                                                                          | einstellbar |
| -------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ----------- |
| Zuordnung einer Ausleihe der Schülerbücherei zur Person              | 1 Tag nach der Rückgabe, dann getrennt; ein offener Schadensfall hält sie                       | ja          |
| Zuordnung der Ausleihe eines Buchs des Landes zur Person             | 730 Tage nach der Rückgabe, sonst wie oben                                                      | ja          |
| bearbeitende Person an einer Ausleihe                                | 14 Tage nach der Rückgabe                                                                       | nein        |
| Abgänger ohne offene Vorgänge                                        | anonymisiert 90 Tage nach dem späteren von Abgang und letztem Vorgang, endgültig gelöscht ab dem 30. Januar des Folgejahres | die 90 Tage |
| Schülerin oder Schüler im Papierkorb                                 | anonymisiert nach 180 Tagen                                                                     | nein        |
| Kollegin oder Kollege im Papierkorb                                  | endgültig gelöscht nach 180 Tagen, außer eine Ausleihe oder eine unbezahlte Forderung ist offen; das Zugangskonto geht schon beim Löschen | nein        |
| erledigte Wünsche, Meldungen und Klassensatz-Reservierungen des Kollegiums                       | 365 Tage nach der Erledigung                                                                    | ja          |
| quittierte Meldungen der Theke nach einem Netzausfall                | Frist der Schülerbücherei, höchstens 30 Tage                                                    | über diese  |
| Protokollzeile einer Vormerkung, die mit ihrem Titel gelöscht wurde  | Kennung und Name fallen nach der Frist der Schülerbücherei, die Zeile bleibt                    | über diese  |
| Protokoll                                                            | 24 Monate                                                                                       | ja, mindestens 6 |
| Sicherung jede Nacht                                                 | die jüngsten 14 bleiben, dazu von den älteren je Kalenderwoche eine für 12 Wochen (etwa drei Monate) | nein        |
| Sicherung vor einem Update                                           | gelöscht beim ersten Update, bei dem sie älter als 30 Tage ist                                  | nein        |
| Sicherung von Hand                                                   | gelöscht beim ersten Lauf von Hand, bei dem sie älter als 7 Tage ist                            | nein        |
| unverschlüsselte Sicherung (misslungenes Update, Verschlüsselung nicht möglich) | gelöscht beim ersten Update oder Lauf von Hand nach 2 Tagen; jeder Lauf meldet, wie viele noch liegen | nein        |

Beim Anonymisieren leert das Programm Name, Anschrift, Geburtsdatum, Schuleintritt, LUSD-ID und
Eltern-E-Mail, löscht das Foto, ersetzt die Ausweisnummer und tilgt die Spuren der Person im
Protokoll und in den Vormerkungen. Eine vom Programm vergebene Ausweisnummer bleibt danach als
bloße Zahl gesperrt, damit keine andere Person sie bekommt. Wer eine offene Ausleihe oder eine
offene Forderung hat, wird nicht gelöscht; die Selbstprüfung des Programms meldet solche Fälle.

## 5. Wie die Daten geschützt sind

Kurzfassung; vollständig im Anhang des [Verzeichnisses](vvt_entwurf.md) und im
[Sicherheitskonzept](../SECURITY.md).

- **Zugang:** Anmeldung über das Schul-Postfach; nach fünf Fehlversuchen ist die Anmeldung eines
  Kontos von derselben Netzadresse aus 15 Minuten gesperrt. Eine Sitzung gilt 12 Stunden. Die
  Theke leert sich nach 5 Minuten ohne Bedienung, nach 15 Minuten kommt der Sperrbildschirm
  (beides einstellbar); er sperrt die Anmeldung am Server, Neuladen öffnet sie nicht. Ein deaktiviertes oder herabgestuftes Konto verliert seine Rechte bei der
  nächsten Anfrage, nicht erst mit dem Ablauf der Sitzung.
- **Verschlüsselung:** Fotos und das Passwort des Mailversands liegen verschlüsselt in der
  Datenbank, die Sicherungen verschlüsselt auf der Platte; unverschlüsselt bleibt eine
  Sicherung nur in den zwei Fällen aus Abschnitt 4. Die Verbindung zum Browser ist
  verschlüsselt; Mail verschickt das Programm nur über eine verschlüsselte Verbindung.
- **Browser:** Antworten mit Personendaten legt der Browser nicht in seinem Zwischenspeicher
  ab (seit dem 30.09.2026), auch nicht an Rechnern, die mehrere Personen benutzen. Von den
  Antworten des Programms behält er nur Buchcover und die Buchnummern der Theke.
- **Geheimnisse:** Mit den Beispielschlüsseln aus der Vorlage verweigert der Server im Betrieb
  den Start, außer jemand schaltet diese Prüfung ausdrücklich ab.
- **Protokoll:** Verwaltungsvorgänge stehen mit Zeitpunkt und bearbeitender Person im Protokoll,
  administrative Eingriffe zusätzlich mit der Netzadresse des Arbeitsplatzes. Ausleihe und
  Rückgabe schreiben ihren Eintrag im selben Schritt wie die Buchung: ohne Eintrag keine
  Buchung.
- **Sicherung:** jede Nacht verschlüsselt; jeden Sonntag spielt das Programm die jüngste
  Sicherung probeweise in eine Wegwerf-Datenbank ein und meldet einen Fehlschlag per Mail.
- **Pflege:** Sicherheitsprüfungen der Abhängigkeiten laufen bei jeder Änderung und jede Woche;
  wie Korrekturen auf den Server kommen, regelt das [Pflegekonzept](../PFLEGEKONZEPT.md).

## 6. Woran sich die Zusagen prüfen lassen

Jede Zusage unten prüft ein automatischer Test bei jeder Änderung am Programm. Wird die Zusage
falsch, wird der Test rot, und aus diesem Stand entsteht kein Release.

| Zusage                                                                                       | Test                                                                                                   |
| -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| Katalog, Monitor und Bilder liefern keine Personendaten; jede Adresse liefert höchstens ihre Stufe | `api/pii_antwort_gate_pg_test.go` ruft jede lesende Adresse mit dem Recht ihrer Zeile auf und sucht in der Antwort nach eingestreuten Prüfwerten |
| Jede Adresse des Programms ist eingestuft und verlangt ein Recht                              | `api/pii_matrix_test.go`, `api/routes_authz_coverage_test.go`                                           |
| Ohne das Recht wird jeder schreibende Aufruf abgewiesen, auch am Menü vorbei                  | `api/rechte_schreibwege_pg_test.go`                                                                     |
| Die Löschläufe halten Frist und Reihenfolge; Löschlauf und Warnung folgen derselben Regel     | `jobs/cron_dsgvo_karenz_pg_test.go`, `jobs/cron_dsgvo_abgaenger_pg_test.go`, `jobs/cron_dsgvo_lesehistorie_pg_test.go`, `jobs/cron_dsgvo_anliegen_pg_test.go`, `jobs/loeschpraedikat_ratsche_test.go` |
| Die Anonymisierung entfernt, was an der Person hängt                                          | `api/dsgvo_paar_rundreise_pg_test.go` (Grenze: Abschnitt 9)                                              |
| Die Auskunft druckt jede Angabe, die sie enthält                                              | `api/dsgvo_pdf_vollstaendig_test.go`                                                                    |
| Eine Sicherung lässt sich zurückspielen                                                       | `jobs/backup_drill_pg_test.go`, `jobs/restore_probe_pg_test.go`; im Betrieb die Probe jeden Sonntag      |
| Update und Sicherung von Hand löschen je nur ihre eigenen verschlüsselten Sicherungen; jeden unverschlüsselten Rest löschen und melden beide | `docs/backup_ablage_test.go`                                                                            |
| Antworten mit Personendaten legt der Browser nicht ab                                       | `api/pii_antwort_gate_pg_test.go` verlangt an jeder lesenden Adresse ab Stufe 1 den Kopf `Cache-Control: no-store` |
| Mail geht nie unverschlüsselt hinaus                                                          | `mailservice/versand_test.go`                                                                           |
| Die Theke leert sich und sperrt nach der eingestellten Zeit                                   | `frontend/src/lib/stores/idleLock.test.js`                                                              |

## 7. Auskunft, Berichtigung, Löschung

- **Auskunft:** in der Akte jedes Lesers als PDF, ab Werk für Admin und Leitung. Hat die Person
  ein Zugangskonto, braucht es zusätzlich das Recht, Konten zu verwalten (ab Werk nur Admin).
  Jeder Abruf steht im Protokoll.
- **Berichtigung:** Stammdaten in der Akte. Anschrift und Eltern-E-Mail lassen sich einzeln
  leeren; Name, Klasse und Ausweisnummer lassen sich ändern, aber nicht leeren. Das
  Geburtsdatum bleibt, es verbindet die Person mit dem LUSD-Export.
- **Löschung:** in den Papierkorb, von dort endgültig von Hand (ab Werk Admin und Leitung) oder
  nach den Fristen in Abschnitt 4. Widerruft jemand die Einwilligung für die Schülerbücherei,
  lässt sich der Datensatz sofort in den Papierkorb legen; nach 180 Tagen wird er
  anonymisiert. Laufende Lernmittel-Ausleihen bleiben davon unberührt, für sie gilt keine
  Einwilligung.

## 8. Wenn etwas passiert: Ablauf bei einer Datenpanne

Beispiele: Eine Sicherungsdatei, ein Ausdruck mit Namen oder ein angemeldeter Rechner gerät in
fremde Hände; ein Brief geht an die falsche Anschrift; jemand hat unter einem fremden Konto
gearbeitet; die Schlüssel des Servers sind bekannt geworden.

1. **Melden, sofort.** Wer es bemerkt, sagt es der Bibliotheksleitung und dem Admin. Nichts
   löschen und nichts aufräumen: Das Protokoll ist der Beleg.
2. **Eindämmen, in der ersten Stunde (Admin).**
   - Ein Konto sperren: unter Benutzer & Rechte das Konto öffnen und „Benutzerkonto ist aktiv"
     ausschalten. Das wirkt bei der nächsten Anfrage dieses Kontos und beendet damit auch eine
     laufende Sitzung.
   - Alle Sitzungen beenden: einen neuen `JWT_SECRET` setzen und das Programm neu starten
     ([DEPLOYMENT.md](../DEPLOYMENT.md)); danach muss sich jeder neu anmelden.
   - Ist `APP_ENCRYPTION_KEY` bekannt geworden: Fotos und Mail-Passwort mit dem Werkzeug aus
     dem [Sicherheitskonzept](../SECURITY.md) auf einen neuen Schlüssel umschlüsseln.
   - Ist `BACKUP_ENCRYPTION_KEY` bekannt geworden: Jede vorhandene Sicherung gilt als offen.
     Neue Sicherungen mit einem neuen Schlüssel; die älteren lassen sich danach nur noch mit dem
     alten öffnen.
3. **Feststellen, wer betroffen ist (Admin).**
   - Das Protokoll zeigt, wer wann welchen Verwaltungsvorgang ausgelöst hat. Anmeldungen und das
     bloße Ansehen einer Akte stehen nicht darin; protokolliert wird der Abruf einer Auskunft.
   - Eine Sicherung enthält alle Personen, die zu ihrem Zeitpunkt im Programm standen.
   - Die Sicherungsdatei eines Theken-Rechners nach einem Netzausfall enthält Ausweis- und
     Buchnummern, keine Namen.
   - Was über eine einzelne Person gespeichert ist, zeigt ihre Auskunft (Abschnitt 7).
4. **Entscheiden und melden (Schulleitung mit dem Datenschutzbeauftragten).** Ob und an wen
   gemeldet wird (Schulamt, Schulträger, Aufsichtsbehörde, Betroffene), entscheidet die
   Schulleitung mit dem Datenschutzbeauftragten. Die Entscheidung eilt; die Kontakte stehen
   deshalb auf dem Blatt bei der Schule ([Pflegekonzept](../PFLEGEKONZEPT.md), Abschnitt 7.3).
5. **Aufschreiben:** was geschehen ist, wann es bemerkt wurde, was getan wurde und wer
   informiert ist.
6. **Danach:** die Entwicklung informieren, damit sie die Ursache behebt, nicht über das
   öffentliche Repository.

## 9. Bekannte Lücken

Stand und Reihenfolge führt [OFFEN.md](../OFFEN.md); die Nummer steht dabei.

- Alle Sicherungen liegen auf demselben Server; ein zweiter Ort ist nicht eingerichtet (7.3).
  Wird er als S3-Speicher eingerichtet, löscht das Programm dort nie: Ohne eine Löschregel am
  Speicher bliebe jede Sicherung dort unbegrenzt, mit allen Personen, die zu ihrem Zeitpunkt im
  Programm standen.
- Seit dem 29.09.2026 decken die Sicherungen die Sommerferien ab; gelöschte Personen bleiben
  damit bis zu etwa drei Monate in den Sicherungen. Wer einen älteren Stand zurückspielt, holt
  Personen zurück, die seitdem von Hand endgültig gelöscht wurden, und das Protokoll dieser
  Löschung ist mit zurückgespielt. Die Anleitung zum Zurückspielen hält deren Kennungen vorher
  fest und nennt danach, wer erneut zu löschen ist
  ([resilience_and_recovery.md](../resilience_and_recovery.md), Abschnitt 2a, Schritte 5b und
  8); das Nachholen ist Handarbeit.
- Seit dem 29.09.2026 fällt ein gelöschter Kollege nach 180 Tagen im Papierkorb endgültig, und
  erledigte Klassensatz-Reservierungen fallen nach der Frist für erledigte Wünsche und Meldungen;
  eine Reservierung, die vor dem Einspielen von Migration 089 (31.08.2026) erledigt wurde, hat
  keinen Zeitpunkt und bleibt stehen.
- Vom Internet aus soll nur die Bestätigungsseite für Lieferanten erreichbar sein, alles andere
  nur aus dem Schulnetz (entschieden am 28.09.2026); die Sperre am Eingang ist nicht eingerichtet
  (4.23).
- Ein gelöschtes Zugangskonto nennt die Auskunft seit dem 29.09.2026; Löschungen von davor
  trugen den Bezug zum Leser nicht und bleiben für sie unsichtbar. Die Selbstanmeldung und der
  Nachtrag einer Schul-E-Mail stehen danach nicht unter dem früheren Konto (5.19).
- Sechsmal gelangte ein Wert einer Person ins Protokoll, den die Anonymisierung nicht kannte
  oder dessen Eintrag sie nicht fand; jedes Mal behoben, zuletzt am 08.10.2026 der getippte
  Grund einer Stornierung und die Löschspur einer zusammengeführten Leserzeile. Seit dem
  29.09.2026 prüft ein Test jede Stelle im Code, die einen Protokolleintrag mit der Kennung
  eines Lesers baut. Offen sind Einträge von vor der jeweiligen Behebung: zu Lesern, die schon
  gelöscht waren, und Stornierungen, die die Kennung noch nicht trugen (5.35).
- Anmeldungen stehen nicht im Protokoll; nach einem Missbrauch lässt sich nicht nachsehen, wann
  und von wo ein Konto angemeldet war (6.1).

## 10. Was bei der Schule liegt

Im Einzelnen stehen die Punkte in der Tabelle unter der Liste; andere Unterlagen nennen sie mit
ihrer Nummer (B1 bis B7).

- Verzeichnis und Datenschutzhinweis ausfüllen und beschließen; für die Schülerbücherei eine der
  beiden Fassungen wählen.
- Das Foto auf dem Schülerausweis klären, bevor ein Ausweis mit Foto gedruckt wird.
- Den Datenschutzbeauftragten beteiligen und schriftlich festhalten, ob eine
  Datenschutz-Folgenabschätzung nötig ist.
- Das IT-Sicherheitskonzept mit dem Schulträger aufstellen, dabei den Platz von Server und
  Theken-Rechnern im Netz festlegen.
- Regeln, in welcher Rolle die Person, die das Programm wartet, Zugang zu Schülerdaten hat.
- Für die Übernahme aus dem bisherigen Programm Littera festlegen, was übernommen wird: nur,
  was aktuell ist und einen Zweck hat.

| # | Punkt |
|---|---|
| B1 | **VVT** nach HBDI-Muster — **Entwurf liegt vor:** [vvt_entwurf.md](vvt_entwurf.md) (drei Tätigkeiten: Lernmittel, Schülerbücherei, Personal/Protokolle; TOM-Anhang). Schule füllt Klammern, DSB prüft, Schulleitung beschließt. |
| B2 | **Datenschutzhinweis** (Art. 13) — **Entwurf liegt vor:** [datenschutzhinweis_art13.md](datenschutzhinweis_art13.md), Fassung A (Lernmittel, Schulaufnahme-Information § 5 (2) SchDSV) und B (Schülerbücherei, mit Einwilligungsfeld). |
| B3 | **Foto auf dem Schülerausweis**: Anlage 1 kennt kein Foto. Entweder deckt es der Schülerausweis-Erlass (prüfen) oder es braucht eine Einwilligung (§ 3). Vor dem ersten Ausweisdruck klären. |
| B4 | **Schulischer DSB** beteiligen; Schwellwertanalyse DSFA dokumentieren (Minderjährige, Fotos — Ergebnis vermutlich „nicht erforderlich", aber schriftlich). |
| B5 | **IT-Sicherheitskonzept im Benehmen mit dem Schulträger** (§ 6 (3)); **Netzplatzierung** klären: Schülerdaten ins Verwaltungsnetz, Theke/Kiosk und Lehrer-Browser sitzen meist im pädagogischen Netz. |
| B6 | **Rolle des Entwicklers/Wartenden**: Wer Admin-Zugang hat und Schülerdaten sehen kann, braucht eine Regelung — als Lehrkraft dienstlich, als Externer AVV/Wartungsvertrag. |
| B7 | **Löschkonzept gegenüber Littera**: „Littera hatte die Daten auch" ist keine Rechtsgrundlage. Beim Übergang nur übernehmen, was aktuell ist und einen Zweck hat. |

## 11. Bewusst kein Befund

- Geburtsdatum: Anlage 1 (1.7), Zweck Dubletten-Wachhund + LUSD-Abgleich → im VVT begründen.
- Anschrift: „ggf. Anschrift" im HBDI-Muster; Zweck gedruckte Rechnung/Elternbrief.
- Rollen ab Werk: nur Admin, Leitung und Mitarbeiter sehen Stufe 3 (Adresse, Eltern-Mail, Foto); Kollegium und Helfer nicht (Abschnitt 3).
- IP-Adressen stehen nur im Verwaltungsprotokoll (`audit_logs.ip_adresse`, administrative Eingriffe) und werden mit ihm nach der Aufbewahrungsfrist gelöscht (`audit_aufbewahrung_monate`, Vorgabe 24 Monate, Untergrenze 6); das Anfrage-Log schreibt keine IP; Fotos und Backups verschlüsselt; Auskunft Art. 15 vorhanden; Statistik ohne Personenbezug.
- `inventur_sessions` und `bestellungen_verlauf` bleiben **bewusst** unbefristet — die eine ist der Nachweis der Bestandskartei (HKM-Leitfaden LMF 11.3: Ausleihe UND Rücklauf), die andere ein Belegwesen mit eigener Aufbewahrungspflicht; beide tragen keine Schülerdaten.

## Anhang: Rechtsrahmen (gelesen, nicht nacherzählt)

Das Programm ist gegen den hessischen Rahmen bewertet.

| Quelle | Was daraus folgt |
|---|---|
| **SchDSV** (Schul-Datenschutzverordnung, 1.12.2023, ABl. S. 763) § 2 + Anlage 1 | Erlaubter Datenkatalog: Anschrift (1.4), Geburtsdatum (1.7), Eltern-E-Mail (1.14), Schüler-ID (1.20). **Kein Foto.** |
| SchDSV § 3 | Alles außerhalb Anlage 1 braucht eine dokumentierte **Einwilligung**. |
| SchDSV § 5 | Schule führt VVT (Art. 30), schließt AVV bei Fremdverarbeitung, hält Datenschutzhinweise aktuell, löscht regelmäßig, meldet Vorfälle an Schulamt/Schulträger. |
| SchDSV § 6 | TOMs nach Art. 25/32, BSI-Grundschutz beachten, **IT-Sicherheitskonzept im Benehmen mit dem Schulträger**. |
| SchDSV § 13 | Schülerausweis = Teil-B-Datensatz (Name, Vorname, Anschrift, Geburtsdatum, Schulen, Schüler-ID); Näheres per Erlass. |
| SchDSV § 15 | Eltern-E-Mail nur für erforderliche Schulkommunikation; Weitergabe an Dritte nur mit Einwilligung. |
| SchDSV § 17 + Anlage 3 | Löschen, sobald Zweck erreicht. |
| **§ 83a HSchG Nr. 2** | Schule darf eine digitale Anwendung selbst einführen, wenn sie als Verantwortliche Datenschutz und Sicherheit gewährleistet. |
| **HBDI-Muster-VVT „Schulbibliothek"** | Datenumfang: Name, Klasse, ggf. Anschrift, Ausleihdaten. Rechtsgrundlage: Einwilligung (freiwillige Ausleihe). Löschfrist: „unverzüglich, wenn nicht mehr notwendig". |
| **HKM-Leitfaden Lernmittelfreiheit** | 11.3 Bestandskartei weist Ausleihe **und** Rücklauf nach; 12.3–12.7 Schadensersatz ist öffentlich-rechtlich (Leistungsbescheid der Schulaufsicht); 18: Littera zulässig, Datenschutz beachten. |

Links: [SchDSV Volltext](https://www.glb-hessen.de/wp-content/uploads/2024/01/DLHRatgeberAktuell_SchDSV_NEU.pdf) ·
[HBDI-Muster Schulbibliothek](https://datenschutz.hessen.de/sites/datenschutz.hessen.de/files/2024-02/verfahrensverzeichnis_schulbibliothek_v1_0.pdf) ·
[HKM-Leitfaden LMF](https://kultus.hessen.de/sites/kultus.hessen.de/files/2021-06/lernmittelfreiheit_in_hessen_-_leitfaden_fuer_das_verfahren.pdf) ·
[§ 83a HSchG](https://gesetze.co/HE/HSchG/83a)

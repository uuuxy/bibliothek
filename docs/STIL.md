# Stil — wie der Code geschrieben ist

Stand: 14.10.2026

Eine Seite mit den Regeln für Namen, Sichtbarkeit und Kommentare im Go-Code, über `gofmt` und
den Linter hinaus. Maßstab ist Googles Regelwerk für Go
([Guide](https://google.github.io/styleguide/go/guide),
[Decisions](https://google.github.io/styleguide/go/decisions),
[Best Practices](https://google.github.io/styleguide/go/best-practices)). Dort steht an erster
Stelle die Klarheit, gesehen „through the lens of the reader, not the author", und erreicht
wird sie durch „effective naming, helpful commentary, and efficient code organization". Wo
etwas steht, sagt der [Wegweiser](WEGWEISER.md); was ein Fachwort bedeutet, das Glossar in
[ARCHITEKTUR.md](ARCHITEKTUR.md), Kapitel 12.

Der Bestand folgt diesen Regeln noch nicht überall. Er wird Paket für Paket umgestellt; welche
Pakete fertig sind, steht in [OFFEN.md](OFFEN.md), und `stil_ratsche_test.go` hält je Paket
fest, was erreicht ist. Neuer Code folgt den Regeln von Anfang an.

## 1. Sprache

- Was die Bibliothek und das Projekt benennen, heißt deutsch, ohne Umlaute geschrieben (`ae`,
  `oe`, `ue`, `ss`): `Leser`, `Rueckgabe`, `PruefeFrist`.
- Lehnwörter bleiben, wie man sie im Deutschen sagt: Barcode, Cover, Scan, Import, Export,
  Backup, Mail, Token, Cache, Cookie, Router, Middleware, Upload, Host, Snapshot, Bucket.
- Englisch ist nur, was Go oder eine Bibliothek vorgibt: `New…` und klein `new…` für den
  Erzeuger eines Typs (`NewClient`, `newKalender`; nicht `Neuer`, `Neue`, `Neues`, die mit dem
  Geschlecht des Typs wechselten), `Err…` für einen Fehlerwert, `Test…`, `Must…`, `main`,
  `init` und die Methoden fremder Schnittstellen (`Error`, `String`, `ServeHTTP`, `Scan`,
  `Close`).
- Pakete der Technik tragen die im Go-Umfeld üblichen Kurznamen (`api`, `db`, `auth`, `jobs`,
  `repository`, `service`, `pkg`, `internal`); ein Paket mit Fachbezug heißt deutsch (`leser`,
  `mahnwesen`).

Eine Sprache schreibt Googles Regelwerk nicht vor. Es verlangt, dass eine Sache einen Namen
hat: „It can be very jarring if […] the same concept has many names within a file" (Guide,
Consistency).

## 2. Eine Sache, ein Wort

| Sache                                               | Im Code                              | Abgelöst                          |
| --------------------------------------------------- | ------------------------------------ | --------------------------------- |
| jeder, der ausleiht (Tabelle `leser`)               | `Leser`                              | `Student`, `Borrower`             |
| Leser der Art Schüler (Sicht `schueler`)            | `Schueler`                           | `Student`                         |
| Eintrag im Katalog, eine Auflage                    | `Titel`                              | `Book`, `Title`                   |
| das einzelne Stück                                  | `Exemplar`                           | `Copy`, `Item`, `Book`            |
| Ausleihe, Rückgabe, Frist                           | `Ausleihe`, `Rueckgabe`, `Frist`     | `Loan`, `Checkout`, `Return`, `Due` |
| Vormerkung eines Lesers, Reservierung einer Klasse  | `Vormerkung`, `Reservierung`         | `Reservation`, `Hold`             |
| Klassensatz, Klasse, Jahrgang, Fach                 | `Klassensatz`, `Klasse`, `Jahrgang`, `Fach` | `ClassBooks`, `Class`, `Grade`, `Subject` |
| Bestellung, Nachbestellung, Zulauf                  | `Bestellung`, `Nachbestellung`, `Zulauf` | `Order`, `Reorder`, `Shipment` |
| Lieferant, Gerät                                    | `Lieferant`, `Geraet`                | `Supplier`, `Device`              |
| Schaden und die Forderung daraus                    | `Schaden`, `Forderung`               | `Damage`, `Fee`                   |
| Mahnung, überfällig                                 | `Mahnung`, `Ueberfaellig`            | `Reminder`, `Overdue`             |
| Zugang einer Person zum Programm (Tabelle `benutzer`) | `Konto`                            | `User`, `Account`                 |
| Recht, Rolle                                        | `Recht`, `Rolle`                     | `Permission`, `Role`              |
| was die Verwaltung einstellt; was der Start liest   | `Einstellung`; `Konfig`              | `Setting`, `Config`               |
| der Wert, der gilt, solange nichts eingestellt ist  | `…Vorgabe`                           | `Standard…`, `Default…`           |
| Etikett, Foto, Sperre, Papierkorb                   | `Etikett`, `Foto`, `Sperre`, `Papierkorb` | `Label`, `Photo`, `Lock`, `Deleted` |
| Abgänger, Lehrkraft                                 | `Abgaenger`, `Lehrkraft`             | `Graduate`, `Teacher`             |
| Inventur und ihr einzelner Lauf                     | `Inventur`, `Inventurlauf`           | `Inventory`, `Session`            |
| Suche, Anmeldung                                    | `Suche`, `Anmeldung`                 | `Search`, `Login`                 |
| was die Verwaltung nachlesen kann (Tabelle `audit_log`) | `Protokoll`                      | `Audit`                           |
| nimmt eine Anfrage der Oberfläche an                | Tür, am Typ `Tueren` ihres Themas    | `Handler`                         |
| was eine Tür liest und zurückgibt                   | `…Anfrage`, `…Antwort`               | `Request`, `Response`             |
| liest und schreibt eine Sache in der Datenbank      | `…Abfragen`, an Postgres `pg…Abfragen` | `…Repository`, `…Repo`          |

„Buch" steht nur in festen Wörtern (Buchakte, Zugangsbuch); sonst ist es der Titel oder das
Exemplar. Das technische Log des Servers heißt weiter `Log`.

Wiederkehrende Bausteine tragen dasselbe Wort: Die Angaben eines neuen Datensatzes enden auf
`Neu` (`AusleiheNeu`, `ExemplarNeu`), der Helfer, der eine Zeile der Datenbank liest, beginnt
mit `lese` (`leseExemplar`), der Schlüssel einer Sperre der Datenbank endet auf `Sperre`
(`auflagenSperre`), und was ein Stück SQL liefert, beginnt mit `sql` oder `SQL`
(`sqlKarenzUhr`, `SQLTitelHatExemplar`). Der Lauf einer Inventur hat einen Bereich
(`InventurBereich`), keinen Scope.

## 3. Verben

Eine Funktion, die etwas tut, beginnt mit dem Verb in der Befehlsform; ein trennbares Verb
stellt seinen Rest ans Ende (`LegeBestellungAn`). Eine Funktion, die nur etwas liefert, heißt
wie das Gelieferte (`FristEnde`), ohne `Get` davor (Decisions, „Getters").

| Verb                    | Bedeutung                                                    | Abgelöst                              |
| ----------------------- | ------------------------------------------------------------ | ------------------------------------- |
| `Lade`                  | liest aus der Datenbank                                      | `Get`, `Load`, `Fetch`, `List`, `Lies` |
| `Finde`                 | sucht nach einem Merkmal; nichts zu finden ist kein Fehler   | `Find`, `Lookup`, `Search`            |
| `Zaehle`                | zählt in der Datenbank                                       | `Count`                               |
| `Lese`                  | liest aus einer Eingabe: Datei, Anfrage, Text                | `Read`, `Parse`                       |
| `LegeAn`                | legt einen Datensatz neu an                                  | `Create`, `Insert`, `Add`             |
| `Aendere`, `Setze`      | ändert einen Datensatz; setzt einen einzelnen Wert           | `Update`, `Edit`, `Set`               |
| `Schreibe`              | schreibt, ohne dass der Rufer neu und geändert unterscheidet | `Save`, `Upsert`, `Write`, `Store`    |
| `Loesche`               | löscht                                                       | `Delete`, `Remove`                    |
| `Pruefe`                | prüft und liefert einen Fehler                               | `Check`, `Validate`, `Verify`         |
| `Ist`, `Hat`, `Darf`    | liefert wahr oder falsch                                     | `Is`, `Has`, `Can`                    |
| `Berechne`              | rechnet, ohne Datenbank                                      | `Calculate`, `Compute`, `Resolve`     |
| `Baue`, `Erzeuge`       | setzt im Speicher zusammen; erzeugt Nummer, Token oder Datei | `Build`, `Make`, `Generate`           |
| `Zeichne`               | zeichnet auf ein Blatt                                       | `Draw`, `Render`, `Print`             |
| `Sende`                 | verschickt                                                   | `Send`                                |
| `Protokolliere`         | schreibt ins Protokoll der Verwaltung                        | `Log`, `Audit`                        |
| `Verarbeite`            | führt einen Schritt einer Tür oder eines Laufs aus           | `Handle`, `Process`                   |
| `Fuehre…Aus`            | führt einen Lauf aus                                         | `Run`, `Execute`                      |

Die Tabelle legt fest, wo zwei Wörter für dieselbe Tätigkeit standen. Jede andere Tätigkeit
trägt ihr deutsches Verb (`Kuerze`, `Zerlege`, `Schliesse`, `Entschaerfe`), in der Befehlsform
auf e (`Verwerfe`, `Uebergebe`).

Wer aus der Datenbank liest, beginnt mit `Lade`, `Finde` oder `Zaehle`, auch wenn er nur einen
Wert liefert (`LadeKontoEmail`); den Namen ohne Verb trägt, wer rechnet, ohne zu lesen
(`FristEnde`, `AbgaengerJahr`). Dass eine Funktion in der Transaktion des Rufers läuft, sagt
ihr Parameter; der Name trägt dafür keine Endung (`LegeAn`, nicht `LegeAnTx`).

Wer eine übergebene Funktion in einem Rahmen ausführt, heißt nach dem Rahmen: `InTx`,
`ImSavepoint`.

## 4. Schreibweise

- Kürzel stehen durchgehend groß oder, am Anfang eines nicht öffentlichen Namens, durchgehend
  klein (Decisions, „Initialisms"): `LMFPlan`, `lmfPlan`, `DSGVOAuskunft`, `LUSDImport`. Das gilt
  für `ID`, `URL`, `ISBN`, `PDF`, `CSV`, `JSON`, `SQL`, `HTTP`, `SMTP`, `IMAP`, `UUID`, `SSE`,
  `CSRF`, `JWT`, `DNB`, `EAN`, `LMF`, `DSGVO`, `LUSD` und `OPAC`.
- Ein Name wiederholt weder sein Paket noch seinen Empfänger (Decisions, „Repetition"): nicht
  `leser.LeserListe`, sondern `leser.Liste`; nicht `LieferantenAbfragen.GetSupplierByID`,
  sondern `LieferantenAbfragen.Lade`.
- Die Länge eines Namens richtet sich nach seiner Reichweite: Ein Zähler in drei Zeilen heißt
  `n`, eine Variable der Paketebene sagt in Worten, was sie hält.
- Der Empfänger einer Methode ist ein Kürzel seines Typs, in jeder Methode dasselbe
  (Decisions, „Receiver names"): `t` für `Tueren`.
- Die üblichen Kurznamen von Go bleiben: `err`, `ok`, `ctx`, `tx`, `w`, `r`, `t`, in Tests `got`
  und `want`.

## 5. Türen

- Eine Tür ist eine öffentliche Methode am Typ `Tueren` ihres Themas. Ihr Name nennt die Sache
  und, wo sie etwas ändert, die Tätigkeit in der Grundform, ohne Endung: `Leserakte`,
  `LeserListe`, `LeserSperren`, `LMFPlanVeroeffentlichen`.
- Die Typen ihrer Anfrage und Antwort enden auf `Anfrage` und `Antwort` und stehen in der
  Datei der Tür.

## 6. Sichtbarkeit

Groß geschrieben ist, was ein anderes Paket beim Namen nennt, auch in seinen Tests, und der
Typ, den eine solche Funktion annimmt oder zurückgibt: Wer das Ergebnis in der Hand hält, muss
seinen Typ nachschlagen können. Alles andere ist klein. In einem Thema unter `api/` sind öffentlich der
Typ `Tueren` und seine Türen. Der Grund steht in Best Practices: „Every exported type increases
the cognitive load for the reader."

Konstanten, die zusammen eine Aufzählung bilden, sind als Ganzes öffentlich oder als Ganzes
klein (`lmf.FachMathematik` bis `lmf.FachNaWi`).

Eine Naht, die es nur gibt, damit ein Test etwas austauschen kann, endet auf `FuerTest`.

## 7. Dateien

Eine Datei heißt wie die Sache, die in ihr steht, deutsch und mit Unterstrich: `leser_sperre.go`.
Tests heißen `…_test.go`, Tests an der Datenbank `…_pg_test.go`, Prüfungen über den Quelltext
`…_ratsche_test.go`.

## 8. Kommentare

- Kommentare sind deutsch.
- Jeder öffentliche Name trägt einen Kommentar, der als ganzer Satz mit dem Namen beginnt
  (Decisions, „Doc comments"); der Linter hält es (`.golangci.yml`, revive `exported`).
- Ein Kommentar sagt in ein bis zwei Sätzen, warum der Code so ist. Was der Code tut, sagen
  die Namen. „It is often better for comments to explain why something is done, not what the
  code is doing" (Guide, Clarity).
- Kein Datum, keine Vorgeschichte, kein Verweis auf eine Nummer aus OFFEN.md oder einen Pull
  Request, kein Wort in Großbuchstaben zur Betonung. Die Geschichte steht in der
  Commit-Nachricht.
- Stehen bleibt ein Datum, das zu einer Quelle gehört: „Erlass vom 17.12.2014", „Beschluss vom
  21.09.2022", „abgerufen am 24.09.2026". Marke und Datum stehen in einer Zeile; daran erkennt
  die Prüfung sie.

## 9. Was seinen Namen behält

Adressen, die Felder in JSON und die Tabellen und Spalten der Datenbank mischen ebenfalls
Deutsch und Englisch (`idempotency_key` neben `idempotenz_schluessel`). Sie sind der Vertrag
mit der Oberfläche und mit den gespeicherten Daten und werden mit dem Code nicht umbenannt: Ein
Theken-Fenster mit altem Stand schickt seine gesammelten Buchungen unter den alten Namen. An
dieser Grenze trägt das Feld in Go den deutschen Namen und sein Tag den Namen des Vertrags.

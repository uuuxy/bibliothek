# Stil — wie der Code geschrieben ist

Stand: 16.10.2026

Eine Seite mit den Regeln für Namen, Sichtbarkeit und Kommentare im Go-Code, über `gofmt` und
den Linter hinaus. Maßstab ist Googles Regelwerk für Go
([Guide](https://google.github.io/styleguide/go/guide),
[Decisions](https://google.github.io/styleguide/go/decisions),
[Best Practices](https://google.github.io/styleguide/go/best-practices)). Dort steht an erster
Stelle die Klarheit, gesehen „through the lens of the reader, not the author", und erreicht
wird sie durch „effective naming, helpful commentary, and efficient code organization". Wo
etwas steht, sagt der [Wegweiser](WEGWEISER.md); was ein Fachwort bedeutet, das Glossar in
[ARCHITEKTUR.md](ARCHITEKTUR.md), Kapitel 12.

Programmcode und Tests folgen diesen Regeln; `stil_ratsche_test.go` meldet jeden neuen Fund, im
Programmcode auch an lokalen Namen, in Tests an Namen der Paketebene und Dateinamen,
`sichtbarkeit_ratsche_test.go` jeden öffentlichen Namen ohne fremden Nutzer gegen Abschnitt 6,
`fehlertext_ratsche_test.go` jeden Fehlertext und `meldung_ratsche_test.go` jede Meldung gegen
Abschnitt 11, `paketzustand_ratsche_test.go` jeden Zustand der Paketebene,
`testmeldung_ratsche_test.go` jede Meldung in Tests und `hilfsfunktion_ratsche_test.go` jede
Hilfsfunktion ohne `t.Helper()` gegen Abschnitt 10, `leerimport_ratsche_test.go` jeden Leerimport
und `kontext_ratsche_test.go` jeden selbst angelegten Kontext gegen Abschnitt 12, `ireturn` im
Linter jeden Erzeuger, der eine Schnittstelle liefert, gegen Abschnitt 14. Für die Oberfläche
gilt dasselbe Wörterbuch (Abschnitt 13). Neuer Code folgt den Regeln von Anfang an.

## 1. Sprache

- Was die Bibliothek und das Projekt benennen, heißt deutsch, ohne Umlaute geschrieben (`ae`,
  `oe`, `ue`, `ss`): `Leser`, `Rueckgabe`, `PruefeFrist`.
- Lehnwörter bleiben, wie man sie im Deutschen sagt: Barcode, Cover, Scan, Import, Export,
  Backup, Mail, Token, Cache, Cookie, Router, Middleware, Upload, Host, Snapshot, Bucket,
  Batch.
- Englisch ist nur, was Go oder eine Bibliothek vorgibt: `New…` und klein `new…` für den
  Erzeuger eines Typs (`NewClient`, `newKalender`; nicht `Neuer`, `Neue`, `Neues`, die mit dem
  Geschlecht des Typs wechselten), `Err…` für einen Fehlerwert, `Test…`, `Must…`, `main`,
  `init` und die Methoden fremder Schnittstellen (`Error`, `String`, `ServeHTTP`, `Scan`,
  `Close`).
- Ein Name, der ein Kopffeld von HTTP, dessen Wert oder einen Status nennt, folgt dem Protokoll
  (`HeaderContentType`, `ContentTypePDF`, `apierrors.BadRequest`): Wer ihn liest, sucht das Feld
  oder den Status unter seinem Namen.
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
| liest und schreibt eine Sache in der Datenbank      | `…Abfragen`                          | `…Repository`, `…Repo`, `pg…Abfragen` |
| führt einen Vorgang mit seinen Regeln aus (`internal/service`) | `…Dienst`                 | `…Service`, `…Svc`                |
| der Arbeitsplatz mit Scanner an der Ausleihe        | `Theke`                              | `Tresen`, `Counter`               |
| das eine Eingabefeld der Theke für Scan und Suche   | `Omnibox` (Glossar)                  | —                                 |

„Buch" steht nur in festen Wörtern (Buchakte, Zugangsbuch); sonst ist es der Titel oder das
Exemplar. Das technische Log des Servers heißt weiter `Log`.

Wiederkehrende Bausteine tragen dasselbe Wort: Die Angaben eines neuen Datensatzes enden auf
`Neu` (`AusleiheNeu`, `ExemplarNeu`), was ein Dienst zurückgibt, auf `Ergebnis`
(`AusleihErgebnis`), der Helfer, der eine Zeile der Datenbank liest, beginnt
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

`sichtbarkeit_ratsche_test.go` hält die Regel mit `go/types` über alle Pakete samt Tests.
Ausgenommen sind Methoden, die eine Schnittstelle verlangt (`Error`, `String`, `ServeHTTP`);
was die Prüfung nicht sieht, steht in ihrem Kopf.

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
- In einem Doc-Kommentar schreibt `gofmt` zwei Apostrophe als `”` und zwei Gravis als `“`.
  Der leere Text aus SQL (`''`) steht dort deshalb in Worten („auf den leeren Text"); im Rumpf
  einer Funktion bleibt er, wie er ist.

## 9. Was seinen Namen behält

Adressen, die Felder in JSON und die Tabellen und Spalten der Datenbank mischen ebenfalls
Deutsch und Englisch (`idempotency_key` neben `idempotenz_schluessel`). Sie sind der Vertrag
mit der Oberfläche und mit den gespeicherten Daten und werden mit dem Code nicht umbenannt: Ein
Theken-Fenster mit altem Stand schickt seine gesammelten Buchungen unter den alten Namen. An
dieser Grenze trägt das Feld in Go den deutschen Namen und sein Tag den Namen des Vertrags.

## 10. Tests

- Geprüft wird mit dem Paket `testing`. Bibliotheken für Zusicherungen wie testify gibt es nicht:
  „assertion libraries and third-party testing frameworks are not allowed" (Decisions, „Use
  package `testing`"). Der Linter sperrt testify (depguard in `.golangci.yml`).
- Einfache Werte vergleicht `==`, zusammengesetzte `cmp.Diff` oder `cmp.Equal` aus
  `github.com/google/go-cmp` (Decisions, „Equality comparison and diffs"). `reflect.DeepEqual`
  nicht: Es „is sensitive to changes in unexported fields and other implementation details".
  Der Import heißt `github.com/google/go-cmp/cmp`; goimports setzt sonst auch ein anderes Paket
  namens cmp ein. Für einen Typ des eigenen Pakets mit kleinen Feldern nennt der Vergleich
  `cmp.AllowUnexported`.
- Die Meldung nennt die Funktion, dann was sie lieferte, dann was erwartet war:
  `sichereCoverURL(%q) = %q, erwartet %q` (Decisions, „Identify the function", „Got before
  want"). Ein Unterschied aus `cmp.Diff` nennt seine Richtung: `(-erwartet +ist)`.
  `testmeldung_ratsche_test.go` meldet englische Wörter für Ergebnis und Erwartetes, eine Meldung,
  die mit „erwartet" beginnt, und das Erwartete vor dem Ergebnis.
- Eine Abweichung meldet `t.Error`, damit ein Lauf alle zeigt; `t.Fatal` steht dort, wo die
  folgenden Prüfungen sinnlos würden, etwa nach einem unerwarteten Fehler (Decisions, „Keep
  going").
- Ein Test setzt keine Variable der Paketebene um, auch nicht mit Zurücksetzen danach: Was er
  austauschen will, gehört einem Typ, den der Rufer anlegt und weiterreicht, wie die Basis der
  Türen mit Mailversand und Cache der Rechte (Best Practices, „Global state").
  `paketzustand_ratsche_test.go` hält es.
- Eine Hilfsfunktion, die den Test führt (`*testing.T`, `*testing.B`, `*testing.F`,
  `testing.TB`), beginnt mit `t.Helper()` und nimmt `t` vorn, vor ihm steht nur ein Kontext:
  „call t.Helper to attribute failures in the test helper to the line where the helper is
  called" (Decisions, „Test helpers"). Rückrufe wie der Rumpf von `t.Run` und die Funktionen in
  einer Tabelle von Fällen rufen es nicht, denn ihre eigene Zeile nennt den Fall: „Do not use
  this pattern when it obscures the connection between a test failure and the conditions that
  led to it" (ebenda). `hilfsfunktion_ratsche_test.go` hält es.
- Lokale Namen in Tests (Variablen, Parameter, Felder einer Tabelle von Fällen) werden nicht
  übersetzt; ein englischer Name dort bleibt stehen. `stil_ratsche_test.go` zählt in Testdateien
  nur die Namen der Paketebene und die Dateinamen.

## 11. Fehler und Meldungen

- Ein Fehlertext, also der Text von `errors.New`, `fmt.Errorf` und die Rückgabe einer Methode
  `Error()`, beginnt klein und endet ohne Satzzeichen: „Error strings should not be capitalized
  (unless beginning with an exported name, a proper noun or an acronym) and should not end with
  punctuation" (Decisions, „Error strings"). Das gilt auch für ein Substantiv am Anfang
  (`leser nicht gefunden`); ein Kürzel (`LUSD-Export …`), ein Go-Name oder ein Eigenname
  (`Excel-Datei …`) darf vorn stehen. `fehlertext_ratsche_test.go` hält es, auch in Tests; ein
  `nolint` dafür gibt es nicht.
- Den Satz für die Oberfläche bestimmt die Tür ([ARCHITEKTUR.md](ARCHITEKTUR.md), „Tür und
  Abfrage"). Er geht als Meldung hinaus, nicht als Fehlertext: hinter `MitFehlerantwort` mit
  `apierrors.Conflict("Satz", err)`, sonst mit `apierrors.AntworteFehlerMitMeldung(w, status,
  "Satz", err)`. Eine Prüfung neben der Tür liefert `apierrors.BadRequest("Satz", nil)`;
  `AntworteFehler` schickt dessen Meldung, die Ursache steht nur im Log. Ein Fehler ohne Meldung
  geht unter keinem Status als Text hinaus: `AntworteFehler` antwortet dann mit „Die Anfrage ließ
  sich nicht ausführen“.
- Eine Meldung ist ein Satz nach M3 (Content design, Grammar and punctuation): groß begonnen,
  deutsch, ein einzelner Satz ohne Punkt, mehrere Sätze mit Punkt, kein Gedankenstrich. Sätze, die
  mehrere Türen teilen, stehen als Konstante beim Thema (`auth.MeldungNichtAngemeldet`).
  `meldung_ratsche_test.go` hält es für jeden festen Text mit einem Fehlerstatus außer 500 (dessen
  Meldung ist neutral), auch 502 und 503, auch für den Formattext von `fmt.Sprintf`, für jede
  Textkonstante, deren Name mit `meldung` oder `Meldung` beginnt, für die festen Rückgaben einer
  Funktion `meldung…` oder `…Meldung` und für den Satz einer erfolgreichen Antwort (Schlüssel
  `message`); ein Satz, der über eine Variable in die Antwort kommt, steht deshalb als solche
  Konstante.
- Einen Fehler aus einer tieferen Schicht oder einer fremden Bibliothek gibt die Tür nicht als
  Meldung weiter, auch nicht mit `err.Error()`: Sein Text beginnt klein und ist oft englisch
  (`unexpected EOF`). Die Tür erkennt ihn am Wert und setzt ihren Satz; `tuer.LeseAnfrage` tut
  das für den Rumpf jeder Anfrage (`apierrors.MeldungAnfrageUnlesbar`, `tuer.Pruefmeldung`). Die
  eine Ausnahme ist die Diagnose des Mailservers (502): Ihr Text ist für das Formular der
  Testmail gebaut und geht mit Absicht hinaus (`api/system/mail_einstellungen.go`).
- Braucht der Satz Einzelheiten aus einer tieferen Schicht, trägt der Fehler sie als Felder,
  und die Tür formuliert (`repository.KontoHatAusleihenFehler`, `lusd.MassenabgangFehler`): „If
  there is extra information in the error that the caller needs programmatically, it should
  ideally be presented structurally" (Best Practices, „Error structure").
- Erklärt eine Prüfung ihre Regel selbst, steht der Grund als Feld neben der Regel
  (`lmfplan.SommerferienFehler.Grund`); die Tür setzt den Namen der Einstellung davor. In
  `repository/` baut `regelVerletzt` einen `regelFehler` aus dem benannten Fehler und dem Grund
  in Worten der Bibliothek („höchstens 60 Zeichen"); die Tür erkennt den Fehler mit `errors.Is`
  und setzt den Grund mit `repository.MitGrund` hinter ihren Satz („Standort nicht geändert:
  höchstens 60 Zeichen"). Die
  Sätze der Theke entstehen im Dienst, weil mehrere Türen sie zeigen; der Fehler trägt den Satz
  als Feld neben seiner Art, `service.MeldungAus` liefert ihn, `Error()` beginnt mit der Art.

## 12. Zustand der Paketebene

- Eine Variable der Paketebene hält nur, was sich nach dem Bau nicht ändert: Fehlerwerte,
  Tabellen und Texte, übersetzte reguläre Ausdrücke. Sicher ist solcher Zustand nur, wenn er
  „logically constant" ist, und „Libraries should not force their clients to use APIs that rely
  on global state" (Best Practices, „Global state").
- Was sich ändert, gehört einem Typ, den der Rufer anlegt und weiterreicht: die Bremse der
  Fehlversuche an `auth.Anmeldung`, die Sperre des Cover-Abgleichs an `service.CoverDienst`
  (einer für Start, Zeitplan und Knopf, `main.go`), die vertrauten Proxys am Server
  (`api.ServerKonfig`). Was nur eine Anfrage betrifft, reist in ihrem Kontext
  (`clientip.AusAnfrage`); was eine Tür allein braucht, bekommt sie als Parameter.
- Ein Programm (`package main`) darf Werte der Paketebene halten; Google lässt sie dort zu
  („must only be used by binary build targets").
- `paketzustand_ratsche_test.go` hält es: Ein Wert, dessen Anfangswert nicht erkennbar fest ist,
  steht mit Grund in `paketwerteOhneZustand`, und keine Funktion außer `init` weist einer
  Variable der Paketebene etwas zu. In Tests gilt Abschnitt 10.
- Ein Paket, das nur für seine Nebenwirkung eingebunden wird (`import _`), steht in einem
  Programm oder im Test, der es braucht; ausgenommen ist `embed` (Decisions, „Import blank":
  „Avoid blank imports in library packages, even if the library indirectly depends on them").
  Welche Bildformate der Server liest und dass er die API-Beschreibung anmeldet, legt `main.go`
  fest. `leerimport_ratsche_test.go` hält beides.
- Der Kontext einer Anfrage reist als erster Parameter bis zur Abfrage, die Läufe bekommen den
  des Servers; kein Typ trägt einen als Feld (Decisions, „Contexts": „Do not add a context member
  to a struct type"). `context.Background()` steht nur in Programmen, Tests und Prüfhilfen. Muss
  Arbeit das Ende der Anfrage überdauern (der Widerruf beim Abmelden, ein Sammelversand, der
  Abgleich der Cover), nimmt sie `context.WithoutCancel(ctx)`. Es halten `containedctx` und
  `contextcheck` im Linter und `kontext_ratsche_test.go`.

## 13. Oberfläche

- Für die Svelte- und JS-Dateien unter `frontend/src` gilt dasselbe Wörterbuch wie im Go-Code
  (Abschnitte 1 und 2): Was die Bibliothek benennt, heißt deutsch, ohne Umlaute, mit denselben
  Lehnwörtern. Die Komponente der Leserakte heißt `Leserakte`, nicht `StudentProfile`.
- Eine Komponente heißt wie die Sache, die sie zeigt, groß und ohne Trenner
  (`BestellDetail.svelte`); im Code trägt sie denselben Namen. Ein Modul mit Funktionen beginnt
  klein (`fokusFalle.js`), ein Modul mit Runen endet auf `.svelte.js`, sein Test heißt wie die
  Datei mit `.test.js`.
- Die Bauteile von M3 heißen deutsch, wo das Deutsche ein Wort hat: Karte (Card), Liste (List),
  Menue (Menu), Reiter (Tabs), Tabelle (Table), Feld (Text field), Kaestchen (Checkbox),
  Ladekreis (Progress indicator), Suchfeld (Search). Die übrigen behalten ihren Namen: Button,
  Chip, Dialog, Snackbar, Switch, Select, Radio, Badge, Tooltip. So stehen sie schon in
  `components/ui/`.
- Englisch bleibt, was Svelte, das Web oder eine Bibliothek vorgibt: Runen (`$state`),
  Ereignisse (`onclick`), die Ordner `stores`, `actions`, `utils`, `layout`, `ui`, Wörter wie
  `route` und `fetch`, und die Vorsilbe `use` eines Moduls mit Zustand (`useLeserakte`).
- Ein Ordner mit Fachbezug heißt deutsch wie sein Thema unter `api/` (`leser`, `etiketten`,
  `einstellungen`, `statistik`).
- `frontend/src/lib/frontend-hygiene-sprache.test.js` hält es für Datei- und Ordnernamen: Ihr
  Bestand nennt je Pfad, was noch umzubenennen ist, und darf nur sinken. Für Namen innerhalb
  der Dateien gilt dieselbe Regel; der zweite Teil derselben Prüfung zählt je Datei die
  erklärten Namen (let, const, var, function, class) mit einem englischen Wort, auch dieser
  Bestand darf nur sinken.

## 14. Schnittstellen

- Eine Schnittstelle entsteht erst, wenn es sie braucht: „Avoid creating interfaces until a
  real need exists" (Decisions, „Interfaces"). Den Bedarf schafft eine zweite Umsetzung
  (`tuer.Mailversand`: der eingesetzte Versand oder der Ersatz ohne Versand) oder eine Attrappe
  im Test.
- Sie steht beim Nutzer und trägt nur die Methoden, die er ruft: „The consumer of the interface
  should define it (not the package implementing the interface), ensuring it includes only the
  methods they actually use". Bekommt eine Tür eine Attrappe als Parameter, erklärt das Paket der
  Tür die Schnittstelle; tauschen die Tests eines Dienstes ein Feld aus, erklärt der Dienst sie.
- Sie heißt nach dem Vorgang, für den der Nutzer sie braucht, und ist klein geschrieben, weil nur
  ihr Paket sie nennt (`leserLoeschung` in `jobs`, `kontoPflege` in `api/benutzer`): „Focus on
  the required behavior rather than just abstract named patterns like “service” or
  “repository”" und „Keep interface types unexported if they are only used internally within a
  package" (Decisions, „Interfaces"). Jede Methode trägt einen Kommentar mit dem, worauf sich der
  Nutzer verlässt: „Multi-method interfaces: each individual method requires its own
  documentation" (Best Practices, „Designing effective interfaces").
- Brauchen mehrere Nutzer eines Pakets für denselben Vorgang dieselben Methoden, teilen sie sich
  eine Schnittstelle (`kontoPflege` für Anlegen und Ändern). Eine größere setzt sich aus kleineren
  zusammen (`leserErmittlung` aus `leserAbruf` und `ausweisErkennung`). Was die Dienste von
  `repository` brauchen, steht in `internal/service/abfragen.go`.
- Fehlt einer Attrappe eine Methode ihrer Schnittstelle, bettet sie diese kleine Schnittstelle
  ein, nicht den Typ aus `repository`: Ein Aufruf der fehlenden Methode endet dann am nil der
  eingebetteten Schnittstelle. Wer die echten Abfragen weiterreicht und nur einzelne Methoden
  ändert, bettet den Zeiger auf den Typ ein (`protokollScheitert` in `internal/service`).
- Ein Erzeuger liefert den Typ: „Functions should take interfaces as arguments but return
  concrete types". Eine Schnittstelle als Parameter steht dort, wo Verschiedenes hineinpasst
  (`repository.Datenbank`, `KontoSchreiber` und `SpurenDatenbank`: Pool oder Transaktion).
- `ireturn` im Linter meldet jede Funktion, die eine Schnittstelle dieses Projekts liefert
  (`.golangci.yml`); eine Ausnahme in der Konfiguration gibt es nicht. Wer mit Grund eine
  Schnittstelle liefert, schreibt den Grund an die Stelle (`//nolint:ireturn` an
  `tuer.Basis.Post`).


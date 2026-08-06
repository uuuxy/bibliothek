# Befund-Register

**Zweck:** Ein Ort für Dinge, die beim Arbeiten auffallen, aber nicht sofort angefasst
werden. Ohne diese Liste gibt es nur zwei Umgangsformen mit einem Fund — sofort
reparieren oder vergessen — und beide sind falsch, wenn das System kurz vor dem
Pilotbetrieb steht.

Ergänzt [`invarianten.md`](invarianten.md): Dort steht, was immer wahr sein *muss*.
Hier steht, was aufgefallen ist und noch nicht entschieden oder erledigt wurde.

---

## Die Einordnung

Vor jedem Fund steht dieselbe Frage — **nicht** „ist das hässlich?", sondern
**„kann das still jemandem schaden?"**:

| | Kategorie | Umgang |
|---|---|---|
| **A** | Kann **stillschweigend** ein falsches Ergebnis für einen echten Menschen erzeugen: doppelte Mahnung an Eltern, Daten beim falschen Empfänger, verlorene Eingabe, ein Gate, das nicht rot werden kann | **Sofort**, eigener Commit, eigener Test, eigene Deploy-Entscheidung |
| **B** | Fehler, der sich **laut** meldet, oder Unordnung ohne Wirkung nach außen: totes Codestück, wackeliger Test, Doppelung | **Hier notieren**, gebündelt nach dem Pilotstart |
| **C** | „Wenn ich schon mal hier bin" — Umbenennungen, Stilfragen, Refactorings ohne Anlass | **Nicht** vor dem Pilotstart |

Zwei Regeln dazu:

1. **Ein Fund = ein Commit.** Kein Anhängen verwandter Aufräumarbeiten. Was beim
   Reparieren zusätzlich auffällt, kommt in diese Liste, nicht in denselben Commit.
2. **Kategorie A wird belegt, nicht behauptet.** Ein Fund dieser Klasse braucht einen
   Test, der mit dem alten Code rot wird. Ohne diese Gegenprobe ist unklar, ob
   überhaupt etwas repariert wurde.

---

## Offen

| Fund | Warum offen |
|---|---|
| `golang.org/x/crypto/openpgp` gilt als unwartbar (`GO-2026-5932`) | Kein Fix verfügbar (`Fixed in: N/A`), transitive Abhängigkeit, **kein** Aufrufer im eigenen Code (`govulncheck`: „your code doesn't appear to call these"). Beobachten, nicht jagen. |
| `pgtest_support_test.go` liegt echt dupliziert in fünf Paketen | Sauber wäre ein `internal/pgtest`-Paket; fasst die Test-Infrastruktur von fünf Paketen auf einmal an und ist nur gegen echtes PostgreSQL beweisbar. Steht als Befund in `sonar-project.properties`. |
| Frontend liefert keine Coverage an SonarQube | `@vitest/coverage-v8` ist nicht installiert. |
| Etikettenraster stehen an vier Stellen | Maßgeblich ist `api/label_formats.go`; das Druck-Center führt eigene Kopien (`LabelLayoutOptionen.svelte`, `stores/labels.svelte.js`). Der Umbau auf die Server-Liste — wie bei der Lieferantenseite gemacht — wäre eine Verhaltensänderung an einem täglich benutzten Bildschirm. Bis dahin hält `etikettformate-konsistenz.test.js` die Kopien deckungsgleich. |
| Zwei verschiedene Vorgaben für dasselbe Raster | Druck-Center `avery_3475`, Lieferanten-Weg `zweckform_l4760`. Fällt praktisch nie auf, weil beide Oberflächen immer ein Format mitschicken — die Vorgabe greift nur bei einem Aufruf ohne `?format=`. Zu entscheiden ist, welche gelten soll. |

---

## Erledigt (2026-08-06) — Etiketten nach Rückmeldung Naacher

Zwei Wünsche aus einem Telefonat, beide am fertigen PDF abgesichert:

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| Große Lernmittel-Etiketten | „ein Etikett je Seite (A6)" | Auf einem A4-Drucker kam damit **ein** Etikett je Blatt heraus. Vier passen ohne jede Skalierung darauf — A6 ist exakt ein Viertel von A4. Jetzt 2×2 mit Schnittlinien. |
| Kleine Etiketten im Lieferanten-Link | „Bogen wie im Druck-Center" | Der Weg hatte `"zweckform_l4760"` **fest verdrahtet**, während das Druck-Center längst drei Raster anbietet. Wer andere Bögen im Drucker hatte, bekam einen Ausdruck, der danebenliegt. |

**Der Fund kam aus der Rückbau-Probe, nicht aus dem Code.** Der erste Test zur
Formatwahl rief den Generator direkt auf und blieb grün, als der Lieferanten-Weg
probeweise wieder auf das feste Raster gesetzt wurde — er sah die Durchreichung gar
nicht. Erst ein Test über den echten Endpunkt (`bestellbestaetigung_format_pg_test.go`)
wurde rot. Genau die Bugklasse dieses Projekts: ein Gate, das eine Stelle prüft, die
niemand kaputtmacht.

Dazu ein drittes Fundstück am Rand: Die Rasterdaten stehen an **vier** Stellen
(`api/label_formats.go`, `LabelLayoutOptionen.svelte`, `stores/labels.svelte.js`, dazu
zwei verschiedene Vorgaben — `avery_3475` im Druck-Center, `zweckform_l4760` im
Lieferanten-Weg). Der Umbau des Druck-Centers auf die Server-Liste wäre eine
Verhaltensänderung an einem täglich benutzten Bildschirm und ist **nicht** gemacht;
stattdessen hält `etikettformate-konsistenz.test.js` die Kopien deckungsgleich.

Stand: 2026-08-06

---

## Erledigt (2026-08-06) — Audit-Nachlese

Ein externes Audit meldete 16 Punkte. Nachgeprüft haben sich davon acht bestätigt, drei
in anderer Ursache als gemeldet, zwei waren falsch. Vier weitere Funde kamen erst beim
Nachprüfen dazu — sie standen in keiner Meldung.

**Der teuerste Fund stand nicht auf der Liste:** `internal/crypto` las den
Master-Schlüssel vorrangig aus `ENCRYPTION_KEY` und erst danach aus
`APP_ENCRYPTION_KEY`. Geprüft wird beim Start aber nur der zweite Name — auf Länge, auf
Hex-Form und (mit `ENFORCE_PROD_SECRETS`) gegen die bekannten Default-Werte. Ein
gesetztes `ENCRYPTION_KEY` hätte alle drei Prüfungen umgangen und still gewonnen; ver-
und entschlüsselt worden wäre mit einem anderen Schlüssel als dem geprüften.
Schülerfotos und das gespeicherte SMTP-Passwort wären damit nicht falsch, sondern
**weg**. Dieselbe Machart wie „zwei Türen zum selben Zustand": ein zweiter Weg zu einem
Wert, der nur an einem Weg abgesichert ist.

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `SECURITY.md`: „docker-compose.yml erzwingt per `${VAR:?}`, dass **alle** Secrets gesetzt sind" | Absicherung durch Compose | Nur `POSTGRES_PASSWORD` und `IMAP_HOST`. `JWT_SECRET` und `APP_ENCRYPTION_KEY` fallen auf die **committeten** Defaults zurück; einzige Absicherung ist der Code-Guard, und der ist standardmäßig aus. |
| `starttlsWennMoeglich` | „verifiziertes STARTTLS" | Bot der Server keines an, ging die Mahnung im Klartext raus — mit Schülernamen und Elternadressen. Ein MITM konnte das durch Streichen der Erweiterung selbst herbeiführen. |
| Cover-Proxy `/api/images/cover` | öffentlicher Bild-Cache | Einziger unauthentifizierter Pfad, der eine ausgehende Verbindung **und** ein volles `image.Decode` auslöst — ohne Größengrenze, ohne Dimensionsprüfung, ohne Bremse, mit dem Query-Parameter als Dateinamen. |
| `maskiereToken` „in beiden Logzeilen" | Token nirgends im Log | Die Panic-Recovery schrieb den rohen Pfad. Beim ersten Fix schlicht übersehen. |
| `ReadHeaderTimeout` am Server | Anfragen sind zeitlich begrenzt | Nur die Kopfzeilen. Ein Rumpf durfte beliebig langsam kommen; der Kontext-Deadline der `TimeoutMiddleware` bricht kein blockierendes `Read` ab. |
| `docker-compose.local.yml` | Entwicklungsstack, „nicht laxer als Prod" | Der DB-Port war auf `127.0.0.1` gebunden und begründet — das Backend eine Zeile darunter auf `0.0.0.0`, mit `IMAP_HOST=mock`, das jedes Passwort durchwinkt. |
| `SCRIPTS.md`: Backup-Pipeline „→ AES-GCM → 0600" | gilt für alle Backups | Galt nur für den nächtlichen Job. `scripts/backup.sh` **und** `./update.sh` legten unverschlüsselte Volldumps mit Standardrechten ab — letzterer war überhaupt nicht dokumentiert. |
| `scripts/backup.sh` meldete „Backup erfolgreich" | pg_dump lief durch | Ohne `pipefail` lieferte die Pipe den Status von `gzip`, und `gzip` gelingt auch dann, wenn `pg_dump` abgebrochen ist. |
| `Caddyfile` im Repo-Root | Vorlage für den Reverse-Proxy | Zeigte auf `bibliothek-backend-local:8083` — ein Container dieses Namens existiert nur lokal und lauscht auf 8084. |
| `useStudentEditForm({ student, … })` | Formular des ausgewählten Schülers | Destrukturieren nimmt einen Schnappschuss. `save()` hätte das PATCH an die **zuvor** geöffnete ID geschickt — mit Erfolgsmeldung. |
| `bind:this` auf `let` in `UnifiedInventory` | `$effect` setzt den Fokus ins Scanfeld | Ohne `$state` ist die Zuweisung nicht reaktiv: Der Effekt lief einmal, bevor das Feld existierte, und nie wieder. Jeder Scan wäre ins Leere gelaufen — ohne Fehlermeldung. |

**Zwei Meldungen waren falsch**, und beide auf dieselbe Weise: Sie beschrieben eine
Grenze, die es gab, als fehlend. Die XLSX-Importe haben `http.MaxBytesReader`
(100/20 MB) und hängen hinter `RequirePermission("manage_inventory")`. Und der
Klartext-Versand hätte die **SMTP-Zugangsdaten** nie preisgegeben: `smtp.PlainAuth`
verweigert die Übertragung über eine unverschlüsselte Verbindung von sich aus. Der
Inhalt war das Problem, nicht die Anmeldung — bei einem Relay ohne Zugangsdaten fiel
diese Bremse allerdings ganz weg.

**Merksatz dieses Tages:** Ein gemeldeter Befund ist eine Behauptung wie jede andere.
Drei der 16 stimmten in der Wirkung, aber nicht in der Ursache — wer sie so repariert
hätte, wie sie dastanden, hätte die falsche Stelle angefasst und den Fund für erledigt
gehalten.

Stand: 2026-08-06

---

## Erledigt (2026-08-04) — Littera-Schreibpfad

Fünf Funde, alle derselben Machart wie am 30.07.: Etwas behauptete etwas, und die
Behauptung war nie gegen die Wirklichkeit gehalten worden. Nur diesmal war die
Wirklichkeit nicht der Code, sondern **das Buch im Regal**.

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `barcode_id` aus der Exemplarnummer | „Auf dem Etikett steht die Exemplarnummer" | Der Scanner liest eine EAN-13: aufgedruckt `105785`, gescannt `1057850039567`. Keines der 61.520 Exemplare wäre auffindbar gewesen — gemerkt hätte man es an der Theke. |
| Schülerausweis `[0395] 37` | Der Aufdruck ist der Scanwert | Gescannt kommt `B97601826457`, die Nummer des Kartenherstellers. Sie steht in keinem Stammdatenfeld, nur in Litteras `FremdLeserNummer`. |
| „10.002 Titel mit Autor (93 %)" | Autorenabdeckung | Mitgezählt waren Standortvermerke, die als Personen erfasst sind: `Buchbestand Bibliothek` allein auf 6.711 Titeln. Bei 7.131 Titeln stünde ein Regalvermerk in der Autorenangabe. Echt sind 9.029 (84 %). |
| Geburtsdaten aus Littera | Jahrgänge der Personen | Gos Jahrhundertgrenze liegt bei 69: 69 Lehrkräfte der Jahrgänge 1946–1968 kamen als **2046–2068** an. |
| Präfixlose Omnibox-Auflösung | Buch → Schüler → Suche | Lehrkräfte stehen in `benutzer`, nicht in `schueler`. Ein gescannter Lehrerausweis lief bis in die Volltextsuche. Die passende Abfrage gab es längst — sie hing allein hinter `L-` und ließ sich als rohes SQL am Pool nicht testen. |

**Merksatz dieses Tages:** Aufdruck, Datenbankspalte und Scanwert sind drei verschiedene
Dinge. Für jede Barcode-Quelle einmal real in einen Texteditor scannen, bevor irgendetwas
nach `barcode_id` geschrieben wird.

Dazu, aus demselben Lauf und nicht kleiner: **`npm ci` lief gegen das committete Lockfile
gar nicht durch** (`typescript@^7` gegen `typescript-eslint@8.65`, das `<6.1.0` verlangt).
Lokal war ESLint grün, weil `node_modules` älter war als das Lockfile — auf einem frischen
Klon oder in CI wäre es rot gewesen. Ein grünes Gate beweist nichts über das, was im Repo
steht, wenn die Umgebung abgedriftet ist.

Stand: 2026-08-04

## Erledigt (2026-07-30)

Alle fünf Funde eines Tages waren dieselbe Art Fehler — etwas behauptete zu
funktionieren, und nichts prüfte die Behauptung nach:

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `expect([200, 400, 500])` im Mail-Test | prüft den Testversand | konnte nicht rot werden; schickte Felder, die die API nie gelesen hat |
| CSRF-Ausnahme für `/api/admin` & Co. | „Inventur-Modul hat eigenes CSRF" | dieses System gab es im Code nicht; sechs Admin-Mutationen ungeschützt |
| SMTP-Einstellungen im Admin-Bereich | steuern den Mailversand | wurden nur vom Test-Knopf gelesen; echte Mails nahmen die Umgebung |
| Diagnose bei SMTP-Fehlern | steht im Formular | jede 500 wurde zu „interner Datenbankfehler" eingedampft |
| Ergebnis des Mailversands | Nachricht zugestellt | war das Ergebnis der Verabschiedung; Abbruch danach galt als Fehlschlag |

**Merksatz:** Der Fundort ist immer die Stelle, an der eine Zusicherung steht, die
niemand prüft — ein Kommentar, ein Testname, eine Eingabemaske. Wer dort sucht,
findet; wer im Code sucht, findet Schönheitsfehler.

### Nachtrag desselben Tages: die Liste selbst abgearbeitet

| Fund | Was es behauptete | Was stimmte |
|---|---|---|
| `schadensfall.spec.js` fiel unter Last um | Test prüft den Elternbrief | `popup.url()` wurde gelesen, bevor die Navigation übernommen hatte — allein grün, im vollen Lauf manchmal rot. Jetzt wird die Adresse abgewartet und die Route zusätzlich am Inhaltstyp geprüft. |
| `SendTemplateMail` | Vorlagen-Versand des Systems | kein einziger Aufrufer; die Vorlagen werden anderswo direkt geladen. Entfernt. |
| `inventur`-Backup-Modul samt Benachrichtigungs-Mail | zweites Backup-System mit Mail bei Änderungen | `NewAPIHandler` hat das Feld nie gesetzt — unerreichbar. Gesichert wird über `jobs.BackupJob`. Die `.env.example` lud dazu ein, `BACKUP_EMAIL_TO` zu setzen und auf Mails zu warten, die nie kommen konnten. Ersatzlos entfernt. |

Der dritte Fund kam aus der Prüfung des zweiten: Der Eintrag stand hier als
„bewusst so entschieden" — und die Begründung hielt der Nachfrage nicht stand. Auch
eine Notiz in dieser Liste ist eine Zusicherung, die jemand prüfen muss.

Stand: 2026-07-30

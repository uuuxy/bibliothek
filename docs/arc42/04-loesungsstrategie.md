# 4. Lösungsstrategie

Stand: 17.09.2026

Dieses Kapitel nennt die tragenden Entscheidungen in Kurzform und ordnet sie den
Qualitätszielen aus [Kapitel 1](01-einfuehrung-und-ziele.md#12-qualitätsziele) zu. Die
ausführliche Fassung mit Datum, Anlass und Folge steht in
[Kapitel 9](09-architekturentscheidungen.md).

---

## 4.1 Die acht Leitentscheidungen

| #  | Entscheidung                                                                                     | Trägt                | Kurzbegründung                                                                                                                                                                                                 |
| -- | ------------------------------------------------------------------------------------------------ | -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| L1 | **Geschichteter Monolith**, ein Deployable: Handler → Service → Repository                        | Q5, Betrieb          | Ein Betreiber, ein Host, eine Datenbank. Ein Schnitt in Dienste würde Transaktionsgrenzen zerreißen, die heute die Korrektheit tragen (Ausleihe = eine Transaktion über Schüler, Ausleihe, Exemplar).            |
| L2 | **Die Datenbank ist die letzte Instanz**, nicht die Applikation                                   | Q1                   | Zwei Stationen können dieselbe Zeile ohne gemeinsame Sperre anfassen. Erst der partielle Unique-Index macht die zweite aktive Ausleihe *strukturell unmöglich*; Code-Prüfungen sind umgehbar, sobald ein zweiter Schreibpfad entsteht. |
| L3 | **Autorisierung pro Route, nicht global**                                                          | Q2                   | Eine globale Kette kennt die Route nicht, die sie schützt. `RequirePermission` sitzt **hinter** dem Routing (nur dort ist `r.PathValue` gefüllt) und prüft Recht, Kontostatus und UUID-Form in einem Zug. Ein Coverage-Gate zählt jede ungeschützte Route auf. |
| L4 | **Echtzeit über SSE, ohne Event-Loop**                                                             | Q1, Q4, Betrieb      | Alle Stationen sehen denselben Zustand nach dem Commit. Der Broker hält seinen Zustand hinter einem `RWMutex` statt hinter Kanälen — die frühere Kanal-Bauweise verhinderte das Herunterfahren und machte **jeden** Deploy zum `os.Exit(1)`. |
| L5 | **Idempotenz als Vertrag der Schreibtüren**                                                        | Q1, Q4               | Ein Scan darf doppelt ankommen (Netz, Nachbuchen, nervöse Hand). Der Idempotenz-Key liefert die gespeicherte Antwort zurück; 5xx wird nie gecacht, damit ein Wiederholen möglich bleibt.                          |
| L6 | **Der Tresen arbeitet auch ohne Netz weiter**                                                      | Q4                   | Scans landen in IndexedDB und gehen später durch die **Nachbuch-Tür**, die den Scan-Zeitpunkt buchen, einen Schlüssel genau einmal buchen und jede Abweichung als Meldung festhalten kann. Schweigen des Servers gilt nicht als Erfolg. |
| L7 | **Jede Bugklasse bekommt einen Detektor**                                                          | Q5                   | Ein Fix ohne Ratsche ist ein Rückfall auf Zeit. Das Register dieser Klassen und ihrer Detektoren ist [sweeps.md](../sweeps.md); die Landkarte nennt zu jeder Ratsche auch, was sie systembedingt **nicht** sieht. |
| L8 | **Das System muss sich selbst melden**                                                             | Q3                   | Die wiederkehrende Fehlerart ist nicht der Absturz, sondern die fertige Funktion, die still nichts tut, weil eine Einstellung fehlt. Dagegen stehen die Selbstprüfung, der tägliche Bereitschafts-Wächter per Mail und die wöchentliche Restore-Probe. |

---

## 4.2 Wie die Qualitätsziele technisch erreicht werden

### Q1 Korrektheit unter Nebenläufigkeit

Vier Ebenen, absteigend nach Verlässlichkeit:

1. **Struktur (DB):** `uniq_ausleihen_aktiv_exemplar` / `uniq_ausleihen_aktiv_geraet`
   (partielle Unique-Indizes), `check_loan_item` (Exemplar XOR Gerät), `check_return_date`,
   Fremdschlüssel auf `leser`, Enum-Vokabular für Rollen und Status.
2. **Transaktion + Zeilensperre:** `READ COMMITTED` mit `SELECT … FOR UPDATE`. Die
   **Sperrreihenfolge** ist festgelegt — Schüler → Ausleihe → Exemplar —, weil Online-Scan
   und Nachbuchen sich sonst gegeneinander verklemmen. Die Rückgabe mit Vormerkung nimmt
   `FOR UPDATE OF v SKIP LOCKED`, damit eine fremde Sperre den Rückgabevorgang nicht anhält.
3. **Idempotenz:** `idempotency_keys` mit gespeicherter Antwort, stündlicher TTL-Lauf,
   Wartezeit auf eine laufende Anfrage desselben Schlüssels. Der Unique-Index deckt den
   TOCTOU-Fall ab, in dem zwei Anfragen die Idempotenz-Prüfung gleichzeitig passieren.
4. **Fehler ehrlich abbilden:** Eine Unique-Verletzung wird zu **409 Conflict**
   (`mapLoanCreateErr`), nicht zu 500 und nicht zu einem stillen Erfolg. Die Testklasse
   dazu heißt `phantom_erfolg_test.go` — „hat es geklappt?" muss beantwortbar sein.

### Q2 Datenschutz-Konformität

- **Ein Ort für Schülerdaten, eine gefilterte Sicht für „wirklich Schüler":** Seit
  Migration 123 stehen Schüler und Kollegium in `leser` mit der Spalte `art`; `schueler`
  ist eine **Sicht** mit `WHERE art = 'schueler'` und `WITH CHECK OPTION`. Klassenlisten,
  LUSD-Abgleich, Mahnlauf und die Löschfristen lesen die Sicht und sehen das Kollegium nicht.
- **Verschlüsselung am Ruhepunkt:** Schülerfotos und das gespeicherte SMTP-Passwort liegen
  AES-256-GCM verschlüsselt in der Datenbank (`internal/crypto`); Backups zusätzlich mit
  scrypt-abgeleitetem Schlüssel (`internal/backupkrypto`). Für den Schlüsselwechsel gibt es
  ein Werkzeug mit Probelauf (`cmd/rotate-encryption-key`).
- **Fristen als Code, nicht als Vorsatz:** Anonymisierung, Karenz und Hard-Delete der
  Abgänger laufen als Cronjob in festgelegter Reihenfolge (Anonymisierung **vor** Löschung,
  damit die Karenz für beides gilt).
- **Einstufung jeder Route:** [PII_MATRIX.de.md](../PII_MATRIX.de.md) ordnet jede Route
  einer Stufe 0–3 zu; `api/pii_matrix_test.go` hält Dokument und Code deckungsgleich.
- **Kein Personenbezug in Logs:** Die Logzeile je Anfrage trägt keine IP; der
  Bestätigungstoken im Pfad wird maskiert.

### Q3 Betriebstransparenz

| Mittel                                        | Beantwortet die Frage                                                         |
| --------------------------------------------- | ----------------------------------------------------------------------------- |
| `GET /health` (mit DB-Ping)                   | Läuft der Prozess und erreicht er die Datenbank?                              |
| Selbstprüfung der Betriebsbereitschaft        | Was ist **eingerichtet, aber nicht in Betrieb** (Mail, Selbstanmeldung, Geheimnisse, Backup)? |
| Bereitschafts-Wächter (3 min nach Start, dann täglich) | Muss jemand hinsehen? Der Wächter meldet sich per Mail, statt gelesen werden zu müssen; auf Spielwiesen schweigt er von selbst. |
| Restore-Probe (wöchentlich So 03:30)          | Ist das Backup **wiederherstellbar** — nicht nur vorhanden?                    |
| Start-Verweigerung bei Default-Geheimnissen   | Läuft der Schulserver mit dem Schlüssel aus dem Repository?                    |
| Skip-Bilanz im pre-push-Hook                  | Was hat dieser Lauf **nicht** geprüft?                                         |

### Q4 Bedienbarkeit am Tresen

- **Ein Feld für alles.** Die Omnibox löst ohne Präfix in der Reihenfolge Buch → Ausweis →
  Volltextsuche auf; Präfixe (`B-`, `A-`, `G-`, historisch `S-`/`L-`) sind eine Abkürzung.
  Die Vorsilbe bleibt trotzdem nötig, weil sie **offline** die einzige Information ist, an
  der ein Buchscan von einem Ausweisscan zu unterscheiden ist.
- **Nachsicht für alte Aufdrucke.** Bleibt ein Scan ohne Treffer, wird ein mögliches
  Code-39-Prüfzeichen abgeschnitten und **ein zweiter Versuch** gestartet — nur als zweiter
  Versuch, weil im Schnitt jeder 43. gültige Code zufällig so aussieht, als hinge eines dran.
- **Sichtschutz statt Logout.** Nach 5 Minuten leert sich die Theke, nach 15 kommt der
  Sperrbildschirm; die Sitzung läuft weiter (Mehrplatzrechner).
- **Barrierefreiheit gemessen, nicht behauptet:** axe über den Anfangszustand aller
  Hauptansichten, dazu Gates für Fokusfalle, Tabellen und Bewegung.

### Q5 Änderbarkeit

- **Testebenen mit klarer Zuständigkeit:** Unit (Regeln), `*_pg_test.go` (Constraints gegen
  echtes Postgres), e2e gegen den **gebauten Container** (was ausgeliefert wird).
- **Drift-Gates gegen auseinanderlaufende Wahrheiten:** Swagger gegen Annotationen,
  Migrationsliste gegen `schema.sql`, Compose-Variablen gegen den Code, Dokumentzahlen
  gegen die Rechenfunktion, Fundstellen gegen `schema.sql`.
- **Deadcode-Gate mit Baseline:** Unerreichbarer Code wird sichtbar, statt als
  Scheinsicherheit dazustehen.
- **Eine Wahrheitsquelle je Frage.** Menü und Router fragen dieselbe Funktion; die
  Host-Allowlist steht in einem Paket; die Rechte der Leitung werden aus den Admin-Rechten
  **abgeleitet** statt abgeschrieben.

---

## 4.3 Bewusst nicht gewählte Alternativen

| Alternative                                  | Warum nicht                                                                                                                                                                        |
| -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Microservices**                            | Die Korrektheit der Ausleihe hängt an einer Transaktion über drei Tabellen. Verteilte Transaktionen wären ein Vielfaches an Komplexität für null fachlichen Gewinn bei acht Arbeitsplätzen. |
| **ORM (GORM/ent)**                            | Die kritischen Stellen sind Sperren, partielle Indizes und Sichten mit `CHECK OPTION`. Genau das ist der Teil, den ein ORM verdeckt.                                               |
| **Web-Framework (Gin/Chi/Echo)**              | Der Bedarf ist Methoden-Routing plus sechs Middlewares — das kann `net/http` seit Go 1.22 selbst. Der Preis (eigene Kette, eigene Gates) ist bezahlt und dokumentiert.             |
| **WebSockets statt SSE**                       | Der Datenfluss ist einseitig (Server → Stationen). SSE kommt ohne Protokoll-Upgrade durch den Reverse Proxy und reconnectet von selbst.                                            |
| **Eigene Benutzerverwaltung mit Passwörtern**   | Ein zweiter Passwortspeicher in einer Schule ist ein Risiko ohne Nutzen. Es gibt keine Passwortspalte — und damit auch keinen Passwort-Leak.                                        |
| **Redis/Memcached für Cache und Rate-Limit**    | Ein Prozess, ein Host: In-Memory reicht und spart eine Betriebskomponente. Der Preis ist die fehlende horizontale Skalierbarkeit (dokumentiert in [Kapitel 11](11-risiken-und-technische-schulden.md)). |
| **TypeScript im Frontend**                     | JSDoc mit `checkJs` und `svelte-check --fail-on-warnings` liefert die Prüfung ohne den Umbau von 286 Komponenten.                                                                   |
| **Kubernetes**                                  | Ein Schulserver. `docker compose` plus `update.sh` ist die Betriebsform, die eine Person im Ernstfall noch versteht.                                                               |
| **Soft-Delete überall**                        | Wo die DSGVO Löschung verlangt, ist ein Soft-Delete keine Löschung. Es gibt Soft-Deletes im Bestand (Aussonderung mit Grund), aber die Abgänger-Tilgung ist ein Hard-Delete.        |

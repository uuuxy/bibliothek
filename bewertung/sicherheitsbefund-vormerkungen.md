# Security Finding — Vormerkungsliste gibt Schülernamen an die Helfer-Rolle heraus

**Schweregrad:** Mittel–Hoch
**Kategorie:** Fehlerhafte Autorisierung / Datenpreisgabe (Rolle unter der vorgesehenen Schwelle)
**Status:** Bestätigt durch Code-Analyse

## Kurzfassung

`GET /api/vormerkungen` ist nur mit dem Recht `view_books` geschützt. Dieses Recht besitzt
auch die Kiosk-Rolle `HELFER`. Der Handler gibt für jede Vormerkung den **Namen und die
Klasse des vormerkenden Schülers** sowie dessen `schueler_id` zurück. Ohne Filterparameter
liefert er die **gesamte** Vormerkungsliste. Ein Theken-Helfer kann damit auslesen, welcher
namentlich genannte Schüler welches Buch vorgemerkt hat — eine Verknüpfung von Identität und
Leseinteresse, die der Rolle bewusst verschlossen sein soll.

Das widerspricht direkt der dokumentierten Annahme in `db/seed.go` (Zeilen 169–174), mit der
`view_books` an `HELFER` vergeben wurde:

> `Das Recht ist rein lesend und öffnet keine Personendaten — die stecken hinter
> view_students …`

Die Vormerkungsliste öffnet sehr wohl Personendaten.

## Betroffene Stellen

- Route: `api/routes_books.go:51` — `mux.Handle("GET /api/vormerkungen", s.RequirePermission("view_books")(...))`
- Handler: `api/vormerkungen.go:20` (`ListVormerkungHandler`) — gibt `[]repository.Vormerkung` unverändert zurück
- Query/Struct: `repository/vormerkung.go` — `List(...)` selektiert
  `s.vorname || ' ' || s.nachname || ', ' || s.klasse` als `SchuelerName` und `s.id` als `SchuelerID`
- Rechtevergabe: `db/seed.go:174` (`HELFER` → `view_books = true`)

Betroffen sind neben `GET /api/vormerkungen` auch `POST /api/vormerkungen` und
`DELETE /api/vormerkungen/{id}` (ebenfalls `view_books`): Ein Helfer kann Vormerkungen für
beliebige `schueler_id` anlegen und fremde löschen. Der Lesezugriff ist die primäre
PII-Preisgabe; die Schreibrechte sind ein zusätzlicher Nebenbefund.

## Repro (Angreifermodell: niedrig-privilegierter Kiosk-Helfer)

1. Als `helfer`-Konto anmelden (lokal: Mock-IMAP akzeptiert jedes Passwort).
2. Mit dem `session_token`-Cookie ohne weitere Parameter aufrufen:
   ```
   GET /api/vormerkungen
   ```
3. Die Antwort ist ein Array, in dem jeder Eintrag `schueler_name` ("Vorname Nachname,
   Klasse"), `schueler_id` und den vorgemerkten Buchtitel (`titel`) enthält.

## Empfohlene Behebung

Zwei Wege, je nach fachlicher Absicht:

- **Wenn der Helfer keine Vormerkungen sehen soll:** Die Vormerkungs-Endpunkte auf ein
  Recht heben, das der Helfer nicht besitzt (z. B. `view_students` für den Lesezugriff,
  passend zur übrigen Schüler-PII), und Schreibzugriff an ein Verwaltungsrecht binden.
- **Wenn der Helfer die Warteschlange fachlich braucht:** Für den `view_books`-Zugriff eine
  reduzierte Projektion ohne `schueler_name`/`schueler_id` zurückgeben (nur Titel, Position,
  Zeitstempel) und die namentliche Sicht separat hinter `view_students` legen.

Der Kernpunkt: Das Recht `view_books` wurde ausdrücklich als „keine Personendaten"
eingeführt. Entweder muss die Vormerkungsliste diese Zusage einhalten (Projektion ohne
Schülerbezug), oder der Endpunkt gehört hinter ein stärkeres Recht.

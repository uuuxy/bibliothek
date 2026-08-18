# Security Finding — Kiosk-Suche gibt vollständige Schüler-PII an die Helfer-Rolle heraus

**Schweregrad:** Hoch
**Kategorie:** Fehlerhafte Autorisierung / Datenpreisgabe (horizontale Über-Exposition an eine unterprivilegierte Rolle)
**Status:** Bestätigt durch Code-Analyse (Repro-Pfad unten)

## Kurzfassung

Der Endpunkt `GET /api/search` ist nur mit dem Recht `perform_actions` geschützt. Dieses
Recht besitzt auch die Kiosk-Rolle `HELFER`. Der Handler liefert jedoch den **vollständigen**
Schüler-Datensatz als JSON zurück — inklusive **Wohnanschrift, Eltern-E-Mail und
Geburtsdatum**. Damit kann ein Theken-Helfer die Meldeadressen und Elternkontakte der
gesamten Schülerschaft auslesen, obwohl die Rolle bewusst *keinen* Zugriff auf Schülerprofile
haben soll.

## Betroffene Stellen

- Route: `api/routes_misc.go:35` — `mux.Handle("GET /api/search", s.RequirePermission("perform_actions")(searchHandler))`
- Handler: `api/search.go:23` (`SearchHandler`) — serialisiert `repository.Student` unverändert
- Query: `repository/student_queries.go:146` (`SearchStudentsFuzzy`)
- Struct: `repository/models.go:31` (`repository.Student`)
- Rechtevergabe: `db/seed.go:162` (`HELFER` → `perform_actions = true`)

## Worin der Fehler besteht

`SearchHandler` gibt die komplette `repository.Student`-Struktur zurück. Die zugrunde
liegende Abfrage selektiert den vollen Datensatz:

```sql
... coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''),
    coalesce(ort, ''), coalesce(eltern_email, ''), TO_CHAR(geburtsdatum, 'YYYY-MM-DD') ...
```

Alle diese Felder haben einen JSON-Tag ohne `omitempty` und landen damit in der Antwort:
`strasse`, `hausnummer`, `plz`, `ort`, `eltern_email`, `geburtsdatum`.

Die Route hängt an `perform_actions`. Dieses Recht ist der Kiosk-Rolle `HELFER` zugeteilt.
Genau derselbe Seed-Block dokumentiert die gegenteilige Absicht:

> `HELFER … Bewusst KEIN view_students: das gäbe Zugriff auf Schülerlisten, Profile,
> Mahnwesen …`

Alle anderen PII-Flächen (`/api/schueler`, `/api/schueler/{id}`, `/api/mahnwesen*`, das
Foto-Endpunkt) sind korrekt an `view_students` gebunden, das `HELFER` (und `KOLLEGIUM`)
verwehrt bleibt. `/api/search` ist der einzige Pfad, der dieselben Profildaten hinter dem
schwächeren `perform_actions`-Recht herausgibt — und durchbricht damit die dokumentierte
Grenze.

## Repro (Angreifermodell: niedrig-privilegierter Kiosk-Helfer)

1. Als `helfer`-Konto anmelden (Standard-Rolle des geteilten Thekenterminals). Lokal
   akzeptiert Mock-IMAP (`IMAP_HOST=mock`, `APP_ENV=local`) jedes Passwort für jeden
   bekannten Benutzer.
2. Mit dem `session_token`-Cookie aufrufen:
   ```
   GET /api/search?q=a
   ```
3. Das `students[]`-Array der Antwort enthält je Treffer (bis zu 10 pro Anfrage) den
   vollen Namen, die Klasse, **Straße + Hausnummer + PLZ + Ort**, die **Eltern-E-Mail**
   und das **Geburtsdatum**. Durch Iterieren über Einzelbuchstaben und häufige
   Namensbestandteile lässt sich die gesamte Schülerschaft samt Meldeadressen abziehen.

Der legitime Bedarf des Kiosks bei der Ausleihe ist Name + Klasse + Sperrstatus zur
Identitätsbestätigung — nicht Wohnanschrift, Elternkontakt oder Geburtsdatum.

## Empfohlene Behebung

Auf dem Suchpfad eine schlanke Projektion statt des vollen `Student`-Datensatzes
zurückgeben. Bevorzugt spaltenscharf, damit die PII gar nicht erst in den Prozess gelangt:

- Eigenes Kiosk-DTO (`id`, `barcode_id`, `vorname`, `nachname`, `klasse`, `ist_gesperrt`,
  Block-Status) einführen und `SearchStudentsFuzzy` nur diese Spalten selektieren lassen; **oder**
- Die Abfrage belassen, aber in `SearchHandler` auf eine reduzierte Antwortstruktur
  mappen und `strasse`, `hausnummer`, `plz`, `ort`, `eltern_email`, `geburtsdatum`
  weglassen, sofern der Aufrufer nicht zusätzlich `view_students` besitzt.

Der spaltenscharfe Weg ist robuster: Er hält die PII vollständig aus dem Kiosk-Pfad heraus,
statt sich auf einen Serialisierungsfilter zu verlassen.

## Gleiche Ursache, zweiter Pfad: `POST /api/action` (Barcode-Scan)

Derselbe volle `Student`-Datensatz wird auch beim Kiosk-Scan zurückgegeben:

- Route: `api/routes_misc.go:30` — `POST /api/action`, Recht `perform_actions` (Helfer)
- `ActionResponse.Student` ist `*repository.Student` (voller Datensatz) — `api/action_types.go:41`
- Befüllt bei jedem Schüler-Barcode-Scan: `internal/service/omnibox_service.go:194` (`handleStudentAction` → `resp.Student = student`)

Etwas geringere Schwere als die Suche, weil ein gültiger Schüler-Barcode bekannt/gescannt
sein muss (ein Nicht-Barcode-Freitext fällt in die reine Buchsuche). Der Ausleihvorgang
liefert dem Kiosk-Bediener aber weiterhin die volle Wohnanschrift und die Eltern-E-Mail des
gescannten Schülers — über den Kiosk-Zweck hinaus. Die Behebung ist dieselbe: reduzierte
Projektion für `perform_actions`-Aufrufer.

## Anmerkung zum Umfang

Der volle `Student`-Datensatz ist auf den `view_students`-geschützten Profil-/Listen-Endpunkten
(Admin/Mitarbeiter) angemessen. Der Fehler ist ausschließlich, dass der Fuzzy-Search-Handler
dieselbe volle Struktur auf einer `perform_actions`-Route wiederverwendet.

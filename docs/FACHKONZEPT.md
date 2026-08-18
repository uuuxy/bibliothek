# Fachkonzept & Feature-Spezifikation

Beschreibt die funktionale Business-Logik der Bibliothekssoftware auf Basis der Go-Implementierung. Zielgruppe: Administratoren, Betreiber und Entwickler.

---

## 1. Zentrale Scanner-Omnibox (Kiosk)

Die Software nutzt ein eingabefokussiertes Kiosk-Design für die Ausleihe und Rückgabe. Ein
einziges Eingabefeld (Omnibox) verarbeitet alle Scans.

**Der Normalfall ist ohne Präfix.** Die Ausweise und Buchetiketten aus dem Littera-
Altbestand tragen nackte Nummern und dürfen nicht neu gedruckt werden; ein Scan wird
deshalb der Reihe nach aufgelöst:

```
Buch-Exemplar → Schülerausweis → Lehrerausweis → Volltextsuche im Katalog
```

Das geht auf, weil die Formen verschieden sind: Buchetiketten liefern eine 13-stellige
EAN-13, Schülerausweise die Nummer ihres Kartenherstellers. Die Vorgängersoftware Littera
kennt überhaupt keine Präfixe — sie hat getrennte Suchfelder, wir haben eines für alles.

**Präfixe sind eine Abkürzung**, kein Muss. Sie überspringen die Auflösung und sprechen
den Bereich direkt an:

- **`S-[Barcode]` (Schüler):** Lädt das Konto eines Schülers (inkl. offener Ausleihen, Mahnungen und Sperren).
- **`L-[Barcode]` (Lehrer):** Lädt das Konto eines Lehrers (Handapparat).
- **`B-[Barcode]` (Buch-Exemplar):** Führt eine Aktion mit einem Buch aus.
- **`G-[Barcode]` (Gerät):** Führt eine Aktion mit Hardware (z. B. Laptops, iPads) aus.

**Ablauflogik:**
1. Wird ein Buch gescannt, *ohne* dass ein Schüler/Lehrer aufgerufen ist, wird das Buch **zurückgegeben**.
2. Wird ein Schüler/Lehrer gescannt und danach Bücher gescannt, werden diese an die Person **ausgeliehen**.

---

## 2. Ausleih-Regelwerk und Fristen

Das System unterscheidet zwischen verschiedenen Medien und Leihertypen:

### 2.1. Fristenberechnung
- **Lernmittelfreiheit (LMF) - "Schulbücher":** Haben ein fixes Rückgabedatum: den **31. Juli** des laufenden (oder bei Sommer-Ausleihe des kommenden) Schuljahres.
- **Freihand-Bestand (Sonderbestände):** CDs, DVDs, Hörbücher etc. haben eine rollierende Frist (z. B. +14 oder +28 Tage ab Ausleihe), keine starre Jahresfrist.
- **Ferien-Logik:** Fällt das berechnete Rückgabedatum in die Schulferien, wird die Frist automatisch bis zum ersten Schultag nach den Ferien verlängert.
- **Lehrer (Handapparat):** Erhalten pauschal eine Frist von einem Jahr — `AddDate(1, 0, 0)`, also ein **Kalenderjahr**, nicht 365 Tage (im Schaltjahr sind es 366). Wie jede andere Frist läuft sie durch `tagesEndeInSchulzeitzone`; eine zweite, rohe Berechnung gibt es bewusst nicht.
- **Verlängerungen:** Ausleihen können verlängert werden, es sei denn, der Schüler ist gesperrt oder hat das Ausleihlimit überschritten.

### 2.2. Blockaden und Limits
- **Ausleihlimit:** Es gibt ein konfigurierbares Maximum an gleichzeitigen Ausleihen pro Schüler (LMF-Bücher ausgenommen).
- **Sperre bei Überfälligkeit:** Hat ein Schüler mehr als `MaxOverdueItems` überfällige Medien, wird das Konto automatisch für neue Ausleihen gesperrt.
- **Manuelle Sperre:** Administratoren können Schüler manuell sperren (z. B. bei massivem Fehlverhalten). Ein verpflichtender Begründungstext (`block_reason`) wird stets verlangt und den Helfern angezeigt.

---

## 3. Mahnwesen

Das Mahnsystem durchläuft einen 3-stufigen, rechtlich bindenden Eskalationsprozess.

- **Stufe 1 (Erinnerung):** Kann als kostenlose E-Mail ("Friendly Reminder") oder als PDF-Ausdruck versendet werden.
- **Stufe 2 & 3 (Kostenpflichtig):** Erzeugen Mahngebühren. Diese **Mahnstufe erhöht sich ausschließlich beim physischen PDF-Druck**, da dies den rechtlichen Verwaltungsakt darstellt. Der reine E-Mail-Versand einer Erinnerung führt *nicht* zur Erhöhung der Mahnstufe oder zu neuen Gebühren.
- **Sperr-Folge:** Bei Erreichen von Stufe 3 (oder Nichtzahlung) kann der Schüler gesperrt werden.

---

## 4. Vormerkungen und Klassensatz-Reservierungen

Das System verwaltet den Mangel an verfügbaren Büchern durch zwei Konzepte:

### 4.1. Einzel-Vormerkungen
- Ein Schüler kann ein Buch vormerken, wenn kein Exemplar mehr frei ist.
- **Rückgabe-Match:** Wird ein Exemplar dieses Titels zurückgegeben, prüft das System, ob eine Vormerkung vorliegt.
- **Abholbereitschaft:** Das Buch wird nicht freigegeben, sondern direkt dem wartenden Schüler zugeteilt (Status `abholbereit`). Es landet physisch im Bereitstellungsregal.
- *Schutz:* Es ist technisch unmöglich, dass ein Schüler ein Buch vormerkt, das er aktuell selbst ausleiht (Vermeidung von Monopolisierung).

### 4.2. Klassensatz-Reservierungen

**Reservieren heißt Anstellen, nicht Sperren** (Betreiber-Entscheidung 16.08.2026 — bis
dahin beschrieb dieser Abschnitt ein Blockier-Modell, das nie gebaut war):

- Eine Lehrkraft reserviert im Kollegiums-Portal Titel + Klasse + Anzahl. Die einzige
  harte Grenze: nicht mehr, als die Bibliothek physisch besitzt.
- Die Reservierung **blockiert nichts** — weder Exemplare noch Rückgaben. Sie ist ein
  Arbeitsauftrag an die Bibliothek und reiht sich in eine **sichtbare Warteschlange**
  ein (älteste zuerst).
- Das Portal zeigt bestehende Reservierungen **vor** dem Klick am Treffer
  („28 reserviert für 8a“); wer trotzdem reserviert, erfährt in der Bestätigung, hinter
  wem sein Satz an der Reihe ist.
- Die Bibliothek arbeitet die Schlange unter Bestellungen → Klassensatz-Reservierungen
  ab: je Zeile Anfragende(r) mit Namen und der aktuelle Regal-Bestand („N verfügbar“).
  „Abschließen“ beendet den Vorgang nach der physischen Übergabe — die Ausleihe selbst
  läuft über die normalen Wege (Kiosk je Schüler oder Lehrer-Handapparat).

### 4.3. Wünsche & Meldungen der Lehrkräfte

Seit 18.08.2026 (Betreiber-Entscheidung: bewusst schlank, kein Ticketsystem):

- **Ein Mechanismus für zwei Fälle:** „Ich möchte in der 8G3 den Markl 2" (Wunsch)
  und „die 8G3 hat die falschen Bücher bekommen" (Meldung). Die Lehrkraft trägt es
  im Kollegiums-Portal ein — Art, Freitext, Klasse/Kurs, optionale Anmerkung.
- **Wünschen geht immer** — keine Wunschphase, kein Stichtag.
- Die Bibliothek arbeitet die Liste unter Bestellungen → „Wünsche & Meldungen" in
  Ruhe ab (älteste zuerst). **Abhaken** schließt das Anliegen und schickt der
  Lehrkraft automatisch eine Mail — mit der optionalen Notiz („bestellt, kommt
  Anfang September"). Ein Doppelklick von zwei Arbeitsplätzen löst keine zweite
  Mail aus (gleiches Muster wie die Klassensatz-Bereit-Mail).
- Die Lehrkraft sieht ihre Anliegen samt Status und Erledigungs-Notiz im Portal.
- Bewusst NICHT gebaut: Prioritäten, Kommentar-Threads, Genehmigungsketten,
  Deckungsprüfung, Packlisten — erst nachrüsten, wenn der Alltag sie vermisst.

---

## 5. Geräteausleihe (Hardware)

Seit 16.08.2026 vollständig in Betrieb (vorher Backend-Torso ohne Oberfläche —
dieser Abschnitt beschrieb die Absicht als Realität):

- **Verwaltung** im Medienkatalog, Bereich „Geräte“ (kein eigener Menüpunkt —
  ausgebucht wird ohnehin am Kiosk): anlegen (Barcode zwingend mit `G-`-Präfix,
  sonst findet der Kiosk-Scan das Gerät nicht), Liste mit aktuellem Ausleiher,
  Zubehör- und Stammdaten-Pflege, Defekt-Schalter (`ist_ausleihbar`).
- **Checklisten-Zwang am Kiosk:** Trägt ein Gerät Zubehör (kommaseparierte Liste),
  unterbricht der Scan mit einem Bestätigungs-Dialog, der jedes Teil nennt —
  bei Ausleihe UND Rückgabe. Erst „Alles vollständig“ schickt den Scan mit
  Bestätigung erneut; Abbrechen bucht nichts.
- **Fristen:** 14 Tage, auf das Tagesende in der Schul-Zeitzone normalisiert wie
  Buch-Fristen; Lehrkräfte leihen Geräte als Handapparat (Dauerleihe).
- **Zustands-Sperre:** Defekte (`ist_ausleihbar = false`) und ausgesonderte Geräte
  verweigern die Ausleihe am Kiosk.
- Fehlendes Zubehör bei der Rückgabe: Das Personal bricht im Dialog ab und meldet
  den Schaden über den bestehenden Schadensfall-Weg am Profil.

---

## 6. Inventur-System

Die Inventur findet im laufenden Betrieb statt, ohne dass die Bibliothek zwingend schließen muss.

- **Session-basiert:** Jede Inventur erhält einen Scope (z. B. "Raum 2, Regal A") und läuft in einer eigenen Session (`inventur_sessions`). Mehrere Mitarbeiter können mit Handscannern parallel inventarisieren, ohne sich gegenseitig zu überschreiben.
- **Fehlmengen-Ausbuchung:** Wird die Session beendet (`Finish`), vergleicht das System alle gescannten Exemplare mit dem theoretischen Bestand in diesem Scope.
- **Schutz aktiver Ausleihen:** Bücher, die laut Datenbank aktuell *verliehen* sind, werden vom System bei der Fehlmengenberechnung ignoriert – sie können nicht versehentlich als Verlust ausgebucht werden, nur weil sie nicht im Regal standen.
- Fehlende, nicht verliehene Exemplare erhalten automatisch den Status `VERLUST`.

---

## 7. Bestellwesen und Wareneingang

Die Software verwaltet Bedarfe und Lieferungen:

- **Bestellbedarf:** Ein Titel gilt als Bedarf, wenn sein Gesamtbestand unter der Schwelle aus den Systemeinstellungen liegt (`bestellbedarf_schwelle`, Vorgabe 3). Die Spalte `meldebestand` am Titel wird nur noch **informativ** mitgeliefert und löst nichts mehr aus — ihr pauschaler Default 5 meldete früher fast jeden Titel. Die Warnung selbst lässt sich abschalten (`bestellbedarf_warnung_aktiv`).
- **Fokus auf Lernmittelfreiheit (LMF):** Die Bedarfsvorschläge sind standardmäßig auf LMF-Medien (Schulbücher) gefiltert. Freihand-Exemplare (Lese-Einzelstücke) werden in der Regel nicht nachbestellt.
- **Exemplare entstehen beim BESTELLEN, nicht beim Eintreffen:** Mit dem Absenden legt das System die Exemplare samt Barcode-Nummern an und markiert sie als „Im Zulauf". Nur so kann der Barcodebogen mit der Bestellung mitgehen. Der Wareneingang bucht diese Exemplare später frei — er erzeugt sie nicht.
- **Lieferanten-Eigenschaften** (Lieferantenverwaltung, je Händler schaltbar):
  - *Händler beklebt die Bücher:* Der Barcodebogen geht mit; die Exemplare gelten sofort als etikettiert und erscheinen nicht auf der Nachdruck-Liste.
  - *Voreingestellt beim Bestellen:* Vorauswahl im Bestellformular, höchstens ein Händler (DB-seitig erzwungen).
  - *Lieferant bestätigt Bestellung selbst:* siehe nächster Punkt.
- **Bestätigungs-Link an den Lieferanten:** Händler, die selbst etikettieren (z. B. Naacher), bekommen mit der Bestellmail einen von Bibliosys erzeugten Link. Dahinter liegt eine Seite ohne Login, auf der der Lieferant die Etiketten dieser Bestellung in klein oder groß druckt (identisch zum Mailanhang) und die Bestellung **einmalig** bestätigt.
  - *Klein:* Bogen für vorgestanztes Etikettenmaterial. Der Lieferant **wählt das Bogenraster selbst** (Zweckform L4760 3×7, Avery 3475 3×8, Kleine Barcodes 4×13) — er druckt auf sein eigenes Material, und davon gibt es verschiedene. Bis zum 06.08.2026 kam der Bogen immer im Zweckform-Raster; wer andere Bögen im Drucker hatte, bekam einen Ausdruck, der danebenliegt. Das gewählte Raster steht anschließend in der Bestellhistorie der Schule.
  - *Groß:* Lernmittel-Etikett mit Ausleihtabelle, **vier Stück auf einem A4-Blatt** (2×2, mit Schnittlinien) — wird ausgeschnitten, nicht auf vorgestanztes Material gedruckt. Vorher war jedes Etikett eine eigene A6-Seite: Auf einem A4-Drucker kam damit ein Etikett je Blatt heraus, außer man stellte im Druckdialog von Hand „4 Seiten pro Blatt" ein (telefonische Rückmeldung Naacher, 06.08.2026). Die Bestätigung erscheint automatisch in der Bestellhistorie und ist dort von einem manuellen Nachtrag aus der Bibliothek unterscheidbar. Voraussetzung ist die **Öffentliche Adresse** in den Einstellungen — fehlt sie, geht die Bestellung ohne Link raus. Sicherheitszuschnitt: 256-Bit-Token, in der Datenbank nur als Hash, 180 Tage gültig, jederzeit durch einen neuen ersetzbar (der alte stirbt dabei). Details in [SECURITY.md](SECURITY.md).
- **Historie zeigt die neuesten 200 Bestellungen** (Grenze serverseitig, max. 500). Die Kennzahlen im Kopf — Gesamtausgaben, Exemplare, wartende Bestätigungen — zählen dagegen **alle** Bestellungen (`/api/bestellhistorie/uebersicht`), damit aus einer Teilsumme keine vermeintliche Gesamtsumme wird. Ohne die Grenze lieferte die Historie auf einer gewachsenen Datenbank 2,45 MB in 3,9 Sekunden, mit ihr 0,10 MB in 0,17 Sekunden.
- **Preise sind optional:** Ist „Preise erfassen" aus, arbeitet das ganze Bestellwesen ohne Geldbeträge — kein Preisfeld, keine Betragsspalten, Berichte zählen Exemplare statt Euro zu summieren.

---

## 8. LUSD-Synchronisation & Datenschutz (DSGVO)

### 8.1. Der LUSD-Import
- Die Landesschuldatenbank (LUSD) ist das führende System für Schülerdaten.
- Der Import überschreibt Namen, Klassen und LUSD-IDs im Bibliothekssystem.
- **Match-Logik:** Identifikation erfolgt primär über die LUSD-ID. Fallback (falls ID fehlt) ist eine Kombination aus Vorname, Nachname und Geburtsdatum.
- **Neue Kontaktdaten:** Es werden Anschriften und Eltern-E-Mails importiert, jedoch *ausschließlich* zum Zweck der Rechnungs- und Mahnungsstellung.

### 8.2. DSGVO und Lösch-Routinen (Abgänger)
- Wenn ein Schüler in der LUSD nicht mehr auftaucht, wird er im System zum "Abgänger" (`ist_abgaenger = true`).
- Das System anonymisiert Abgänger nach einer gesetzlichen Karenzzeit (Cronjob).
- **Retention-Blockade:** Ein Abgänger wird **nicht** gelöscht oder anonymisiert, solange er noch Bücher ausgeliehen hat oder unbezahlte Schadensfälle existieren. In diesem Fall wird der Datensatz eingefroren (`ist_gesperrt = true`, Sperrgrund: "Automatisierte Abgänger-Sperre"). Falls die offenen Vorgänge geklärt werden und der Abgänger im Folgejahr in der LUSD wieder als aktiver Schüler auftaucht, hebt das System die Sperre automatisch wieder auf.
- **Papierkorb:** Manuelles Löschen von Schülern durch den Admin verschiebt diese in einen Papierkorb (Soft-Delete). Ausleihhistorie und Name bleiben vorerst für einen etwaigen Restore erhalten. Erst der `Purge`-Prozess löscht sie endgültig und anonymisiert historische Ausleihen (`schueler_id = NULL`).

---

## 9. Druck-Center und Ausweise

Das System bietet einen zentralen Druck-Manager für physische Objekte:

- **Barcode-Etiketten:** Das System generiert PDF-Bögen mit Code-128 oder QR-Codes für neu eingetroffene Bücher. Diese können direkt auf vorgefertigte Etikettenbögen (z. B. Avery Zweckform) gedruckt werden.
- **Schülerausweise:** Mit den (verschlüsselten) LUSD-Fotos generiert das System druckfertige Schülerausweise mit persönlichem Barcode für die Ausleihe am Kiosk.

---

## 10. System-Audit & Protokollierung

Um Nachvollziehbarkeit bei sensiblen Schuldaten zu garantieren, gibt es ein strenges, unveränderliches Audit-Log:

- Jede administrative Aktion (Benutzer gelöscht, Schadensfall storniert, Schüler manuell gesperrt) wird in der Tabelle `audit_logs` mit `Akteur`, `Zeitstempel`, `IP` und Vorher-/Nachher-Details protokolliert.
- Das Audit-Log ist **append-only** (nur anhängend). Weder Admins noch Lehrer können Einträge über die UI verändern oder löschen.
- Die Daten dienen der Fehlerbehebung und DSGVO-Rechenschaftspflicht.
- **Aufbewahrungsfrist** (seit 16.08.2026): Auch Protokolle brauchen ein „wie lange“ —
  IP-Adressen und Bearbeiter-Bezüge sind personenbezogen (Speicherbegrenzung, Art. 5).
  Ein nächtlicher Job (03:00, nach dem Backup) löscht Einträge beider Tabellen jenseits
  der Frist aus `audit_aufbewahrung_monate` (Vorgabe 24 Monate, Untergrenze 6 gegen
  Fehlkonfiguration) und hinterlässt die Löschung selbst als eine Meta-Zeile mit den
  Zahlen — sonst sähe eine spätere Prüfung nur ein Protokoll mit unerklärlicher
  Vorderkante.

---

## 11. Statistiken & Dashboards

Für die Schulleitung und Bibliotheks-Administration aggregiert das System Echtzeit-Metriken:

- Auswertung von Ausleihen pro Jahrgang/Klasse.
- Hitlisten der beliebtesten Medien (LMF vs. Freihand).
- Warn-Dashboards für offene Schäden und eskalierte Mahnungen.
- Export-Funktionen (CSV/PDF) für die Jahresberichte an die Schulleitung.

---

## 12. Authentifizierung & Rollenmodell (RBAC)

Der Zugang zum System ist strikt reglementiert und wird durch ein Role-Based Access Control (RBAC) System gesteuert.

### 12.1. Login & Sicherheit
- **Verfahren:** E-Mail und Passwort **gegen den Schul-Mailserver (IMAP)**, nicht gegen die eigene Datenbank. Eine lokale Passwortspalte gibt es seit Migration 012 nicht, und es wird nirgends ein Passwort gehasht oder gespeichert — wer ein Konto anlegt, legt kein Passwort fest, sondern setzt eine E-Mail-Adresse, die auf dem Schulserver existiert (`auth/handlers.go`, `verifyIMAPCredentials`). Hier stand bis zum 11.08.2026 „E-Mail und Passwort (Bcrypt-gehasht)" — das war nie so, und zwei Absätze weiter unten stand bereits das Gegenteil.
- **Folge für die Kontoverwaltung:** Die E-Mail **ist** die Identität. Wer die Spalte `benutzer.email` schreiben darf, übernimmt damit ein Konto; ein Rechte-Audit, das nur auf `rolle` schaut, sieht diesen Weg nicht.
- **Session-Management:** Stateless via JWT (JSON Web Tokens) in HttpOnly-Cookies.
- **Brute-Force-Schutz:** Strenges Rate-Limiting beim Login (Sperre nach mehreren Fehlversuchen pro IP/E-Mail-Kombination).

### 12.2. Das 4-Rollen-Konzept
Das System kennt vier fest verdrahtete Rollen, deren genaue Rechte (z.B. `view_students`, `manage_users`, `checkout_books`) vom Admin in der Datenbank konfiguriert werden können:

1. **Admin (`admin`):** Uneingeschränkter Zugriff auf alle Systembereiche, Einstellungen, Audits und Datenschutz-Routinen.
2. **Mitarbeiter (`mitarbeiter`):** Das Personal für das Tagesgeschäft. Hat Zugriff auf die Scanner-Omnibox, Buchkatalog, Mahnwesen und Schülerverwaltung, darf aber keine Systemeinstellungen ändern.
3. **Kollegium (`kollegium`):** Zugang zum Kollegiums-Portal, und zwar für **genau eine** Funktion: einen Klassensatz reservieren. Erteilt ist ein einziges Recht, `create_reservations` (Migration 070); die Suche darin läuft über den öffentlichen OPAC, das Absenden über `create_reservations`. Beides fasst keine Personendaten an.

   **Die Rolle hieß bis zum 10.08.2026 `lehrer`** (Migration 069). Das Wort war doppelt belegt — als Anmelde-Rolle *und* als Entleihertyp `schueler.klasse = 'lehrer'` (Handapparat, eigene Behandlung im Mahnwesen). `kollegium` benennt jetzt die Personengruppe mit Zugang, `lehrer` bleibt für den Entleiher frei. Die Umbenennung selbst war keine Rechteänderung.

   **Der Rechteumfang war es** (Migration 070): Auf dem Schulserver sah ein Kollegiums-Konto am 10.08.2026 zehn von fünfzehn Menüpunkten, darunter Schülerdatei, Mahnwesen, System-Logs und Einstellungen — `role_permissions` führte `manage_users`, `audit_logs`, `view_stats`, `view_students`, `view_books` und `perform_actions` auf `true`. Das war keine reine Anzeigefrage: Dieselbe Tabelle entscheidet in `RequirePermission`, die API hätte es ebenfalls zugelassen. Alles außer `create_reservations` ist entzogen. Wer einer Lehrkraft gezielt mehr geben will, tut das im PermissionManager — Migrationen laufen nur einmal, eine spätere Vergabe wird nicht zurückgedreht.
4. **Helfer (`helfer`):** Stark limitierte Rolle für studentische Hilfskräfte oder Eltern. Kiosk-Ansicht (Omnibox) für Ausleihe und Rückgabe, dazu **lesender Katalogzugriff** (Entscheidung vom 30.07.2026, Migration 055): Ein Helfer an der Theke ist die erste Anlaufstelle für „Habt ihr Band 3 noch da?" und musste die Frage sonst weiterreichen. Die Grenze zu Personendaten zieht weiterhin `view_students`.

   **Ein Helfer braucht ein Postfach auf dem Schul-Mailserver.** Das ist die Frage, die in der Praxis zuerst kommt, und sie hatte bis zum 08.08.2026 keine Antwort in dieser Doku. Die Anmeldung läuft ausschließlich über E-Mail + Passwort gegen IMAP (`auth/handlers.go`); eine lokale Passwortspalte gibt es seit Migration 012 nicht, und einen Code- oder Barcode-Anmeldeweg gibt es nicht — die Felder `barcode_id`/`pin` standen einmal im `LoginRequest`, wurden nie ausgewertet und sind entfernt. Wer eine Hilfskraft aufnehmen will, lässt also zuerst ein Postfach anlegen und trägt dann unter Einstellungen → Benutzerverwaltung die Person mit der Rolle „Helfer" ein. Die E-Mail ist dabei die Identität: Wer die Spalte `benutzer.email` schreibt, übernimmt das Konto.

   Erteilt sind genau zwei Rechte (`db/seed.go`): `perform_actions` (Scannen, Ausleihe, Rückgabe) und `view_books` (Katalog). Erreichbar sind damit Ausleihe, Medienkatalog, Signaturen und Schulklassen — Letztere seit dem 08.08.2026, weil der Klassensatz-Reiter im Katalog aufgelöst wurde und der Blick darauf sonst verloren gegangen wäre. Die Pflege-Aktionen auf diesen Seiten hängen an `edit_books` und bleiben dem Helfer verborgen.

   **Welche Seiten eine Rolle erreicht, entscheidet `canSeeItem()` in `frontend/src/lib/menu.js` — und nur diese Funktion.** Der Router fragt dieselbe. Bis zum 08.08.2026 führte er eine zweite, handgepflegte Liste; als das Recht für „Schulklassen" wechselte, liefen beide auseinander, und der Helfer bekam einen Menüpunkt, der ihn beim Klick wortlos an die Theke zurückwarf. Wer eine Rolle oder ein Recht ändert, fasst deshalb `menu.js` an und sonst nichts. Abgesichert durch `e2e/menue-fuehrt-irgendwohin.spec.js`, das für Helfer, Mitarbeiter und Lehrkraft jeden sichtbaren Menüpunkt anklickt.

---

## 13. Katalogisierung & Medienverwaltung

Das System bietet umfassende Werkzeuge zur Pflege des Buchkatalogs:

- **Systematiken & Signaturen:** Bücher können hierarchisch nach Systematiken (Kategorien/Themen) und spezifischen Signaturen (Regal-/Standort-Kennung) klassifiziert werden.
- **Automatische Cover-Synchronisation:** Ein Hintergrund-Worker (`Cover-Sync`) sucht über ISBNs automatisch in externen Buch-APIs (z.B. Google Books) nach Buchcovern, lädt diese herunter und speichert sie datensparsam im WebP-Format.
- **Legacy-Import-Engine:** Für die initiale Einrichtung oder Datenübernahme bietet das System eine dynamische Import-Schnittstelle (`/api/import/littera`), um Altbestände aus Legacy-Programmen (wie z. B. *Littera*) per CSV einzulesen und zu mappen.

---

## 14. Schadensmanagement

Nicht nur bei Hardware, sondern auch bei Büchern greift ein dediziertes Schadensmanagement:

- Wenn ein Buch als "Verlust" oder "Beschädigt" ausgebucht wird (z.B. bei der Inventur oder manuell am Kiosk), kann das System automatisch eine Kostenforderung (Schadensfall) gegen den verursachenden Schüler anlegen.
- Offene Schäden blockieren die DSGVO-Löschung eines Schülers und können per PDF-Rechnung ausgedruckt werden.
- **Erledigt wird eine Gebühr in der Schülerakte** (seit 16.08.2026) auf genau zwei Wegen:
  **„Bezahlt“** (Barzahlung am Tresen) oder **„Stornieren“** mit Pflicht-Begründung
  (Erlass, Buch wiedergefunden, Kulanz). Beides verlangt das Recht `edit_students` —
  bewusst nicht die Kiosk-/Helfer-Rolle. Der Betrag im revisionssicheren Audit-Eintrag
  stammt immer aus der Datenbank, nie aus der Anfrage; eine bereits erledigte Gebühr
  meldet dem zweiten Bearbeiter einen Konflikt (409) statt einer Doppelbuchung.
  Stornierte wie bezahlte Gebühren geben Schülerlöschung, LUSD-Abgleich und Ausleihe
  wieder frei; die DSGVO-Auskunft weist Stornierungszeitpunkt und -grund transparent aus.

---

## 15. Selbstprüfung der Betriebsbereitschaft

Erreichbar unter **System → Einstellungen → Tab „Betriebsbereitschaft“** (Recht `manage_users`,
`GET /api/admin/system/betriebsbereitschaft`). Die Seite beantwortet **eine** Frage:
*Was ist eingerichtet, aber nicht in Betrieb?*

Der Anlass ist eine wiederkehrende Fehlerart, und sie ist immer dieselbe: Eine Funktion
ist fertig programmiert, getestet und verdrahtet — und tut nichts, weil eine Einstellung
fehlt. Kein Fehler, kein Statuscode, kein Logeintrag, der jemandem auffiele. Dreimal
gefunden, jedes Mal von Hand: die Auslagerung der Backups (vier leere Variablen), der
nächtliche Backup-Job (Schlüssel stand in der `.env`, kam aber nicht im Container an)
und der Bestell-Bestätigungslink (`oeffentliche_adresse` nie gesetzt — die Mails gingen
raus, nur ohne den Link, um dessentwillen es sie gibt).

**Geprüft werden neun Bereiche:** Auslagerung der Backups, Geheimnisse, Anmeldung
(IMAP), Bestell-Bestätigungslink, Mailversand (Mahnwesen), Demo-Daten, Admin-Konten
(wer hat Vollzugriff und erhält die Kritisch-Alarme — der Alarm-Mail-Vorfall vom
16.08.2026 zeigte vier aktive Admins, drei davon dem Betreiber unbekannt; Konten-Anlage
und -Änderung werden seither auditiert), Rechte-Vorgabe
(Abgleich der Live-Tabelle `role_permissions` gegen die Code-Vorgabe — der Seed fasst
bestehende Zeilen nie an, eine geänderte Vorgabe erreicht Alt-Anlagen sonst nie; eine
Abweichung ist bewusst nur eine Warnung, denn sie kann eine Admin-Entscheidung sein)
und die Klassen-Zuordnung (seit 18.08.2026, Befund F3: Klassennamen verbinden Schüler,
Klassenlehrer-Zuordnung und Bücherlisten nur als übereinstimmender Text — die Prüfung
benennt Klassen ohne Lehrkraft, verwaiste Zuordnungen und Bücherlisten ohne Klasse).

Jeder Befund trägt vier Angaben, weil drei nicht reichen: **Befund** („was ist"),
**Folge** („warum das zählt") und **Abhilfe** („was zu tun ist") — ohne die letzte landet
die Meldung auf einem Zettel statt in der `.env`. Dazu eine **Stufe**: `ok` (in Betrieb),
`warnung` (läuft, aber nicht wie gedacht), `kritisch` (vor dem Echtbetrieb zwingend zu
klären). Bewusst nur drei — eine feinere Skala liest niemand.

**Der Wächter meldet sich selbst** (seit 16.08.2026): Kritische Befunde gehen täglich
per Mail — solange sie bestehen, mit Befund, Folge und Abhilfe je Punkt. Der
Empfängerkreis ist seit 17.08.2026 einstellbar (Einstellungen → Allgemein,
`alarm_empfaenger`, mehrere Adressen mit Komma); ist er leer, gehen die Mails als
sicherer Rückfall an alle aktiven Admin-Konten — ein Alarm, der niemanden erreicht,
ist keiner. Die Mail nennt im Fußtext, an wen sie ging und in welchem Modus. Warnungen lösen bewusst keine Mail aus (Dauerrauschen stumpft ab);
ihre Zahl steht als Fußnote. Auf Spielwiesen (`APP_ENV=local/development/test`)
schweigt der Alarm — die lokale `.env` zeigt auf den echten Schul-SMTP.

Die Seite ist eine **reine Prüffunktion**: Sie ändert nichts, sie schaltet nichts frei.
Die Urteile sind als reine Funktion über eine eingesammelte Lage gebaut
(`api/betriebsbereitschaft.go`) und damit vollständig testbar, ohne
Umgebungsvariablen zu verbiegen oder eine Datenbank zu brauchen; das Zusammentragen der
Lage steht daneben im Handler.

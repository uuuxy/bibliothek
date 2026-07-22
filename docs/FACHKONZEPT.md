# Fachkonzept & Feature-Spezifikation

Beschreibt die funktionale Business-Logik der Bibliothekssoftware auf Basis der Go-Implementierung. Zielgruppe: Administratoren, Betreiber und Entwickler.

---

## 1. Zentrale Scanner-Omnibox (Kiosk)

Die Software nutzt ein eingabefokussiertes Kiosk-Design für die Ausleihe und Rückgabe. Ein einziges Eingabefeld (Omnibox) verarbeitet alle Scans anhand spezifischer Präfixe:

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
- **Lehrer (Handapparat):** Erhalten pauschal eine Frist von einem Jahr (365 Tage).
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
- Lehrkräfte können `N` Exemplare eines Titels für einen Zeitraum blockieren (Klassensatz).
- **Erfüllung:** Solange die Reservierung läuft, werden zurückgegebene Bücher dieses Titels im System zurückgehalten, bis die benötigte Anzahl für den Klassensatz erreicht ist.

---

## 5. Geräteausleihe (Hardware)

Die Ausleihe von teurer Hardware (Laptops, Tablets, Beamer) folgt strikteren Regeln als Bücher:

- **Checklisten-Zwang:** Bei jeder Ausleihe und Rückgabe eines Geräts muss das Personal eine Checkliste bestätigen (z. B. "Ist das Ladekabel vorhanden?", "Ist der Stift da?", "Bildschirm intakt?").
- **Schadens-Zuweisung:** Fehlt ein Zubehörteil bei der Rückgabe, kann direkt in der Rückgabe-Transaktion ein kostenpflichtiger Schadensfall für den Schüler generiert werden.
- **Zustands-Sperre:** Geräte mit Status "Defekt" können vom System technisch nicht ausgeliehen werden.

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

- **Meldebestand:** Titel haben einen Meldebestand. Sinkt der Bestand darunter, schlägt das System den Titel zur Nachbestellung vor.
- **Fokus auf Lernmittelfreiheit (LMF):** Die Bedarfsvorschläge sind standardmäßig auf LMF-Medien (Schulbücher) gefiltert. Freihand-Exemplare (Lese-Einzelstücke) werden in der Regel nicht nachbestellt.
- **Zulauf:** Erstellte Bestellungen (mit Lieferant, Preis, Menge) tauchen im Wareneingang auf. Beim Eintreffen der Pakete generiert das System aus der Bestellposition direkt die passenden Buch-Exemplare inklusive Barcode-Nummern.

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
- **Verfahren:** E-Mail und Passwort (Bcrypt-gehasht).
- **Session-Management:** Stateless via JWT (JSON Web Tokens) in HttpOnly-Cookies.
- **Brute-Force-Schutz:** Strenges Rate-Limiting beim Login (Sperre nach mehreren Fehlversuchen pro IP/E-Mail-Kombination).

### 12.2. Das 4-Rollen-Konzept
Das System kennt vier fest verdrahtete Rollen, deren genaue Rechte (z.B. `view_students`, `manage_users`, `checkout_books`) vom Admin in der Datenbank konfiguriert werden können:

1. **Admin (`admin`):** Uneingeschränkter Zugriff auf alle Systembereiche, Einstellungen, Audits und Datenschutz-Routinen.
2. **Mitarbeiter (`mitarbeiter`):** Das Personal für das Tagesgeschäft. Hat Zugriff auf die Scanner-Omnibox, Buchkatalog, Mahnwesen und Schülerverwaltung, darf aber keine Systemeinstellungen ändern.
3. **Lehrer (`lehrer`):** Eingeschränkter Zugriff. Kann den eigenen Handapparat verwalten, Klassensätze reservieren und den Katalog durchsuchen. Hat keinen Zugriff auf sensible Schüler- oder Mahndaten.
4. **Helfer (`helfer`):** Stark limitierte Rolle für studentische Hilfskräfte oder Eltern. Kann ausschließlich in die Kiosk-Ansicht (Omnibox) gelangen, um einfache Rückgaben oder Scans durchzuführen.

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
- Ist ein Schaden bezahlt, wird die Rechnung als beglichen markiert und der Schüler ist wieder "frei".

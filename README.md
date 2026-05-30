
# 📚 Schulbibliothek-Verwaltungssystem

Eine moderne, datenschutzkonforme und hochperformante Webanwendung zur Verwaltung von Schulbibliotheken. Entwickelt für den anspruchsvollen Schulalltag mit Fokus auf einfache Bedienung, Ausfallsicherheit und DSGVO-Konformität.

---

## 🛠️ Tech-Stack

Das System besteht aus einer schlanken und robusten Architektur:

* **Backend**: **Go (Golang)**
  * Standardbibliothek und `pgx/v5` für hochperformante PostgreSQL-Verbindungen.
  * `gofpdf` für serverbasierte PDF-Generierung.
  * Event-basiertes Server-Sent Events (SSE) System für Echtzeit-Synchronisation.
* **Frontend**: **Svelte 5** + **Vite** + **TailwindCSS**
  * Modernes Svelte 5 (Runes) für reaktive und schlanke Benutzeroberflächen.
  * `html5-qrcode` für den Kamera-basierten Barcode-Scan direkt im Browser.
  * Web Audio API für synthesierte Audio-Rückmeldungen.
* **Datenbank**: **PostgreSQL**
  * Transaktionssicherheit (ACID) mit `Read Committed` Isolation.
  * `pg_trgm` Extension und GIN-Indizes für schnelle Volltextsuchen.
  * JSONB-Spalten zur flexiblen Eigenschaftserweiterung ohne Schemaänderungen.

---

## 🚀 Kernfunktionen für den Schulalltag

### 1. LUSD-Upsert (Schüler-Import)
Der Import von Schülern erfolgt direkt aus der offiziellen hessischen Lehrer- und Schülerdatenbank (LUSD) über CSV-Dateien (unterstützt Komma und Semikolon als Trennzeichen).
* **Idempotenz**: Das Backend führt einen ACID-Transaktions-Upsert durch (`ON CONFLICT (barcode_id) DO UPDATE`).
* **Datenaktualität**: Bei Namensgleichheit oder Klassenwechsel werden die Klassenbezeichnungen und Abgangsjahre automatisch aktualisiert, ohne bestehende Ausleihen oder offene Gebühren zu beeinträchtigen.

### 2. Kiosk-Massenbetrieb (Scannen am Tresen)
Der Kiosk-Ausleihtresen ist für schnellen Durchsatz (z. B. während kurzer Pausen) optimiert:
* **Audio-Feedback**: Integrierte Web Audio API erzeugt synthetische Erfolgs- ("Pling") und Fehler-Töne ("Buzz"), damit das Bibliotheksteam das Feedback hört, ohne auf den Bildschirm schauen zu müssen.
* **Visuelle Signale**: Farbiges Aufblinken des Bildschirmrands (Grün = Erfolg, Rot = Fehler) und Schütteleffekt der Omnibox bei Fehlern.
* **Offline-Ausfallsicherheit**: Bei Netzwerkausfällen (`navigator.onLine == false`) werden Scans lokal in einer Warteschlange (`localStorage`) zwischengespeichert und bei Wiederherstellung der Verbindung automatisch nachgesendet.
* **Kamera-Scanner**: Direktes Scannen von Barcodes über die integrierte Webcam des Laptops/Tablets.

### 3. Automatisiertes PDF-Mahnwesen
Überfällige Ausleihen werden klassenweise ermittelt und können automatisiert angemahnt werden:
* **PDF-Generierung**: Erstellt im Speicher hochauflösende, kompakte PDF-Mahnlisten mit Buch-Covern.
* **SMTP-Versand**: Versendet die Mahnliste per Knopfdruck als E-Mail-Anhang direkt an die zuständigen Klassenlehrer, damit diese die Zettel in der Klasse verteilen können.

### 4. Antolin-Schnittstelle
* **Live-Punkteabfrage**: Das System fragt bei Büchern mit gültiger ISBN automatisch über einen integrierten Proxy die Antolin-Klassenstufen und Punkte ab und stellt sie übersichtlich im OPAC sowie in den Buchdetails dar.
* **Leistungsfähiger Cache**: Ein 24-Stunden-Servercache minimiert externe API-Anfragen und beschleunigt Ladezeiten.

### 5. Digital Signage (Info-Monitor)
* **Dynamisches Dashboard**: Das integrierte Slideshow-Karussell (Route `/monitor`) bereitet beliebte Bücher, Neuerscheinungen und das "Buch des Monats" für Displays im Schulflur oder im Bibliotheksbereich grafisch ansprechend auf.

### 6. Ferien-Leseclub
* **Saisonale Ausleihregeln**: Über die Systemeinstellungen lässt sich ein Leseclub mit festem Zieldatum aktivieren. Alle in diesem Zeitraum getätigten Ausleihen werden automatisch auf den letzten Ferientag terminiert.

---

## 🛡️ DSGVO-Konzept (Datenschutz & Anonymisierung)

Die Anwendung ist standardmäßig nach den Prinzipien *Privacy by Design* und *Datensparsamkeit* konzipiert:

1. **Automatisierte 14-Tage-Anonymisierung (Cron Job)**:
   Ein täglicher Cron-Job (jeden Tag um 00:00 Uhr) setzt bei allen zurückgegebenen Büchern, deren Rückgabe länger als 14 Tage zurückliegt, die IDs des ausleihenden Personals (`bearbeiter_id` und `rueckgabe_bearbeiter_id`) auf `NULL`. Damit ist im Nachhinein nicht mehr nachvollziehbar, welcher Mitarbeiter welches Buch entgegengenommen hat.
2. **Abgänger-Löschung nach Schuljahresende**:
   Ehemalige Schüler (`ist_abgaenger = true`) werden nach Ablauf einer Karenzzeit (mindestens 30 Tage nach Beginn des Folgejahres) automatisch vollständig gelöscht, sofern sie keine offenen Ausleihen oder unbezahlten Schadensfälle mehr haben.
3. **Schüler-Löschung & Historien-Anonymisierung**:
   Wird ein Schüler manuell gelöscht, werden alle personenbezogenen Daten rückstandslos aus der Datenbank entfernt. Die historischen Ausleihdatensätze bleiben für anonyme Statistiken (z. B. Ausleihhäufigkeit eines Buchtitels) erhalten, die `schueler_id` wird jedoch auf `NULL` gesetzt.
4. **Audit-Log**:
   Sicherheitsrelevante Aktionen (z. B. Rechteänderungen, Schülerlöschung) werden revisionssicher in einem Append-Only-Audit-Log protokolliert.

---

## 💻 Setup & Installation

### 1. Umgebungsvariablen (.env)
Erstelle eine `.env`-Datei im Hauptverzeichnis mit folgenden Konfigurationen:

```env
# Server & Datenbank
PORT=8080
DATABASE_URL=postgres://postgres:password@localhost:5432/bibliothek?sslmode=disable
COOKIE_SECURE=false # Auf true setzen im HTTPS-Produktivbetrieb

# Authentifizierung
JWT_SECRET=dein_super_geheimes_jwt_token_signier_key
JWT_ISSUER=bibliothek-auth
JWT_AUDIENCE=bibliothek-app
COOKIE_DOMAIN=localhost

# Backups
BACKUP_DIR=./backups
BACKUP_ENCRYPTION_KEY=dein_backup_passwort_32_zeichen_!!
BACKUP_EMAIL_TO=admin@schule.de

# Inventur & administrative Sonderpasswörter
ADMIN_PASSWORD=admin123
GUEST_PASSWORD=gast123

# SMTP-Konfiguration für Mahnwesen
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=bibliothek@schule.de
SMTP_PASSWORD=dein_smtp_passwort
SMTP_FROM=bibliothek@schule.de
SMTP_TO=sekretariat@schule.de
SUPPLIER_EMAIL=bestellungen@buchhaendler.de
ALLOWED_ORIGIN=http://localhost:5173
```

### 2. Backend starten
```bash
# Abhängigkeiten installieren
go mod download

# Backend starten
export $(cat .env | xargs)
go run main.go
```

### 3. Frontend starten
```bash
cd frontend

# Abhängigkeiten installieren
npm install

# Entwicklungs-Server starten
npm run dev

# Produktions-Build erstellen
npm run build
```

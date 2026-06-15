# Kommandozeilen-Skripte und Migrationen

Diese Dokumentation listet die in der Applikation verfügbaren Kommandozeilen-Werkzeuge auf. Diese befinden sich primär in den Verzeichnissen `cmd/` und `scripts/`.

## 1. LITTERA-Import (`cmd/littera-import`)

Das Tool migriert Altbestände aus LITTERA-Exporten in die neue Datenbankstruktur.
- **Funktionsweise:** Es verarbeitet CSV-Dumps der LITTERA-Software, parst Titelinformationen sowie die Barcodes physischer Exemplare.
- **Verwendung:** Die Ausführung erfolgt über das dedizierte Go-Binary, wobei die CSV-Dateien als Argument oder über Standard-Eingabe (stdin) bereitgestellt werden.
- **Architektur:** Der Import ist transaktional. Buchtitel (`buecher_titel`) und ihre assoziierten Exemplare (`buecher_exemplare`) werden kohärent angelegt.

## 2. Foto-Migration (`cmd/migrate-fotos`)

Dieses Hilfsprogramm migriert bestehende unverschlüsselte Bilddateien aus dem Dateisystem in die relationale PostgreSQL-Datenbank.
- **Funktionsweise:** Es iteriert über ein Zielverzeichnis mit Schülerfotos, validiert diese und wandelt sie in einen Bytestrom (`BYTEA`) um.
- **Speicherung:** Die Bilder werden anschließend direkt in die Tabelle `schueler_fotos` injiziert. Dies dient der Konsolidierung der Infrastruktur und der Steigerung der Datensicherheit.

## 3. Datenbank-Backup (`scripts/backup.sh`)

Ein Shell-Skript zur Erzeugung periodischer Datenbank-Backups.
- **Funktionsweise:** Es nutzt das Programm `pg_dump`, um den Zustand der Datenbank zu exportieren. Das Skript wird idealerweise über einen System-Cronjob oder den internen Scheduler aufgerufen.

## 4. Concurrency & Load Testing (`cmd/stresstest`)

Ein isoliertes Go-Skript zur Durchführung von parallelen Lasttests und zur Simulation von Race-Conditions.
- **Funktionsweise:** Es feuert mithilfe von `sync.Cond` und Go-Routinen in derselben Millisekunde dutzende gleichzeitige Requests gegen den Ausleih-Endpunkt (`/api/action`).
- **Verwendung:** `go run cmd/stresstest/main.go -port 8084` (der Port kann an die lokale Ziel-Umgebung angepasst werden).
- **Zweck:** Sicherstellung der Transaktionssicherheit (ACID) der PostgreSQL-Datenbank und der Mutex-Locks im Backend bei zeitgleichem Scannen desselben Barcodes.

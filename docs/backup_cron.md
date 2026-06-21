# Datenbank Backup & Cronjob Einrichtung

Zur automatischen Sicherung der Datenbank existiert das Skript `scripts/db_backup.sh`. Dieses Skript erstellt einen Dump der PostgreSQL Datenbank und rotiert alte Backups nach 7 Tagen automatisch.

## 1. Voraussetzungen prüfen
- Sicherstellen, dass das Skript ausführbar ist:
  ```bash
  chmod +x /pfad/zur/Bibliothek/scripts/db_backup.sh
  ```
- Eine eventuelle `.env` Datei im Hauptverzeichnis mit Datenbank-Zugangsdaten (z. B. `POSTGRES_USER`) wird vom Skript automatisch geladen.

## 2. Cronjob unter Ubuntu/Linux einrichten
Um das Backup täglich automatisch ausführen zu lassen (z. B. nachts um 02:00 Uhr), richten wir einen Cronjob ein.

1. Öffne den Cron-Editor auf dem Server:
   ```bash
   crontab -e
   ```
2. Füge ganz am Ende der Datei folgende Zeile ein (Pfade müssen an das tatsächliche System angepasst werden):
   ```bash
   0 2 * * * cd /pfad/zur/Bibliothek && ./scripts/db_backup.sh >> /pfad/zur/Bibliothek/backups/backup.log 2>&1
   ```
3. Speichere und schließe die Datei. Der Cron-Daemon übernimmt die Aufgabe automatisch.

## 3. Wiederherstellung (Papierkorb & Backup)
- Einzelne versehentlich gelöschte Schüler können direkt über die Weboberfläche im **Papierkorb** (Administration > Schüler > Reiter "Papierkorb") wiederhergestellt werden (Soft-Delete).
- Bei einem schwerwiegenden Datenverlust kann ein komplettes Backup über `psql` eingespielt werden (siehe [resilience_and_recovery.md](./resilience_and_recovery.md)).

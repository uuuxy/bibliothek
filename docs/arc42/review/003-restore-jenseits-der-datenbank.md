# 003 — Der Restore endet an der `.env`, nicht an der Datenbank

Stand: 19.09.2026

**Befund.** Backup und Wiederherstellung der **Datenbank** sind vollständig gelöst und
sogar wöchentlich erprobt. Für das **System** gilt das nicht: Der Schlüssel, der die
Sicherung aufschließt, liegt nur auf dem Host, der im Ernstfall gerade weg ist.

**Beleg.** Am 19.09.2026 am Code und an der Compose-Datei nachgemessen:

| Frage | Stand |
| --- | --- |
| Ist die `.env` in einer Sicherung? | **Nein.** `jobs/backup.go` sichert ausschließlich den `pg_dump` |
| Was steht darin, das der Restore braucht? | `BACKUP_ENCRYPTION_KEY` (entschlüsselt die Sicherung — **Ringabhängigkeit**), `APP_ENCRYPTION_KEY` (`schueler_fotos.foto_encrypted`, gespeichertes SMTP-Passwort), `POSTGRES_PASSWORD`, `JWT_SECRET`, IMAP/SMTP, `TRUSTED_PROXIES`, `ALLOWED_ORIGIN` |
| Beschreibt die Anleitung einen Totalverlust? | Bis zum 19.09.2026 **nein** — `resilience_and_recovery.md` kannte nur den Datenbank-Restore (Abschnitte 2a–2e). Abschnitt 2f ist mit diesem Befund ergänzt |
| Sind die Cover wirklich reproduzierbar? | **Nicht alle.** `inventur/upload_handler.go`, `handleUploadCover` nimmt Bilddateien von Hand entgegen; die stammen aus keiner externen Quelle. `DEPLOYMENT.md` behauptete das Gegenteil und ist korrigiert |
| Wurde ein Restore je an fremder Hardware gefahren? | Nein (offen: `OFFEN.md` 7.4). Die Wochenprobe läuft auf demselben Server, in dieselbe Postgres-Instanz |

**Was es kostet.** Im Ernstfall genau das, wofür die Sicherung existiert. Zwei Stufen:

1. **Ohne `BACKUP_ENCRYPTION_KEY` ist die Sicherung eine Datei ohne Inhalt.** Kein Werkzeug
   öffnet sie, auch nicht das eigene.
2. **Mit dem Backup-, aber ohne den `APP_ENCRYPTION_KEY`** gelingt der Restore und wirkt
   erfolgreich — die Schülerfotos und das gespeicherte SMTP-Passwort bleiben trotzdem für
   immer Chiffrat. Das ist der gefährlichere Fall, weil er wie ein Erfolg aussieht.

Dasselbe gilt lautlos beim **Wechsel** des `BACKUP_ENCRYPTION_KEY`: Danach sind die alten
Sicherungen unlesbar, ohne dass irgendetwas rot wird (`OFFEN.md`, Abschnitt 5).

**Was daraus folgt.** Drei Dinge, zwei davon erledigt:

- ✅ Die Anleitung kennt den Totalverlust: `resilience_and_recovery.md` Abschnitt 2f, mit
  der Ringabhängigkeit als erstem Schritt.
- ✅ Die falschen Aussagen sind korrigiert (`DEPLOYMENT.md`, `07-verteilungssicht.md` 7.7).
- ⬜ **Offen und nicht dokumentierbar, sondern zu entscheiden:** wohin die Kopie der `.env`
  gehört und wie sie dort geschützt ist. Vorschlag als
  [Idee 006](../ideen/006-zweiter-ort-ohne-s3.md); ohne diese Entscheidung bleibt Abschnitt
  2f eine Anleitung mit einem Schritt, den niemand ausführen kann.

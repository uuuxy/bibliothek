# Das Blatt bei der Schule — Vorlage

Stand: 28.09.2026

Zum Ausdrucken und Ausfüllen von Hand. Das ausgefüllte Blatt liegt auf Papier bei der Schule,
nicht im Repository: Es trägt Zugänge und den Ort der Schlüssel. Wozu die Angaben dienen, steht
im [Pflegekonzept](PFLEGEKONZEPT.md), Abschnitt 7.3.

Ausgefüllt am: ________________ von: ______________________________

## Erreichbarkeit

| Rolle                   | Name                         | E-Mail                         | Telefon              |
| ----------------------- | ---------------------------- | ------------------------------ | -------------------- |
| Entwicklung             | ____________________________ | ______________________________ | ____________________ |
| Vertretung              | ____________________________ | ______________________________ | ____________________ |
| IT des Schulträgers     | ____________________________ | ______________________________ | ____________________ |
| Schulleitung            | ____________________________ | ______________________________ | ____________________ |
| Datenschutzbeauftragter | ____________________________ | ______________________________ | ____________________ |

**Einen Fehler melden:** per E-Mail an die Entwicklung, die Vertretung in Kopie; bei Stillstand
zusätzlich anrufen. Nie über GitHub ([Pflegekonzept](PFLEGEKONZEPT.md), Abschnitt 1).

**Bei einer Datenpanne:** sofort Bibliotheksleitung und Admin, dann der Ablauf im
[Datenschutz-Nachweis](datenschutz/nachweis.md), Abschnitt 8.

## Server

| Was                               | Eintrag                                              |
| --------------------------------- | ---------------------------------------------------- |
| Adresse des Servers               | ____________________________________________________ |
| Konto für die Anmeldung am Server | ____________________________________________________ |
| Programmverzeichnis               | ____________________________________________________ |

## Schlüssel

Die beiden Schlüssel aus der `.env` liegen als Kopie außerhalb des Servers. Ohne
`BACKUP_ENCRYPTION_KEY` lässt sich keine Sicherung öffnen, ohne `APP_ENCRYPTION_KEY` bleiben
Schülerfotos und das Mail-Passwort unlesbar ([Pflegekonzept](PFLEGEKONZEPT.md), Abschnitt 3.2).

| Schlüssel               | Ort der Kopie außerhalb des Servers                  |
| ----------------------- | ---------------------------------------------------- |
| `APP_ENCRYPTION_KEY`    | ____________________________________________________ |
| `BACKUP_ENCRYPTION_KEY` | ____________________________________________________ |

## Weiteres

| Was                                                                       | Eintrag                          |
| ------------------------------------------------------------------------- | -------------------------------- |
| Ort der Quelldokumente ([Pflegekonzept](PFLEGEKONZEPT.md), Abschnitt 7.1) | ________________________________ |
| Zugang zum Repository mit Schreibrecht, falls die Pflege übergeben wird   | ________________________________ |

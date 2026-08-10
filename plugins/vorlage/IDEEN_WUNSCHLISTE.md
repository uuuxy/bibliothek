# 💡 Entwickler-Wunschliste & Ideen (IDEEN_WUNSCHLISTE.md)

Dieses Dokument dient als Brainstorming-Plattform und Wunschliste für zukünftige optionale Erweiterungen und Plugins der Schulbibliothek, die über die Plugin-Schnittstellen (Frontend-Registry und Go-Events) realisiert werden können.

---

## 🔮 Zukünftige Plugin-Ideen (Noch nicht umgesetzt)

### 1. Slack / Discord / Matrix Webhook-Notifier
* **Beschreibung**: Sendet automatisierte Benachrichtigungen an Schul-Chat-Kanäle (z. B. bei Systemfehlern, kritischen Log-Meldungen oder erfolgreichen Backups).
* **Backend-Integration**: Kann über die Go-Hooks (siehe [hooks.go](file:///Users/peterflasch/Developer/Bibliothek/plugins/hooks.go)) an Events wie `OnBookReturned` oder neu zu erstellende Events gekoppelt werden.

### 2. Thermo-Bondrucker-Integration (Receipt Printer)
* **Beschreibung**: Generiert nach erfolgreichem Checkout/Return einen Beleg im 58mm- oder 80mm-Format für handelsübliche Belegdrucker. Zeigt dem Schüler eine Liste seiner aktuell entliehenen Medien und Fälligkeitstermine.
* **Frontend-Integration**: Platzierung als Button in der Kiosk-Sidebar oder im Schülerprofil-Tab.

### 3. RFID-Unterstützung
* **Beschreibung**: Unterstützung von RFID-Etiketten anstelle von 1D/2D-Barcode-Scannern. Ermöglicht das gleichzeitige Erfassen mehrerer Bücher auf einem Scan-Pad.
* **Backend/Frontend-Integration**: Anbindung über serielle/USB-Web-Schnittstellen (z. B. WebUSB/WebSerial API) zur direkten Einspeisung der IDs in das Omnibox-Protokoll.

### 4. Schüler-Selbstbedienungs-Kiosk (Self-Service)
* **Beschreibung**: Ein stark vereinfachter Kiosk-Modus mit PIN-Schutz oder Ausweis-Scan, an dem Schüler eigenständig Medien ausleihen und zurückgeben können, ohne dass Personal am Tresen anwesend sein muss.

### 5. Ausleih-Übersicht für Schüler im Schulportal (abschaltbar)
*Wunsch von Peter, 09.08.2026.*

* **Beschreibung**: Schüler (und ggf. Eltern) sehen über das Schulportal Hessen, welche Medien sie aktuell entliehen haben und wann die Frist abläuft. Rein lesend — Ausleihen und Verlängern bleiben im Haus.
* **Muss abschaltbar sein**: Ein Schalter in den Systemeinstellungen, mit dem die Schule die Funktion ganz aus- oder anschaltet. Vorgabe: **aus**. Solange sie aus ist, verlässt kein Datensatz das Haus.
* **Backend-Integration**: Eigener, schmaler Endpunkt, der NUR die Ausleihen des angemeldeten Schülers liefert — nicht der bestehende Profil-Endpunkt, der weit mehr Personendaten führt (Adresse, Eltern-Mail, Sperrgründe). Siehe das Datenminimierungs-Muster in `api/stats.go`: anonymisiert bzw. beschnitten wird serverseitig, nie erst im Frontend.
* **Offene Fragen, vor dem Bauen zu klären**:
  1. **Wie authentifiziert das Schulportal?** Ohne belastbares SSO (OAuth/OIDC oder signiertes Token) darf der Endpunkt gar nicht existieren — eine Ausleihliste je Schüler ist ein lohnendes Ziel für Neugierige.
  2. **Datenschutz**: Ausleihdaten sind Lesedaten. Es braucht eine Rechtsgrundlage bzw. die Aufnahme in das Verarbeitungsverzeichnis, bevor sie nach außen gehen — auch schulintern.
  3. **Elternzugriff ja/nein?** Fachliche Entscheidung, keine technische. Getrennt vom Schülerzugriff schaltbar halten.
  4. Push oder Pull — holt das Portal die Daten ab, oder schiebt die Bibliothek sie? Pull ist einfacher zu widerrufen.

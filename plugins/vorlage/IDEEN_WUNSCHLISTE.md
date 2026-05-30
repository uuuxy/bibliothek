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

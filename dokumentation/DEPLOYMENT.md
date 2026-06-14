# Deployment Guide (Hetzner Server)

Um die Bibliothek-Anwendung auf dem Hetzner-Server zu aktualisieren und unter der Domain `flasch3.herzog-dupont.de` bereitzustellen, gehst du wie folgt vor.

## 1. Aktualisieren und Neustarten (deploy.sh)
Das mitgelieferte Skript erledigt den Pull des Codes und den Neustart der Docker-Container sicher.

1. Verbinde dich per SSH mit deinem Hetzner-Server.
2. Wechsle in das Verzeichnis der Anwendung:
   ```bash
   cd /pfad/zur/bibliothek
   ```
3. Führe das Deployment-Skript aus:
   ```bash
   ./scripts/deploy.sh
   ```

Das Skript führt `git pull` aus und baut die Container für das Frontend (Port 3000) und Backend (Port 8083) mit `docker compose up -d --build` neu, ohne Ausfallzeiten für andere Container zu verursachen.

## 2. Caddy Reverse Proxy konfigurieren (Automatisiert)
Das Deployment-Skript prüft automatisch, ob die Datei `/root/Caddyfile` existiert.
Wenn sie gefunden wird, prüft es, ob die Domain `flasch3.herzog-dupont.de` bereits konfiguriert ist.
Falls nicht, wird folgender Block sicher **ans Ende** der Datei angehängt:

```caddyfile
flasch3.herzog-dupont.de {
    # Routen zur Go Backend-API
    handle /api/* {
        reverse_proxy localhost:8083
    }
    
    # Routen zum Frontend SvelteKit Server
    handle /* {
        reverse_proxy localhost:3000
    }
}
```

## 3. Caddy sanft neu laden (Zero Downtime)
Um die neue Konfiguration zu aktivieren, darfst du Caddy **nicht** mit `restart` oder `stop/start` neustarten, da dies aktive Verbindungen zu den anderen Webseiten kappen würde. 

Verwende stattdessen den "Soft Reload"-Befehl:

* **Wenn Caddy als Docker-Container läuft (empfohlen):**
  ```bash
  docker exec caddy_container_name caddy reload -c /root/Caddyfile
  ```
  *(Ersetze `caddy_container_name` durch den tatsächlichen Namen deines Caddy-Containers, z.B. `caddy`)*

* **Wenn Caddy als direkter Linux-Dienst läuft:**
  ```bash
  systemctl reload caddy
  ```

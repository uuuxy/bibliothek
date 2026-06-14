#!/usr/bin/env bash
set -e

# Wechsle in das Hauptverzeichnis des Projekts (ein Verzeichnis über dem script)
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$DIR/.."

echo "=== Hole neueste Änderungen aus dem Git-Repository ==="
git pull origin main

echo "=== Baue und starte Docker-Container neu ==="
docker compose up -d --build

CADDYFILE="/root/Caddyfile"
DOMAIN="flasch3.herzog-dupont.de"

if [ -f "$CADDYFILE" ]; then
    echo "=== Caddyfile im Root-Verzeichnis ($CADDYFILE) gefunden ==="
    if grep -q "$DOMAIN" "$CADDYFILE"; then
        echo "Domain $DOMAIN ist bereits im Caddyfile konfiguriert."
    else
        echo "Füge $DOMAIN zum Caddyfile hinzu..."
        cat << EOF >> "$CADDYFILE"

$DOMAIN {
    # Routen zur Go Backend-API
    handle /api/* {
        reverse_proxy bibliothek-backend:8083
    }
    
    # Routen zum Frontend SvelteKit Server
    handle /* {
        reverse_proxy bibliothek-frontend:3000
    }
}
EOF
        echo "Neuer Block hinzugefügt. Bitte Caddy sanft neu laden (Zero Downtime) mit:"
        echo "  - Nativ: systemctl reload caddy"
        echo "  - Docker: docker exec <caddy_container> caddy reload -c /root/Caddyfile"
    fi
else
    echo "=== ACHTUNG: Caddyfile nicht unter $CADDYFILE gefunden ==="
    echo "Bitte ergänze dein Caddyfile manuell um diesen Block:"
    cat << EOF
$DOMAIN {
    handle /api/* { reverse_proxy bibliothek-backend:8083 }
    handle /* { reverse_proxy bibliothek-frontend:3000 }
}
EOF
fi

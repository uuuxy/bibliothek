#!/bin/bash
# Schreibt die Caddy-Konfiguration des Testservers und lädt sie. Die neue Fassung wird im Container
# geprüft, bevor sie die laufende ersetzt: Eine ungültige Datei an deren Stelle ließe den nächsten
# Neustart von Caddy scheitern, und mit ihm alle drei Dienste.
set -eu
# Zum Ausprobieren mit einem Wegwerf-Caddy lassen sich Datei und Container umlenken.
ziel=${CADDY_DATEI:-/root/caddy/Caddyfile}
container=${CADDY_CONTAINER:-caddy}
neu=$(mktemp)
trap 'rm -f "$neu"' EXIT

cat << 'EOF' > "$neu"
{
    # Global options
    email admin@philipp-reis-schule.de

    # Admin-API nur auf localhost binden (nicht im Container-Netzwerk erreichbar)
    admin 127.0.0.1:2019

    # Caddy speichert LE-Zertifikate und ACME-Account-Keys in /data.
    # Dieses Verzeichnis wird auf das externe Volume schul-orga_caddy_data
    # gemountet → Zertifikate überleben Container-Neustarts und Rebuilds.
}

# =============================================================================
# Schul-Orga (Go-Backend + Svelte-Frontend, alles in einem Container)
# =============================================================================
orga.fla-dev.de {
    reverse_proxy school-calendar-app:8080 {
        # Timeouts für 150+ Nutzer mit SSE-Verbindungen
        transport http {
            response_header_timeout 300s
            read_timeout 600s
            write_timeout 600s
        }
    }

    # Security Headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
    }
}

# =============================================================================
# Bibliothek
# =============================================================================
# Keine tls-Zeile. Caddy holt und erneuert das Zertifikat selbst (ACME/Let's Encrypt).
#
# Ein von Hand geladenes Wildcard-Zertifikat deckt auch die anderen Namen dieses Hosts ab,
# und Caddy erneuert dann deren eigene ACME-Zertifikate nicht mehr — es sieht ja eine
# gültige Abdeckung. Läuft das Wildcard ab, stehen alle drei Dienste ohne Rückfall da.
#
# Wildcards brauchen eine DNS-01-Challenge, die Caddy ohne DNS-Plugin nicht kann —
# ein solches Zertifikat muss also immer von Hand nachgelegt werden. Für drei feste
# Subdomains braucht es das nicht: HTTP-01 auf Port 80 genügt, und das erneuert sich
# von allein.
#
# Von außerhalb des Schulnetzes kommt nur die Seite durch, auf der der Hauptlieferant seine
# Bestellung bestätigt, mit den Dateien, die sie lädt, und /health für die Ausfallmeldung
# (ARCHITEKTUR.md A29). Eine Firewall öffnet nur den ganzen Port, die Seite sieht erst Caddy.
# SCHULNETZ steht in der Umgebung des Caddy-Containers; fehlt er oder ist er leer, bleibt alles
# offen wie auf dem Testserver. Die Zeile mit expression hält den leeren Wert offen: Ohne sie
# gälte bei SCHULNETZ= keine Adresse als Schulnetz, auch nicht die Theke.
#
# Der zweite handle-Block hat bewusst keinen Pfad: Caddy sortiert handle-Blöcke selbst, und ein
# Block mit Pfad wie /* liefe vor dem mit @draussen. Welche Dateien die Seite vom Wurzelpfad lädt,
# hält caddy_von_aussen_test.go gegen index.html und frontend/public.
bibliothek.fla-dev.de {
    @draussen {
        expression `{env.SCHULNETZ} != ""`
        not remote_ip {$SCHULNETZ:0.0.0.0/0 ::/0}
        not path /bestellung/* /api/public/bestellung/* /assets/* /darstellung.js /favicon.svg /manifest.webmanifest /health
    }
    handle @draussen {
        respond "Die Bibliothek ist nur im Schulnetz erreichbar." 403
    }

    handle {
        reverse_proxy bibliothek-backend:8083 {
            transport http {
                response_header_timeout 300s
                read_timeout 600s
                write_timeout 600s
            }
        }
    }
}

# =============================================================================
# Schadenmeldung (eigener Stack in /root/schadenmeldung)
# =============================================================================
# Die Fehlversuchsbremse der Schadenmeldung zählt nach X-Real-IP (PROXY_VERTRAUEN=1). Ohne die
# Zeile zählte sie alle Anmeldungen als eine Adresse, und ein Client könnte den Kopf selbst setzen.
schaden.fla-dev.de {
    encode zstd gzip
    reverse_proxy schaden:8090 {
        header_up X-Real-IP {remote_host}
    }
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
}
EOF

# Erst prüfen, dann ersetzen. `caddy validate` liest die Datei, ohne sie zu übernehmen. Die laufende
# ist als einzelne Datei in den Container eingehängt; die neue kommt darum per docker cp hinein.
docker cp "$neu" "$container:/tmp/Caddyfile.neu"
if ! docker exec "$container" caddy validate --config /tmp/Caddyfile.neu --adapter caddyfile; then
    docker exec "$container" rm -f /tmp/Caddyfile.neu || true
    echo "FEHLER: Die neue Caddyfile ist ungültig — die laufende bleibt, wie sie ist." >&2
    exit 1
fi
docker exec "$container" rm -f /tmp/Caddyfile.neu

# An Ort und Stelle überschreiben (cat >): Eine ersetzte Datei sähe der Container nicht.
cat "$neu" > "$ziel"

# reload statt restart: Caddy übernimmt die neue Konfiguration unterbrechungsfrei und
# behält bestehende Verbindungen. `docker restart` trennt jede laufende SSE-Verbindung
# und wirft für ein paar Sekunden 502 — bei einem Reverse Proxy für drei produktive
# Dienste ist das der teurere Weg.
docker exec "$container" caddy reload --config /etc/caddy/Caddyfile

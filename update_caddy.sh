#!/bin/bash
cat << 'EOF' > /root/caddy/Caddyfile
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
flasch.herzog-dupont.de {
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
# Inventur-Programm (Bücherverwaltung)
# =============================================================================
flasch2.herzog-dupont.de {
    # API-Anfragen an das Go-Backend
    handle /api/* {
        reverse_proxy inventur-backend-1:8080
    }

    # Statische Upload-Dateien (Buchcover etc.) vom Backend
    handle /uploads/* {
        reverse_proxy inventur-backend-1:8080
    }

    # Alles andere: Svelte-Frontend (SvelteKit Node-Server)
    handle {
        reverse_proxy inventur-frontend-1:3000
    }

    # Security Headers
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "SAMEORIGIN"
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
# Block mit Pfad wie /* liefe vor dem mit @draussen.
flasch3.herzog-dupont.de {
    @draussen {
        expression `{env.SCHULNETZ} != ""`
        not remote_ip {$SCHULNETZ:0.0.0.0/0 ::/0}
        not path /bestellung/* /api/public/bestellung/* /assets/* /favicon.svg /icons.svg /manifest.webmanifest /health
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
EOF

# Erst prüfen, dann laden. `caddy validate` liest die Datei, ohne sie zu übernehmen —
# ein Tippfehler fällt damit auf, bevor er den Reverse Proxy aller drei Dienste trifft.
docker exec caddy caddy validate --config /etc/caddy/Caddyfile || {
    echo "FEHLER: Caddyfile ist ungültig — nichts geladen, alter Stand läuft weiter." >&2
    exit 1
}

# reload statt restart: Caddy übernimmt die neue Konfiguration unterbrechungsfrei und
# behält bestehende Verbindungen. `docker restart` trennt jede laufende SSE-Verbindung
# und wirft für ein paar Sekunden 502 — bei einem Reverse Proxy für drei produktive
# Dienste ist das der teurere Weg.
docker exec caddy caddy reload --config /etc/caddy/Caddyfile

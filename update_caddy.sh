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
# KEINE tls-Zeile. Caddy holt und erneuert das Zertifikat selbst (ACME/Let's Encrypt).
#
# Hier stand bis zum 06.08.2026 `tls /etc/caddy/certs/flasch3.crt …` — ein am
# 14.06.2026 von Hand abgelegtes Wildcard *.herzog-dupont.de. Das hat nicht nur
# flasch3 betroffen: Ein manuell geladenes Zertifikat deckt per Wildcard AUCH
# flasch und flasch2 ab, und Caddy erneuert dann die eigenen ACME-Zertifikate
# dieser Namen nicht mehr — es sieht ja eine gültige Abdeckung.
#
# Die Folge stand am 06.08.2026 im Log: flasch2 war 15 Tage abgelaufen und wurde
# aufgeräumt, das Erneuerungsfenster von flasch (25.–27.07.) war verstrichen.
# Ein Wildcard mit Ablauf 15.08. war damit der EINZIGE Schutz für drei Dienste,
# ohne Rückfall und ohne Erneuerung.
#
# Wildcards brauchen eine DNS-01-Challenge, die Caddy ohne DNS-Plugin nicht kann —
# ein solches Zertifikat muss also immer von Hand nachgelegt werden. Für drei feste
# Subdomains braucht es das nicht: HTTP-01 auf Port 80 genügt, und das erneuert sich
# von allein.
flasch3.herzog-dupont.de {
    handle /* {
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

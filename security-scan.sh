#!/bin/bash

echo "========================================="
echo "🛡️  Starte Security-Audit für Go-Backend..."
echo "========================================="

# 1. Statische Code-Analyse (SAST) mit gosec
echo -e "\n---> 1. Führe gosec aus (Quellcode-Analyse)..."
go run github.com/securego/gosec/v2/cmd/gosec@latest ./...

# 2. Abhängigkeiten und Konfigurationen mit Trivy prüfen
# (Voraussetzung: Trivy ist installiert, z. B. via 'brew install trivy')
echo -e "\n---> 2. Führe Trivy aus (Bibliotheken- & Schwachstellen-Scan)..."
trivy fs --scanners vuln,config .

# 3. Dynamischer API-Scan mit OWASP ZAP (DAST) über Docker
echo -e "\n---> 3. Führe OWASP ZAP API-Scan aus..."

# Ziel ist der lokale Stack (docker-compose.local.yml, Port 8084): Swagger gibt es nur
# bei APP_ENV=local/development. Die Spezifikation nennt bewusst keinen Host (kein
# @host in main.go), ZAP nimmt dann den Host dieser URL. Kein -O: Mit "http://host:port"
# scheitert der Import im OpenAPI-Add-on („http: Name or service not known"), mit
# "host:port" zerlegt zap-api-scan.py den Wert per urlparse als Schema und scannt ein
# Ziel ohne http:// (beides am 13.09.2026 ausprobiert).
ZIEL="host.docker.internal:8084"

# Die Anwendung liest die Sitzung aus dem Cookie session_token, einen Authorization-
# Header wertet sie nicht aus (bis 13.09.2026 schickte dieses Skript einen Bearer-Header —
# jede Anfrage lief unangemeldet). ADMIN_TOKEN = Wert des Cookies session_token nach der
# Anmeldung am lokalen Stack. Die geschriebenen Anfragen des Scans scheitern trotzdem an
# CSRF (die Regel ersetzt den ganzen Cookie-Header, csrf_token fehlt) — angemeldet liest
# er also nur. Ohne ADMIN_TOKEN läuft der Scan unangemeldet.
AUTH_KONFIG=""
if [ -n "${ADMIN_TOKEN:-}" ]; then
  AUTH_KONFIG="-config replacer.full_list(0).description=auth \
      -config replacer.full_list(0).enabled=true \
      -config replacer.full_list(0).matchtype=req_header \
      -config replacer.full_list(0).matchstr=Cookie \
      -config replacer.full_list(0).regex=false \
      -config replacer.full_list(0).replacement=session_token=$ADMIN_TOKEN"
else
  echo "ADMIN_TOKEN ist nicht gesetzt — der ZAP-Scan läuft unangemeldet."
fi

docker run -v "$(pwd)":/zap/wrk/:rw -t zaproxy/zap-stable zap-api-scan.py \
  -t "http://$ZIEL/swagger/doc.json" \
  -f openapi \
  -r zap_api_report.html \
  ${AUTH_KONFIG:+-z "$AUTH_KONFIG"}
ZAP_EXIT=$?

# zap-api-scan.py: 0 = nichts gefunden, 1 = FAIL, 2 = WARN, 3 = Lauf abgebrochen (z. B.
# „Failed to import any URLs"). Bis 13.09.2026 stand hier unbedingt „Scan komplett" —
# auch nach einem Lauf, der keine einzige URL importiert hatte.
if [ "$ZAP_EXIT" -ge 3 ]; then
  echo -e "\n❌ ZAP-Scan abgebrochen (Exit $ZAP_EXIT) — Ausgabe oben lesen, es gibt keinen Report." >&2
  exit "$ZAP_EXIT"
fi
echo -e "\n✅ Scan komplett (Exit $ZAP_EXIT)! Der ZAP-Report liegt als 'zap_api_report.html' in deinem Ordner."

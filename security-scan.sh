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

# Trage hier den Token ein, den dir das Seed-Skript im Terminal ausgegeben hat
# Token über Umgebungsvariable setzen: export ADMIN_TOKEN="dein-jwt-token"
ADMIN_TOKEN="${ADMIN_TOKEN:-DEIN_ADMIN_TOKEN_HIER_EINTRAGEN}"

# ZAP holt sich die Endpunkte direkt aus deiner neuen Swagger.json
docker run -v $(pwd):/zap/wrk/:rw -t zaproxy/zap-stable zap-api-scan.py \
  -t http://host.docker.internal:8080/swagger/doc.json \
  -f openapi \
  -r zap_api_report.html \
  -z "-config replacer.full_list(0).description=auth \
      -config replacer.full_list(0).enabled=true \
      -config replacer.full_list(0).matchtype=req_header \
      -config replacer.full_list(0).matchstr=Authorization \
      -config replacer.full_list(0).regex=false \
      -config replacer.full_list(0).replacement=\"Bearer $ADMIN_TOKEN\""

echo -e "\n✅ Scan komplett! Der ZAP-Report liegt als 'zap_api_report.html' in deinem Ordner."

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
ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiZmZlOThlZDEtYzQ0Ni00ZGE0LWEwMmEtYmQ4ZGQ4ZjI0MGRhIiwiYmFyY29kZV9pZCI6IkFETUlOLVNDQU5ORVItVEVTVCIsInJvbGxlIjoiYWRtaW4iLCJpc3MiOiJiaWJsaW90aGVrLXN5c3RlbSIsImV4cCI6MTgxMzU3OTY3NCwibmJmIjoxNzgyMDQzNjc0LCJpYXQiOjE3ODIwNDM2NzR9.g76Sft-XCzoXlmmFzn7rBfG7WKQ5bQ4Ojz48dlaOX74"

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

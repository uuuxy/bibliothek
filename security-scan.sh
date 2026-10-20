#!/bin/bash

echo "========================================="
echo "🛡️  Starte Security-Audit für Go-Backend..."
echo "========================================="

# 1. Statische Code-Analyse (SAST) mit gosec
# Die Fassung steht in scripts/gosec-gate.sh. Hier läuft gosec ohne dessen Ausnahmen: Der
# Sammel-Scan zeigt auch die Stellen der global ausgenommenen Regeln.
echo -e "\n---> 1. Führe gosec aus (Quellcode-Analyse)..."
GOSEC_FASSUNG="$(sed -n 's/^GOSEC_FASSUNG="\(.*\)"$/\1/p' scripts/gosec-gate.sh)"
go run "github.com/securego/gosec/v2/cmd/gosec@${GOSEC_FASSUNG}" ./...

# 2. Abhängigkeiten und Konfigurationen mit Trivy prüfen
# (Voraussetzung: Trivy ist installiert, z. B. via 'brew install trivy')
echo -e "\n---> 2. Führe Trivy aus (Bibliotheken- & Schwachstellen-Scan)..."
trivy fs --scanners vuln,config .

# 3. Dynamischer API-Scan mit OWASP ZAP (DAST) über Docker
echo -e "\n---> 3. Führe OWASP ZAP API-Scan aus..."

# Ziel ist der lokale Stack (docker-compose.local.yml, Port 8084). Die Eingabe für ZAP ist
# eine OpenAPI-Datei aus allen registrierten Routen (scripts/api_inventar.sh --routen): Pfade
# und Methoden, Pfadparameter als Text, keine Rümpfe. Geschriebene Anfragen scheitern im Scan
# ohnehin an CSRF (siehe unten). Teilbaum-Einhängungen ohne Methode gehen nicht mit. Der Host
# steht unter servers in der Datei.
ZIEL="http://host.docker.internal:8084"
./scripts/api_inventar.sh --routen | python3 -I -c '
import json, re, sys
pfade = {}
for zeile in sys.stdin:
    methode, _, pfad = zeile.strip().partition(" ")
    if not pfad.startswith("/"):
        continue
    operation = {"responses": {"default": {"description": "Antwort"}}}
    namen = re.findall(r"\{([^}]+)\}", pfad)
    if namen:
        operation["parameters"] = [
            {"name": n, "in": "path", "required": True, "schema": {"type": "string"}} for n in namen
        ]
    pfade.setdefault(pfad, {})[methode.lower()] = operation
json.dump({"openapi": "3.0.3", "info": {"title": "Bibliothek", "version": "1"},
           "servers": [{"url": sys.argv[1]}], "paths": pfade}, sys.stdout, indent=1)
' "$ZIEL" > zap_openapi.json

# Die Anwendung liest die Sitzung aus dem Cookie session_token, einen Authorization-
# Header wertet sie nicht aus. ADMIN_TOKEN = Wert des Cookies session_token nach der
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
  -t /zap/wrk/zap_openapi.json \
  -f openapi \
  -r zap_api_report.html \
  ${AUTH_KONFIG:+-z "$AUTH_KONFIG"}
ZAP_EXIT=$?

# zap-api-scan.py: 0 = nichts gefunden, 1 = FAIL, 2 = WARN, 3 = Lauf abgebrochen (z. B.
# „Failed to import any URLs"); ohne diese Prüfung stünde „Scan komplett" auch nach einem
# Lauf, der keine einzige URL importiert hat.
if [ "$ZAP_EXIT" -ge 3 ]; then
  echo -e "\n❌ ZAP-Scan abgebrochen (Exit $ZAP_EXIT) — Ausgabe oben lesen, es gibt keinen Report." >&2
  exit "$ZAP_EXIT"
fi
echo -e "\n✅ Scan komplett (Exit $ZAP_EXIT)! Der ZAP-Report liegt als 'zap_api_report.html' in deinem Ordner."

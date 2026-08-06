#!/bin/bash
#
# pruefe_secrets.sh — Prüft die .env eines Servers auf die Geheimnisse, deren
# Fehlkonfiguration NICHT auffällt, solange man nicht danach sucht.
#
# Warum es das gibt: Alle drei Fälle unten laufen still. Ein Default-Secret meldet
# niemand, ein fehlender Backup-Schlüssel lässt den nächtlichen Job sich selbst
# überspringen, und ein ungesetztes ENFORCE_PROD_SECRETS heißt bloß, dass der Server
# startet. Man merkt es erst, wenn jemand ein Backup braucht oder ein fremdes JWT
# vorlegt.
#
# Das Skript ÄNDERT NICHTS. Es liest, meldet und liefert einen Exit-Code.
#
# Aufruf auf dem Zielserver:
#   ./scripts/pruefe_secrets.sh                 # nutzt ./.env
#   ./scripts/pruefe_secrets.sh /opt/bibliothek/.env
#
# Exit: 0 = alles gut, 1 = mindestens ein kritischer Befund.

set -uo pipefail

ENV_FILE="${1:-$(dirname "$0")/../.env}"

ROT='\033[0;31m'; GELB='\033[0;33m'; GRUEN='\033[0;32m'; AUS='\033[0m'
befunde=0
warnungen=0

if [ ! -f "$ENV_FILE" ]; then
	printf "${ROT}FEHLER${AUS}: %s nicht gefunden.\n" "$ENV_FILE"
	exit 1
fi

printf "Prüfe %s\n\n" "$ENV_FILE"

# lies VARNAME -> gibt den Wert aus (leer, wenn nicht gesetzt). Kommentare ignoriert.
lies() {
	grep -E "^${1}=" "$ENV_FILE" 2>/dev/null | tail -1 | cut -d= -f2- | tr -d '\r'
}

kritisch() { printf "  ${ROT}✗ %s${AUS}\n    %s\n" "$1" "$2"; befunde=$((befunde + 1)); }
warnung() { printf "  ${GELB}! %s${AUS}\n    %s\n" "$1" "$2"; warnungen=$((warnungen + 1)); }
gut() { printf "  ${GRUEN}✓ %s${AUS}\n" "$1"; }

# Die Default-Werte aus docker-compose.yml und main.go. Wer einen davon einsetzt,
# nutzt ein Geheimnis, das öffentlich im Repository steht.
ist_bekannter_default() {
	case "$1" in
	"super-secret-default-key-at-least-32-bytes" | "super-secure-aes-key-32-chars-ok" | \
		"supergeheim_lokal" | "local-dev-jwt-secret-min-32-characters" | "admin" | "postgrespassword")
		return 0
		;;
	*) return 1 ;;
	esac
}

echo "── Geheimnisse ──────────────────────────────────────────"

# Mindestlängen. Ein gesetzter, aber kurzer Wert ist KEIN erledigter Punkt — genau das
# verdeckte die erste Fassung dieses Skripts: Ein sechsstelliges Datenbankpasswort auf
# dem Schulserver bekam ein grünes Häkchen, nur weil es kein bekannter Default war.
mindestlaenge() {
	case "$1" in
	JWT_SECRET) echo 32 ;;         # HMAC-Schlüssel, main.go verlangt >= 32
	APP_ENCRYPTION_KEY) echo 32 ;; # AES-256: exakt 32 Byte oder 64 Hex-Zeichen
	POSTGRES_PASSWORD) echo 16 ;;  # kein harter Zwang im Code, aber alles darunter ist ratbar
	*) echo 0 ;;
	esac
}

# zerlegt_die_dsn meldet Zeichen, die in einer URL eine Bedeutung haben. POSTGRES_PASSWORD
# landet unverändert in postgres://postgres:PASSWORT@postgres-db:5432/bibliothek — ein "/"
# beendet dort den Host-Teil, ein "+" verschiebt den Rest, ein "@" trennt Anmeldedaten von
# Host.
#
# Anlass (06.08.2026): Ein Passwortwechsel mit `openssl rand -base64 24` erzeugte "/" und
# "+". ALTER ROLE lief durch, der Container meldete "Started", und DIESES Skript meldete
# "Alles in Ordnung" — die Anwendung war trotzdem unten:
#   failed to parse as URL (invalid port ":sGWO+wgLlTIj" after host)
# Eine Konfigurationsprüfung, die nur Länge und Bekanntheit misst, sieht so etwas nicht.
zerlegt_die_dsn() {
	case "$1" in
	*/* | *+* | *@* | *\?* | *\#* | *%*) return 0 ;;
	*) return 1 ;;
	esac
}

for var in JWT_SECRET APP_ENCRYPTION_KEY POSTGRES_PASSWORD; do
	wert="$(lies "$var")"
	min="$(mindestlaenge "$var")"
	if [ -z "$wert" ]; then
		kritisch "$var ist nicht gesetzt" \
			"Compose setzt dann seinen Default ein — der steht öffentlich im Repository (docker-compose.yml). Bei APP_ENCRYPTION_KEY heißt das: Schülerfotos und das SMTP-Passwort sind mit einem bekannten Schlüssel verschlüsselt."
	elif ist_bekannter_default "$wert"; then
		kritisch "$var nutzt einen bekannten Default-Wert" \
			"Dieser Wert steht öffentlich im Repository. Bei APP_ENCRYPTION_KEY NICHT einfach ersetzen — sonst sind Schülerfotos und das SMTP-Passwort verloren. Weg: docs/SECURITY.md (cmd/rotate-encryption-key)."
	elif [ "${#wert}" -lt "$min" ]; then
		kritisch "$var ist nur ${#wert} Zeichen lang (erwartet: >= $min)" \
			"Gesetzt heißt nicht sicher — ein kurzes Geheimnis ist ratbar."
	elif [ "$var" = "POSTGRES_PASSWORD" ] && zerlegt_die_dsn "$wert"; then
		kritisch "POSTGRES_PASSWORD enthält ein Zeichen, das die DATABASE_URL zerlegt" \
			"Das Passwort steht unverändert in postgres://postgres:PASSWORT@host:5432/db. Die Zeichen / + @ ? # beenden dort den Host- oder Passwortteil, und der Server startet nicht mehr ('failed to parse as URL'). Neu erzeugen mit: openssl rand -hex 24"
	else
		gut "$var ist gesetzt (${#wert} Zeichen)"
	fi
done

echo
echo "── Backups ──────────────────────────────────────────────"

backup_key="$(lies BACKUP_ENCRYPTION_KEY)"
if [ -z "$backup_key" ]; then
	kritisch "BACKUP_ENCRYPTION_KEY ist nicht gesetzt" \
		"Der nächtliche Job überspringt sich STILL (jobs/backup.go). Es gibt keine Backups."
elif [ "${#backup_key}" -lt 32 ]; then
	warnung "BACKUP_ENCRYPTION_KEY ist nur ${#backup_key} Zeichen lang" \
		"Die Ableitung läuft per SHA-256 und ist schnell — kurze Passphrasen sind an einer entwendeten Backup-Datei offline angreifbar. Empfohlen: >= 32."
else
	gut "BACKUP_ENCRYPTION_KEY ist gesetzt (${#backup_key} Zeichen)"
fi

echo
echo "── Produktionsschalter ──────────────────────────────────"

pruefe_schalter() {
	local var="$1" erwartet="$2" begruendung="$3"
	local wert
	wert="$(lies "$var")"
	if [ "$(echo "$wert" | tr '[:upper:]' '[:lower:]')" = "$erwartet" ]; then
		gut "$var=$erwartet"
	else
		kritisch "$var ist nicht $erwartet (aktuell: ${wert:-<nicht gesetzt>})" "$begruendung"
	fi
}

pruefe_schalter ENFORCE_PROD_SECRETS true \
	"Ohne diesen Schalter startet der Server auch mit den Default-Secrets, statt den Start zu verweigern."
pruefe_schalter COOKIE_SECURE true \
	"Hinter Caddy-HTTPS gehört das Sitzungscookie auf Secure."

imap="$(lies IMAP_HOST)"
if [ "$imap" = "mock" ]; then
	kritisch "IMAP_HOST=mock" "Der Mock akzeptiert JEDES Passwort für jede bekannte E-Mail. Die Anmeldung steht damit offen."
elif [ -z "$imap" ]; then
	warnung "IMAP_HOST ist nicht gesetzt" "Der Prod-Stack startet dann gar nicht (Pflichtvariable in docker-compose.yml)."
else
	gut "IMAP_HOST=$imap"
fi

appenv="$(lies APP_ENV)"
if [ "$appenv" = "local" ] || [ "$appenv" = "development" ]; then
	kritisch "APP_ENV=$appenv" "Damit sind die Swagger-Docs öffentlich und der IMAP-Mock erlaubt."
else
	gut "APP_ENV=${appenv:-production (Compose-Default)}"
fi

echo
echo "─────────────────────────────────────────────────────────"
if [ "$befunde" -gt 0 ]; then
	printf "${ROT}%d kritische(r) Befund(e)${AUS}, %d Warnung(en).\n" "$befunde" "$warnungen"
	echo "Vorgehen: docs/DEPLOYMENT.md (Checkliste) und docs/SECURITY.md (Schlüsselwechsel)."
	exit 1
fi
if [ "$warnungen" -gt 0 ]; then
	printf "${GELB}Keine kritischen Befunde, %d Warnung(en).${AUS}\n" "$warnungen"
	exit 0
fi
printf "${GRUEN}Alles in Ordnung.${AUS}\n"

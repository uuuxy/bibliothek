#!/usr/bin/env bash
# ==============================================================================
# backup_krypto.sh — gemeinsame Verschlüsselungs-Helfer für die Shell-Backups
# ==============================================================================
# Wird von scripts/backup.sh und ./update.sh eingebunden (source), nicht selbst
# aufgerufen.
#
# Warum es das gibt (Befund A5, docs/datenschutz_offene_punkte.md): Beide Skripte
# dumpten am nächtlichen Job vorbei und legten `.sql.gz` im KLARTEXT ab — jeder
# Schülername, jede Adresse, jede Ausleihe, 7 bzw. 30 Tage lang. Geschützt waren die
# Dateien nur durch `0600`; wer den Datenträger, ein Datei-Backup des Servers oder das
# Verzeichnis in die Hand bekam, las alles ohne Passphrase.
#
# Der Schlüssel bleibt dabei im Container. `docker exec` reicht ihn NICHT durch — das
# Werkzeug liest ihn aus der Umgebung des Backend-Containers, in der er ohnehin steht
# (docker-compose.yml). Auf dem Host taucht er weder in der Prozessliste noch in einer
# Variablen auf.
# ==============================================================================

BACKEND_CONTAINER="${BACKEND_CONTAINER:-bibliothek-backend}"

# krypto_moeglich prüft alle drei Voraussetzungen und sagt in $KRYPTO_GRUND, welche
# fehlt. Kein Zugriff auf den Schlüsselwert selbst: `sh -c '[ -n "$VAR" ]'` läuft IM
# Container und liefert nur einen Exit-Code zurück — `printenv` hätte die Passphrase in
# die Pipe des Hosts geschrieben.
krypto_moeglich() {
    KRYPTO_GRUND=""
    if ! docker ps --filter "name=${BACKEND_CONTAINER}" --filter "status=running" \
        --format '{{.Names}}' | grep -q "${BACKEND_CONTAINER}"; then
        KRYPTO_GRUND="Backend-Container '${BACKEND_CONTAINER}' läuft nicht"
        return 1
    fi
    if ! docker exec "${BACKEND_CONTAINER}" test -x ./encrypt-backup 2>/dev/null; then
        KRYPTO_GRUND="Werkzeug ./encrypt-backup fehlt im laufenden Image (Stand vor dem 23.08.2026?)"
        return 1
    fi
    if ! docker exec "${BACKEND_CONTAINER}" sh -c '[ -n "$BACKUP_ENCRYPTION_KEY" ]' 2>/dev/null; then
        KRYPTO_GRUND="BACKUP_ENCRYPTION_KEY ist im Container nicht gesetzt"
        return 1
    fi
    return 0
}

# krypto_pipe verschlüsselt stdin nach stdout — für Pipelines, in denen der Klartext
# NIE die Platte berühren soll (scripts/backup.sh).
krypto_pipe() {
    docker exec -i "${BACKEND_CONTAINER}" ./encrypt-backup
}

# pruefe_enc_datei ist die billige Formprüfung: Datei da, nicht leer, Format-Kennung
# `BKDF` am Anfang.
pruefe_enc_datei() {
    local datei="$1"
    [ -s "${datei}" ] || return 1
    [ "$(head -c 4 "${datei}" 2>/dev/null)" = "BKDF" ] || return 1
    return 0
}

# pruefe_enc_rundweg beweist den Rückweg an der DATEI, die auf der Platte liegt.
#
# Die Formprüfung allein genügt nicht, und der Fall ist nicht theoretisch: Läuft die
# Platte während des Schreibens voll, trägt die abgeschnittene Datei trotzdem ihre
# `BKDF`-Kennung und hat plausible Größe — bemerkt würde der Verlust erst beim
# Wiederherstellen. Das Werkzeug prüft zwar seinen eigenen Rundweg, aber zwischen ihm
# und der Datei liegt noch die Umleitung der Shell; geprüft werden muss, was ANKOMMT.
#
# Deshalb geht die fertige Datei denselben Weg wie im Ernstfall: durch das
# `restore-backup` im Container. Der entschlüsselte Dump landet dabei in /dev/null —
# er darf nirgends abgelegt werden, und gebraucht wird nur der Exit-Code.
pruefe_enc_rundweg() {
    local datei="$1"
    docker exec -i "${BACKEND_CONTAINER}" sh -c '
        tmp=$(mktemp) || exit 1
        cat > "$tmp" && ./restore-backup "$tmp" > /dev/null
        rc=$?
        rm -f "$tmp"
        exit $rc
    ' < "${datei}" > /dev/null 2>&1
}

# verschluessele_datei wandelt eine bestehende Klartext-Datei in ihr `.enc`-Gegenstück
# und löscht das Original — aber erst, NACHDEM die Gegenprobe bestanden ist. Bei jedem
# Fehler bleibt der Klartext liegen: eine ungeprüfte Verschlüsselung ist kein Grund,
# die einzige lesbare Sicherung wegzuwerfen.
# Rückgabe: 0 = verschlüsselt und Klartext entfernt, 1 = Klartext liegt unverändert da.
verschluessele_datei() {
    local quelle="$1"
    local ziel="${quelle}.enc"

    if ! (umask 077; krypto_pipe < "${quelle}" > "${ziel}"); then
        rm -f "${ziel}"
        return 1
    fi
    if ! pruefe_enc_datei "${ziel}" || ! pruefe_enc_rundweg "${ziel}"; then
        rm -f "${ziel}"
        return 1
    fi

    rm -f "${quelle}"
    KRYPTO_ZIEL="${ziel}"
    return 0
}

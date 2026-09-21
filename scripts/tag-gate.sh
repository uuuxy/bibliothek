#!/bin/bash
#
# Das Tag-Gate: Ein v-Tag erzeugt ein Release und ein versioniertes Image nur, wenn ALLE
# Pflicht-Prüfungen des getaggten Commits grün sind.
#
# Warum es dieses Skript gibt (21.09.2026): Die Pflichtliste stand zweimal wörtlich in den
# Workflows — in release.yml und in docker-publish.yml —, samt derselben Auswertung. Zwei
# Abschriften, die ein Test zusammenhielt. Beim Aufnehmen der Security-Jobs (entschieden am
# 21.09.2026) hätten beide Kopien an drei Stellen zugleich anders werden müssen; stattdessen steht
# die Liste jetzt EINMAL hier, und beide Workflows rufen dieses Skript. Nebenbei lässt sich
# das Gate damit am echten Commit nachstellen, statt nur im nächsten Tag-Lauf.
#
# Was sich gegenüber den eingebetteten Fassungen geändert hat:
#
#   1. Die vier Security-Jobs gehören dazu (govulncheck, gosec, npm audit, Trivy). Ein Tag
#      auf einen Commit mit rotem Trivy erzeugte bis hierher trotzdem ein Release. Der
#      übliche Einwand — eine fremde Lücke ohne Fix sperrt das Release — trifft hier nicht:
#      Trivy läuft mit ignore-unfixed, govulncheck hat die Ausnahmeliste mit Wiedervorlage,
#      npm audit läuft mit --omit=dev ab HIGH. Rot heißt bei allen: Es gibt einen Fix, der
#      nicht eingespielt ist.
#   2. Eine Zeile je Name statt einer Liste mit Leerzeichen: GitHub führt einen Prüflauf
#      unter dem Feld `name:` des Jobs („Go – govulncheck"), nicht unter dem Job-Schlüssel
#      (`go-vuln-scan`) — und diese Namen tragen Leerzeichen.
#   3. ALLE Läufe eines Namens müssen grün sein. Ein Commit kann denselben Namen zweimal
#      tragen: einmal vom Push, einmal vom Wochenlauf (gesehen an 1bedaec7, 21.09.2026).
#      Die alte Auswertung nahm den ersten Treffer — welcher das ist, legt niemand fest.
#   4. Die Abfrage blättert (--paginate, 100 je Seite). Die Vorgabe der Schnittstelle sind
#      30 Einträge; ein Name auf Seite zwei hätte als „fehlt" gezählt.
#
# Die Liste wird von docs/umgebung_paritaet_test.go gegen ci.yml und security-scan.yml
# gehalten, als MENGE in beide Richtungen: Ein neuer Job, den diese Liste nicht kennt, ist
# genauso ein Loch wie ein Name hier, den kein Workflow mehr baut. Ein Name, der ins Leere
# zeigt, ist ein Fehler („fehlt") und kein Freifahrtschein.
#
# Aufruf in den Workflows (nach dem Checkout; GH_TOKEN setzt der Schritt):
#   ./scripts/tag-gate.sh
# Am echten Commit nachstellen (braucht nur eine Anmeldung bei gh):
#   GITHUB_REPOSITORY=uuuxy/bibliothek GITHUB_SHA="$(git rev-parse origin/main)" ./scripts/tag-gate.sh
set -euo pipefail

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY fehlt (Form: besitzer/repository)}"
: "${GITHUB_SHA:?GITHUB_SHA fehlt (der zu prüfende Commit)}"

PFLICHT="actionlint
build-and-test
frontend-test
e2e
Go – govulncheck
Go – gosec static analysis
npm – audit (frontend)
Docker – Trivy image scan"

RUNS=$(gh api --paginate "repos/${GITHUB_REPOSITORY}/commits/${GITHUB_SHA}/check-runs?per_page=100" \
    --jq '.check_runs[] | "\(.name)\t\(.conclusion // "pending")"')

FEHLT=""
ROT=""
while IFS= read -r name; do
    [ -n "$name" ] || continue
    ERGEBNISSE=$(printf '%s\n' "$RUNS" | awk -F'\t' -v n="$name" '$1 == n { print $2 }')
    if [ -z "$ERGEBNISSE" ]; then
        FEHLT="${FEHLT}${FEHLT:+, }${name}"
        continue
    fi
    NICHT_GRUEN=$(printf '%s\n' "$ERGEBNISSE" | grep -vx 'success' | sort -u | tr '\n' '/' || true)
    if [ -n "$NICHT_GRUEN" ]; then
        ROT="${ROT}${ROT:+, }${name} = ${NICHT_GRUEN%/}"
    fi
done <<< "$PFLICHT"

if [ -n "$FEHLT" ]; then
    echo "::error::Für ${GITHUB_SHA} fehlen Prüfungen ganz: ${FEHLT}. Entweder laufen sie noch, oder ein Job wurde umbenannt — dann die Liste in scripts/tag-gate.sh nachziehen."
    exit 1
fi
if [ -n "$ROT" ]; then
    echo "::error::Die Prüfungen für ${GITHUB_SHA} sind nicht grün: ${ROT}. Erst grün machen, dann diesen Workflow erneut starten (Re-run)."
    exit 1
fi
echo "Alle Pflicht-Prüfungen grün ($(printf '%s\n' "$PFLICHT" | wc -l | tr -d ' ') Namen) für ${GITHUB_SHA}."

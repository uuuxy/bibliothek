#!/usr/bin/env bash
# gosec mit der Fassung und den Ausnahmen dieses Repositorys. Der Sicherheits-Prüflauf
# (.github/workflows/security-scan.yml) und der Hook vor dem Push (scripts/git-hooks/pre-push)
# rufen beide dieses Skript: Ein Fund soll am Arbeitsplatz auffallen, nicht erst auf GitHub.
#
# Aufruf: ./scripts/gosec-gate.sh   (vom Repo-Root)
set -euo pipefail
cd "$(dirname "$0")/.."

# Eine feste Fassung: Ein neues gosec bringt neue Regeln mit, und die sollen sich mit einem
# eigenen Commit melden. Es ist ein Entwicklungsstand, weil die veröffentlichte v2.29.0
# (golang.org/x/tools 0.49) die Paketdaten von Go 1.27.2 nicht liest; auf die nächste
# veröffentlichte Fassung heben, sobald sie erscheint (docs/OFFEN.md 5.10). `go run` besorgt
# das Werkzeug selbst, ein fehlendes Werkzeug überspringt die Prüfung deshalb nicht.
GOSEC_FASSUNG="v2.29.1-0.20261005092323-d2b649ec0182"

# Ausgenommen sind nur Regeln, die Stellen melden; die Zahl je Regel steht in
# docs/ARCHITEKTUR.md 11.5. Eine global ausgenommene Regel meldet auch eine neue Stelle nicht.
#   G706  Log-Einschleusung: log.Printf läuft über slog.SetDefault durch den JSON-Handler, ein
#         Umbruch bleibt im String (docs/SECURITY.md, „Log-Injection").
#   G704  SSRF: Abrufe bei DNB, Open Library, Google Books und Cover-Quellen; Cover-Adressen
#         gegen eine feste Host-Liste (docs/SECURITY.md, „Cover-Proxy").
#   G703  Pfad aus Eingabe: Cover-Cache und Oberfläche über os.Root gebunden, Backup-Pfade aus
#         Umgebung und Kommandozeile des Betreibers.
#   G120  Formular ohne Grenze: Importe und Uploads; die Body-Grenze der Anfragekette begrenzt
#         jeden Rumpf.
#   G124  Cookie-Attribute: Secure hängt an COOKIE_SECURE (im Betrieb true), HttpOnly und
#         SameSite=Strict stehen fest; zwei Stellen im Lasttest.
#   G404  math/rand: nur im Demo-Seed (cmd/seed).
#   G115  int nach rune: Prüfziffern der Littera-Etiketten (Ziffern 0 bis 9) und der Demo-Seed.
#   G101  „Zugangsdaten": der Kopfname X-CSRF-Token im Lasttest.
AUSNAHMEN="G706,G704,G703,G120,G124,G404,G115,G101"

BERICHT="$(mktemp)"
trap 'rm -f "$BERICHT"' EXIT

# Ohne weitere Schalter: Mit -quiet endet gosec ohne Fund mit Exit 0, auch wenn es kein Paket
# laden konnte.
if ! go run "github.com/securego/gosec/v2/cmd/gosec@${GOSEC_FASSUNG}" \
  -exclude="$AUSNAHMEN" ./... >"$BERICHT" 2>&1; then
  cat "$BERICHT"
  echo "✗ gosec meldet einen Fund oder konnte nicht laufen. Die Ausgabe steht oben, die Funde unter Results."
  exit 1
fi

# Ein Lauf, der keine Datei geprüft hat, ist kein Ergebnis.
DATEIEN="$(sed -n 's/^ *Files *: *\([0-9][0-9]*\) *$/\1/p' "$BERICHT")"
if [ -z "$DATEIEN" ] || [ "$DATEIEN" -eq 0 ]; then
  cat "$BERICHT"
  echo "✗ gosec endete ohne Fund, nennt aber keine geprüfte Datei."
  exit 1
fi
VERMERKE="$(sed -n 's/^ *Nosec *: *\([0-9][0-9]*\) *$/\1/p' "$BERICHT")"
echo "✓ gosec: kein Fund in ${DATEIEN} Dateien (${VERMERKE} Vermerke #nosec)."

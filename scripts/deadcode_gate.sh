#!/bin/sh
# Gate gegen unerreichbaren Go-Code.
#
# Nutzt golang.org/x/tools/cmd/deadcode: Das Werkzeug rechnet Erreichbarkeit ab main()
# ueber den echten Aufrufgraphen und nimmt dabei ALLE main-Pakete als Wurzeln, auch die
# unter cmd/. Ein grep kann das nicht — es zaehlt Textvorkommen und haelt schon die
# Erwaehnung eines Namens im eigenen Doc-Kommentar fuer eine Verwendung.
#
# Warum es dieses Gate gibt (04.08.2026): An einem Tag wurde zweimal ein Einstiegspunkt
# entfernt und der Rumpf stehen gelassen. Die verwaisten Funktionen wurden danach nur
# noch von ihren eigenen Tests am Leben gehalten — gruene Tests fuer Code, den die
# Anwendung nie erreicht. Beide Faelle hat erst dieses Werkzeug aufgedeckt.
#
# -test zaehlt Testbinaries als Einstiegspunkt mit. Gemeldet wird deshalb nur, was WEDER
# von der Anwendung NOCH von einem Test erreicht wird — die echten Leichen. Funktionen,
# die nur Tests benutzen, meldet "deadcode ./..." (ohne -test); das ist eine eigene,
# schwaechere Frage und bewusst nicht Teil dieses Gates.
#
# Diese schwaechere Frage wurde am 11.08.2026 einmal vollstaendig beantwortet: 17 Treffer,
# davon vier echte Leichen (entfernt: inventur.NewDB als zweiter Pool-Konstruktor neben
# db.Connect, crypto.EncryptUpload als zweite Verschluesselungstuer neben crypto.Encrypt,
# repository.QueryUeberfaelligeByAusleiheIDs als Rest eines Tx-Umbaus, und
# scripts/migrate_photos.go als Doppel von cmd/migrate-fotos). Eine fuenfte,
# auth.TokenBlacklist.Stop, wurde nicht geloescht, sondern im geordneten Herunterfahren
# von main.go verdrahtet — sie fehlte dort.
#
# Die verbleibenden 13 sind BEGRUENDET und sollen bleiben. Wer die schwaechere Frage
# erneut stellt, findet genau diese und muss sie nicht noch einmal untersuchen:
#
#   internal/smtptest/*   (6)  Test-Mailserver. Von Tests benutzt zu werden IST sein Zweck.
#   internal/littera/*    (5)  Pruefwerkzeug fuer die noch ausstehende Littera-Migration.
#                              BarcodeInhalt entschluesselt die alte Littera-Druckzeichen-
#                              kette und dient als Gegenprobe zur Exemplarnummer, aus der
#                              die EAN-13 gerechnet wird — 61.520 von 61.520 stimmen
#                              ueberein. Der Lauf haengt an LITTERA_CSV_DIR und braucht die
#                              echten Altdaten, die nicht im Repo liegen.
#   sse.istGeschlossen    (1)  Nahtstelle, damit ein Test das Herunterfahren pruefen kann.
#   uebernahme.Leeren     (1)  Zuruecksetzen zwischen PG-Testlaeufen.
#
# Aufruf: ./scripts/deadcode_gate.sh   (vom Repo-Root)
set -eu

if ! command -v deadcode >/dev/null 2>&1; then
	# Bewusst KEIN stilles Ueberspringen: Ein Waechter, der sich abschaltet, sobald sein
	# Werkzeug fehlt, meldet jahrelang gruen und hat nie etwas geprueft.
	echo "deadcode ist nicht installiert — das Gate kann nicht laufen." >&2
	echo "  go install golang.org/x/tools/cmd/deadcode@latest" >&2
	exit 1
fi

# node_modules enthaelt eine fremde Go-Datei in einem npm-Paket (flatted) und gehoert
# nicht zu diesem Projekt.
LEICHEN=$(deadcode -test ./... 2>/dev/null | grep -v node_modules || true)

if [ -z "$LEICHEN" ]; then
	echo "deadcode: kein unerreichbarer Code."
	exit 0
fi

echo "deadcode: unerreichbarer Code gefunden — entfernen oder verdrahten:" >&2
echo "$LEICHEN" >&2
exit 1

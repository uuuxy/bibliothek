package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestIstZeilenfehler sichert die Weiche ab, an der sich entscheidet, ob „übersprungen"
// die Wahrheit sagt. Fällt ein Verbindungsabbruch versehentlich auf die
// Zeilenfehler-Seite, protokolliert das Werkzeug wieder Tausende Ausfälle, die es nie
// versucht hat — dieselbe Klasse Falschmeldung, die diese Änderung beseitigt.
func TestIstZeilenfehler(t *testing.T) {
	pgFehler := func(code string) error { return &pgconn.PgError{Code: code, Message: code} }

	faelle := []struct {
		name     string
		err      error
		erwartet bool
	}{
		{"doppelte ISBN oder Barcode (23505)", pgFehler("23505"), true},
		{"NOT NULL verletzt (23502)", pgFehler("23502"), true},
		{"CHECK verletzt (23514)", pgFehler("23514"), true},
		{"Wert zu lang für die Spalte (22001)", pgFehler("22001"), true},
		{"Zahl außerhalb des Bereichs (22003)", pgFehler("22003"), true},
		{"Datensatzfehler ohne Postgres", fmt.Errorf("%w: kaputtes JSONB", errZeile), true},
		{"umschlossener Datensatzfehler", fmt.Errorf("Exemplar B-1: %w", pgFehler("23505")), true},

		{"vergiftete Transaktion (25P02)", pgFehler("25P02"), false},
		{"Verbindung verloren (08006)", pgFehler("08006"), false},
		{"Deadlock (40P01)", pgFehler("40P01"), false},
		{"Administrator hat beendet (57P01)", pgFehler("57P01"), false},
		{"interner Fehler (XX000)", pgFehler("XX000"), false},
		{"Kontext abgebrochen", context.Canceled, false},
		{"Zeitüberschreitung", context.DeadlineExceeded, false},
		{"Netzwerkfehler ohne SQLSTATE", &net.OpError{Op: "read", Err: errors.New("broken pipe")}, false},
		{"kein Fehler", nil, false},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := istZeilenfehler(f.err); got != f.erwartet {
				t.Errorf("istZeilenfehler(%v) = %v, erwartet %v", f.err, got, f.erwartet)
			}
		})
	}
}

// TestBeschreibeFehlerBehaeltUrsacheUndKontext: der Grund, warum dieses Werkzeug
// existiert, ist die Frage „was stimmt mit meiner Quelldatei nicht". Die Antwort muss den
// umschließenden Kontext (welcher Barcode) UND die Postgres-Angaben (SQLSTATE,
// Constraint) enthalten — errors.As allein greift bis zum PgError durch und verliert
// alles davor.
func TestBeschreibeFehlerBehaeltUrsacheUndKontext(t *testing.T) {
	pgErr := &pgconn.PgError{
		Code:           "23505",
		Message:        "duplicate key value violates unique constraint",
		ConstraintName: "buecher_exemplare_barcode_id_key",
		Detail:         "Key (barcode_id)=(B-00042) already exists.",
	}
	text := beschreibeFehler(fmt.Errorf("Exemplar mit Barcode B-00042: %w", pgErr))

	for _, erwartet := range []string{"B-00042", "23505", "buecher_exemplare_barcode_id_key", "already exists"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("Fehlerbeschreibung nennt %q nicht: %s", erwartet, text)
		}
	}
}

func TestBeschreibeFehlerOhnePostgres(t *testing.T) {
	if got := beschreibeFehler(errors.New("etwas ging schief")); got != "etwas ging schief" {
		t.Errorf("unerwartete Beschreibung: %q", got)
	}
}

// TestKuerzeAufSpalteZaehltZeichen: Postgres begrenzt varchar in ZEICHEN. Ein byteweiser
// Schnitt zerlegte deutsche Umlaute und schriebe ungültiges UTF-8 in die Datenbank.
func TestKuerzeAufSpalteZaehltZeichen(t *testing.T) {
	el := verwerfenderLogger(t)
	m := mysqlMedium{ID: 1, Titel: "x"}

	lang := strings.Repeat("ä", 300)
	got := kuerzeAufSpalte(el, m, "titel", lang, maxTitelSpalte)

	if r := []rune(got); len(r) != maxTitelSpalte {
		t.Fatalf("%d Zeichen erwartet, geliefert: %d", maxTitelSpalte, len(r))
	}
	if !isValidUTF8(got) {
		t.Error("die Kürzung hat ein Zeichen zerschnitten")
	}
	if el.warnings != 1 {
		t.Errorf("die Kürzung muss protokolliert werden, gezählt: %d", el.warnings)
	}
}

func TestKuerzeAufSpalteLaesstPassendeWerteUnberuehrt(t *testing.T) {
	el := verwerfenderLogger(t)
	m := mysqlMedium{ID: 1}

	grenzwert := strings.Repeat("ö", maxTitelSpalte)
	if got := kuerzeAufSpalte(el, m, "titel", grenzwert, maxTitelSpalte); got != grenzwert {
		t.Error("ein Wert exakt auf Spaltenbreite darf nicht angetastet werden")
	}
	if el.warnings != 0 {
		t.Errorf("keine Warnung erwartet, gezählt: %d", el.warnings)
	}
}

// TestKlaerISBNWertetAbStattZuVerwerfen hält die Grundhaltung dieser Datei fest: eine
// kaputte oder doppelte ISBN kostet die ISBN, nicht das Buch.
func TestKlaerISBNWertetAbStattZuVerwerfen(t *testing.T) {
	t.Run("ungültige Prüfziffer", func(t *testing.T) {
		el := verwerfenderLogger(t)
		m := mysqlMedium{ID: 1, ISBN: sql.NullString{String: "9783161484101", Valid: true}}
		if got := klaerISBN(m, map[string]int{}, el); got != "" {
			t.Errorf("leere ISBN erwartet, geliefert: %q", got)
		}
		if el.warnings != 1 || el.failures != 0 {
			t.Errorf("WARNUNG erwartet, gezählt: %d/%d", el.warnings, el.failures)
		}
	})

	t.Run("Dublette im selben Lauf", func(t *testing.T) {
		el := verwerfenderLogger(t)
		seen := map[string]int{"9783161484100": 7}
		m := mysqlMedium{ID: 9, ISBN: sql.NullString{String: "978-3-16-148410-0", Valid: true}}
		if got := klaerISBN(m, seen, el); got != "" {
			t.Errorf("leere ISBN erwartet, geliefert: %q", got)
		}
		if seen["9783161484100"] != 7 {
			t.Error("die bestehende Reservierung darf nicht überschrieben werden")
		}
	})

	t.Run("gültige ISBN wird reserviert", func(t *testing.T) {
		el := verwerfenderLogger(t)
		seen := map[string]int{}
		m := mysqlMedium{ID: 3, ISBN: sql.NullString{String: "978-3-16-148410-0", Valid: true}}
		if got := klaerISBN(m, seen, el); got != "9783161484100" {
			t.Errorf("normalisierte ISBN erwartet, geliefert: %q", got)
		}
		if seen["9783161484100"] != 3 {
			t.Error("die ISBN muss für den laufenden Import reserviert werden")
		}
		if el.warnings != 0 {
			t.Errorf("keine Warnung erwartet, gezählt: %d", el.warnings)
		}
	})
}

// TestExitCodeSagtDieWahrheit: der Rückgabewert ist die Kurzfassung des Berichts. Ein
// unvollständiger Lauf darf niemals als Erfolg enden — auch dann nicht, wenn die
// Zählungen unauffällig aussehen und nur der Abgleich widerspricht.
func TestExitCodeSagtDieWahrheit(t *testing.T) {
	faelle := []struct {
		name     string
		bericht  abschlussbericht
		erwartet int
	}{
		{"sauber", abschlussbericht{AbgleichOK: true}, exitOK},
		{"nur Warnungen", abschlussbericht{Warnungen: 12, AbgleichOK: true}, exitOK},
		{"übersprungene Titel", abschlussbericht{Uebersprungen: 1, Fehler: 1, AbgleichOK: true}, exitUnvollstaendig},
		{"Abgleich widerspricht", abschlussbericht{AbgleichOK: false}, exitUnvollstaendig},
		{"abgebrochen", abschlussbericht{Abbruch: errors.New("Verbindung weg"), AbgleichOK: true}, exitAbgebrochen},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			if got := f.bericht.exitCode(); got != f.erwartet {
				t.Errorf("exitCode() = %d, erwartet %d", got, f.erwartet)
			}
		})
	}
}

// verwerfenderLogger schreibt in eine temporäre Datei; geprüft werden hier nur die Zähler.
func verwerfenderLogger(t *testing.T) *errLogger {
	t.Helper()
	el, err := newErrLoggerAt(t.TempDir() + "/err.log")
	if err != nil {
		t.Fatalf("Fehlerprotokoll: %v", err)
	}
	t.Cleanup(el.close)
	return el
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '�' {
			return false
		}
	}
	return true
}

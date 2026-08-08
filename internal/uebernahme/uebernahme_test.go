package uebernahme

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// TestIstZeilenfehler sichert die Weiche ab, an der sich entscheidet, ob „übersprungen"
// die Wahrheit sagt. Fällt ein Verbindungsabbruch versehentlich auf die
// Zeilenfehler-Seite, protokolliert das Werkzeug wieder Tausende Ausfälle, die es nie
// versucht hat.
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
		{"Datensatzfehler ohne Postgres", fmt.Errorf("%w: kaputtes JSONB", ErrZeile), true},
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
			if got := IstZeilenfehler(f.err); got != f.erwartet {
				t.Errorf("IstZeilenfehler(%v) = %v, erwartet %v", f.err, got, f.erwartet)
			}
		})
	}
}

// TestBeschreibeFehlerBehaeltUrsacheUndKontext: der Grund, warum diese Werkzeuge
// existieren, ist die Frage „was stimmt mit meiner Quelldatei nicht". Die Antwort muss den
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
	text := BeschreibeFehler(fmt.Errorf("Exemplar mit Barcode B-00042: %w", pgErr))

	for _, erwartet := range []string{"B-00042", "23505", "buecher_exemplare_barcode_id_key", "already exists"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("Fehlerbeschreibung nennt %q nicht: %s", erwartet, text)
		}
	}
}

func TestBeschreibeFehlerOhnePostgres(t *testing.T) {
	if got := BeschreibeFehler(errors.New("etwas ging schief")); got != "etwas ging schief" {
		t.Errorf("unerwartete Beschreibung: %q", got)
	}
}

// TestKuerzeZaehltZeichen: Postgres begrenzt varchar in ZEICHEN. Ein byteweiser Schnitt
// zerlegte deutsche Umlaute und schriebe ungültiges UTF-8 in die Datenbank.
func TestKuerzeZaehltZeichen(t *testing.T) {
	p := testProtokoll(t)

	lang := strings.Repeat("ä", 300)
	got := Kuerze(p, "1", "", "titel", lang, MaxFreitext)

	if r := []rune(got); len(r) != MaxFreitext {
		t.Fatalf("%d Zeichen erwartet, geliefert: %d", MaxFreitext, len(r))
	}
	if strings.ContainsRune(got, '�') {
		t.Error("die Kürzung hat ein Zeichen zerschnitten")
	}
	if p.Warnungen() != 1 {
		t.Errorf("die Kürzung muss protokolliert werden, gezählt: %d", p.Warnungen())
	}
}

func TestKuerzeLaesstPassendeWerteUnberuehrt(t *testing.T) {
	p := testProtokoll(t)

	grenzwert := strings.Repeat("ö", MaxFreitext)
	if got := Kuerze(p, "1", "", "titel", grenzwert, MaxFreitext); got != grenzwert {
		t.Error("ein Wert exakt auf Spaltenbreite darf nicht angetastet werden")
	}
	if p.Warnungen() != 0 {
		t.Errorf("keine Warnung erwartet, gezählt: %d", p.Warnungen())
	}
}

// TestKlaereISBNWertetAbStattZuVerwerfen hält die Grundhaltung fest: eine kaputte oder
// doppelte ISBN kostet die ISBN, nicht das Buch.
func TestKlaereISBNWertetAbStattZuVerwerfen(t *testing.T) {
	t.Run("ungültige Prüfziffer", func(t *testing.T) {
		p := testProtokoll(t)
		if got := KlaereISBN(p, "1", "9783161484101", map[string]string{}); got != "" {
			t.Errorf("leere ISBN erwartet, geliefert: %q", got)
		}
		if p.Warnungen() != 1 || p.FehlerAnzahl() != 0 {
			t.Errorf("WARNUNG erwartet, gezählt: %d/%d", p.Warnungen(), p.FehlerAnzahl())
		}
	})

	t.Run("Dublette im selben Lauf", func(t *testing.T) {
		p := testProtokoll(t)
		gesehen := map[string]string{"9783161484100": "7"}
		if got := KlaereISBN(p, "9", "978-3-16-148410-0", gesehen); got != "" {
			t.Errorf("leere ISBN erwartet, geliefert: %q", got)
		}
		if gesehen["9783161484100"] != "7" {
			t.Error("die bestehende Reservierung darf nicht überschrieben werden")
		}
	})

	t.Run("gültige ISBN wird reserviert", func(t *testing.T) {
		p := testProtokoll(t)
		gesehen := map[string]string{}
		if got := KlaereISBN(p, "3", "978-3-16-148410-0", gesehen); got != "9783161484100" {
			t.Errorf("normalisierte ISBN erwartet, geliefert: %q", got)
		}
		if gesehen["9783161484100"] != "3" {
			t.Error("die ISBN muss für den laufenden Import reserviert werden")
		}
		if p.Warnungen() != 0 {
			t.Errorf("keine Warnung erwartet, gezählt: %d", p.Warnungen())
		}
	})
}

// TestProtokollNenntDenQuellschluessel: ohne die Quell-ID in der Zeile ist das Protokoll
// wertlos — man findet den Datensatz in der Quelldatei nicht wieder.
func TestProtokollNenntDenQuellschluessel(t *testing.T) {
	pfad := t.TempDir() + "/err.log"
	p, err := NeuesProtokoll(pfad, "littera_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	p.Fehler("4711", "B-00042", "Barcode belegt")
	p.Warnung("4712", "", "Titel gekürzt")
	p.Schliessen()

	inhalt := leseDatei(t, pfad)
	for _, erwartet := range []string{"FEHLER", "littera_id=4711", "B-00042", "WARNUNG", "littera_id=4712"} {
		if !strings.Contains(inhalt, erwartet) {
			t.Errorf("Protokoll nennt %q nicht:\n%s", erwartet, inhalt)
		}
	}
	if p.Warnungen() != 1 || p.FehlerAnzahl() != 1 {
		t.Errorf("1/1 erwartet, gezählt: %d/%d", p.Warnungen(), p.FehlerAnzahl())
	}
}

func testProtokoll(t *testing.T) *Protokoll {
	t.Helper()
	p, err := NeuesProtokoll(t.TempDir()+"/err.log", "quell_id")
	if err != nil {
		t.Fatalf("Protokoll: %v", err)
	}
	t.Cleanup(p.Schliessen)
	return p
}

func leseDatei(t *testing.T, pfad string) string {
	t.Helper()
	b, err := os.ReadFile(pfad) // #nosec G304 - Pfad aus t.TempDir()
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	return string(b)
}

func TestNeuesProtokoll(t *testing.T) {
	t.Run("erfolgreich", func(t *testing.T) {
		pfad := t.TempDir() + "/test.log"
		p, err := NeuesProtokoll(pfad, "id")
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		if p == nil {
			t.Fatal("erwartete Protokoll-Instanz, bekam nil")
		}
		p.Schliessen()

		if _, err := os.Stat(pfad); err != nil {
			t.Errorf("Protokolldatei wurde nicht angelegt: %v", err)
		}
	})

	t.Run("verwirft existierende Inhalte", func(t *testing.T) {
		pfad := t.TempDir() + "/test.log"
		err := os.WriteFile(pfad, []byte("altes zeug"), 0644)
		if err != nil {
			t.Fatalf("konnte Testdatei nicht anlegen: %v", err)
		}

		p, err := NeuesProtokoll(pfad, "id")
		if err != nil {
			t.Fatalf("unerwarteter Fehler: %v", err)
		}
		p.Schliessen()

		inhalt, err := os.ReadFile(pfad)
		if err != nil {
			t.Fatalf("konnte Datei nicht lesen: %v", err)
		}
		if len(inhalt) > 0 {
			t.Errorf("erwartete leere Datei nach NeuesProtokoll, fand %q", inhalt)
		}
	})

	t.Run("Fehler bei ungueltigem Pfad", func(t *testing.T) {
		pfad := t.TempDir() + "/gibt-es-nicht/test.log"
		p, err := NeuesProtokoll(pfad, "id")
		if err == nil {
			t.Error("erwartete Fehler für ungültigen Pfad, bekam nil")
			p.Schliessen()
		}
		if err != nil && !strings.Contains(err.Error(), "konnte die Protokolldatei") {
			t.Errorf("Fehlermeldung sollte 'konnte die Protokolldatei' enthalten, war: %v", err)
		}
	})
}

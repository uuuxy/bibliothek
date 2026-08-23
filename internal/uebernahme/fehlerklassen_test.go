package uebernahme

import (
	"context"
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
		{"Code zu kurz", pgFehler("1"), false},
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
		ColumnName:     "barcode_id",
		Detail:         "Key (barcode_id)=(B-00042) already exists.",
	}
	text := BeschreibeFehler(fmt.Errorf("Exemplar mit Barcode B-00042: %w", pgErr))

	for _, erwartet := range []string{"B-00042", "23505", "buecher_exemplare_barcode_id_key", "barcode_id", "already exists"} {
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

// TestBeschreibeFehlerTraegtKeineDatenwerte ist ein Datenschutz-Gate, kein Formattest.
//
// Das Ziel dieser Zeilen ist `littera_import.log`: eine unverschlüsselte Datei im
// Arbeitsverzeichnis, ohne Frist und ohne Löschregel. Postgres schreibt in DETAIL den
// Inhalt der kollidierenden Zeile — bei der Leser-Übernahme also echte E-Mail-Adressen
// und Ausweisnummern. Am laufenden Postgres nachgestellt (23.08.2026):
//
//	DETAIL: Key (email)=(erika.mustermann@philipp-reis-schule.de) already exists.
//
// Spalte und Constraint müssen bleiben — ohne sie ist die Zeile für die Reparatur der
// Quelldatei wertlos.
func TestBeschreibeFehlerTraegtKeineDatenwerte(t *testing.T) {
	adresse := "erika.mustermann@philipp-reis-schule.de"
	pgErr := &pgconn.PgError{
		Code:           "23505",
		Message:        "duplicate key value violates unique constraint",
		ConstraintName: "benutzer_email_key",
		ColumnName:     "email",
		Detail:         "Key (email)=(" + adresse + ") already exists.",
	}

	text := BeschreibeFehler(fmt.Errorf("Leser 4711: %w", pgErr))

	if strings.Contains(text, adresse) {
		t.Errorf("die echte Adresse steht in der Protokollzeile: %s", text)
	}
	if strings.Contains(text, "mustermann") || strings.Contains(text, "philipp-reis-schule") {
		t.Errorf("ein Bestandteil der Adresse steht in der Protokollzeile: %s", text)
	}
	// Diagnosewert muss erhalten bleiben, sonst ist die Schwärzung ein Rückschritt.
	for _, erwartet := range []string{"23505", "benutzer_email_key", "email", "already exists", "Leser 4711"} {
		if !strings.Contains(text, erwartet) {
			t.Errorf("Fehlerbeschreibung nennt %q nicht mehr: %s", erwartet, text)
		}
	}
}

// TestOhneDatenwerteFormen hält die Formen fest, in denen Postgres DETAIL liefert.
func TestOhneDatenwerteFormen(t *testing.T) {
	faelle := []struct{ detail, verboten string }{
		{"Key (email)=(a@b.de) already exists.", "a@b.de"},
		{"Key (barcode_id)=(B-00042) already exists.", "B-00042"},
		{"Key (titel_id)=(9f1c) is not present in table \"buecher_titel\".", "9f1c"},
		{"Key (vorname, nachname)=(Erika, Mustermann) already exists.", "Mustermann"},
	}
	for _, f := range faelle {
		got := ohneDatenwerte(f.detail)
		if strings.Contains(got, f.verboten) {
			t.Errorf("Wert %q überlebt die Schwärzung: %s", f.verboten, got)
		}
		if !strings.Contains(got, ")=(…)") {
			t.Errorf("Form ging verloren, erwartet )=(…): %s", got)
		}
	}
}

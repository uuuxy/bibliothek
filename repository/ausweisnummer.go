package repository

import (
	"context"
	"fmt"
)

// AusweisnummernAbfragen vergibt die laufenden Ausweisnummern („A-10001").
type AusweisnummernAbfragen struct {
	db Datenbank
}

// NewAusweisnummernAbfragen liefert die Vergabe der Ausweisnummern an der übergebenen
// Datenbank.
func NewAusweisnummernAbfragen(db Datenbank) *AusweisnummernAbfragen {
	return &AusweisnummernAbfragen{db: db}
}

// AusweisPraefix steht auf jeder Ausweisnummer, die das Programm vergibt, für Schüler wie für
// das Kollegium: Wer jemand ist, steht in den Stammdaten, nicht auf dem Ausweis. Ohne Netz ist
// die Vorsilbe das Einzige, woran die Theke einen Ausweis von einem Buch unterscheidet
// (frontend/src/lib/stores/omnibox.svelte.js). Die früheren Vorsilben „S-" und „L-" versteht
// der Scanner weiter (internal/service/omnibox.go); vergeben werden sie nicht mehr.
const AusweisPraefix = "A-"

// AusweisNummer setzt eine laufende Zahl in die Form, die auf den Ausweis gedruckt wird:
// fünfstellig mit führenden Nullen, damit die Nummern gleich lang bleiben und sich nach
// Zeichen wie nach Wert gleich sortieren.
func AusweisNummer(n int) string { return fmt.Sprintf("%s%05d", AusweisPraefix, n) }

// ZieheNaechste liefert die nächste freie laufende Ausweisnummer, ohne Vorsilbe; die gedruckte Form
// setzt AusweisNummer. Der Zähler steht in der Datenbank (ausweis_nummer_start), weil auch die
// Trigger aktives_konto_hat_ausweis und aktives_konto_behaelt_ausweis ihn brauchen: Ein
// Nummernkreis hat einen Zähler.
//
// Gezählt wird nach dem Wert der Zahl und nicht nach Zeichen (A-100000 liegt über A-99999),
// ohne Bestand beginnt die Reihe bei 10001, und Nummern mit mehr als 15 Ziffern (ein
// verrutschter Scan) werden übergangen. Die Sperre hält bis zum Ende der Transaktion; wer
// mehrere Nummern braucht, etwa der Lauf der LUSD, zieht einmal und zählt in derselben
// Transaktion selbst weiter.
//
// Die Nummern der Exemplare kommen nicht von hier, sondern aus barcode_seq
// (repository/barcode_vergabe.go).
func (a *AusweisnummernAbfragen) ZieheNaechste(ctx context.Context) (int, error) {
	var n int64
	if err := a.db.QueryRow(ctx, `SELECT ausweis_nummer_start()`).Scan(&n); err != nil {
		return 0, fmt.Errorf("nächste ausweisnummer: %w", err)
	}
	return int(n), nil
}

package betrag

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// Cent ist ein Geldbetrag in ganzen Cent. Gerechnet wird in Cent, weil float64 einen Betrag wie
// 0,10 nicht genau kennt: Eine Summe oder ein Anteil, der auf einen halben Cent fällt, kippt sonst
// beim Runden. Die Datenbank führt Beträge als NUMERIC(10,2), die Oberfläche als Zahl in Euro;
// Cent liest und schreibt beide Formen genau, ohne Umweg über float64.
type Cent int64

// Text ist der Betrag ohne Währung: „12,50".
func (c Cent) Text() string {
	return c.mitTrenner(",")
}

// Euro ist der Betrag mit Euro-Zeichen: „12,50 €".
func (c Cent) Euro() string {
	return c.Text() + " €"
}

// mitTrenner schreibt den Betrag in Euro mit zwei Nachkommastellen und dem Trenner.
func (c Cent) mitTrenner(trenner string) string {
	vorzeichen, wert := "", int64(c)
	if wert < 0 {
		vorzeichen, wert = "-", -wert
	}
	return fmt.Sprintf("%s%d%s%02d", vorzeichen, wert/100, trenner, wert%100)
}

// leseEuro liest einen Betrag in Euro aus Text („12.5", „12,50", „12"). Mehr als zwei
// Nachkommastellen rundet es kaufmännisch auf den Cent, wie Postgres beim Schreiben in
// NUMERIC(10,2): Ein Betrag, den die Datenbank bisher gerundet übernahm, geht nicht verloren.
func leseEuro(text string) (Cent, error) {
	zahl, ok := new(big.Rat).SetString(strings.Replace(strings.TrimSpace(text), ",", ".", 1))
	if !ok {
		return 0, fmt.Errorf("betrag: %q ist keine Zahl", text)
	}
	negativ := zahl.Sign() < 0
	// In Cent und um einen halben Cent vom Nullpunkt weg, damit das Abschneiden rundet.
	zahl.Abs(zahl).Mul(zahl, big.NewRat(100, 1)).Add(zahl, big.NewRat(1, 2))
	cent := new(big.Int).Quo(zahl.Num(), zahl.Denom())
	if !cent.IsInt64() {
		return 0, fmt.Errorf("betrag: %q ist zu groß", text)
	}
	if negativ {
		return Cent(-cent.Int64()), nil
	}
	return Cent(cent.Int64()), nil
}

// MarshalJSON schreibt den Betrag als Zahl in Euro mit zwei Nachkommastellen: 1250 → 12.50.
func (c Cent) MarshalJSON() ([]byte, error) {
	return []byte(c.mitTrenner(".")), nil
}

// UnmarshalJSON liest eine Zahl in Euro wie leseEuro. null lässt den Wert stehen, wie
// encoding/json es für einen Unmarshaler vorsieht.
func (c *Cent) UnmarshalJSON(roh []byte) error {
	if string(roh) == "null" {
		return nil
	}
	wert, err := leseEuro(string(roh))
	if err != nil {
		return err
	}
	*c = wert
	return nil
}

// Scan liest eine Spalte NUMERIC, die pgx als Text hereinreicht. Eine Zahl als float64 lehnt es
// ab: Sie hieße, dass die Abfrage den Betrag mit ::float8 umwandelt.
func (c *Cent) Scan(quelle any) error {
	var text string
	switch q := quelle.(type) {
	case string:
		text = q
	case []byte:
		text = string(q)
	case int64:
		*c = Cent(q * 100)
		return nil
	case nil:
		return errors.New("betrag: NULL statt eines Betrags")
	default:
		return fmt.Errorf("betrag: %T lässt sich nicht als Betrag lesen", quelle)
	}
	wert, err := leseEuro(text)
	if err != nil {
		return err
	}
	*c = wert
	return nil
}

// Value schreibt den Betrag als Text in Euro, den Postgres in NUMERIC genau übernimmt.
func (c Cent) Value() (driver.Value, error) {
	return c.mitTrenner("."), nil
}

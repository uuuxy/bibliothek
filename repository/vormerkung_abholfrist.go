package repository

import (
	"context"
	"time"

	"bibliothek/pkg/lmfplan"
)

// abholfristTage ist die Zahl der Tage, die ein vorgemerktes Buch bereitliegt, gerechnet ab der
// Zuteilung: bei der Rückgabe oder beim Nachrücken (bedieneNaechstenWartenden). Drei Tage,
// keine Einstellung. Das Ende fällt wie bei der Leihfrist auf den nächsten Schultag, damit ein
// Buch, das kurz vor den Ferien zurückkommt, nicht in den Ferien verfällt.
const abholfristTage = 3

// BestimmeAbholfrist liefert das Ende der Abholfrist für eine Zuteilung um jetzt: die
// Tagesfrist (lmfplan.Ferientabelle.Tagesfrist) mit den Sommerferien aus den Einstellungen,
// dieselbe Rechnung wie bei der Leihfrist. db ist der Pool oder die Transaktion der Zuteilung.
func BestimmeAbholfrist(ctx context.Context, db SpurenDatenbank, jetzt time.Time) (time.Time, error) {
	einstellungen, err := LadeEinstellungen(ctx, db)
	if err != nil {
		return time.Time{}, err
	}
	return lmfplan.FerientabelleAus(einstellungen.Sommerferien).Tagesfrist(jetzt, abholfristTage), nil
}

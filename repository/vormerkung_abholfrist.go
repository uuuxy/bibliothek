package repository

import (
	"context"
	"time"

	"bibliothek/pkg/lmfplan"
)

// AbholfristTage ist die Zahl der Tage, die ein vorgemerktes Buch bereitliegt, gerechnet ab
// dem Zeitpunkt, zu dem es zugeteilt wird — bei der Rückgabe (service,
// processReturnVormerkungTx) oder beim Nachrücken (bedieneNaechstenWartenden). Entschieden
// am 28.09.2026: drei Tage, das Ende fällt wie bei der Leihfrist auf den nächsten Schultag;
// keine Einstellung. Bis zum 29.09.2026 stand an beiden Stellen CURRENT_TIMESTAMP + INTERVAL
// '3 days':
// Wochenende und Ferien zählten mit, und ein Buch, das in den letzten drei Tagen vor den
// Herbstferien zurückkam, verfiel in den Ferien.
const AbholfristTage = 3

// Abholfrist ist das Ende der Abholfrist für eine Zuteilung um jetzt: die Tagesfrist
// (lmfplan.Ferientabelle.Tagesfrist) mit den Sommerferien aus den Einstellungen, dieselbe
// Rechnung wie bei der Leihfrist. ex ist der Executor der Zuteilung (Transaktion oder Pool).
func Abholfrist(ctx context.Context, ex SpurenExecutor, jetzt time.Time) (time.Time, error) {
	einstellungen, err := EinstellungenUeber(ctx, ex)
	if err != nil {
		return time.Time{}, err
	}
	return lmfplan.FerientabelleAus(einstellungen.Sommerferien).Tagesfrist(jetzt, AbholfristTage), nil
}

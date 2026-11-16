package leser

import (
	"context"

	"bibliothek/internal/lusd"
)

// Testsichten auf den LUSD-Import: der Lauf über den Pool des Servers, mit der Karenzzeit
// aus den Einstellungen, wie die Tür ihn fährt.

// berechneLUSDLauf fährt den Lauf mit den genannten Vorgaben.
func (t *Tueren) berechneLUSDLauf(ctx context.Context, datei lusd.Datei, lauf lusd.Lauf) (*lusd.Vorschau, error) {
	lauf.KarenzTage = t.abgaengerKarenzTage(ctx)
	return lusd.FuehreAus(ctx, t.DB.Pool, datei, lauf)
}

// berechneLUSD ist der Lauf ohne Umbenennungs-Wahl.
func (t *Tueren) berechneLUSD(ctx context.Context, datei lusd.Datei, apply bool, allowMassGraduation bool) (*lusd.Vorschau, error) {
	return t.berechneLUSDLauf(ctx, datei, lusd.Lauf{Anwenden: apply, MassenabgangBestaetigt: allowMassGraduation})
}

// berechneLUSDAenderungen ist berechneLUSD für Zeilen im Abgleich über die LUSD-ID.
func (t *Tueren) berechneLUSDAenderungen(ctx context.Context, records []lusd.Zeile, apply bool, allowMassGraduation bool) (*lusd.Vorschau, error) {
	return t.berechneLUSD(ctx, lusd.Datei{Zeilen: records, Modus: lusd.ModusID}, apply, allowMassGraduation)
}

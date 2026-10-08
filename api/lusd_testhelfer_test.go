package api

import (
	"context"

	"bibliothek/internal/lusd"
)

// Testsichten auf den LUSD-Import: der Lauf über den Pool des Servers, mit der Karenzzeit
// aus den Einstellungen, wie die Tür ihn fährt.

// computeLusdLauf fährt den Lauf mit den genannten Vorgaben.
func (s *Server) computeLusdLauf(ctx context.Context, datei lusd.Datei, lauf lusd.Lauf) (*lusd.PreviewResult, error) {
	lauf.KarenzTage = s.abgaengerKarenzTage(ctx)
	return lusd.Fuehre(ctx, s.DB.Pool, datei, lauf)
}

// computeLusd ist der Lauf ohne Umbenennungs-Wahl.
func (s *Server) computeLusd(ctx context.Context, datei lusd.Datei, apply bool, allowMassGraduation bool) (*lusd.PreviewResult, error) {
	return s.computeLusdLauf(ctx, datei, lusd.Lauf{Anwenden: apply, MassenabgangBestaetigt: allowMassGraduation})
}

// computeLusdChanges ist computeLusd für Zeilen im Abgleich über die LUSD-ID.
func (s *Server) computeLusdChanges(ctx context.Context, records []lusd.Zeile, apply bool, allowMassGraduation bool) (*lusd.PreviewResult, error) {
	return s.computeLusd(ctx, lusd.Datei{Zeilen: records, Modus: lusd.ModusID}, apply, allowMassGraduation)
}

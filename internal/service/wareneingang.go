package service

import (
	"context"
	"errors"

	"bibliothek/internal/db"
	"bibliothek/internal/repository"
)

// EingebuchtesExemplar ist ein Exemplar, das der Wareneingang eingebucht hat. EtikettGedruckt
// sagt der Oberfläche, ob sie den Druck des Etiketts anbietet.
type EingebuchtesExemplar struct {
	BarcodeID       string `json:"barcode_id"`
	Titel           string `json:"titel"`
	Autor           string `json:"autor"`
	EtikettGedruckt bool   `json:"etikett_gedruckt"`
	// Abholfach nennt den Schüler, für den das Exemplar ins Abholfach gehört; leer heißt ins Regal.
	Abholfach string `json:"abholfach,omitempty"`
}

// ErrNichtsEinzubuchen meldet, dass keines der genannten Exemplare im Zulauf stand: Ein
// anderer Platz war schneller, oder die Liste ist veraltet. Die Tür antwortet mit 404 und
// erkennt den Fall am Fehlerwert, nicht am Wortlaut der Meldung.
var ErrNichtsEinzubuchen = errors.New("keine zu aktualisierenden Exemplare gefunden (bereits freigegeben?)")

// WareneingangAngaben sind die Angaben eines Wareneingangs. parameter_strukturen_test.go
// verlangt, dass jedes Literal jedes Feld setzt: Ohne Bearbeiter oder IP-Adresse stünde der
// Eintrag im Protokoll ohne sie.
type WareneingangAngaben struct {
	ExemplarIDs  []string
	BearbeiterID string
	IPAdresse    string
}

// BucheWareneingang bucht die genannten Exemplare aus dem Zulauf ein und schreibt einen Eintrag
// ins Protokoll. Scheitert nur der Eintrag, bleibt die Buchung, und der Fehler steht im Log
// (meldeProtokollFehler).
func BucheWareneingang(ctx context.Context, pool db.Pool, protokollAbfragen *repository.ProtokollAbfragen, angaben WareneingangAngaben) ([]EingebuchtesExemplar, error) {
	// Eine Transaktion: Das Exemplar kommt nur zusammen mit der Vormerkung, die es bedient, in
	// den Bestand.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Verwerfe(ctx, tx)
	eingebucht, err := repository.BucheZulaufEin(ctx, tx, angaben.ExemplarIDs)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if len(eingebucht) == 0 {
		return nil, ErrNichtsEinzubuchen
	}
	exemplare := make([]EingebuchtesExemplar, 0, len(eingebucht))
	for _, e := range eingebucht {
		exemplare = append(exemplare, EingebuchtesExemplar(e))
	}

	meldeProtokollFehler("wareneingang-bulk", protokollAbfragen.ProtokolliereAdminAktion(ctx, angaben.BearbeiterID, "BULK_RECEIVE_ITEMS", angaben.IPAdresse, map[string]any{
		"received_count": len(exemplare),
		"message":        "Wareneingang als Massen-Freigabe gebucht",
	}))

	return exemplare, nil
}

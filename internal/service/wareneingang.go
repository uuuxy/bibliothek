package service

import (
	"context"
	"errors"

	"bibliothek/db"
	"bibliothek/repository"
)

// EingebuchtesExemplar ist ein Exemplar, das der Wareneingang eingebucht hat. EtikettGedruckt
// sagt der Oberfläche, ob sie den Druck des Etiketts anbietet.
type EingebuchtesExemplar struct {
	BarcodeID       string `json:"barcode_id"`
	Titel           string `json:"titel"`
	Autor           string `json:"autor"`
	EtikettGedruckt bool   `json:"etikett_gedruckt"`
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
func BucheWareneingang(ctx context.Context, pool db.Pool, protokollAbfragen repository.ProtokollAbfragen, angaben WareneingangAngaben) ([]EingebuchtesExemplar, error) {
	eingebucht, err := repository.BucheZulaufEin(ctx, pool, angaben.ExemplarIDs)
	if err != nil {
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
		"message":        "Wareneingang gebucht (Massen-Freigabe)",
	}))

	return exemplare, nil
}

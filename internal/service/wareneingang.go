package service

import (
	"context"
	"errors"

	"bibliothek/db"
	"bibliothek/repository"
)

// EingebuchtesExemplar describes a received exemplar, including whether its
// barcode label still needs printing (drives the print suggestion in the UI).
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

// WareneingangAngaben encapsulates the parameters needed to bulk receive orders.
type WareneingangAngaben struct {
	ExemplarIDs  []string
	BearbeiterID string
	IPAdresse    string
}

// BucheWareneingang marks all pre-allocated items as received.
func BucheWareneingang(ctx context.Context, pool db.PgxPoolIface, protokollAbfragen repository.ProtokollAbfragen, angaben WareneingangAngaben) ([]EingebuchtesExemplar, error) {
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

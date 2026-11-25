package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Die Aktionen der Logbuchzeile zu einem gemeldeten Gerät.
const (
	protokollAktionGeraetDefekt   = "defekt"
	protokollAktionGeraetVerloren = "verloren"
)

// meldeGeraeteSchaden bucht Verlust oder Schaden an einer Geräteausleihe. Die Ausleihe endet, und
// das Gerät ist nicht mehr ausleihbar, bleibt aber in der Geräteverwaltung stehen: beschädigt als
// defekt, bis es repariert wieder freigegeben wird, verloren als verloren (ist_ausgesondert). Die
// Forderung entsteht wie beim Buch, beim Kollegium nicht (ohneForderung); auf einen Bescheid
// kommt ein Gerät nicht (bucheVerluste).
func meldeGeraeteSchaden(ctx context.Context, tx pgx.Tx, meldung schadensMeldung, a schadensAusleihe, grund string) (string, error) {
	if meldung.bearbeiterID == "" {
		return "", ErrAussonderungOhneBearbeiter
	}
	verloren := meldung.art == SchadensArtNichtZurueck
	tag, err := tx.Exec(ctx, `
		UPDATE geraete
		SET ist_ausleihbar = false, ist_ausgesondert = $3, zustand_notiz = $1,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $2`, meldung.beschreibung, a.geraetID, verloren)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() != 1 {
		return "", fmt.Errorf("gerät %s stilllegen: %d zeilen, erwartet 1", a.geraetID, tag.RowsAffected())
	}
	var schadensID string
	if !a.ohneForderung {
		if err := tx.QueryRow(ctx, `
			INSERT INTO schadensfaelle (geraet_id, ausleihe_id, schueler_id, beschreibung, betrag, art)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, a.geraetID, meldung.ausleiheID, a.schuelerID, meldung.beschreibung, meldung.betrag, string(meldung.art)).Scan(&schadensID); err != nil {
			return "", err
		}
	}
	aktion := protokollAktionGeraetDefekt
	if verloren {
		aktion = protokollAktionGeraetVerloren
	}
	if err := protokolliereGeraeteMeldung(ctx, tx, a.geraetID, meldung, aktion, grund, schadensID); err != nil {
		return "", err
	}
	if err := beendeGemeldeteAusleihe(ctx, tx, meldung); err != nil {
		return "", err
	}
	return schadensID, nil
}

// protokolliereGeraeteMeldung schreibt die Meldung an einem Gerät ins Logbuch, wie
// protokolliereAussonderung die an einem Exemplar: Wer sie gebucht hat, steht nur hier.
func protokolliereGeraeteMeldung(ctx context.Context, tx pgx.Tx, geraetID string, meldung schadensMeldung, aktion, grund, schadensID string) error {
	zusatz := map[string]any{}
	if schadensID != "" {
		zusatz["schadensfall_id"] = schadensID
	}
	zusatzJSON, err := json.Marshal(zusatz)
	if err != nil {
		return fmt.Errorf("meldung am gerät protokollieren: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, kontext, details)
		VALUES ('geraete', 'UPDATE', $1, $2::uuid, 'USER', $3,
		        jsonb_build_object('action', $4::text, 'grund', $5::text) || $6::jsonb)`,
		geraetID, meldung.bearbeiterID, meldung.weg, aktion, grund, string(zusatzJSON))
	if err != nil {
		return fmt.Errorf("meldung am gerät protokollieren: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("meldung am gerät protokollieren: %d einträge, erwartet 1", tag.RowsAffected())
	}
	return nil
}

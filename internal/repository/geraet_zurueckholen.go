package repository

import (
	"context"
	"fmt"

	"bibliothek/internal/db"
	"bibliothek/internal/pkg/betrag"
	"bibliothek/internal/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// protokollAktionGeraetZurueck ist die Aktion der Logbuchzeile, wenn ein verlorenes Gerät zurück ist.
const protokollAktionGeraetZurueck = "zurueck"

// HoleVerlorenesGeraetZurueck bringt ein als verloren gemeldetes Gerät (ist_ausgesondert,
// meldeGeraeteSchaden) zurück in den Schrank: wieder ausleihbar, ohne die Notiz der Meldung, und
// die offene Forderung „nicht zurückgegeben“ ist storniert wie beim Buch (verbucheRueckkehr). Auf
// einem Bescheid steht ein Gerät nie (bucheVerluste). Wer es zurückgeholt hat, steht im Logbuch.
// zurueck ist false, wenn das Gerät nicht verloren ist; dann ist nichts geschrieben. Ohne
// Bearbeiter holt niemand ein verlorenes Gerät zurück (ErrAussonderungOhneBearbeiter): Storno und
// Logbuch nennen ihn.
func HoleVerlorenesGeraetZurueck(ctx context.Context, pool db.Pool, geraetID, bearbeiterID string) (zurueck bool, befund RueckkehrBefund, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, befund, fmt.Errorf("gerät zurückholen: %w", err)
	}
	defer db.Verwerfe(ctx, tx)
	tag, err := tx.Exec(ctx, `
		UPDATE geraete
		SET ist_ausgesondert = false, ist_ausleihbar = true, zustand_notiz = NULL,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $1 AND ist_ausgesondert = true`, geraetID)
	if err != nil {
		return false, befund, fmt.Errorf("gerät zurückholen: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, befund, nil
	}
	if bearbeiterID == "" {
		return false, befund, ErrAussonderungOhneBearbeiter
	}
	type forderung struct {
		id    string
		hoehe betrag.Cent
	}
	// FOR UPDATE: Holten zwei Plätze dasselbe Gerät zugleich zurück, stornierte sonst jeder.
	rows, err := tx.Query(ctx, `
		SELECT id, betrag FROM schadensfaelle
		WHERE geraet_id = $1 AND art = 'nicht_zurueckgegeben'
		  AND ist_bezahlt = false AND storniert_am IS NULL
		FOR UPDATE`, geraetID)
	if err != nil {
		return false, befund, fmt.Errorf("offene Forderungen des Geräts lesen: %w", err)
	}
	var offene []forderung
	for rows.Next() {
		var f forderung
		if err := rows.Scan(&f.id, &f.hoehe); err != nil {
			rows.Close()
			return false, befund, fmt.Errorf("offene Forderung des Geräts lesen: %w", err)
		}
		offene = append(offene, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, befund, fmt.Errorf("offene Forderungen des Geräts lesen: %w", err)
	}
	grund := "Rückgabe am " + schulzeit.Jetzt().Format(datumsformat)
	for _, f := range offene {
		if err := storniereWeilZurueck(ctx, tx, f.id, f.hoehe, bearbeiterID, grund, "das Gerät"); err != nil {
			return false, befund, err
		}
		befund.StornierteForderungen++
		befund.StornierterBetrag += f.hoehe
	}
	if err := protokolliereGeraeteRueckkehr(ctx, tx, geraetID, bearbeiterID, befund.StornierteForderungen); err != nil {
		return false, befund, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, befund, fmt.Errorf("gerät zurückholen: %w", err)
	}
	return true, befund, nil
}

// protokolliereGeraeteRueckkehr schreibt die Rückkehr ins Logbuch, wie protokolliereGeraeteMeldung
// die Meldung: Wer das Gerät zurückgeholt hat, steht nur hier.
func protokolliereGeraeteRueckkehr(ctx context.Context, tx pgx.Tx, geraetID, bearbeiterID string, storniert int) error {
	tag, err := tx.Exec(ctx, `
		INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur, kontext, details)
		VALUES ('geraete', 'UPDATE', $1, $2::uuid, 'USER', 'Geräteverwaltung',
		        jsonb_build_object('action', $3::text, 'stornierte_forderungen', $4::int))`,
		geraetID, bearbeiterID, protokollAktionGeraetZurueck, storniert)
	if err != nil {
		return fmt.Errorf("rückkehr des geräts protokollieren: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("rückkehr des geräts protokollieren: %d einträge, erwartet 1", tag.RowsAffected())
	}
	return nil
}

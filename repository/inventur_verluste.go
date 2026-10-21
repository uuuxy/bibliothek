package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// InventurVerlust ist eine Zeile des Fehlbestandsberichts.
type InventurVerlust struct {
	// ExemplarID ist leer, wenn das Exemplar endgültig gelöscht wurde (ON DELETE SET NULL); die
	// Textspalten bleiben lesbar. „Gefunden" und „endgültig löschen" sind dann nicht mehr
	// möglich.
	ExemplarID string `json:"exemplar_id,omitempty"`
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	Signatur   string `json:"signatur"`
	GebuchtAm  string `json:"gebucht_am"`
	// GefundenAm ist gesetzt, sobald das Exemplar beim Nachsuchen gefunden und über „Gefunden"
	// zurück in den Umlauf gebracht wurde.
	GefundenAm string `json:"gefunden_am,omitempty"`
}

// LadeVerluste liefert den Fehlbestand eines abgeschlossenen Laufs, sortiert nach Signatur und
// Titel: in der Reihenfolge, in der man mit der Liste durchs Regal geht.
func (a *InventurAbfragen) LadeVerluste(ctx context.Context, laufID string) ([]InventurVerlust, error) {
	rows, err := a.db.Query(ctx, `
		SELECT coalesce(exemplar_id::text, ''), barcode_id, titel, autor, signatur,
		       to_char(gebucht_am, 'YYYY-MM-DD"T"HH24:MI:SSOF'),
		       coalesce(to_char(gefunden_am, 'YYYY-MM-DD"T"HH24:MI:SSOF'), '')
		FROM inventur_verluste
		WHERE session_id = $1
		ORDER BY signatur COLLATE deutsch, titel COLLATE deutsch, barcode_id
	`, laufID)
	if err != nil {
		return nil, fmt.Errorf("fehlbestand laden fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	// Nie nil: Eine leere Liste kommt bei der Oberfläche als [] an.
	liste := make([]InventurVerlust, 0)
	for rows.Next() {
		var v InventurVerlust
		if err := rows.Scan(&v.ExemplarID, &v.BarcodeID, &v.Titel, &v.Autor, &v.Signatur, &v.GebuchtAm, &v.GefundenAm); err != nil {
			return nil, fmt.Errorf("fehlbestand lesen fehlgeschlagen: %w", err)
		}
		liste = append(liste, v)
	}
	return liste, rows.Err()
}

// MarkiereVerlustAlsGefunden bringt ein als Verlust gebuchtes Exemplar zurück in den Umlauf und
// trägt den Fund im Fehlbestandsbericht ein. false ohne Fehler heißt: kein offener Verlust mit
// dieser ID (falsche ID oder schon endgültig gelöscht).
//
// Mit dem Fund endet auch die Forderung „nicht zurückgegeben", die das Buch abgerechnet hat
// (verbucheRueckkehr, dieselbe Regel wie an der Theke); sonst stünde das Buch wieder im Regal,
// und das Kind bliebe gesperrt. Der Befund sagt, was storniert wurde und ob ein Bescheid schon
// bei der Schulaufsicht liegt. Die Transaktion gehört dem Rufer, damit Umlauf, Bericht und
// Eintrag zusammen gelten.
func (a *InventurAbfragen) MarkiereVerlustAlsGefunden(ctx context.Context, exemplarID, bearbeiterID string) (bool, RueckkehrBefund, error) {
	var barcode, titel string
	err := a.db.QueryRow(ctx,
		`SELECT e.barcode_id, t.titel FROM buecher_exemplare e
		 JOIN buecher_titel t ON t.id = e.titel_id
		 WHERE e.id = $1 AND e.ist_ausgesondert = true AND e.aussonderung_grund = 'VERLUST'`,
		exemplarID,
	).Scan(&barcode, &titel)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, RueckkehrBefund{}, nil
	}
	if err != nil {
		return false, RueckkehrBefund{}, fmt.Errorf("exemplar für Fund-Meldung lesen fehlgeschlagen: %w", err)
	}

	// Derselbe Weg wie an der Theke (HoleExemplarZurueck): Umlauf und Forderung in der
	// Transaktion der Tür.
	befund, err := HoleExemplarZurueck(ctx, a.db, exemplarID, bearbeiterID, nil)
	if err != nil {
		return false, RueckkehrBefund{}, err
	}

	// Alle offenen Verlust-Zeilen des Exemplars schließen, nicht nur die eines Laufs: Ein
	// Exemplar kann mehrfach als Verlust gebucht sein, wenn ein früherer Fund nie eingetragen
	// wurde.
	if _, err := a.db.Exec(ctx, `
		UPDATE inventur_verluste SET gefunden_am = CURRENT_TIMESTAMP
		WHERE exemplar_id = $1 AND gefunden_am IS NULL
	`, exemplarID); err != nil {
		return false, RueckkehrBefund{}, fmt.Errorf("verlustbericht aktualisieren fehlgeschlagen: %w", err)
	}

	if err := schreibeProtokollEintrag(ctx, a.db, protokollEintrag{
		Tabelle: "buecher_exemplare", Aktion: "UPDATE", DatensatzID: exemplarID,
		BearbeiterID: &bearbeiterID, Akteur: "USER",
		Kontext: zeigerAuf("Als Verlust gebuchtes Exemplar beim Nachsuchen gefunden"),
		Details: map[string]any{"barcode_id": barcode, "titel": titel, "action": "verlust_gefunden"},
	}); err != nil {
		return false, RueckkehrBefund{}, err
	}
	return true, befund, nil
}

// LoescheVerlusteEndgueltig entfernt als Verlust gebuchte Exemplare aus dem Katalog. Der
// Fehlbestandsbericht bleibt lesbar (ON DELETE SET NULL, die Textspalten sind eine Abschrift).
// Die Transaktion gehört dem Rufer.
//
// Nur für ausgesonderte Exemplare mit dem Grund VERLUST: Das ist nicht der allgemeine Weg, ein
// Exemplar aus dem Bestand zu nehmen (dafür SondereExemplarAus), sondern räumt auf, was der
// Bericht schon als verloren führt. Geliefert werden die IDs der gelöschten Exemplare; was
// schon weg war oder nicht VERLUST ist, bleibt aus.
func (a *InventurAbfragen) LoescheVerlusteEndgueltig(ctx context.Context, exemplarIDs []string, bearbeiterID string) ([]string, error) {
	if len(exemplarIDs) == 0 {
		return nil, nil
	}

	rows, err := a.db.Query(ctx, `
		SELECT e.id, e.barcode_id, t.titel, `+SQLWarImBestand+`
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.id = ANY($1) AND e.ist_ausgesondert = true AND e.aussonderung_grund = 'VERLUST'
	`, exemplarIDs)
	if err != nil {
		return nil, fmt.Errorf("zu löschende Verlust-Exemplare lesen fehlgeschlagen: %w", err)
	}
	type snapshot struct {
		id, barcode, titel string
		warImBestand       bool
	}
	var treffer []snapshot
	for rows.Next() {
		var s snapshot
		if err := rows.Scan(&s.id, &s.barcode, &s.titel, &s.warImBestand); err != nil {
			rows.Close()
			return nil, fmt.Errorf("verlust-exemplar lesen fehlgeschlagen: %w", err)
		}
		treffer = append(treffer, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(treffer) == 0 {
		return nil, nil
	}

	ids := make([]string, len(treffer))
	barcodeVon := make(map[string]string, len(treffer))
	for i, s := range treffer {
		ids[i] = s.id
		barcodeVon[s.id] = s.barcode
	}

	if err := a.pruefeVerlusteLoeschbar(ctx, ids, barcodeVon); err != nil {
		return nil, err
	}

	// Die abhängigen Zeilen zuerst: ausleihen.exemplar_id und schadensfaelle.exemplar_id stehen
	// auf ON DELETE RESTRICT. Reihenfolge wie beim Löschen eines Titels
	// (inventur/titel_loeschen.go): Schadensfälle, Ausleihen, dann das Exemplar. Was hier
	// fällt, ist erledigt; offene Vorgänge hat pruefeVerlusteLoeschbar abgewiesen. Der
	// Fehlbestandsbericht behält seine Abschrift, und der Barcode wird wieder frei:
	// buecher_exemplare.barcode_id ist ohne Teilindex eindeutig, eine stehengebliebene Zeile
	// hielte den Aufkleber für immer besetzt.
	if _, err := a.db.Exec(ctx,
		`DELETE FROM schadensfaelle WHERE exemplar_id = ANY($1)`, ids,
	); err != nil {
		return nil, fmt.Errorf("erledigte schadensfälle der verlust-exemplare löschen fehlgeschlagen: %w", err)
	}
	if _, err := a.db.Exec(ctx,
		`DELETE FROM ausleihen WHERE exemplar_id = ANY($1) AND rueckgabe_am IS NOT NULL`, ids,
	); err != nil {
		return nil, fmt.Errorf("ausleihhistorie der verlust-exemplare löschen fehlgeschlagen: %w", err)
	}
	if _, err := a.db.Exec(ctx,
		`DELETE FROM buecher_exemplare WHERE id = ANY($1)`, ids,
	); err != nil {
		return nil, fmt.Errorf("verlust-exemplare endgültig löschen fehlgeschlagen: %w", err)
	}

	for _, s := range treffer {
		if err := schreibeProtokollEintrag(ctx, a.db, protokollEintrag{
			Tabelle: "buecher_exemplare", Aktion: "DELETE", DatensatzID: s.id,
			BearbeiterID: &bearbeiterID, Akteur: "USER",
			Kontext: zeigerAuf("Als Verlust gebuchtes Exemplar endgültig gelöscht"),
			Details: map[string]any{
				"barcode_id": s.barcode, "titel": s.titel,
				"action": ProtokollAktionVerlustEndgueltigGeloescht, ProtokollDetailWarImBestand: s.warImBestand,
			},
		}); err != nil {
			return nil, err
		}
	}
	// Die IDs und nicht nur ihre Zahl: Die Oberfläche nimmt genau diese Zeilen aus ihrer Liste.
	// Waren von fünf gewählten zwei inzwischen als gefunden markiert, bleiben sie dort stehen.
	return ids, nil
}

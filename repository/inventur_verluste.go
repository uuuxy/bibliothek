package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// InventurVerlust ist eine Zeile des Fehlbestandsberichts.
type InventurVerlust struct {
	// ExemplarID kann fehlen (leerer String): Das FK-Feld geht bei einer endgültigen
	// Löschung des Exemplars auf NULL (Migration 059, ON DELETE SET NULL) — die
	// Textspalten bleiben trotzdem lesbar. Ohne ExemplarID sind "Gefunden" und
	// "endgültig löschen" nicht mehr möglich, das Exemplar ist ja schon weg.
	ExemplarID string `json:"exemplar_id,omitempty"`
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	Signatur   string `json:"signatur"`
	GebuchtAm  string `json:"gebucht_am"`
	// GefundenAm ist gesetzt, sobald das Exemplar beim Nachsuchen wiedergefunden und
	// über den "Gefunden"-Knopf zurück in Umlauf gebracht wurde (Migration 061).
	GefundenAm string `json:"gefunden_am,omitempty"`
}

// LadeVerluste liefert den Fehlbestand einer abgeschlossenen Session.
//
// Sortiert nach Signatur und Titel — das ist die Reihenfolge, in der man mit der Liste
// durchs Regal geht. Nach Barcode oder Buchungszeit sortiert wäre sie zum Nachsuchen
// unbrauchbar, weil man dann kreuz und quer laufen müsste.
func (a *InventurAbfragen) LadeVerluste(ctx context.Context, laufID string) ([]InventurVerlust, error) {
	rows, err := a.db.Query(ctx, `
		SELECT coalesce(exemplar_id::text, ''), barcode_id, titel, autor, signatur,
		       to_char(gebucht_am, 'YYYY-MM-DD"T"HH24:MI:SSOF'),
		       coalesce(to_char(gefunden_am, 'YYYY-MM-DD"T"HH24:MI:SSOF'), '')
		FROM inventur_verluste
		WHERE session_id = $1
		ORDER BY signatur, titel, barcode_id
	`, laufID)
	if err != nil {
		return nil, fmt.Errorf("fehlbestand laden fehlgeschlagen: %w", err)
	}
	defer rows.Close()

	// Nie nil: Eine leere Liste muss beim Client als [] ankommen.
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

// Zwei Handlungen für den Fehlbestandsbericht (Migration 061). Vorher war er reine
// Anzeige — Rückmeldung aus der Bibliothek (05.08.2026): „was mache ich [dann]? ich kann nicht weiter damit
// machen!" Beim Regal-Absuchen gibt es genau zwei Ausgänge je Zeile: das Buch taucht
// doch wieder auf, oder es bleibt endgültig weg. Beide brauchen einen Knopf.
//
// Transaktionsgrenze liegt bewusst NICHT hier: Wie SchliesseLaufAb erwarten
// diese Methoden ein bereits eröffnetes db (Pool oder Tx) — der Aufrufer (Handler)
// eröffnet die Transaktion, damit Snapshot, Update und Audit-Log atomar bleiben.

// MarkiereVerlustAlsGefunden bringt ein als Verlust gebuchtes Exemplar zurück in
// Umlauf und trägt den Fund im Fehlbestandsbericht ein. false ohne Fehler heisst: kein
// offener Verlust mit dieser ID (falsche ID oder schon endgültig gelöscht) — dann gibt
// es nichts zu tun.
//
// Seit dem 15.09.2026 endet mit dem Fund auch die Forderung „nicht zurückgegeben", die
// das Buch abgerechnet hat (verbucheRueckkehr, dieselbe Regel wie an der Theke): Vorher
// stand das Buch wieder im Regal, das Kind blieb gesperrt und die Forderung offen —
// beendet wurde sie nur, wenn das Buch über die Theke zurückkam (OFFEN.md 5.3). Der
// Befund sagt, was storniert wurde und ob ein Bescheid schon bei der Aufsicht liegt.
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

	// Derselbe Baustein wie an der Theke (seit 15.09.2026, Stufe 2, Commit 8): Umlauf und
	// Forderung in der Transaktion des Handlers. Bis dahin stand das UPDATE hier ein zweites
	// Mal — zwei Schreibweisen für eine Rückkehr, die nur zufällig einig waren.
	befund, err := HoleExemplarZurueck(ctx, a.db, exemplarID, bearbeiterID, nil)
	if err != nil {
		return false, RueckkehrBefund{}, err
	}

	// Alle noch offenen Verlust-Zeilen dieses Exemplars schliessen, nicht nur die einer
	// bestimmten Session — ein Exemplar kann rein rechnerisch mehrfach als Verlust
	// gebucht worden sein, wenn ein Fund vor dem Klick hier nie eingetragen wurde.
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

// LoescheVerlusteEndgueltig entfernt bereits als Verlust gebuchte Exemplare
// unwiderruflich aus dem Katalog. Der Fehlbestandsbericht bleibt lesbar (Migration
// 059: ON DELETE SET NULL, die Textspalten sind eine Abschrift) — nur das Löschen
// beendet den offenen Zustand endgültig.
//
// Bewusst eingeschränkt auf ist_ausgesondert=true (VERLUST): Diese Funktion ist NICHT
// der allgemeine Lösch-Weg für Exemplare (dafür DeleteCopyHandler, der soft löscht) —
// sie darf nur aufräumen, was der Fehlbestandsbericht selbst schon als verloren
// gebucht hat. Liefert die Zahl der tatsächlich gelöschten Exemplare (IDs, die schon
// vorher weg waren oder nicht VERLUST sind, werden still übersprungen).
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

	// Die abhängigen Zeilen zuerst — `ausleihen.exemplar_id` und
	// `schadensfaelle.exemplar_id` stehen auf ON DELETE RESTRICT, die Datenbank hält
	// sonst dagegen. Bis zum 23.08.2026 fehlten diese beiden Schritte hier, während das
	// Geschwister-Verfahren (inventur/db_books_delete.go, „Titel löschen") sie längst
	// hatte: Das endgültige Löschen scheiterte an JEDEM Exemplar, das je ausgeliehen war
	// — in einer Schulbibliothek also an fast jedem. Der Fehler kam als 500 zurück und
	// wurde zu „interner Datenbankfehler" sanitisiert; niemand konnte sehen, woran es lag.
	//
	// Reihenfolge wie beim Titel-Löschen: Schadensfälle, dann Ausleihen, dann das
	// Exemplar. Was hier verschwindet, ist erledigte Geschichte — offene Vorgänge hat
	// pruefeVerlusteLoeschbar bereits abgewiesen. Der Fehlbestandsbericht überlebt es
	// (inventur_verluste hält eine Abschrift, ON DELETE SET NULL), und der Barcode wird
	// wieder frei: `buecher_exemplare.barcode_id` ist UNIQUE ohne Teilindex, eine
	// stehengebliebene Zeile blockierte den Aufkleber für immer.
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
	// Die IDs, nicht nur ihre Anzahl: Die Oberfläche entfernte bisher ALLE angefragten
	// Zeilen aus ihrer Liste, gleich wie viele der Server wirklich gelöscht hat. Wer fünf
	// auswählte, von denen zwei inzwischen als „gefunden" markiert waren, sah alle fünf
	// verschwinden — zwei davon nur auf seinem Bildschirm.
	return ids, nil
}

func zeigerAuf(s string) *string { return &s }

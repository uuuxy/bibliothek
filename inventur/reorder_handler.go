package inventur

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ReorderRequest beschreibt die Eingabe für die Neuordnung von Büchern.
type ReorderRequest struct {
	BookIDs []string `json:"bookIds"`
}

// handleReorderBooks verarbeitet PUT-Anfragen zum Neuordnen der Bücher.
// Die Sortierung wird als Batch-Update durchgeführt (kein N+1 Query).
func (handler *APIHandler) handleReorderBooks(writer http.ResponseWriter, request *http.Request) {
	var input ReorderRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "ungültiges json")
		return
	}

	if len(input.BookIDs) == 0 {
		writeJSON(writer, http.StatusOK, map[string]string{"message": "nichts zu speichern"})
		return
	}

	// Batch-Update: Alle sort_order Werte in einer einzigen Query setzen
	// statt N einzelner UPDATE-Statements (N+1 Schutz).
	ctx := request.Context()
	tx, err := handler.repo.db.Begin(ctx)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "transaktion konnte nicht gestartet werden")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }() //nolint:errcheck

	// Erstelle Arrays für den Batch-Update mittels unnest
	sortOrders := make([]int, len(input.BookIDs))
	for i := range input.BookIDs {
		sortOrders[i] = i + 1
	}

	ergebnis, err := tx.Exec(ctx, `
		UPDATE buecher_titel SET sort_order = daten.neue_reihenfolge
		FROM (SELECT unnest($1::uuid[]) AS buch_id, unnest($2::int[]) AS neue_reihenfolge) AS daten
		WHERE buecher_titel.id = daten.buch_id
	`, input.BookIDs, sortOrders)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "sortierung konnte nicht gespeichert werden")
		return
	}

	// Gezaehlt wird, was die Datenbank GEAENDERT hat, nicht was der Client geschickt hat.
	// Die Meldung nannte bis zum 23.08.2026 len(input.BookIDs) — bei IDs, die es nicht
	// (mehr) gibt, aendert das UPDATE nichts, und die Oberflaeche bekam trotzdem
	// "Sortierung gespeichert". Eine Liste, die nach dem Neuladen wieder in der alten
	// Reihenfolge steht, ist danach ein Raetsel statt einer Fehlermeldung.
	geaendert := ergebnis.RowsAffected()
	if geaendert == 0 {
		writeError(writer, http.StatusNotFound, "keines der uebergebenen Buecher existiert — Sortierung nicht gespeichert")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(writer, http.StatusInternalServerError, "transaktion konnte nicht abgeschlossen werden")
		return
	}

	antwort := map[string]any{
		"message":    fmt.Sprintf("erfolgreich %d bücher sortiert", geaendert),
		"sortiert":   geaendert,
		"uebergeben": len(input.BookIDs),
	}
	if geaendert < int64(len(input.BookIDs)) {
		antwort["message"] = fmt.Sprintf("%d von %d büchern sortiert — %d IDs waren unbekannt",
			geaendert, len(input.BookIDs), int64(len(input.BookIDs))-geaendert)
	}
	writeJSON(writer, http.StatusOK, antwort)
}

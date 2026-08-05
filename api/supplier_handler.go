package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
)

// SupplierResponse represents the supplier data sent to the client.
type SupplierResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	CustomerNumber string    `json:"customerNumber"`
	ErstelltAm     time.Time `json:"erstellt_am"`

	// LiefertMitBarcode: Händler beklebt die Bücher vor der Lieferung mit unseren Barcodes.
	LiefertMitBarcode bool `json:"liefert_mit_barcode"`

	// IstStandard: Vorauswahl im Bestellformular. Höchstens einer trägt true.
	IstStandard bool `json:"ist_standard"`

	// BietetBestellbestaetigung: Lieferant bietet nach der Bestellung eine eigene
	// Etikettengrößen-Wahl + Bestätigung an (z. B. Naacher). Steuert, ob beim Bestellen
	// zusätzlich das große Lernmittel-Etikett mitgeschickt wird und ob die Bestellhistorie
	// den Bestätigen-Schritt zeigt.
	BietetBestellbestaetigung bool `json:"bietet_bestellbestaetigung"`
}

// CreateSupplierRequest holds the payload for creating a new supplier.
type CreateSupplierRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	CustomerNumber string `json:"customerNumber"`

	// LiefertMitBarcode ist bewusst ein einfaches bool und kein *bool: Fehlt das Feld,
	// gilt false — das bisherige Verhalten, bei dem wir selbst etikettieren.
	LiefertMitBarcode bool `json:"liefert_mit_barcode"`

	// IstStandard: Vorauswahl im Bestellformular.
	IstStandard bool `json:"ist_standard"`

	// BietetBestellbestaetigung: siehe SupplierResponse.
	BietetBestellbestaetigung bool `json:"bietet_bestellbestaetigung"`
}

// setzeStandardLieferant macht genau einen Lieferanten zur Vorauswahl.
//
// Die REIHENFOLGE ist der Schutz, nicht nur Kosmetik: Der Teil-Index
// idx_lieferanten_ein_standard lässt nur eine Zeile mit true zu. Würde erst der neue
// gesetzt und danach der alte geräumt, bräche das UPDATE mit einer
// Unique-Verletzung ab — und zwar erst beim zweiten Wechsel, also lange nach dem
// Einbau. Deshalb zuerst räumen, dann setzen, beides in derselben Transaktion.
//
// Ohne Transaktion bliebe zwischen den beiden Schritten ein Moment ohne
// Standardlieferanten; an mehreren Arbeitsplätzen gleichzeitig ist das kein
// theoretischer Fall (siehe docs zum Mehrplatzbetrieb).
func setzeStandardLieferant(ctx context.Context, pool db.PgxPoolIface, id string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	if _, err := tx.Exec(ctx, `UPDATE lieferanten SET ist_standard = false WHERE ist_standard AND id <> $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE lieferanten SET ist_standard = true WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListSuppliersHandler returns a list of all suppliers.
func (s *Server) ListSuppliersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Der Standardlieferant zuerst: Das Bestellformular nimmt sonst den alphabetisch
		// ersten, und die Vorauswahl bliebe wirkungslos.
		rows, err := s.DB.Pool.Query(ctx, `
			SELECT id, name, email, kundennummer, erstellt_am, liefert_mit_barcode, ist_standard, bietet_bestellbestaetigung
			FROM lieferanten
			ORDER BY ist_standard DESC, name ASC
		`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		suppliers := []SupplierResponse{}
		for rows.Next() {
			var sup SupplierResponse
			if err := rows.Scan(&sup.ID, &sup.Name, &sup.Email, &sup.CustomerNumber, &sup.ErstelltAm, &sup.LiefertMitBarcode, &sup.IstStandard, &sup.BietetBestellbestaetigung); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			suppliers = append(suppliers, sup)
		}
		if err := rows.Err(); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		RespondJSON(w, http.StatusOK, suppliers)
	}
}

// CreateSupplierHandler adds a new supplier.
func (s *Server) CreateSupplierHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSupplierRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		if req.Name == "" || req.Email == "" || req.CustomerNumber == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("name, email and customerNumber are required"))
			return
		}

		ctx := r.Context()

		var newID string
		var erstelltAm time.Time
		err := s.DB.Pool.QueryRow(ctx, `
			INSERT INTO lieferanten (name, email, kundennummer, liefert_mit_barcode, bietet_bestellbestaetigung)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, erstellt_am
		`, req.Name, req.Email, req.CustomerNumber, req.LiefertMitBarcode, req.BietetBestellbestaetigung).Scan(&newID, &erstelltAm)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Bewusst NICHT im INSERT: Wäre schon ein anderer Lieferant Standard, bräche der
		// Teil-Index den Anlegevorgang ab. Erst anlegen, dann umschalten — dabei räumt
		// setzeStandardLieferant den bisherigen weg.
		if req.IstStandard {
			if err := setzeStandardLieferant(ctx, s.DB.Pool, newID); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}

		RespondJSON(w, http.StatusCreated, SupplierResponse{
			ID:                        newID,
			Name:                      req.Name,
			Email:                     req.Email,
			CustomerNumber:            req.CustomerNumber,
			ErstelltAm:                erstelltAm,
			LiefertMitBarcode:         req.LiefertMitBarcode,
			IstStandard:               req.IstStandard,
			BietetBestellbestaetigung: req.BietetBestellbestaetigung,
		})
	}
}

// UpdateSupplierHandler updates name, email and customer number of an existing supplier.
func (s *Server) UpdateSupplierHandler() http.HandlerFunc {
	return s.handleUpdateSupplier
}

func (s *Server) handleUpdateSupplier(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing supplier ID"))
		return
	}

	var req CreateSupplierRequest
	if !DecodeAndValidate(w, r, &req) {
		return
	}
	if req.Name == "" || req.Email == "" || req.CustomerNumber == "" {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("name, email and customerNumber are required"))
		return
	}

	ctx := r.Context()
	tag, err := s.DB.Pool.Exec(ctx,
		`UPDATE lieferanten SET name = $1, email = $2, kundennummer = $3, liefert_mit_barcode = $4, bietet_bestellbestaetigung = $5 WHERE id = $6`,
		req.Name, req.Email, req.CustomerNumber, req.LiefertMitBarcode, req.BietetBestellbestaetigung, id,
	)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	if tag.RowsAffected() == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
		return
	}

	// Der Haken wird nur GESETZT, nie hier entfernt: Das Wegnehmen geschieht dadurch,
	// dass ein anderer Lieferant Standard wird. Ein Bestellwesen ganz ohne Vorauswahl
	// wäre der Zustand von vorher — dafür gibt es keinen Anlass, und ein versehentlich
	// entfernter Haken beim Korrigieren einer E-Mail wäre ein stiller Rückschritt.
	if req.IstStandard {
		if err := setzeStandardLieferant(ctx, s.DB.Pool, id); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
	}

	RespondJSON(w, http.StatusOK, SupplierResponse{
		ID:                        id,
		Name:                      req.Name,
		Email:                     req.Email,
		CustomerNumber:            req.CustomerNumber,
		LiefertMitBarcode:         req.LiefertMitBarcode,
		IstStandard:               req.IstStandard,
		BietetBestellbestaetigung: req.BietetBestellbestaetigung,
	})
}

// DeleteSupplierHandler removes a supplier.
func (s *Server) DeleteSupplierHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Go 1.22+ routing path parameter resolution
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("missing supplier ID"))
			return
		}

		ctx := r.Context()

		tag, err := s.DB.Pool.Exec(ctx, "DELETE FROM lieferanten WHERE id = $1", id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if tag.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

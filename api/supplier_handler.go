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

	// IstHauptlieferant: Der EINE Händler, über den die Schule bestellt. Siehe
	// repository.Supplier — an diesem einen Merkmal hängen Vorauswahl, Bestelllink und
	// Nachdruck-Liste gemeinsam.
	IstHauptlieferant bool `json:"ist_hauptlieferant"`
}

// CreateSupplierRequest holds the payload for creating a new supplier.
type CreateSupplierRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	CustomerNumber string `json:"customerNumber"`

	// IstHauptlieferant ist bewusst ein einfaches bool und kein *bool: Fehlt das Feld,
	// gilt false — also ein Händler, der einfach nur die Bestellmail bekommt.
	IstHauptlieferant bool `json:"ist_hauptlieferant"`
}

// setzeHauptlieferant macht genau einen Lieferanten zum Hauptlieferanten und nimmt das
// Merkmal allen anderen — in dieser Reihenfolge, in einer Transaktion.
//
// Die REIHENFOLGE ist der Schutz, nicht nur Kosmetik: Der Teil-Index
// idx_lieferanten_ein_hauptlieferant lässt nur eine Zeile mit true zu. Würde erst der
// neue gesetzt und danach der alte geräumt, bräche das UPDATE mit einer
// Unique-Verletzung ab — und zwar erst beim zweiten Wechsel, also lange nach dem Einbau.
// Deshalb zuerst räumen, dann setzen.
//
// Ohne Transaktion bliebe zwischen den beiden Schritten ein Moment ganz ohne
// Hauptlieferanten; an mehreren Arbeitsplätzen gleichzeitig ist das kein theoretischer
// Fall (siehe docs zum Mehrplatzbetrieb).
func setzeHauptlieferant(ctx context.Context, pool db.PgxPoolIface, id string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	if _, err := tx.Exec(ctx,
		`UPDATE lieferanten SET ist_hauptlieferant = false WHERE ist_hauptlieferant AND id <> $1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE lieferanten SET ist_hauptlieferant = true WHERE id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListSuppliersHandler returns a list of all suppliers.
func (s *Server) ListSuppliersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Der Hauptlieferant zuerst: Das Bestellformular nimmt sonst den alphabetisch
		// ersten, und die Vorauswahl bliebe wirkungslos.
		rows, err := s.DB.Pool.Query(ctx, `
			SELECT id, name, email, kundennummer, erstellt_am, ist_hauptlieferant
			FROM lieferanten
			ORDER BY ist_hauptlieferant DESC, name ASC
		`)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		defer rows.Close()

		suppliers := []SupplierResponse{}
		for rows.Next() {
			var sup SupplierResponse
			if err := rows.Scan(&sup.ID, &sup.Name, &sup.Email, &sup.CustomerNumber, &sup.ErstelltAm, &sup.IstHauptlieferant); err != nil {
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
			INSERT INTO lieferanten (name, email, kundennummer)
			VALUES ($1, $2, $3)
			RETURNING id, erstellt_am
		`, req.Name, req.Email, req.CustomerNumber).Scan(&newID, &erstelltAm)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Bewusst NICHT im INSERT: Gibt es schon einen Hauptlieferanten, bräche der
		// Teil-Index den Anlegevorgang ab — der neue Lieferant wäre gar nicht erst
		// entstanden, nur weil ein Haken gesetzt war. Erst anlegen, dann umschalten;
		// dabei räumt setzeHauptlieferant den bisherigen weg.
		if req.IstHauptlieferant {
			if err := setzeHauptlieferant(ctx, s.DB.Pool, newID); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}

		RespondJSON(w, http.StatusCreated, SupplierResponse{
			ID:                newID,
			Name:              req.Name,
			Email:             req.Email,
			CustomerNumber:    req.CustomerNumber,
			ErstelltAm:        erstelltAm,
			IstHauptlieferant: req.IstHauptlieferant,
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
		`UPDATE lieferanten SET name = $1, email = $2, kundennummer = $3 WHERE id = $4`,
		req.Name, req.Email, req.CustomerNumber, id,
	)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	if tag.RowsAffected() == 0 {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
		return
	}

	// Das Merkmal steht bewusst NICHT im UPDATE oben: Trägt es schon ein anderer, bräche
	// der Teil-Index das ganze UPDATE ab — und damit auch die Korrektur einer E-Mail.
	// Erst die Stammdaten, dann das Merkmal über den Setzer, der vorher räumt.
	//
	// Abschalten ist erlaubt: „Kein Hauptlieferant" ist ein normaler Zustand — wer
	// aufhört, über Naacher zu bestellen, muss ihn loswerden können, ohne ihn erst
	// jemand anderem zu geben. Der Aufrufer schickt beim Bearbeiten immer den aktuellen
	// Stand mit (startEdit liest ihn aus der Zeile), ein versehentliches Abschalten beim
	// Korrigieren der E-Mail ist damit ausgeschlossen.
	if req.IstHauptlieferant {
		if err := setzeHauptlieferant(ctx, s.DB.Pool, id); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
	} else if _, err := s.DB.Pool.Exec(ctx,
		`UPDATE lieferanten SET ist_hauptlieferant = false WHERE id = $1`, id); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	RespondJSON(w, http.StatusOK, SupplierResponse{
		ID:                id,
		Name:              req.Name,
		Email:             req.Email,
		CustomerNumber:    req.CustomerNumber,
		IstHauptlieferant: req.IstHauptlieferant,
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

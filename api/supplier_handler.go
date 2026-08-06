package api

import (
	"context"
	"errors"
	"fmt"
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
	return setzeExklusivesMerkmal(ctx, pool, merkmalStandard, id)
}

// setzeBestelllinkLieferant bestimmt den einen Lieferanten, der den Bestelllink bekommt.
// Über diesen Link wählt er die Etikettengröße und bestätigt die Bestellung selbst.
func setzeBestelllinkLieferant(ctx context.Context, pool db.PgxPoolIface, id string) error {
	return setzeExklusivesMerkmal(ctx, pool, merkmalBestelllink, id)
}

// exklusivesMerkmal ist eine Eigenschaft, die höchstens EIN Lieferant tragen darf. Beide
// Werte sind Konstanten aus diesem Paket und stammen nie aus einer Anfrage.
type exklusivesMerkmal string

const (
	merkmalStandard    exklusivesMerkmal = "ist_standard"
	merkmalBestelllink exklusivesMerkmal = "bietet_bestellbestaetigung"
)

// setzeExklusivesMerkmal gibt das Merkmal genau einem Lieferanten und nimmt es allen
// anderen — in dieser Reihenfolge, in einer Transaktion.
//
// Die REIHENFOLGE ist der Schutz, nicht nur Kosmetik: Die Teil-Indizes
// idx_lieferanten_ein_standard bzw. idx_lieferanten_ein_bestelllink lassen nur eine Zeile
// mit true zu. Würde erst der neue gesetzt und danach der alte geräumt, bräche das UPDATE
// mit einer Unique-Verletzung ab — und zwar erst beim zweiten Wechsel, also lange nach dem
// Einbau. Deshalb zuerst räumen, dann setzen, beides in derselben Transaktion.
//
// Ohne Transaktion bliebe zwischen den beiden Schritten ein Moment ohne Träger; an
// mehreren Arbeitsplätzen gleichzeitig ist das kein theoretischer Fall (siehe docs zum
// Mehrplatzbetrieb).
//
// Das SQL steht je Merkmal wörtlich da, statt den Spaltennamen in einen String zu
// formatieren: Eine zusammengesetzte Abfrage wäre hier zwar ungefährlich, aber sie nimmt
// jedem Leser (und jedem Linter) die Möglichkeit, das ohne Nachdenken zu sehen.
func setzeExklusivesMerkmal(ctx context.Context, pool db.PgxPoolIface, merkmal exklusivesMerkmal, id string) error {
	var raeumen, setzen string
	switch merkmal {
	case merkmalStandard:
		raeumen = `UPDATE lieferanten SET ist_standard = false WHERE ist_standard AND id <> $1`
		setzen = `UPDATE lieferanten SET ist_standard = true WHERE id = $1`
	case merkmalBestelllink:
		raeumen = `UPDATE lieferanten SET bietet_bestellbestaetigung = false WHERE bietet_bestellbestaetigung AND id <> $1`
		setzen = `UPDATE lieferanten SET bietet_bestellbestaetigung = true WHERE id = $1`
	default:
		return fmt.Errorf("unbekanntes exklusives Lieferanten-Merkmal %q", merkmal)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer db.SafeRollback(ctx, tx)

	if _, err := tx.Exec(ctx, raeumen, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, setzen, id); err != nil {
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
			INSERT INTO lieferanten (name, email, kundennummer, liefert_mit_barcode)
			VALUES ($1, $2, $3, $4)
			RETURNING id, erstellt_am
		`, req.Name, req.Email, req.CustomerNumber, req.LiefertMitBarcode).Scan(&newID, &erstelltAm)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// Beide exklusiven Merkmale bewusst NICHT im INSERT: Trägt sie schon ein anderer
		// Lieferant, bräche der jeweilige Teil-Index den Anlegevorgang ab — der neue
		// Lieferant wäre gar nicht erst entstanden, nur weil ein Haken gesetzt war. Erst
		// anlegen, dann umschalten; dabei räumt der Setzer den bisherigen Träger weg.
		if req.IstStandard {
			if err := setzeStandardLieferant(ctx, s.DB.Pool, newID); err != nil {
				apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if req.BietetBestellbestaetigung {
			if err := setzeBestelllinkLieferant(ctx, s.DB.Pool, newID); err != nil {
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
		`UPDATE lieferanten SET name = $1, email = $2, kundennummer = $3, liefert_mit_barcode = $4 WHERE id = $5`,
		req.Name, req.Email, req.CustomerNumber, req.LiefertMitBarcode, id,
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

	// Beim Bestelllink ist das ABSCHALTEN dagegen erlaubt — anders als beim Standard.
	// „Kein Standardlieferant" wäre ein Rückschritt, „kein Händler mit Bestelllink" ist
	// ein völlig normaler Betriebszustand: Wer aufhört, über Naacher zu bestellen, muss
	// den Link auch wieder loswerden können, ohne ihn erst jemand anderem zu geben.
	if req.BietetBestellbestaetigung {
		if err := setzeBestelllinkLieferant(ctx, s.DB.Pool, id); err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
	} else if _, err := s.DB.Pool.Exec(ctx,
		`UPDATE lieferanten SET bietet_bestellbestaetigung = false WHERE id = $1`, id); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
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

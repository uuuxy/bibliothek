package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"

	"github.com/jackc/pgx/v5"
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

	// KundennummerSchultraeger: zweites Kundenkonto für Bestellungen der Schülerbücherei
	// (Mittel des Schulträgers). Leer = dieselbe Nummer (Migration 109).
	KundennummerSchultraeger string `json:"kundennummer_schultraeger"`
}

// CreateSupplierRequest holds the payload for creating a new supplier.
type CreateSupplierRequest struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	CustomerNumber string `json:"customerNumber"`

	// IstHauptlieferant ist bewusst ein einfaches bool und kein *bool: Fehlt das Feld,
	// gilt false — also ein Händler, der einfach nur die Bestellmail bekommt.
	IstHauptlieferant bool `json:"ist_hauptlieferant"`

	// KundennummerSchultraeger: optional; leer heißt „dieselbe Nummer wie customerNumber".
	//
	// Zeiger, weil das FEHLENDE Feld etwas anderes bedeutet als das LEERE (Bugklasse
	// „Fehlendes Feld, zwei Bedeutungen"): nil = unverändert lassen, "" = ausdrücklich
	// löschen. Ein plain string machte jede Anfrage ohne dieses Feld zum stillen Blanking
	// — eine Kundennummer, die verschwindet, fällt erst auf, wenn der Händler die Rechnung
	// auf das falsche Konto stellt. Beim Anlegen (POST) ist nil schlicht leer.
	KundennummerSchultraeger *string `json:"kundennummer_schultraeger"`
}

// kundennummerSchultraeger liefert den getrimmten Wert für das Anlegen — dort ist ein
// fehlendes Feld schlicht leer, es gibt noch keinen Stand, der erhalten bleiben könnte.
func kundennummerSchultraeger(req CreateSupplierRequest) string {
	if req.KundennummerSchultraeger == nil {
		return ""
	}
	return strings.TrimSpace(*req.KundennummerSchultraeger)
}

// kundennummerSchultraegerOderNil liefert nil, wenn das Feld in der Anfrage fehlt — das
// COALESCE im UPDATE lässt die hinterlegte Nummer dann unangetastet.
func kundennummerSchultraegerOderNil(req CreateSupplierRequest) *string {
	if req.KundennummerSchultraeger == nil {
		return nil
	}
	getrimmt := strings.TrimSpace(*req.KundennummerSchultraeger)
	return &getrimmt
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
			SELECT id, name, email, kundennummer, erstellt_am, ist_hauptlieferant, kundennummer_schultraeger
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
			if err := rows.Scan(&sup.ID, &sup.Name, &sup.Email, &sup.CustomerNumber, &sup.ErstelltAm, &sup.IstHauptlieferant, &sup.KundennummerSchultraeger); err != nil {
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
			INSERT INTO lieferanten (name, email, kundennummer, kundennummer_schultraeger)
			VALUES ($1, $2, $3, $4)
			RETURNING id, erstellt_am
		`, req.Name, req.Email, req.CustomerNumber, kundennummerSchultraeger(req)).Scan(&newID, &erstelltAm)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		// An diese Adresse gehen Bestellungen. Der Eintrag nennt den Händler, nicht die Adresse.
		s.protokolliereVerwaltung(ctx, auditLieferantAngelegt,
			map[string]any{"lieferant_id": newID, "name": req.Name})

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
			ID:                       newID,
			Name:                     req.Name,
			Email:                    req.Email,
			CustomerNumber:           req.CustomerNumber,
			ErstelltAm:               erstelltAm,
			IstHauptlieferant:        req.IstHauptlieferant,
			KundennummerSchultraeger: kundennummerSchultraeger(req),
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
	// COALESCE mit einem Zeiger-Parameter: Fehlt das Feld in der Anfrage (nil → SQL NULL),
	// bleibt die hinterlegte Nummer stehen; ein leerer String löscht sie ausdrücklich.
	//
	// RETURNING statt Exec + RowsAffected, damit die Antwort den GESPEICHERTEN Stand nennt
	// und nicht die Eingabe zurückspiegelt — sonst meldete sie eine leere zweite
	// Kundennummer, während in der Datenbank die alte steht.
	//
	// Der Stand davor (alt) sagt dem Protokoll, welche Felder sich geändert haben.
	var gespeicherteZweitnummer string
	var alt lieferantStand
	err := s.DB.Pool.QueryRow(ctx, `
		WITH alt AS (
			SELECT name, email, kundennummer, kundennummer_schultraeger, ist_hauptlieferant
			  FROM lieferanten WHERE id = $4 FOR UPDATE
		)
		UPDATE lieferanten l
		   SET name = $1, email = $2, kundennummer = $3,
		       kundennummer_schultraeger = COALESCE($5, l.kundennummer_schultraeger)
		  FROM alt
		 WHERE l.id = $4
		RETURNING l.kundennummer_schultraeger,
		          alt.name, alt.email, alt.kundennummer, alt.kundennummer_schultraeger, alt.ist_hauptlieferant`,
		req.Name, req.Email, req.CustomerNumber, id, kundennummerSchultraegerOderNil(req),
	).Scan(&gespeicherteZweitnummer, &alt.name, &alt.email, &alt.kundennummer, &alt.zweitnummer, &alt.haupt)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
		return
	}
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	felder := geaenderteFelder(
		feldWechsel{"name", alt.name != req.Name},
		feldWechsel{"email", alt.email != req.Email},
		feldWechsel{"kundennummer", alt.kundennummer != req.CustomerNumber},
		feldWechsel{"kundennummer_schultraeger", alt.zweitnummer != gespeicherteZweitnummer},
	)
	eintrag := map[string]any{"lieferant_id": id, "name": req.Name}

	// Das Merkmal steht bewusst NICHT im UPDATE oben: Trägt es schon ein anderer, bräche
	// der Teil-Index das ganze UPDATE ab — und damit auch die Korrektur einer E-Mail.
	// Erst die Stammdaten, dann das Merkmal über den Setzer, der vorher räumt.
	//
	// Abschalten ist erlaubt: „Kein Hauptlieferant" ist ein normaler Zustand — wer
	// aufhört, über Naacher zu bestellen, muss ihn loswerden können, ohne ihn erst
	// jemand anderem zu geben. Der Aufrufer schickt beim Bearbeiten immer den aktuellen
	// Stand mit (startEdit liest ihn aus der Zeile), ein versehentliches Abschalten beim
	// Korrigieren der E-Mail ist damit ausgeschlossen.
	if err := s.schreibeHauptlieferant(ctx, id, req.IstHauptlieferant); err != nil {
		// Die Stammdaten sind geschrieben: Der Eintrag nennt, was sich daran geändert hat.
		s.protokolliereGeaenderteFelder(ctx, auditLieferantGeaendert, eintrag, felder)
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	felder = append(felder, geaenderteFelder(feldWechsel{"hauptlieferant", alt.haupt != req.IstHauptlieferant})...)
	s.protokolliereGeaenderteFelder(ctx, auditLieferantGeaendert, eintrag, felder)

	RespondJSON(w, http.StatusOK, SupplierResponse{
		ID:                       id,
		Name:                     req.Name,
		Email:                    req.Email,
		CustomerNumber:           req.CustomerNumber,
		IstHauptlieferant:        req.IstHauptlieferant,
		KundennummerSchultraeger: gespeicherteZweitnummer,
	})
}

// lieferantStand sind die Stammdaten eines Lieferanten vor einer Änderung.
type lieferantStand struct {
	name, email, kundennummer, zweitnummer string
	haupt                                  bool
}

// schreibeHauptlieferant gibt dem Lieferanten das Merkmal oder nimmt es ihm.
func (s *Server) schreibeHauptlieferant(ctx context.Context, id string, soll bool) error {
	if soll {
		return setzeHauptlieferant(ctx, s.DB.Pool, id)
	}
	_, err := s.DB.Pool.Exec(ctx, `UPDATE lieferanten SET ist_hauptlieferant = false WHERE id = $1`, id)
	return err
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

		// Den Hauptlieferanten nicht einfach wegnehmen.
		//
		// Gefunden am 07.09.2026 beim Befragen von `bestellungen_verlauf.lieferant_id ->
		// lieferanten` (Frage 12): Der Fremdschlüssel selbst ist harmlos — die Bestellung
		// hält Name und E-Mail als eigene Abschrift, SET NULL nimmt ihr nichts. Der
		// LÖSCHWEG daneben war das Problem. „Löschen" in der Lieferantenverwaltung fragt
		// nicht nach, und getroffen werden konnte auch der EINE Händler, an dem der ganze
		// Bestellweg hängt: Bestellmail, Bestätigungs-Link (bestellbestaetigung_handler.go)
		// und die Etiketten-Entscheidung „der Händler beklebt selbst" (pdf_service.go).
		// Danach gab es keinen Hauptlieferanten mehr, und niemand erfuhr davon — die
		// Oberfläche zeigte nur einen Händler weniger.
		//
		// Kein Sonderfall in der Oberfläche, sondern hier: Die Verwaltung ist nicht die
		// einzige Tür, und ein Hinweis, den nur ein Formular kennt, ist keine Regel. Der
		// Weg bleibt offen — erst einen anderen zum Hauptlieferanten machen (oder den
		// Schalter abwählen), dann löschen.
		var istHaupt bool
		var name string
		if err := s.DB.Pool.QueryRow(ctx,
			"SELECT ist_hauptlieferant, name FROM lieferanten WHERE id = $1", id).Scan(&istHaupt, &name); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if istHaupt {
			//nolint:staticcheck // ST1005: ganze Sätze mit Satzzeichen — diese Meldung steht so vor der Bibliothekskraft.
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(
				"Dieser Händler ist der Hauptlieferant — über ihn läuft die Bestellung. "+
					"Erst einen anderen zum Hauptlieferanten machen oder den Schalter abwählen, dann löschen."))
			return
		}

		tag, err := s.DB.Pool.Exec(ctx, "DELETE FROM lieferanten WHERE id = $1", id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if tag.RowsAffected() == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
			return
		}
		s.protokolliereVerwaltung(ctx, auditLieferantGeloescht,
			map[string]any{"lieferant_id": id, "name": name})

		w.WriteHeader(http.StatusNoContent)
	}
}

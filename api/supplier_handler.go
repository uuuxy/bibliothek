package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"

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

// UpdateSupplierRequest nennt, was an einem Lieferanten geändert wird. Ein fehlendes Feld
// bleibt, wie es ist: Die Maske schickt nur, was sie seit dem Öffnen geändert hat, damit sie
// nichts überschreibt, was ein anderer Platz inzwischen gespeichert hat — auch nicht das
// Merkmal Hauptlieferant. Ein unbekanntes Feld lehnt die Tür ab. Eine leere zweite
// Kundennummer heißt „dieselbe wie die erste".
type UpdateSupplierRequest struct {
	Name                     *string `json:"name"`
	Email                    *string `json:"email"`
	CustomerNumber           *string `json:"customerNumber"`
	IstHauptlieferant        *bool   `json:"ist_hauptlieferant"`
	KundennummerSchultraeger *string `json:"kundennummer_schultraeger"`
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
	if _, err := repository.SetzeHauptlieferant(ctx, tx, id); err != nil {
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
		zeilen, err := repository.ListeLieferanten(ctx, s.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		suppliers := make([]SupplierResponse, 0, len(zeilen))
		for _, z := range zeilen {
			suppliers = append(suppliers, SupplierResponse(z))
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

		newID, erstelltAm, err := repository.LegeLieferantAn(ctx, s.DB.Pool, req.Name, req.Email, req.CustomerNumber, kundennummerSchultraeger(req))
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

	var req UpdateSupplierRequest
	// Streng: Ein vertippter Feldname nennte sonst nichts, und die Tür meldete Erfolg.
	if !DecodeStrictAndValidate(w, r, &req) {
		return
	}
	for _, wert := range []*string{req.Name, req.Email, req.CustomerNumber} {
		if wert != nil && *wert == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("name, email and customerNumber must not be empty"))
			return
		}
	}
	if req.KundennummerSchultraeger != nil {
		getrimmt := strings.TrimSpace(*req.KundennummerSchultraeger)
		req.KundennummerSchultraeger = &getrimmt
	}

	ctx := r.Context()
	alt, neu, err := s.aendereLieferant(ctx, id, req)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
		return
	}
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return
	}

	s.protokolliereGeaenderteFelder(ctx, auditLieferantGeaendert,
		map[string]any{"lieferant_id": id, "name": neu.Name},
		geaenderteFelder(
			feldWechsel{"name", alt.Name != neu.Name},
			feldWechsel{"email", alt.Email != neu.Email},
			feldWechsel{"kundennummer", alt.Kundennummer != neu.Kundennummer},
			feldWechsel{"kundennummer_schultraeger", alt.Zweitnummer != neu.Zweitnummer},
			feldWechsel{"hauptlieferant", alt.Haupt != neu.Haupt},
		))

	// Die Antwort nennt den gespeicherten Stand, nicht die Eingabe.
	RespondJSON(w, http.StatusOK, SupplierResponse{
		ID:                       id,
		Name:                     neu.Name,
		Email:                    neu.Email,
		CustomerNumber:           neu.Kundennummer,
		IstHauptlieferant:        neu.Haupt,
		KundennummerSchultraeger: neu.Zweitnummer,
	})
}

// aendereLieferant schreibt Merkmal und Stammdaten in einer Transaktion und liefert den Stand
// davor und danach. Bis zum 08.10.2026 waren es zwei Schritte: Scheiterte das Merkmal, waren
// die Stammdaten schon geschrieben.
//
// Zuerst das Merkmal, dann die Stammdaten: Der Setzer sperrt den bisherigen Hauptlieferanten
// vor der eigenen Zeile, wie beim Anlegen. Ein unbekannter Lieferant ist pgx.ErrNoRows; was der
// Setzer bis dahin geräumt hat, rollt mit zurück.
func (s *Server) aendereLieferant(ctx context.Context, id string, req UpdateSupplierRequest) (alt, neu repository.LieferantStand, err error) {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return alt, neu, err
	}
	defer db.SafeRollback(ctx, tx)

	merkmalGeaendert := false
	if req.IstHauptlieferant != nil {
		if *req.IstHauptlieferant {
			merkmalGeaendert, err = repository.SetzeHauptlieferant(ctx, tx, id)
		} else {
			// Abschalten ist erlaubt: „Kein Hauptlieferant" ist ein normaler Zustand.
			merkmalGeaendert, err = repository.NimmHauptlieferant(ctx, tx, id)
		}
		if err != nil {
			return alt, neu, err
		}
	}

	alt, neu, err = repository.AendereLieferantStammdaten(ctx, tx, id, req.Name, req.Email, req.CustomerNumber, req.KundennummerSchultraeger)
	if err != nil {
		return alt, neu, err
	}
	alt.Haupt = neu.Haupt != merkmalGeaendert
	return alt, neu, tx.Commit(ctx)
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
		istHaupt, name, err := repository.LieferantHauptUndName(ctx, s.DB.Pool, id)
		if err != nil {
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

		geloescht, err := repository.LoescheLieferant(ctx, s.DB.Pool, id)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if geloescht == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("supplier not found"))
			return
		}
		s.protokolliereVerwaltung(ctx, auditLieferantGeloescht,
			map[string]any{"lieferant_id": id, "name": name})

		w.WriteHeader(http.StatusNoContent)
	}
}

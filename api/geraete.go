package api

// geraete.go — Verwaltungs-Endpunkte der Geräteausleihe (Bereich im Medienkatalog).
//
// Rechte wie beim Buchbestand: Lesen view_books, Pflegen edit_books — Geräte SIND
// Bestand. Das G--Präfix ist Pflicht: Die Omnibox routet Scans darüber zum
// device_service; ein Gerät ohne G- wäre am Kiosk unauffindbar (die präfixlose
// Auflösung kennt nur Buch → Schüler → Lehrer → Suche).

import (
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// GeraetRequest ist die Eingabe für Anlegen und Pflegen.
type GeraetRequest struct {
	Modellname string `json:"modellname"`
	// Zeiger: Der Defekt-Knopf schickt die Seriennummer nicht mit, das Bearbeiten-
	// Formular schon. nil heisst "nicht angefasst", "" heisst "geloescht".
	Seriennummer  *string `json:"seriennummer,omitempty"`
	BarcodeID     string  `json:"barcode_id,omitempty"` // nur beim Anlegen; Barcodes kleben, sie wandern nicht
	Zubehoer      string  `json:"zubehoer"`
	ZustandNotiz  string  `json:"zustand_notiz,omitempty"`
	IstAusleihbar *bool   `json:"ist_ausleihbar,omitempty"`
}

// ListGeraeteHandler liefert die Geräteliste samt aktuellem Ausleiher.
// GET /api/geraete
func (s *Server) ListGeraeteHandler(repo repository.GeraeteRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		geraete, err := repo.ListGeraete(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Geräte", err)
		}
		// Die Route hängt an view_books (Katalog-Auskunft: "sind noch Geräte
		// da?"). Der NAME des Ausleihers ist aber eine Personenangabe und
		// bleibt Aufrufern ohne view_students verborgen — verliehen ja/nein
		// sieht man weiterhin (ausgeliehen_an wird zu "", nicht nil).
		if !s.BesitztRecht(r, "view_students") {
			maskiereAusleiher(geraete)
		}
		RespondJSON(w, http.StatusOK, map[string]any{"data": geraete})
		return nil
	})
}

// maskiereAusleiher ersetzt Ausleiher-Namen durch leere Strings, um den
// Datenschutz (fehlendes view_students-Recht) zu wahren, während der
// Ausleihstatus (verliehen ja/nein) erhalten bleibt.
func maskiereAusleiher(geraete []repository.GeraetMitStatus) {
	leer := ""
	for i := range geraete {
		if geraete[i].AusgeliehenAn != nil {
			geraete[i].AusgeliehenAn = &leer
		}
	}
}

// CreateGeraetHandler legt ein Gerät an. POST /api/geraete
func (s *Server) CreateGeraetHandler(repo repository.GeraeteRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req GeraetRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		req.Modellname = strings.TrimSpace(req.Modellname)
		req.BarcodeID = strings.TrimSpace(req.BarcodeID)
		if req.Modellname == "" {
			return apierrors.BadRequest("Ein Modellname ist erforderlich", nil)
		}
		if !strings.HasPrefix(req.BarcodeID, "G-") || len(req.BarcodeID) < 4 {
			return apierrors.BadRequest("Der Barcode muss mit G- beginnen (z. B. G-IPAD-01) — "+
				"nur so findet der Kiosk-Scan das Gerät", nil)
		}

		var seriennummer *string
		if getrimmt := getrimmterZeiger(req.Seriennummer); getrimmt != nil && *getrimmt != "" {
			seriennummer = getrimmt
		}
		id, err := repo.CreateGeraet(r.Context(), req.Modellname,
			seriennummer, req.BarcodeID, strings.TrimSpace(req.Zubehoer))
		if err != nil {
			if errors.Is(err, repository.ErrGeraetBarcodeVergeben) {
				return apierrors.Conflict(err.Error(), err)
			}
			return apierrors.Internal("Fehler beim Anlegen des Geräts", err)
		}
		RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
		return nil
	})
}

// UpdateGeraetHandler pflegt Stammdaten und den Defekt-Status.
// PUT /api/geraete/{id}
func (s *Server) UpdateGeraetHandler(repo repository.GeraeteRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req GeraetRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		req.Modellname = strings.TrimSpace(req.Modellname)
		if req.Modellname == "" {
			return apierrors.BadRequest("Ein Modellname ist erforderlich", nil)
		}
		// req.IstAusleihbar wird DURCHGEREICHT, nicht auf true vorbelegt: Ein fehlendes
		// Feld ist "unveraendert" und kein Wert. Vorher stand hier `istAusleihbar := true`,
		// und weil der Bearbeiten-Dialog das Kennzeichen nicht kennt (es liegt auf einem
		// eigenen Knopf), hob jede Stammdaten-Aenderung die Defekt-Markierung auf —
		// mit der Meldung "Geraet gespeichert" daneben.
		err := repo.UpdateGeraet(r.Context(), r.PathValue("id"), req.Modellname,
			strings.TrimSpace(req.Zubehoer), nullableString(strings.TrimSpace(req.ZustandNotiz)),
			getrimmterZeiger(req.Seriennummer), req.IstAusleihbar)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Gerät nicht gefunden", err)
			}
			return apierrors.Internal("Fehler beim Speichern des Geräts", err)
		}
		RespondSuccess(w)
		return nil
	})
}

// getrimmterZeiger reicht einen optionalen Textwert getrimmt weiter: nil bleibt nil
// ("nicht mitgeschickt"), ein leerer Wert wird zu einem Zeiger auf "" ("geloescht").
func getrimmterZeiger(wert *string) *string {
	if wert == nil {
		return nil
	}
	getrimmt := strings.TrimSpace(*wert)
	return &getrimmt
}

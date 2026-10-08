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

// GeraetRequest sind die Angaben eines neuen Geräts.
type GeraetRequest struct {
	Modellname   string  `json:"modellname"`
	Seriennummer *string `json:"seriennummer,omitempty"`
	BarcodeID    string  `json:"barcode_id,omitempty"`
	Zubehoer     string  `json:"zubehoer"`
	ZustandNotiz string  `json:"zustand_notiz,omitempty"`
}

// GeraetAenderungRequest nennt, was an einem Gerät geändert wird. Ein fehlendes Feld bleibt,
// wie es ist: Der Bearbeiten-Dialog schickt nur, was seit dem Öffnen geändert wurde, der
// Defekt-Knopf nur das Kennzeichen. Ein unbekanntes Feld lehnt die Tür ab. barcode_id nimmt sie
// an und liest es nicht: Barcodes kleben, sie wandern nicht, und ein Dialog aus einem älteren
// Stand schickt das Feld noch mit.
type GeraetAenderungRequest struct {
	Modellname    *string `json:"modellname"`
	Seriennummer  *string `json:"seriennummer"`
	BarcodeID     *string `json:"barcode_id"`
	Zubehoer      *string `json:"zubehoer"`
	ZustandNotiz  *string `json:"zustand_notiz"`
	IstAusleihbar *bool   `json:"ist_ausleihbar"`
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
			leer := ""
			for i := range geraete {
				if geraete[i].AusgeliehenAn != nil {
					geraete[i].AusgeliehenAn = &leer
				}
			}
		}
		RespondJSON(w, http.StatusOK, map[string]any{"data": geraete})
		return nil
	})
}

// CreateGeraetHandler legt ein Gerät an. POST /api/geraete
func (s *Server) CreateGeraetHandler(repo repository.GeraeteRepository) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req GeraetRequest
		// Streng: Ein Feld, das die Tür nicht liest, ist ein Fehler und kein stiller Verlust.
		if !DecodeStrictAndValidate(w, r, &req) {
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
			seriennummer, req.BarcodeID, strings.TrimSpace(req.Zubehoer), strings.TrimSpace(req.ZustandNotiz))
		if err != nil {
			if istGeraetKonflikt(err) {
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
		var req GeraetAenderungRequest
		// Streng: Ein vertippter Feldname nennte sonst nichts, und die Tür meldete Erfolg.
		if !DecodeStrictAndValidate(w, r, &req) {
			return nil
		}
		aenderung := repository.GeraetAenderung{
			Modellname:   getrimmterZeiger(req.Modellname),
			Zubehoer:     getrimmterZeiger(req.Zubehoer),
			ZustandNotiz: getrimmterZeiger(req.ZustandNotiz),
			Seriennummer: getrimmterZeiger(req.Seriennummer),
			// Durchgereicht, nicht vorbelegt: Ein fehlendes Kennzeichen ist „unverändert". Mit
			// einer Vorgabe hob jede Änderung der Stammdaten die Defekt-Markierung auf.
			IstAusleihbar: req.IstAusleihbar,
		}
		if aenderung.Modellname != nil && *aenderung.Modellname == "" {
			return apierrors.BadRequest("Ein Modellname ist erforderlich", nil)
		}
		err := repo.UpdateGeraet(r.Context(), r.PathValue("id"), aenderung)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Gerät nicht gefunden", err)
			}
			if istGeraetKonflikt(err) {
				return apierrors.Conflict(err.Error(), err)
			}
			return apierrors.Internal("Fehler beim Speichern des Geräts", err)
		}
		RespondSuccess(w)
		return nil
	})
}

// istGeraetKonflikt erkennt die Eingabe, die an einem anderen Gerät schon steht: Barcode
// oder Seriennummer. Der Satz des Fehlers nennt das Feld.
func istGeraetKonflikt(err error) bool {
	return errors.Is(err, repository.ErrGeraetBarcodeVergeben) ||
		errors.Is(err, repository.ErrGeraetSeriennummerVergeben)
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

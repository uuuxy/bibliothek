package katalog

// Die Pflege der Geräte, die die Bücherei verleiht (ein Bereich im Medienkatalog).
//
// Rechte wie beim Buchbestand: Lesen view_books, Pflegen edit_books, denn Geräte sind Bestand.
// Die Vorsilbe G- ist Pflicht: Die Omnibox leitet einen Scan darüber zum Gerätedienst
// (internal/service/geraet.go); ein Gerät ohne G- fände die Theke nicht, weil die Auflösung
// ohne Vorsilbe nur Buch, Ausweis und Suche kennt.

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/auth"
	"bibliothek/internal/repository"

	"github.com/jackc/pgx/v5"
)

// geraetAnlegenAnfrage sind die Angaben eines neuen Geräts.
type geraetAnlegenAnfrage struct {
	Modellname   string  `json:"modellname"`
	Seriennummer *string `json:"seriennummer,omitempty"`
	BarcodeID    string  `json:"barcode_id,omitempty"`
	Zubehoer     string  `json:"zubehoer"`
	ZustandNotiz string  `json:"zustand_notiz,omitempty"`
}

// geraetAendernAnfrage nennt, was an einem Gerät geändert wird. Ein fehlendes Feld bleibt,
// wie es ist: Der Bearbeiten-Dialog schickt nur, was seit dem Öffnen geändert wurde, der
// Defekt-Knopf nur das Kennzeichen. Ein unbekanntes Feld lehnt die Tür ab. barcode_id nimmt sie
// an und liest es nicht: Barcodes kleben, sie wandern nicht, und ein Dialog aus einem älteren
// Stand schickt das Feld noch mit.
type geraetAendernAnfrage struct {
	Modellname    *string `json:"modellname"`
	Seriennummer  *string `json:"seriennummer"`
	BarcodeID     *string `json:"barcode_id"`
	Zubehoer      *string `json:"zubehoer"`
	ZustandNotiz  *string `json:"zustand_notiz"`
	IstAusleihbar *bool   `json:"ist_ausleihbar"`
}

// geraeteListe liefert alle Geräte im Bestand, das neueste zuerst, mit dem Namen dessen, der das
// Gerät gerade hat (repository.GeraeteAbfragen).
type geraeteListe interface {
	LadeAlle(ctx context.Context) ([]repository.GeraetMitStatus, error)
}

// Geraete liefert die Geräteliste samt aktuellem Ausleiher.
// GET /api/geraete
func (t *Tueren) Geraete(geraeteAbfragen geraeteListe) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		geraete, err := geraeteAbfragen.LadeAlle(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Geräte", err)
		}
		// Die Route hängt an view_books (Auskunft des Katalogs: „sind noch Geräte da?"). Der
		// Name des Ausleihers ist eine Personenangabe und bleibt Aufrufern ohne view_students
		// verborgen; ob ein Gerät verliehen ist, sieht man weiterhin (ausgeliehen_an wird zu "",
		// nicht nil).
		if !t.HatRecht(r, "view_students") {
			leer := ""
			for i := range geraete {
				if geraete[i].AusgeliehenAn != nil {
					geraete[i].AusgeliehenAn = &leer
				}
			}
		}
		tuer.AntworteJSON(w, http.StatusOK, geraete)
		return nil
	})
}

// GeraetAnlegen legt ein Gerät an. POST /api/geraete
func (t *Tueren) GeraetAnlegen(geraeteAbfragen *repository.GeraeteAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		var req geraetAnlegenAnfrage
		// Streng: Ein Feld, das die Tür nicht liest, ist ein Fehler und kein stiller Verlust.
		if !tuer.LeseAnfrageStreng(w, r, &req) {
			return nil
		}
		req.Modellname = strings.TrimSpace(req.Modellname)
		req.BarcodeID = strings.TrimSpace(req.BarcodeID)
		if req.Modellname == "" {
			return apierrors.BadRequest("Ein Modellname ist erforderlich", nil)
		}
		if !strings.HasPrefix(req.BarcodeID, "G-") || len(req.BarcodeID) < 4 {
			return apierrors.BadRequest("Der Barcode muss mit G- beginnen, etwa G-IPAD-01. "+
				"Nur so findet der Scan an der Theke das Gerät.", nil)
		}

		var seriennummer *string
		if getrimmt := getrimmterZeiger(req.Seriennummer); getrimmt != nil && *getrimmt != "" {
			seriennummer = getrimmt
		}
		id, err := geraeteAbfragen.LegeAn(r.Context(), req.Modellname,
			seriennummer, req.BarcodeID, strings.TrimSpace(req.Zubehoer), strings.TrimSpace(req.ZustandNotiz))
		if err != nil {
			if meldung, ok := geraetKonflikt(err); ok {
				return apierrors.Conflict(meldung, err)
			}
			return apierrors.Internal("Fehler beim Anlegen des Geräts", err)
		}
		tuer.AntworteJSON(w, http.StatusCreated, map[string]string{"id": id})
		return nil
	})
}

// GeraetAendern pflegt Stammdaten und den Defekt-Status und holt ein verlorenes Gerät zurück.
// PUT /api/geraete/{id}
func (t *Tueren) GeraetAendern(geraeteAbfragen *repository.GeraeteAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		return t.aendereGeraet(w, r, geraeteAbfragen)
	})
}

// aendereGeraet ist der Rumpf von GeraetAendern.
func (t *Tueren) aendereGeraet(w http.ResponseWriter, r *http.Request, geraeteAbfragen *repository.GeraeteAbfragen) error {
	var req geraetAendernAnfrage
	// Streng: Ein vertippter Feldname nennte sonst nichts, und die Tür meldete Erfolg.
	if !tuer.LeseAnfrageStreng(w, r, &req) {
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
	// „Wieder freigeben“ an einem verlorenen Gerät holt es zurück: Dieselbe Tür macht ein
	// defektes und ein wiedergefundenes Gerät ausleihbar, die Rückkehr bucht der Server.
	var befund repository.RueckkehrBefund
	zurueck := false
	if req.IstAusleihbar != nil && *req.IstAusleihbar {
		var err error
		if befund, zurueck, err = t.holeVerlorenesGeraetZurueck(r); err != nil {
			return err
		}
	}
	if err := geraeteAbfragen.Aendere(r.Context(), r.PathValue("id"), aenderung); err != nil {
		return geraetAenderungsFehler(err)
	}
	if zurueck {
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{"status": "success", "zurueckgeholt": true,
			"stornierte_forderungen": befund.StornierteForderungen, "stornierter_betrag": befund.StornierterBetrag})
		return nil
	}
	tuer.AntworteErfolg(w)
	return nil
}

// holeVerlorenesGeraetZurueck holt das Gerät der Anfrage zurück, wenn es als verloren gemeldet
// ist (repository.HoleVerlorenesGeraetZurueck); zurueck ist false, wenn es nicht verloren war.
func (t *Tueren) holeVerlorenesGeraetZurueck(r *http.Request) (befund repository.RueckkehrBefund, zurueck bool, err error) {
	bearbeiter := ""
	if claims, ok := auth.ClaimsAusKontext(r.Context()); ok {
		bearbeiter = claims.KontoID
	}
	zurueck, befund, err = repository.HoleVerlorenesGeraetZurueck(r.Context(), t.DB.Pool, r.PathValue("id"), bearbeiter)
	switch {
	case errors.Is(err, repository.ErrAussonderungOhneBearbeiter):
		return befund, false, apierrors.Unauthorized(auth.MeldungNichtAngemeldet, err)
	case err != nil:
		return befund, false, apierrors.Internal("Fehler beim Zurückholen des Geräts", err)
	}
	return befund, zurueck, nil
}

// geraetAenderungsFehler ordnet den Fehler einer Änderung Status und Satz zu.
func geraetAenderungsFehler(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apierrors.NotFound("Gerät nicht gefunden", err)
	}
	if meldung, ok := geraetKonflikt(err); ok {
		return apierrors.Conflict(meldung, err)
	}
	return apierrors.Internal("Fehler beim Speichern des Geräts", err)
}

// geraetKonflikt erkennt die Eingabe, die an einem anderen Gerät schon steht, und nennt im Satz
// das Feld: Barcode oder Seriennummer. ok ist false für jeden anderen Fehler.
func geraetKonflikt(err error) (meldung string, ok bool) {
	switch {
	case errors.Is(err, repository.ErrGeraetBarcodeVergeben):
		return "Barcode ist bereits an ein anderes Gerät vergeben", true
	case errors.Is(err, repository.ErrGeraetSeriennummerVergeben):
		return "Seriennummer ist bereits an einem anderen Gerät eingetragen", true
	}
	return "", false
}

// getrimmterZeiger reicht einen optionalen Textwert getrimmt weiter: nil bleibt nil
// („nicht mitgeschickt"), ein leerer Wert wird zu einem Zeiger auf "" („gelöscht").
func getrimmterZeiger(wert *string) *string {
	if wert == nil {
		return nil
	}
	getrimmt := strings.TrimSpace(*wert)
	return &getrimmt
}

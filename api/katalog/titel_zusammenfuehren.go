package katalog

import (
	"errors"
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/repository"
)

// Zwei Titel desselben Buchs zu einem. Regeln und SQL stehen in
// repository/titel_zusammenfuehren.go; hier nur Recht, Rumpf, Fehlerbild und Protokoll. Das
// Recht ist delete_books wie beim Löschen eines Titels: Danach gibt es einen der beiden nicht mehr.

// meldungZweiISBN lehnt zwei Titel ab, die beide eine ISBN tragen.
const meldungZweiISBN = "Beide Titel tragen eine ISBN, sind also zwei Ausgaben. Zwei Ausgaben bleiben eigene Titel; für sie gibt es „Andere Auflage zuordnen“."

type titelZusammenfuehrenAnfrage struct {
	QuelleID string `json:"quelle_id" validate:"required,uuid_oder_leer"`
}

// TitelZusammenfuehrungVorschau bedient GET /api/buecher/titel/{id}/zusammenfuehren?quelle_id=:
// was das Zusammenführen täte, ohne es zu tun. Die Vorschau läuft durch denselben Weg und nimmt
// ihn am Ende zurück (repository.TitelZusammenfuehrenAuftrag.NurVorschau).
func (t *Tueren) TitelZusammenfuehrungVorschau() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		quelleID, err := tuer.LeseUUIDParameter(r, "quelle_id")
		if err != nil {
			return err
		}
		erg, err := t.fuehreTitelZusammen(r, quelleID, true)
		if err != nil {
			return err
		}
		tuer.AntworteJSON(w, http.StatusOK, erg)
		return nil
	})
}

// FuehreTitelZusammen bedient POST /api/buecher/titel/{id}/zusammenfuehren: {id} bleibt,
// quelle_id geht darin auf.
func (t *Tueren) FuehreTitelZusammen(protokollAbfragen *repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		var rumpf titelZusammenfuehrenAnfrage
		if !tuer.LeseAnfrage(w, r, &rumpf) {
			return nil
		}
		erg, err := t.fuehreTitelZusammen(r, rumpf.QuelleID, false)
		if err != nil {
			return err
		}
		// Das Protokoll der Verwaltung nennt, welcher Titel in welchem aufging; den Rückweg mit
		// allen gewanderten Zeilen schreibt die Transaktion in audit_log.
		if claims, ok := auth.ClaimsAusKontext(r.Context()); ok {
			if err := protokollAbfragen.ProtokolliereAdminAktion(r.Context(), claims.KontoID, "TITEL_ZUSAMMENGEFUEHRT", clientip.AusAnfrage(r), map[string]any{
				"titel_id":         erg.TitelID,
				"titel":            erg.Titel,
				"aufgeloest_id":    erg.QuelleID,
				"aufgeloest_titel": erg.QuelleTitel,
				"exemplare":        erg.Exemplare,
			}); err != nil {
				log.Printf("Titel zusammenführen: Eintrag im Protokoll der Verwaltung fehlgeschlagen: %v", err)
			}
		}
		tuer.AntworteJSON(w, http.StatusOK, erg)
		return nil
	})
}

// fuehreTitelZusammen prüft Kennungen und Anmeldung und übersetzt die fachlichen Fehler.
func (t *Tueren) fuehreTitelZusammen(r *http.Request, quelleID string, nurVorschau bool) (*repository.TitelZusammenfuehrung, error) {
	zielID, err := titelIDAusPfad(r)
	if err != nil {
		return nil, err
	}
	if quelleID == "" {
		return nil, apierrors.BadRequest("Die Kennung des anderen Titels fehlt", nil)
	}
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		return nil, apierrors.Unauthorized(auth.MeldungNichtAngemeldet, nil)
	}
	erg, err := repository.FuehreTitelZusammen(r.Context(), t.DB.Pool, repository.TitelZusammenfuehrenAuftrag{
		ZielID: zielID, QuelleID: quelleID,
		BearbeiterID: claims.KontoID, NurVorschau: nurVorschau,
	})
	switch {
	case errors.Is(err, repository.ErrTitelZusammenfuehrenGleich):
		return nil, apierrors.BadRequest("Ein Titel lässt sich nicht mit sich selbst zusammenführen", err)
	case errors.Is(err, repository.ErrTitelNichtGefunden):
		return nil, apierrors.NotFound(tuer.MeldungTitelNichtGefunden, err)
	case errors.Is(err, repository.ErrTitelZusammenfuehrenZweiISBN):
		return nil, apierrors.Conflict(meldungZweiISBN, err)
	case err != nil:
		return nil, apierrors.Internal("Zusammenführen fehlgeschlagen", err)
	}
	return erg, nil
}

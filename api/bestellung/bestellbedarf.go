package bestellung

import (
	"context"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// Bestellbedarf liefert den Bestellbedarf.
//
// Default ist der LMF-Bestand (Lernmittel): Nachbestellt werden praktisch nur
// Lernmittel-Klassensätze; im Freihandbestand steht meist ein einzelnes Prüf- oder
// Leseexemplar, das bewusst ein Einzelstück bleibt. Ohne diese Vorauswahl bestand die
// Liste zu ~99% aus Titeln, die niemand nachbestellen will — bei realem Bestand
// tausende Einträge, die die Ansicht unbenutzbar machten.
//
// ?type=freihand oder ?type=alle bleiben möglich; für Einzelfälle ausserhalb der
// Lernmittel gibt es ausserdem die Titelsuche im Bestellworkspace.
func (t *Tueren) Bestellbedarf() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bedarf, err := t.bestellbedarfMitEinstellungen(r.Context(), r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		tuer.AntworteJSON(w, http.StatusOK, bedarf)
	}
}

// bestellbedarfMitEinstellungen liefert den Bestellbedarf gemäß Konfiguration: Warnung aus →
// leere Liste; sonst gefiltert auf gesamt < Schwelle. Eine Quelle für Ansicht UND
// PDF-Export, damit beide nie auseinanderlaufen.
func (t *Tueren) bestellbedarfMitEinstellungen(ctx context.Context, r *http.Request) ([]repository.Bestellbedarf, error) {
	einstellungen, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return nil, err
	}
	if !einstellungen.BestellbedarfWarnungAktiv {
		return []repository.Bestellbedarf{}, nil
	}
	return repository.LadeBestellbedarf(ctx, t.DB.Pool, bestellbedarfFilter(r), einstellungen.BestellbedarfSchwelle)
}

// bestellbedarfFilter liest ?type= und fällt auf den LMF-Bestand zurück (siehe
// Bestellbedarf). Das SQL-Fragment ist serverkontrolliert, der Parameter wählt
// nur zwischen festen Varianten.
func bestellbedarfFilter(r *http.Request) string {
	typ := r.URL.Query().Get("type")
	if typ == "" {
		typ = "lmf"
	}
	fragment, _ := repository.BestandsFilterBedingung(typ)
	return fragment
}

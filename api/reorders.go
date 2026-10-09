package api

import (
	"context"
	"net/http"

	"bibliothek/apierrors"
	"bibliothek/repository"
)

// GetReordersHandler liefert den Bestellbedarf.
//
// Default ist der LMF-Bestand (Lernmittel): Nachbestellt werden praktisch nur
// Lernmittel-Klassensätze; im Freihandbestand steht meist ein einzelnes Prüf- oder
// Leseexemplar, das bewusst ein Einzelstück bleibt. Ohne diese Vorauswahl bestand die
// Liste zu ~99% aus Titeln, die niemand nachbestellen will — bei realem Bestand
// tausende Einträge, die die Ansicht unbenutzbar machten.
//
// ?type=freihand oder ?type=alle bleiben möglich; für Einzelfälle ausserhalb der
// Lernmittel gibt es ausserdem die Titelsuche im Bestellworkspace.
func (s *Server) GetReordersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reorders, err := s.reordersMitSettings(r.Context(), r)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		RespondJSON(w, http.StatusOK, reorders)
	}
}

// reordersMitSettings liefert den Bestellbedarf gemäß Konfiguration: Warnung aus →
// leere Liste; sonst gefiltert auf gesamt < Schwelle. Eine Quelle für Ansicht UND
// PDF-Export, damit beide nie auseinanderlaufen.
func (s *Server) reordersMitSettings(ctx context.Context, r *http.Request) ([]repository.ReorderTitle, error) {
	settings, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.BestellbedarfWarnungAktiv {
		return []repository.ReorderTitle{}, nil
	}
	return repository.ListeBestellbedarf(ctx, s.DB.Pool, reorderFilter(r), settings.BestellbedarfSchwelle)
}

// reorderFilter liest ?type= und fällt auf den LMF-Bestand zurück (siehe
// GetReordersHandler). Das SQL-Fragment ist serverkontrolliert, der Parameter wählt
// nur zwischen festen Varianten.
func reorderFilter(r *http.Request) string {
	typ := r.URL.Query().Get("type")
	if typ == "" {
		typ = "lmf"
	}
	fragment, _ := repository.BestandsFilterBedingung(typ)
	return fragment
}

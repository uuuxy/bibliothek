package bestellung

import (
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"
)

// bestellKonfiguration nennt die Systemeinstellungen, nach denen sich die Oberfläche des
// Bestellwesens richtet.
type bestellKonfiguration struct {
	PreiseErfassen bool `json:"preise_erfassen"`
	// BestelllinkOhneAdresse: Es gibt einen Hauptlieferanten, aber keine öffentliche
	// Adresse — seine Bestellmails gehen also ohne Bestätigungs-Link raus.
	//
	// Beides zusammen ergibt erst die Aussage: Ohne Hauptlieferant verschickt das System
	// gar keine Links, dann fehlt auch nichts. Die Oberfläche soll das vor dem Bestellen
	// sagen können, nicht erst als Warnung hinterher (siehe bestellVersandMeldung).
	BestelllinkOhneAdresse bool `json:"bestelllink_ohne_adresse"`
}

// BestellKonfiguration liefert die Anzeige-Regeln des Bestellwesens.
//
// Eigene, schmale Route statt GET /api/einstellungen: Die vollständigen Einstellungen
// verlangen manage_settings, das Bestellwesen aber nur view_orders. Wer bestellen darf,
// müsste sonst das Recht auf alle Einstellungen bekommen, nur damit sein Warenkorb weiss, ob
// er ein Preisfeld zeigen soll; ein Recht auszuweiten, um eine Anzeige zu steuern, ist der
// falsche Weg herum.
//
// Zugeschnitten auf das Bestellwesen und auch so benannt: Braucht später eine andere
// Ansicht eine Anzeige-Regel, gehört sie nicht hierher, sondern an ihre eigene Domäne —
// sonst wächst hier still ein zweiter Einstellungs-Endpunkt heran.
func (t *Tueren) BestellKonfiguration(einstellungsAbfragen *repository.EinstellungsAbfragen, lieferantenAbfragen *repository.LieferantenAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		einstellungen, err := einstellungsAbfragen.Lade(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Bestell-Konfiguration", err)
		}

		hauptlieferant, err := lieferantenAbfragen.HatHauptlieferant(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Bestell-Konfiguration", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, bestellKonfiguration{
			PreiseErfassen:         einstellungen.PreiseErfassen,
			BestelllinkOhneAdresse: hauptlieferant && einstellungen.AdresseFuerLinks() == "",
		})
		return nil
	})
}

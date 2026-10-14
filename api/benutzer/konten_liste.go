package benutzer

// konten_liste.go — Handlers for system user and role-permission management.
// Covers: listing/creating/updating/deleting staff accounts and reading/writing
// the role_permissions table.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
)

// kontoAntwort holds public user data sent to administrative screens.
type kontoAntwort struct {
	ID         string    `json:"id"`
	BarcodeID  string    `json:"barcode_id"`
	Vorname    string    `json:"vorname"`
	Nachname   string    `json:"nachname"`
	Email      string    `json:"email"`
	Rolle      string    `json:"rolle"`
	Aktiv      bool      `json:"aktiv"`
	ErstelltAm time.Time `json:"erstellt_am"`
	// ZugangBeantragtAm: offener Antrag aus der Selbstanmeldung (Migration 086);
	// null = keiner. Die Oberfläche unterscheidet daran „wartet" von „deaktiviert".
	ZugangBeantragtAm *time.Time `json:"zugang_beantragt_am"`
	// LeserID ist die Leserzeile dieses Kontos (Migration 125). Die Freischaltungs-Zeile
	// braucht sie, um von einem offenen Antrag aus nach der gleichnamigen Leserzeile zu
	// suchen, die noch KEIN Konto hat — der Altbestand, bei dem die Selbstanmeldung sonst
	// einen zweiten Eintrag erzeugt (UserManagementZugangsanfragen.svelte).
	LeserID string   `json:"leser_id"`
	Rechte  []string `json:"permissions"`
}

// Konten returns a list of all system users.
// @Summary      List system users
// @Description  Retrieves all administrative and staff users registered in the system.
// @Tags         admin
// @Accept       json
// @Produce      json
// @Success      200  {array}   kontoAntwort
// @Failure      500  {object}  map[string]string
// @Router       /benutzer [get]
func (t *Tueren) Konten(kontoAbfragen repository.KontoAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		konten, err := kontoAbfragen.LadeAlle(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		// EINMAL fuer alle Rollen laden, nicht je Benutzer: Sonst wird aus einer Liste
		// mit zwanzig Eintraegen eine Abfragelawine gegen dieselben vier Rollen.
		rechteJeRolle, err := t.rechteAllerRollen(ctx)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		liste := []kontoAntwort{}
		for _, konto := range konten {
			eintrag := kontoAntwort{
				ID:         konto.ID,
				BarcodeID:  konto.BarcodeID,
				Vorname:    konto.Vorname,
				Nachname:   konto.Nachname,
				Email:      konto.Email,
				Rolle:      strings.ToLower(konto.Rolle),
				Aktiv:      konto.Aktiv,
				ErstelltAm: konto.ErstelltAm,

				ZugangBeantragtAm: konto.ZugangBeantragtAm,
				LeserID:           konto.LeserID,
			}

			// Die echten Rechte aus role_permissions, nicht mehr eine feste Liste.
			//
			// Hier stand eine hartkodierte Zuordnung, die sich „analog zum Login" nannte,
			// es aber nicht war: Sie nannte manage_settings, print_classes und view_media —
			// Rechte, die es im System gar nicht gibt (kein RequirePermission, kein Seed).
			// Wer die Benutzerliste ansah, bekam also erfundene Angaben, waehrend Login und
			// /api/auth/me die tatsaechlichen Rechte laden. Audit-Befund vom 01.08.2026.
			eintrag.Rechte = rechteJeRolle[eintrag.Rolle]
			if eintrag.Rechte == nil {
				eintrag.Rechte = []string{}
			}

			liste = append(liste, eintrag)
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// rechteAllerRollen liefert die freigeschalteten Rechte je Rolle aus role_permissions —
// derselben Tabelle, aus der auch Login und RequirePermission lesen.
//
// Admin bekommt "*" wie im Login und im Middleware-Bypass; ein Admin hat implizit alle
// Rechte, unabhaengig davon, was in der Tabelle steht.
//
// Die Rollen kommen kleingeschrieben zurueck, passend zu UserResponse.Rolle.
func (t *Tueren) rechteAllerRollen(ctx context.Context) (map[string][]string, error) {
	rechte, err := repository.LadeErlaubteRechteJeRolle(ctx, t.DB.Pool)
	if err != nil {
		return nil, err
	}
	rechte[strings.ToLower(string(auth.RoleAdmin))] = []string{"*"}
	return rechte, nil
}

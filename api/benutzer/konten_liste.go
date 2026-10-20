package benutzer

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

// kontoAntwort ist ein Konto, wie die Benutzerverwaltung es zeigt.
type kontoAntwort struct {
	ID         string    `json:"id"`
	BarcodeID  string    `json:"barcode_id"`
	Vorname    string    `json:"vorname"`
	Nachname   string    `json:"nachname"`
	Email      string    `json:"email"`
	Rolle      string    `json:"rolle"`
	Aktiv      bool      `json:"aktiv"`
	ErstelltAm time.Time `json:"erstellt_am"`
	// ZugangBeantragtAm ist der offene Antrag aus der Selbstanmeldung, null heißt keiner. Die
	// Oberfläche unterscheidet daran „wartet" von „deaktiviert".
	ZugangBeantragtAm *time.Time `json:"zugang_beantragt_am"`
	// LeserID ist die Leserzeile dieses Kontos. Die Zeile zum Freischalten braucht sie, um von
	// einem offenen Antrag aus nach der gleichnamigen Leserzeile zu suchen, die noch kein Konto
	// hat: der Altbestand, bei dem die Selbstanmeldung sonst einen zweiten Eintrag erzeugt
	// (Zugangsanfragen.svelte).
	LeserID string   `json:"leser_id"`
	Rechte  []string `json:"permissions"`
}

// Konten liefert alle Konten mit den Rechten ihrer Rolle.
func (t *Tueren) Konten(kontoAbfragen *repository.KontoAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		konten, err := kontoAbfragen.LadeAlle(ctx)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		// Einmal für alle Rollen laden, nicht je Konto: Sonst würde aus einer Liste mit zwanzig
		// Einträgen eine Lawine von Abfragen an dieselben vier Rollen.
		rechteJeRolle, err := t.rechteAllerRollen(ctx)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
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

			// Die Rechte kommen aus role_permissions wie bei der Anmeldung und /api/auth/me,
			// damit die Liste die Rechte zeigt, die gelten.
			eintrag.Rechte = rechteJeRolle[eintrag.Rolle]
			if eintrag.Rechte == nil {
				eintrag.Rechte = []string{}
			}

			liste = append(liste, eintrag)
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// rechteAllerRollen liefert die freigeschalteten Rechte je Rolle aus role_permissions,
// derselben Tabelle, aus der Anmeldung und verlangeRecht lesen.
//
// Admin bekommt "*" wie bei der Anmeldung und in der Prüfung der Rechte: Ein Admin hat alle
// Rechte, gleich was in der Tabelle steht. Die Rollen kommen klein geschrieben zurück, wie
// kontoAntwort.Rolle.
func (t *Tueren) rechteAllerRollen(ctx context.Context) (map[string][]string, error) {
	rechte, err := repository.LadeErlaubteRechteJeRolle(ctx, t.DB.Pool)
	if err != nil {
		return nil, err
	}
	rechte[strings.ToLower(string(auth.RolleAdmin))] = []string{"*"}
	return rechte, nil
}

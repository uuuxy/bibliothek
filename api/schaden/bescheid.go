package schaden

import (
	"context"
	"errors"
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// Die Türen des Schadensersatz-Bescheids.
//
// Der Ablauf hat drei Schritte, und jeder ist ein eigener Aufruf: Der Dialog holt den Vorschlag
// (offene Forderungen mit Staffel-Betrag, bescheid_vorschlag.go), der Mensch prüft und erstellt
// (bescheid_erstellen.go), und danach lässt sich der Brief nachdrucken (bescheid_druck.go). Nach
// Fristablauf kommt die Übergabe dazu. Hier stehen die Listen, die Übergabe und was mehrere
// Türen brauchen; was der Brief einfriert, steht in bescheid_schnappschuss.go.
//
// Nichts davon läuft automatisch: Jede Referenznummer wird unwiderruflich verbraucht, und
// jeder Betrag ist eine Entscheidung der Schule.

// aktionBescheidErstellt und aktionBescheidUebergeben sind die Aktionen im Protokoll der
// Verwaltung.
const (
	aktionBescheidErstellt   = "SCHADENSERSATZ_BESCHEID_ERSTELLT"
	aktionBescheidUebergeben = "SCHADENSERSATZ_BESCHEID_UEBERGEBEN"
)

// bescheidAngaben liest Einstellungen und Schul-Identität in einem Zug.
func (t *Tueren) bescheidAngaben(ctx context.Context) (repository.BescheidAngaben, repository.SchuleAngaben, error) {
	einst, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil {
		return repository.BescheidAngaben{}, repository.SchuleAngaben{}, err
	}
	return repository.BescheidAngabenAus(einst), repository.SchuleAngabenAus(einst), nil
}

// bescheidAkteur liefert die ID des angemeldeten Bearbeiters (leer ohne Anmeldung).
func bescheidAkteur(ctx context.Context) string {
	if claims, ok := auth.ClaimsAusKontext(ctx); ok {
		return claims.KontoID
	}
	return ""
}

// protokolliereBescheid protokolliert einen Eingriff. Dasselbe Muster wie der LMF-Plan: ohne
// Anmeldung oder ohne Datenbank gibt es nichts zu schreiben.
func protokolliereBescheid(ctx context.Context, t *Tueren, protokollAbfragen *repository.ProtokollAbfragen, aktion string, details map[string]any, ip string) {
	claims, ok := auth.ClaimsAusKontext(ctx)
	if !ok || t.DB == nil || t.DB.Pool == nil || protokollAbfragen == nil {
		return
	}
	if err := protokollAbfragen.ProtokolliereAdminAktion(ctx, claims.KontoID, aktion, ip, details); err != nil {
		log.Printf("Audit %s fehlgeschlagen: %v", aktion, err)
	}
}

// bescheidListe liefert die Bescheide, die mit abgelaufener Frist zuerst, danach die neuesten;
// nurOffen beschränkt auf die, die weder übergeben noch erledigt sind
// (repository.BescheidAbfragen).
type bescheidListe interface {
	LadeListe(ctx context.Context, nurOffen bool) ([]repository.Bescheid, error)
}

// Bescheide liefert die Bescheide — die Arbeitsliste des Sekretariats.
func (t *Tueren) Bescheide(bescheidAbfragen bescheidListe) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		liste, err := bescheidAbfragen.LadeListe(r.Context(), r.URL.Query().Get("status") == "offen")
		if err != nil {
			return apierrors.Internal("Bescheide konnten nicht gelesen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}

// BescheideDesSchuelers liefert die Bescheide eines Schülers für die Akte.
func (t *Tueren) BescheideDesSchuelers(bescheidAbfragen *repository.BescheidAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}
		liste, err := bescheidAbfragen.LadeZuSchueler(r.Context(), id)
		if err != nil {
			return apierrors.Internal("Bescheide konnten nicht gelesen werden", err)
		}
		tuer.AntworteJSON(w, http.StatusOK, liste)
		return nil
	})
}

// BescheidUebergeben setzt einen Bescheid nach Fristablauf auf „übergeben".
//
// Kein Cron: Die Übergabe ist eine Handlung des Sekretariats (Original und Buchungsbeleg
// gehen aus dem Haus). Eine Automatik über Nacht wäre eine stille Zustandsänderung.
func (t *Tueren) BescheidUebergeben(bescheidAbfragen *repository.BescheidAbfragen, protokollAbfragen *repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest(tuer.MeldungIDLeer, errors.New("missing id"))
		}
		ctx := r.Context()
		if err := bescheidAbfragen.Uebergebe(ctx, id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Entweder gibt es den Bescheid nicht, oder sein Zustand passt nicht.
				// Die Meldung nennt beides, statt einen 404 zu behaupten.
				return apierrors.Conflict(
					"Übergabe nicht möglich: Der Bescheid ist nicht (mehr) offen oder seine Frist läuft noch.",
					err)
			}
			return apierrors.Internal("Übergabe konnte nicht gebucht werden", err)
		}

		protokolliereBescheid(ctx, t, protokollAbfragen, aktionBescheidUebergeben, map[string]any{"bescheid_id": id}, clientip.AusAnfrage(r))
		tuer.AntworteJSON(w, http.StatusOK, map[string]any{"id": id, "status": "uebergeben"})
		return nil
	})
}

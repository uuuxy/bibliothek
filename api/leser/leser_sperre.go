package leser

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/httpresp"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// meldungSperreFehler geht hinaus, wenn das Umschalten an der Datenbank scheitert.
const meldungSperreFehler = "Fehler beim Aktualisieren der Sperre"

// LeserSperren sperrt einen Leser von Hand oder hebt jede Sperre an ihm auf — die
// eine Tür zu diesem Zustand (schueler_sperre_eine_tuer_pg_test.go).
//
// Aufheben nimmt beide Sperren weg: die von Hand und die, die das Programm den Ehemaligen
// setzt (ist_gesperrt: Versetzung, LUSD-Import). Beide lassen an der Theke nur die Rückgabe
// zu (service.pruefeAusleihSperren); aufgehoben werden sie hier, in der Akte oder aus dem
// Dialog der Theke — wie in Littera in den Leserdaten. Sonst fiele die Sperre der
// Ehemaligen erst, wenn der nächste LUSD-Import den Schüler wiederfände.
//
// Nicht umschalten lässt sich: ein Leser im Papierkorb (zurück nur über Wiederherstellen),
// ein anonymisierter Datensatz (keine Person mehr) und die Sperre eines Kollegen — ein
// Kollege wird nie gesperrt.
//
// Jede Änderung steht im Protokoll (LESER_GESPERRT, LESER_ENTSPERRT), wie das Übergehen an
// der Theke (OVERRIDE_BLOCK).
func (t *Tueren) LeserSperren(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		return t.sperreLeser(w, r, protokollAbfragen)
	})
}

// sperreLeser schaltet die Sperre in einer Transaktion um und schreibt den Eintrag ins
// Protokoll.
func (t *Tueren) sperreLeser(w http.ResponseWriter, r *http.Request, protokollAbfragen repository.ProtokollAbfragen) error {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		return apierrors.Unauthorized("missing session information", nil)
	}
	id := r.PathValue("id")
	if id == "" {
		return apierrors.BadRequest("fehlende Schüler-ID", nil)
	}

	var req struct {
		IsLocked bool   `json:"is_locked"`
		Reason   string `json:"reason"`
	}
	if !tuer.LeseAnfrage(w, r, &req) {
		// LeseAnfrage hat schon geantwortet.
		return nil
	}

	// Eine manuelle Sperre ohne Grund ist genau die "Zombie-Sperre", die der
	// DB-Constraint chk_schueler_block_reason verhindern soll: Das Personal sähe nur die
	// rote Markierung ohne Kontext. Daher ist der Grund beim Sperren Pflicht (beim Entsperren
	// irrelevant).
	grund := strings.TrimSpace(req.Reason)
	if req.IsLocked && grund == "" {
		return apierrors.BadRequest("Für eine manuelle Sperre ist ein Grund erforderlich.", nil)
	}

	ctx := r.Context()
	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		return apierrors.Internal(meldungSperreFehler, err)
	}
	defer db.Verwerfe(ctx, tx)

	// `leser`, nicht die Sicht `schueler`: Die Akte eines Kollegen ruft die Tür auch, und
	// über die Sicht antwortete sie mit „schüler nicht gefunden".
	alt, err := repository.SperreLeserZeileMitStand(ctx, tx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return apierrors.NotFound("leser nicht gefunden", err)
	}
	if err != nil {
		return apierrors.Internal(meldungSperreFehler, err)
	}
	switch {
	case alt.Geloescht:
		return apierrors.Conflict("Der Leser liegt im Papierkorb. Erst wiederherstellen.", nil)
	case alt.Anonymisiert:
		return apierrors.Conflict("Ein anonymisierter Datensatz bleibt gesperrt.", nil)
	case req.IsLocked && !leserart.IstSchueler(alt.Art):
		return apierrors.BadRequest("Kollegen werden nicht gesperrt.", nil)
	}

	// Sperren setzt die Sperre von Hand samt Grund. Aufheben nimmt beide Sperren und den
	// Grund weg; chk_schueler_block_reason verlangt ihn nur, solange eine besteht.
	var leser struct {
		ID                string `json:"id"`
		Vorname           string `json:"vorname"`
		Nachname          string `json:"nachname"`
		Klasse            string `json:"klasse"`
		IsManuallyBlocked bool   `json:"is_manually_blocked"`
		IstGesperrt       bool   `json:"ist_gesperrt"`
	}
	neu, err := repository.SetzeSperreVonHand(ctx, tx, id, req.IsLocked, grund)
	if err != nil {
		return apierrors.Internal(meldungSperreFehler, err)
	}
	leser.ID, leser.Vorname, leser.Nachname, leser.Klasse = neu.ID, neu.Vorname, neu.Nachname, neu.Klasse
	leser.IsManuallyBlocked, leser.IstGesperrt = neu.VonHand, neu.Gesperrt
	if err := tx.Commit(ctx); err != nil {
		return apierrors.Internal(meldungSperreFehler, err)
	}

	protokolliereSperre(r, protokollAbfragen, claims.KontoID, id, req.IsLocked, grund, alt)

	w.Header().Set("Content-Type", "application/json")
	httpresp.SchreibeJSON(w, leser)
	return nil
}

// protokolliereSperre schreibt das Sperren und das Aufheben ins Protokoll. Wie bei
// FRIST_OVERRIDE gilt die Änderung auch, wenn das Protokoll klemmt; der Fehlversuch steht dann
// im Log. Aufheben ohne bestehende Sperre ändert nichts und schreibt nichts.
func protokolliereSperre(r *http.Request, protokollAbfragen repository.ProtokollAbfragen, bearbeiterID, id string, sperren bool, grund string, alt repository.LeserSperrStand) {
	aktion, details := "LESER_GESPERRT", map[string]any{"schueler_id": id, "grund": grund}
	if !sperren {
		if !alt.VonHand && !alt.VomProgramm {
			return
		}
		aktion, details = "LESER_ENTSPERRT", map[string]any{
			"schueler_id":  id,
			"grund":        alt.Grund,
			"von_hand":     alt.VonHand,
			"vom_programm": alt.VomProgramm,
		}
	}
	if err := protokollAbfragen.ProtokolliereAdminAktion(r.Context(), bearbeiterID, aktion, clientip.AusAnfrage(r), details); err != nil {
		log.Printf("Audit für %s fehlgeschlagen: %v", aktion, err)
	}
}

package katalog

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// meldungSachgruppeNichtGeaendert ist die Meldung, die der Benutzer bei jedem Abbruch der
// Umbenennung sieht — Begin, Update und Commit sprechen bewusst mit einer Stimme:
// Für den Benutzer ist es derselbe Vorgang, der nicht stattgefunden hat (go:S1192).
const meldungSachgruppeNichtGeaendert = "Sachgruppe konnte nicht geändert werden"

// sachgruppeAnfrage ist die Eingabe für Anlegen und Ändern einer Sachgruppe.
type sachgruppeAnfrage struct {
	Kuerzel     string `json:"kuerzel"`
	Bezeichnung string `json:"bezeichnung"`
}

// normalisiere trimmt beide Felder. Ein Kürzel mit Leerzeichen am Rand landet sonst
// als Signatur-Präfix am Buchrücken und passt danach zu keinem Regalbereich mehr.
func (a *sachgruppeAnfrage) normalisiere() {
	a.Kuerzel = strings.TrimSpace(a.Kuerzel)
	a.Bezeichnung = strings.TrimSpace(a.Bezeichnung)
}

// pruefe verlangt beide Felder und schließt Leerzeichen im Kürzel aus.
func (a sachgruppeAnfrage) pruefe() error {
	if a.Kuerzel == "" {
		return errors.New("kürzel ist erforderlich")
	}
	if a.Bezeichnung == "" {
		return errors.New("bezeichnung ist erforderlich")
	}
	// Die Signatur wird als "BIB <kuerzel>" gebildet und am Leerzeichen in
	// Regalbereiche geschnitten. Ein Kürzel mit Leerzeichen zerlegte diese Grenze.
	if strings.ContainsAny(a.Kuerzel, " \t") {
		return errors.New("kürzel darf keine Leerzeichen enthalten")
	}
	return nil
}

// istDoppelteSachgruppe erkennt die doppelte Sachgruppe (Kürzel oder — seit Migration
// 078 — Bezeichnung, auch case-insensitiv).
func istDoppelteSachgruppe(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// meldungDoppelteSachgruppe benennt, WELCHES Feld kollidiert ist — "Kürzel existiert bereits" bei
// einer doppelten Bezeichnung schickte den Benutzer auf die falsche Fährte.
func meldungDoppelteSachgruppe(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		(pgErr.ConstraintName == "uniq_systematik_bezeichnung" || pgErr.ConstraintName == "uniq_systematik_bezeichnung_ci") {
		return "eine Sachgruppe mit dieser Bezeichnung existiert bereits"
	}
	return "eine Sachgruppe mit diesem Kürzel existiert bereits"
}

// SachgruppeAnlegen legt eine Sachgruppe an.
//
// Diese Liste ist das Vokabular, aus dem das Buchformular die Signatur vorschlägt
// ("BIB Deu"). Sie war bisher nur lesbar — es gab keinen Weg, sie zu füllen, während
// der Vorschlag im Formular von ihr abhängt.
//
// @Summary      Create a systematic category
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        body  body      sachgruppeAnfrage  true  "Category"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Router       /systematics [post]
func (t *Tueren) SachgruppeAnlegen() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req sachgruppeAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return nil
		}
		req.normalisiere()
		if err := req.pruefe(); err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}

		id, err := repository.LegeSachgruppeAn(r.Context(), t.DB.Pool, req.Kuerzel, req.Bezeichnung)
		if err != nil {
			if istDoppelteSachgruppe(err) {
				return apierrors.Conflict(meldungDoppelteSachgruppe(err), err)
			}
			return apierrors.Internal("Sachgruppe konnte nicht angelegt werden", err)
		}

		tuer.AntworteJSON(w, http.StatusCreated, map[string]string{
			"id":          id,
			"kuerzel":     req.Kuerzel,
			"bezeichnung": req.Bezeichnung,
		})
		return nil
	})
}

// SachgruppeAendern ändert Kürzel oder Bezeichnung einer Sachgruppe.
//
// Die BEZEICHNUNG wird auf die Titel mitgezogen (buecher_titel.subject), damit die
// Fach-Auswahl nicht lautlos auseinanderdriftet (Prüfbericht F3). Die Signatur (mit dem
// KÜRZEL) bleibt unangetastet: Sie klebt als physisches Etikett am Buch. Die Antwort
// meldet, wie viele Titel mitgezogen wurden (titel_mitgezogen).
//
// @Summary      Update a systematic category
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        id    path      string             true  "Category ID"
// @Param        body  body      sachgruppeAnfrage  true  "Category"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Router       /systematics/{id} [put]
func (t *Tueren) SachgruppeAendern() http.HandlerFunc {
	return apierrors.Wrap(t.aendereSachgruppe)
}

func (t *Tueren) aendereSachgruppe(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	var req sachgruppeAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return nil
	}
	req.normalisiere()
	if err := req.pruefe(); err != nil {
		return apierrors.BadRequest(err.Error(), err)
	}

	ctx := r.Context()
	// Sachgruppe umbenennen — die Titel zieht seit Migration 078 der Fremdschlüssel
	// mit (buecher_titel.subject → systematik_kategorien.bezeichnung, ON UPDATE
	// CASCADE). Der Handler zählt die betroffenen Titel nur noch VOR dem Update, für
	// die Antwort (titel_mitgezogen). Das KÜRZEL wird bewusst NICHT in die Signaturen
	// zurückgeschrieben: Signaturen stehen als physische Etiketten auf den Büchern;
	// sie folgen einem Umlabeln, nicht einem DB-Update.
	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		return apierrors.Internal(meldungSachgruppeNichtGeaendert, err)
	}
	defer db.SafeRollback(ctx, tx)

	// Die alte Bezeichnung vor dem Update lesen: Nur sie verbindet die Sachgruppe
	// mit den offenen Inventur-Sessions (und die Zählung mit den Büchern).
	alteBezeichnung, err := ladeSachgruppenBezeichnung(ctx, tx, id)
	if err != nil {
		return err
	}

	var mitgezogen int64
	if alteBezeichnung != req.Bezeichnung {
		if mitgezogen, err = repository.ZaehleTitelDerSachgruppe(ctx, tx, alteBezeichnung); err != nil {
			return apierrors.Internal("Verwendung der Sachgruppe konnte nicht geprüft werden", err)
		}
	}

	if err := repository.AendereSachgruppe(ctx, tx, id, req.Kuerzel, req.Bezeichnung); err != nil {
		if istDoppelteSachgruppe(err) {
			return apierrors.Conflict(meldungDoppelteSachgruppe(err), err)
		}
		return apierrors.Internal(meldungSachgruppeNichtGeaendert, err)
	}

	// Offene Inventur-Sessions mit Filter auf dieses Fach folgen der Umbenennung —
	// ihr Scope (scope_subject) ist Text, kein FK: Bliebe er auf dem alten Namen,
	// zählte die Session ab sofort null Exemplare und jeder Scan liefe auf 409
	// "außer Scope". Abgeschlossene Sessions sind Historie und bleiben unangetastet.
	if alteBezeichnung != req.Bezeichnung {
		if err := repository.BenenneFachOffenerInventurenUm(ctx, tx, alteBezeichnung, req.Bezeichnung); err != nil {
			return apierrors.Internal("laufende Inventuren konnten nicht auf das neue Fach umgestellt werden", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return apierrors.Internal(meldungSachgruppeNichtGeaendert, err)
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]any{
		"id":               id,
		"kuerzel":          req.Kuerzel,
		"bezeichnung":      req.Bezeichnung,
		"titel_mitgezogen": mitgezogen,
	})
	return nil
}

// ladeSachgruppenBezeichnung liest die Bezeichnung einer Sachgruppe; eine unbekannte Kennung ist
// ein 404.
func ladeSachgruppenBezeichnung(ctx context.Context, tx pgx.Tx, id string) (string, error) {
	bezeichnung, err := repository.LadeSachgruppenBezeichnung(ctx, tx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", apierrors.NotFound("Sachgruppe nicht gefunden", err)
		}
		return "", apierrors.Internal("Sachgruppe konnte nicht geladen werden", err)
	}
	return bezeichnung, nil
}

// SachgruppeLoeschen entfernt eine Sachgruppe — aber nicht unter den Büchern weg.
//
// buecher_titel.subject hält die Bezeichnung als Text, ohne Fremdschlüssel. Ein Löschen
// bliebe deshalb von der Datenbank unbemerkt und würde die betroffenen Titel auf ein
// Fach zeigen lassen, das es nicht mehr gibt: Sie verschwänden aus der Fach-Auswahl,
// ohne dass irgendwo etwas fehlschlägt. Deshalb prüft der Handler das selbst.
//
// @Summary      Delete a systematic category
// @Tags         books
// @Produce      json
// @Param        id  path  string  true  "Category ID"
// @Success      204
// @Failure      404  {object}  map[string]string
// @Failure      409  {object}  map[string]string
// @Router       /systematics/{id} [delete]
func (t *Tueren) SachgruppeLoeschen() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")

		bezeichnung, err := repository.LadeSachgruppenBezeichnung(r.Context(), t.DB.Pool, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierrors.NotFound("Sachgruppe nicht gefunden", err)
			}
			return apierrors.Internal("Sachgruppe konnte nicht geladen werden", err)
		}

		betroffen, err := t.zaehleTitelMitFach(r.Context(), bezeichnung)
		if err != nil {
			return err
		}
		if betroffen > 0 {
			return apierrors.Conflict(
				"Diese Sachgruppe hängt noch an Büchern und kann nicht gelöscht werden",
				errors.New("systematik noch in verwendung"))
		}

		if err := repository.LoescheSachgruppe(r.Context(), t.DB.Pool, id); err != nil {
			// Rückfallebene zur Zählung oben: Hängt sich ZWISCHEN Prüfung und Löschen
			// ein Titel an das Fach, schlägt der FK (ON DELETE RESTRICT) zu — das ist
			// derselbe fachliche Konflikt, kein interner Fehler.
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return apierrors.Conflict(
					"Diese Sachgruppe hängt noch an Büchern und kann nicht gelöscht werden", err)
			}
			return apierrors.Internal("Sachgruppe konnte nicht gelöscht werden", err)
		}

		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

// zaehleTitelMitFach zählt Titel, deren Fach (subject) auf die Bezeichnung zeigt.
func (t *Tueren) zaehleTitelMitFach(ctx context.Context, bezeichnung string) (int, error) {
	anzahl, err := repository.ZaehleTitelMitFach(ctx, t.DB.Pool, bezeichnung)
	if err != nil {
		return 0, apierrors.Internal("Verwendung der Sachgruppe konnte nicht geprüft werden", err)
	}
	return anzahl, nil
}

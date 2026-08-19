package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"bibliothek/apierrors"
	"bibliothek/db"

	"github.com/jackc/pgx/v5/pgconn"
)

// systematikRequest ist die Eingabe für Anlegen und Ändern einer Sachgruppe.
type systematikRequest struct {
	Kuerzel     string `json:"kuerzel"`
	Bezeichnung string `json:"bezeichnung"`
}

// normalisiert trimmt beide Felder. Ein Kürzel mit Leerzeichen am Rand landet sonst
// als Signatur-Präfix am Buchrücken und passt danach zu keinem Regalbereich mehr.
func (req *systematikRequest) normalisiert() {
	req.Kuerzel = strings.TrimSpace(req.Kuerzel)
	req.Bezeichnung = strings.TrimSpace(req.Bezeichnung)
}

// pruefe verlangt beide Felder und schließt Leerzeichen im Kürzel aus.
func (req systematikRequest) pruefe() error {
	if req.Kuerzel == "" {
		return errors.New("kürzel ist erforderlich")
	}
	if req.Bezeichnung == "" {
		return errors.New("bezeichnung ist erforderlich")
	}
	// Die Signatur wird als "BIB <kuerzel>" gebildet und am Leerzeichen in
	// Regalbereiche geschnitten. Ein Kürzel mit Leerzeichen zerlegte diese Grenze.
	if strings.ContainsAny(req.Kuerzel, " \t") {
		return errors.New("kürzel darf keine Leerzeichen enthalten")
	}
	return nil
}

// istUniqueVerletzung erkennt das doppelte Kürzel (systematik_kategorien.kuerzel UNIQUE).
func istUniqueVerletzung(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// CreateSystematikHandler legt eine Sachgruppe an.
//
// Diese Liste ist das Vokabular, aus dem das Buchformular die Signatur vorschlägt
// ("BIB Deu"). Sie war bisher nur lesbar — es gab keinen Weg, sie zu füllen, während
// der Vorschlag im Formular von ihr abhängt.
//
// @Summary      Create a systematic category
// @Tags         books
// @Accept       json
// @Produce      json
// @Param        body  body      systematikRequest  true  "Category"
// @Success      201   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Router       /systematics [post]
func (s *Server) CreateSystematikHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		var req systematikRequest
		if !DecodeAndValidate(w, r, &req) {
			return nil
		}
		req.normalisiert()
		if err := req.pruefe(); err != nil {
			return apierrors.BadRequest(err.Error(), err)
		}

		var id string
		err := s.DB.Pool.QueryRow(r.Context(), `
			INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
			VALUES ($1, $2)
			RETURNING id::text
		`, req.Kuerzel, req.Bezeichnung).Scan(&id)
		if err != nil {
			if istUniqueVerletzung(err) {
				return apierrors.Conflict("eine Sachgruppe mit diesem Kürzel existiert bereits", err)
			}
			return apierrors.Internal("Sachgruppe konnte nicht angelegt werden", err)
		}

		RespondJSON(w, http.StatusCreated, map[string]string{
			"id":          id,
			"kuerzel":     req.Kuerzel,
			"bezeichnung": req.Bezeichnung,
		})
		return nil
	})
}

// UpdateSystematikHandler ändert Kürzel oder Bezeichnung einer Sachgruppe.
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
// @Param        body  body      systematikRequest  true  "Category"
// @Success      200   {object}  map[string]any
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Failure      409   {object}  map[string]string
// @Router       /systematics/{id} [put]
func (s *Server) UpdateSystematikHandler() http.HandlerFunc {
	return apierrors.Wrap(s.handleUpdateSystematik)
}

func (s *Server) handleUpdateSystematik(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	var req systematikRequest
	if !DecodeAndValidate(w, r, &req) {
		return nil
	}
	req.normalisiert()
	if err := req.pruefe(); err != nil {
		return apierrors.BadRequest(err.Error(), err)
	}

	ctx := r.Context()
	// Sachgruppe umbenennen und die Titel MITZIEHEN — atomar in einer Transaktion.
	//
	// buecher_titel.subject hält die Bezeichnung als Text (ohne Fremdschlüssel). Bis
	// zum 19.08.2026 änderte dieser Handler nur die Sachgruppe und zählte die Titel,
	// die nun auf dem ALTEN Namen zurückblieben — genau der lautlose Drift aus dem
	// Prüfbericht (F3): "Fach umbenannt → Titel behalten den alten Namen, Filter finden
	// nichts". Jetzt wird die Umbenennung auf die Titel propagiert, sodass die
	// Fach-Auswahl (buecher_titel.subject) mit der Sachgruppe deckungsgleich bleibt.
	// Das KÜRZEL wird bewusst NICHT in die Signaturen zurückgeschrieben: Signaturen
	// stehen als physische Etiketten auf den Büchern; sie folgen einem Umlabeln, nicht
	// einem DB-Update.
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		return apierrors.Internal("Sachgruppe konnte nicht geändert werden", err)
	}
	defer db.SafeRollback(ctx, tx)

	// Die alte Bezeichnung VOR dem Update lesen: Nur sie verbindet die Sachgruppe
	// mit den Büchern.
	var alteBezeichnung string
	if err := tx.QueryRow(ctx,
		`SELECT bezeichnung FROM systematik_kategorien WHERE id = $1::uuid`, id).Scan(&alteBezeichnung); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return apierrors.NotFound("Sachgruppe nicht gefunden", err)
		}
		return apierrors.Internal("Sachgruppe konnte nicht geladen werden", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE systematik_kategorien
		SET kuerzel = $2, bezeichnung = $3
		WHERE id = $1::uuid
	`, id, req.Kuerzel, req.Bezeichnung); err != nil {
		if istUniqueVerletzung(err) {
			return apierrors.Conflict("eine Sachgruppe mit diesem Kürzel existiert bereits", err)
		}
		return apierrors.Internal("Sachgruppe konnte nicht geändert werden", err)
	}

	var mitgezogen int64
	if !strings.EqualFold(strings.TrimSpace(alteBezeichnung), strings.TrimSpace(req.Bezeichnung)) {
		tag, err := tx.Exec(ctx,
			`UPDATE buecher_titel SET subject = $2 WHERE btrim(COALESCE(subject, '')) = btrim($1)`,
			alteBezeichnung, req.Bezeichnung)
		if err != nil {
			return apierrors.Internal("Titel konnten nicht auf das neue Fach umgestellt werden", err)
		}
		mitgezogen = tag.RowsAffected()
	}

	if err := tx.Commit(ctx); err != nil {
		return apierrors.Internal("Sachgruppe konnte nicht geändert werden", err)
	}

	RespondJSON(w, http.StatusOK, map[string]any{
		"id":               id,
		"kuerzel":          req.Kuerzel,
		"bezeichnung":      req.Bezeichnung,
		"titel_mitgezogen": mitgezogen,
	})
	return nil
}

// DeleteSystematikHandler entfernt eine Sachgruppe — aber nicht unter den Büchern weg.
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
func (s *Server) DeleteSystematikHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")

		var bezeichnung string
		err := s.DB.Pool.QueryRow(r.Context(),
			`SELECT bezeichnung FROM systematik_kategorien WHERE id = $1::uuid`, id).Scan(&bezeichnung)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				return apierrors.NotFound("Sachgruppe nicht gefunden", err)
			}
			return apierrors.Internal("Sachgruppe konnte nicht geladen werden", err)
		}

		betroffen, err := s.zaehleTitelMitFach(r.Context(), bezeichnung)
		if err != nil {
			return err
		}
		if betroffen > 0 {
			return apierrors.Conflict(
				"Diese Sachgruppe hängt noch an Büchern und kann nicht gelöscht werden",
				errors.New("systematik noch in verwendung"))
		}

		if _, err := s.DB.Pool.Exec(r.Context(),
			`DELETE FROM systematik_kategorien WHERE id = $1::uuid`, id); err != nil {
			return apierrors.Internal("Sachgruppe konnte nicht gelöscht werden", err)
		}

		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}

// zaehleTitelMitFach zählt Titel, deren Fach (subject) auf die Bezeichnung zeigt.
func (s *Server) zaehleTitelMitFach(ctx context.Context, bezeichnung string) (int, error) {
	var anzahl int
	err := s.DB.Pool.QueryRow(ctx,
		`SELECT count(*) FROM buecher_titel WHERE btrim(COALESCE(subject, '')) = btrim($1)`,
		bezeichnung).Scan(&anzahl)
	if err != nil {
		return 0, apierrors.Internal("Verwendung der Sachgruppe konnte nicht geprüft werden", err)
	}
	return anzahl, nil
}

package system

import (
	"errors"
	"net/http"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// MailVorlagen gibt alle Mail-Vorlagen zurück
func (t *Tueren) MailVorlagen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vorlagen, err := repository.LadeMailVorlagen(ctx, t.DB.Pool)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Laden der Vorlagen"))
			return
		}

		type mailVorlage struct {
			ID          string `json:"id"`
			Typ         string `json:"typ"`
			Betreff     string `json:"betreff"`
			TextBody    string `json:"text_body"`
			GeaendertAm string `json:"updated_at"`
		}

		var liste []mailVorlage
		for _, v := range vorlagen {
			liste = append(liste, mailVorlage{
				ID: v.ID, Typ: v.Typ, Betreff: v.Betreff, TextBody: v.TextBody,
				GeaendertAm: v.AktualisiertAm.Format(time.RFC3339),
			})
		}

		tuer.AntworteJSON(w, http.StatusOK, liste)
	}
}

// MailVorlageAendern aktualisiert eine Mail-Vorlage
func (t *Tueren) MailVorlageAendern() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ID fehlt"))
			return
		}

		var req struct {
			Betreff  string `json:"betreff"`
			TextBody string `json:"text_body"`
		}
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		ctx := r.Context()
		// Der Stand davor sagt, was sich geändert hat: Im Protokoll stehen die Vorlage und
		// die Namen der geänderten Felder, nicht der Wortlaut.
		typ, betreffNeu, textNeu, err := repository.AendereMailVorlage(ctx, t.DB.Pool, id, req.Betreff, req.TextBody)
		// Eine unbekannte Vorlage ist ein Fehler, kein „Erfolgreich gespeichert".
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("mail-Vorlage nicht gefunden"))
			return
		}
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, errors.New("fehler beim Aktualisieren der Vorlage"))
			return
		}
		t.ProtokolliereGeaenderteFelder(ctx, tuer.AktionMailvorlageGeaendert, map[string]any{"vorlage": typ},
			tuer.GeaenderteFelder(tuer.FeldWechsel{Name: "betreff", Geaendert: betreffNeu}, tuer.FeldWechsel{Name: "text", Geaendert: textNeu}))

		tuer.AntworteJSON(w, http.StatusOK, map[string]string{"message": "Erfolgreich gespeichert"})
	}
}

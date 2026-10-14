package leser

import (
	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"
	"context"
	"fmt"
	"net/http"
)

// leseFotoURL liefert die Foto-URL eines Schülers, falls ein (verschlüsseltes) Foto
// hinterlegt ist, sonst einen leeren String.
func leseFotoURL(ctx context.Context, leserAbfragen repository.LeserAbfragen, leser *repository.Leser) string {
	if leser.BarcodeID == "" {
		return ""
	}
	hatFoto, err := leserAbfragen.HatFoto(ctx, leser.ID)
	if err == nil && hatFoto {
		return fmt.Sprintf("/api/schueler/%s/photo", leser.BarcodeID)
	}
	return ""
}

// leserakteAntwort ist die Akte eines Lesers: Stammdaten mit der Adresse des Fotos und die
// laufenden Ausleihen.
type leserakteAntwort struct {
	ID        string `json:"id"`
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	// Art: eine aus pkg/leserart. Die Akte richtet sich danach: Ein Kollege hat
	// keine Klasse, kein Abgangsjahr, keine Elternadresse und keine LUSD-Kennung — ohne
	// die Art zeigte die Akte ihm diese Felder als „Keine Angabe" und behauptete damit,
	// dass sie fehlen. Sie gehören ihm gar nicht.
	Art           string `json:"art"`
	Klasse        string `json:"klasse"`
	AbgaengerJahr int    `json:"abgaenger_jahr"`
	// AusweisGueltigBis ist das Ablaufjahr des Schülerausweises (31.07.), aus dem
	// Bildungsgang gerechnet — nicht AbgaengerJahr, das die DSGVO-Löschung steuert.
	// Der profilseitige Ausweisdruck liest genau dieses Feld; fehlt es hier, druckt
	// die Karte "Gültig bis: 31.07.–", obwohl das Repository den Wert kennt.
	AusweisGueltigBis *int    `json:"ausweis_gueltig_bis,omitempty"`
	IstGesperrt       bool    `json:"ist_gesperrt"`
	FotoURL           string  `json:"foto_url"`
	Geburtsdatum      *string `json:"geburtsdatum,omitempty"`
	LUSDID            *string `json:"lusd_id,omitempty"`
	HatOffeneSchaeden bool    `json:"has_open_damages"`
	// offeneForderungen ist der Kontozustand neben der Sperre: Die Akte zeigt ihn in jedem
	// Reiter, ohne die Liste der Forderungen abzurufen.
	OffeneForderungen  offeneForderungen `json:"offene_forderungen"`
	IstVonHandGesperrt bool              `json:"is_manually_blocked"`
	Sperrgrund         *string           `json:"block_reason"`
	Strasse            string            `json:"strasse"`
	Hausnummer         string            `json:"hausnummer"`
	Plz                string            `json:"plz"`
	Ort                string            `json:"ort"`
	ElternEmail        string            `json:"eltern_email"`
	// Email ist die Schul-Adresse am Konto (benutzer.email), nicht die der Eltern —
	// bei Lehrkraft und LiV die Kennung, an der die Anmeldung die Person erkennt.
	// Leer heißt: Diese Person hat noch kein Konto, die Adresse ist in der Akte
	// nachtragbar. Bei einem Schüler steht hier nie etwas.
	Email             string                        `json:"email"`
	EntlieheneBuecher []repository.LaufendeAusleihe `json:"entliehene_buecher"`
}

// offeneForderungen sind Anzahl und Summe der unbezahlten Forderungen eines Lesers.
type offeneForderungen struct {
	Anzahl int     `json:"anzahl"`
	Summe  float64 `json:"summe"`
}

// Leserakte liefert die Akte eines Lesers: Stammdaten, die Adresse des Ausweisfotos, falls
// eines hochgeladen ist, und die laufenden Ausleihen mit Ausleih- und Rückgabedatum.
// @Summary      Akte eines Lesers
// @Description  Die ganze Akte eines Lesers nach seiner Kennung, mit laufenden Ausleihen und der Adresse des Fotos.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Kennung des Lesers (UUID)"
// @Success      200  {object}  leserakteAntwort
// @Failure      400  {object}  map[string]string
// @Failure      404  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /schueler/{id} [get]
func (t *Tueren) Leserakte(
	leserAbfragen repository.LeserAbfragen,
) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		return t.gibLeserakte(w, r, leserAbfragen)
	})
}

// gibLeserakte stellt die Akte eines Lesers zusammen: Stammdaten, Foto, Ausleihen,
// offene Forderungen und beim Kollegium die Schul-Adresse.
func (t *Tueren) gibLeserakte(w http.ResponseWriter, r *http.Request, leserAbfragen repository.LeserAbfragen) error {
	id := r.PathValue("id")
	if id == "" {
		return apierrors.BadRequest("missing student ID parameter", nil)
	}

	ctx := r.Context()

	// 1. Den Leser laden — Schüler oder Kollegium.
	//
	// LeserAbfragen.Finde liest die Tabelle `leser`: Über die Sicht `schueler` käme eine
	// Lehrkraft als 404 zurück. An der Theke hieße das: Der Kollege ist geladen, aber
	// niemand sieht, welche Bücher er hat.
	leser, err := leserAbfragen.Finde(ctx, id)
	if err != nil {
		return apierrors.Internal("Fehler beim Laden des Lesers", err)
	}
	if leser == nil {
		return apierrors.NotFound("student record not found", nil)
	}

	// 2. Die Adresse des Fotos, falls ein verschlüsseltes Foto in der Datenbank liegt.
	fotoURL := leseFotoURL(ctx, leserAbfragen, leser)

	// 3. Die laufenden Ausleihen des Lesers.
	entliehen, err := leserAbfragen.LadeLaufendeAusleihen(ctx, id)
	if err != nil {
		return apierrors.Internal("Fehler beim Laden der ausgeliehenen Bücher", err)
	}
	if entliehen == nil {
		entliehen = []repository.LaufendeAusleihe{}
	}

	// 3.5 Offene Forderungen. Ein Fehler bricht ab: „nichts offen" wäre sonst eine Auskunft,
	// die niemand geprüft hat.
	anzahl, summe, err := leserAbfragen.ZaehleOffeneSchaeden(ctx, leser.ID)
	if err != nil {
		return apierrors.Internal("Fehler beim Laden der offenen Forderungen", err)
	}

	// 3.6 Die Schul-Adresse am Konto — nur beim Kollegium (ein Schüler hat keins).
	schulEmail := ""
	if !leserart.IstSchueler(leser.Art) {
		schulEmail, err = t.kontoEmail(ctx, leser.ID)
		if err != nil {
			return apierrors.Internal("Fehler beim Laden des Zugangs", err)
		}
	}

	// 4. Die Antwort.
	antwort := leserakteAntwort{
		ID:                 leser.ID,
		BarcodeID:          leser.BarcodeID,
		Vorname:            leser.Vorname,
		Nachname:           leser.Nachname,
		Art:                leser.Art,
		Klasse:             leser.Klasse,
		AbgaengerJahr:      leser.AbgaengerJahr,
		AusweisGueltigBis:  leser.AusweisGueltigBis,
		IstGesperrt:        leser.IstGesperrt,
		FotoURL:            fotoURL,
		Geburtsdatum:       leser.Geburtsdatum,
		LUSDID:             leser.LUSDID,
		HatOffeneSchaeden:  anzahl > 0,
		OffeneForderungen:  offeneForderungen{Anzahl: anzahl, Summe: summe},
		IstVonHandGesperrt: leser.IstVonHandGesperrt,
		Sperrgrund:         leser.Sperrgrund,
		Strasse:            leser.Strasse,
		Hausnummer:         leser.Hausnummer,
		Plz:                leser.Plz,
		Ort:                leser.Ort,
		ElternEmail:        leser.ElternEmail,
		Email:              schulEmail,
		EntlieheneBuecher:  entliehen,
	}

	tuer.AntworteJSON(w, http.StatusOK, antwort)
	return nil
}

// Klassen liefert alle Klassen, die einem Schüler zugeordnet sind, ohne Doppelte.
// @Summary      Klassen
// @Description  Alle Klassennamen, die gerade einem Schüler zugeordnet sind, ohne Doppelte.
// @Tags         students
// @Accept       json
// @Produce      json
// @Success      200  {array}   string
// @Failure      500  {object}  map[string]string
// @Router       /klassen [get]
func (t *Tueren) Klassen(leserAbfragen repository.LeserAbfragen) http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		klassen, err := leserAbfragen.LadeKlassen(r.Context())
		if err != nil {
			return apierrors.Internal("Fehler beim Laden der Klassen", err)
		}

		tuer.AntworteJSON(w, http.StatusOK, klassen)
		return nil
	})
}

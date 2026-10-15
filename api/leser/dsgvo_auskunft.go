package leser

import (
	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/internal/auskunft"
	"bibliothek/repository"
	"context"
	"log"
	"net/http"
	"time"
)

// dsgvoFristen liest Lesehistorie-Befristung, Abgänger-Karenz, Protokoll-Aufbewahrung und
// die Frist erledigter Anliegen aus den Einstellungen; bei Fehlern gelten die Vorgaben (so
// arbeiten auch die Jobs).
func (t *Tueren) dsgvoFristen(ctx context.Context) auskunft.FristWerte {
	einst, err := repository.NewEinstellungsAbfragen(t.DB.Pool).Lade(ctx)
	if err != nil || einst == nil {
		return auskunft.FristWerte{
			LesehistorieTage: repository.LesehistorieTageVorgabe,
			LernmittelTage:   repository.LesehistorieLernmittelTageVorgabe,
			KarenzTage:       repository.AbgaengerKarenzTageVorgabe,
			ProtokollMonate:  repository.ProtokollAufbewahrungMonateVorgabe,
			AnliegenTage:     repository.AnliegenTageVorgabe,
		}
	}
	return auskunft.FristWerte{
		LesehistorieTage: repository.TageOderVorgabe(einst.LesehistorieTage, repository.LesehistorieTageVorgabe),
		LernmittelTage:   repository.TageOderVorgabe(einst.LesehistorieLernmittelTage, repository.LesehistorieLernmittelTageVorgabe),
		KarenzTage:       repository.AbgaengerKarenzTageOderVorgabe(einst),
		ProtokollMonate:  repository.ProtokollAufbewahrungMonateOderVorgabe(einst.ProtokollAufbewahrungMonate),
		AnliegenTage:     repository.TageOderVorgabe(einst.AnliegenTage, repository.AnliegenTageVorgabe),
	}
}

// alsAntwort wandelt die Zeilen einer Abfrage in den Typ der Antwort. Eine leere Liste bleibt
// leer und wird nicht nil: Die Auskunft sagt dann „keine" und nicht „unbekannt".
func alsAntwort[Z, T any](zeilen []Z, err error, als func(Z) T) ([]T, error) {
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(zeilen))
	for _, z := range zeilen {
		out = append(out, als(z))
	}
	return out, nil
}

func (t *Tueren) dsgvoStammdaten(ctx context.Context, id string) (*auskunft.Stammdaten, error) {
	zeile, err := repository.LadeDSGVOStammdaten(ctx, t.DB.Pool, id)
	if err != nil || zeile == nil {
		return nil, err
	}
	st := auskunft.Stammdaten(*zeile)
	return &st, nil
}

func (t *Tueren) dsgvoFoto(ctx context.Context, id string) (auskunft.Foto, error) {
	foto := auskunft.Foto{Hinweis: "Foto wird verschlüsselt gespeichert; Kopie über die Akte in der Leserdatei abrufbar"}
	aktualisiert, err := repository.LadeDSGVOFotoStand(ctx, t.DB.Pool, id)
	if err != nil {
		return foto, err
	}
	if aktualisiert == nil {
		return auskunft.Foto{Vorhanden: false, Hinweis: "Kein Foto gespeichert"}, nil
	}
	foto.Vorhanden = true
	foto.AktualisiertAm = aktualisiert
	return foto, nil
}

func (t *Tueren) dsgvoAusleihen(ctx context.Context, id string) ([]auskunft.Ausleihe, error) {
	zeilen, err := repository.LadeDSGVOAusleihen(ctx, t.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOAusleiheZeile) auskunft.Ausleihe {
		return auskunft.Ausleihe(z)
	})
}

func (t *Tueren) dsgvoSchadensfaelle(ctx context.Context, id string) ([]auskunft.Schadensfall, error) {
	zeilen, err := repository.LadeDSGVOSchadensfaelle(ctx, t.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOSchadensfallZeile) auskunft.Schadensfall {
		return auskunft.Schadensfall(z)
	})
}

func (t *Tueren) dsgvoVormerkungen(ctx context.Context, id string) ([]auskunft.Vormerkung, error) {
	zeilen, err := repository.LadeDSGVOVormerkungen(ctx, t.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOVormerkungZeile) auskunft.Vormerkung {
		return auskunft.Vormerkung(z)
	})
}

func (t *Tueren) dsgvoNachbuchMeldungen(ctx context.Context, id string) ([]auskunft.NachbuchMeldung, error) {
	zeilen, err := repository.LadeDSGVONachbuchMeldungen(ctx, t.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVONachbuchZeile) auskunft.NachbuchMeldung {
		return auskunft.NachbuchMeldung(z)
	})
}

func (t *Tueren) dsgvoBescheide(ctx context.Context, id string) ([]auskunft.Bescheid, error) {
	zeilen, err := repository.LadeDSGVOBescheide(ctx, t.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOBescheidZeile) auskunft.Bescheid {
		return auskunft.Bescheid(z)
	})
}

func (t *Tueren) dsgvoProtokollEintraege(ctx context.Context, id string) ([]auskunft.ProtokollEintrag, error) {
	zeilen, err := repository.LadeDSGVOProtokollEintraege(ctx, t.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOProtokollZeile) auskunft.ProtokollEintrag {
		return auskunft.ProtokollEintrag(z)
	})
}

func (t *Tueren) dsgvoVerwaltungsEintraege(ctx context.Context, id string) ([]auskunft.VerwaltungsEintrag, error) {
	zeilen, err := repository.LadeDSGVOVerwaltungsEintraege(ctx, t.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOVerwaltungZeile) auskunft.VerwaltungsEintrag {
		return auskunft.VerwaltungsEintrag(z)
	})
}

// dsgvoDaten bündelt alle personenbezogenen Daten eines Lesers für die Auskunft.
type dsgvoDaten struct {
	stammdaten         *auskunft.Stammdaten
	foto               auskunft.Foto
	ausleihen          []auskunft.Ausleihe
	schaeden           []auskunft.Schadensfall
	vormerkungen       []auskunft.Vormerkung
	bescheide          []auskunft.Bescheid
	nachbuchMeldungen  []auskunft.NachbuchMeldung
	protokollEintraege []auskunft.ProtokollEintrag
	verwaltung         []auskunft.VerwaltungsEintrag
	zugangskonto       *repository.DSGVOZugangskonto
	fruehereKonten     []repository.DSGVOFrueheresZugangskonto
	verarbeitung       auskunft.Verarbeitungsangaben
}

// dsgvoKontoRecht verlangt die Auskunft zusätzlich zum Recht der Route, sobald auf den Leser
// ein Zugangskonto zeigt oder einmal gezeigt hat. Dann nennt sie das Konto, auch ein
// gelöschtes, seine Einträge im Verwaltungsprotokoll und jeden Vorgang, den die Person selbst
// bearbeitet hat, bei Verwaltungseingriffen mit IP-Adresse. Konten und dieses Protokoll zeigt
// die Anwendung sonst nur mit manage_users (GET /api/benutzer, GET /api/admin/auditlog). Das
// Recht der Route, manage_students_admin, hat ab Werk auch die Leitung, und es ist zum
// Delegieren ans Sekretariat gedacht (db/rechte_vorgabe.go, RechteOptional). Gate:
// TestDSGVOAuskunft_KontoVerlangtKontenrecht.
const dsgvoKontoRecht = "manage_users"

// sammleDSGVODaten lädt alle personenbezogenen Daten eines Lesers — Schüler, Lehrkraft
// oder LiV — für die Art.-15-Auskunft. Fehler sind bereits als HTTP-Fehler (apierrors)
// verpackt. darfKonto sagt, ob der Aufrufer dsgvoKontoRecht hat; ohne das Recht endet die
// Auskunft über einen Leser mit Zugangskonto mit 403, bevor sie protokolliert wird.
func (t *Tueren) sammleDSGVODaten(ctx context.Context, id string, darfKonto bool) (*dsgvoDaten, error) {
	stammdaten, err := t.dsgvoStammdaten(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Stammdaten", err)
	}
	if stammdaten == nil {
		return nil, apierrors.NotFound("reader record not found", nil)
	}

	foto, err := t.dsgvoFoto(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Prüfen des Fotos", err)
	}
	ausleihen, err := t.dsgvoAusleihen(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Ausleihhistorie", err)
	}
	schaeden, err := t.dsgvoSchadensfaelle(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Schadensfälle", err)
	}
	vormerkungen, err := t.dsgvoVormerkungen(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Vormerkungen", err)
	}
	bescheide, err := t.dsgvoBescheide(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Schadensersatz-Bescheide", err)
	}
	nachbuchMeldungen, err := t.dsgvoNachbuchMeldungen(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Nachbuch-Meldungen", err)
	}
	protokollEintraege, err := t.dsgvoProtokollEintraege(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Protokolleinträge", err)
	}
	verwaltung, err := t.dsgvoVerwaltungsEintraege(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Verwaltungsprotokolle", err)
	}
	zugangskonto, err := repository.LadeDSGVOZugangskonto(ctx, t.DB.Pool, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden des Zugangskontos", err)
	}
	fruehereKonten, err := repository.LadeDSGVOFruehereZugangskonten(ctx, t.DB.Pool, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der früheren Zugangskonten", err)
	}
	// Geprüft an den gelesenen Konten, nicht an stammdaten.HatZugangskonto: Entscheidend ist,
	// was die Antwort enthalten würde — auch ein gelöschtes Konto samt seinen Einträgen.
	if (zugangskonto != nil || len(fruehereKonten) > 0) && !darfKonto {
		return nil, apierrors.New(http.StatusForbidden,
			"Die Auskunft über einen Leser mit Zugangskonto verlangt das Recht „Benutzer & Rechte verwalten“.", nil)
	}
	verarbeitung := auskunft.Pflichtangaben(stammdaten.Art, t.dsgvoFristen(ctx))

	return &dsgvoDaten{
		verarbeitung:       verarbeitung,
		zugangskonto:       zugangskonto,
		fruehereKonten:     fruehereKonten,
		stammdaten:         stammdaten,
		foto:               foto,
		ausleihen:          ausleihen,
		schaeden:           schaeden,
		vormerkungen:       vormerkungen,
		bescheide:          bescheide,
		nachbuchMeldungen:  nachbuchMeldungen,
		protokollEintraege: protokollEintraege,
		verwaltung:         verwaltung,
	}, nil
}

// protokolliereDSGVOAuskunft schreibt die Auskunft ins Protokoll (Rechenschaftspflicht).
// Ein Fehler wird nur protokolliert, nicht weitergereicht (die Auskunft geht vor).
func (t *Tueren) protokolliereDSGVOAuskunft(ctx context.Context, id string) {
	akteur := "SYSTEM"
	var bearbeiterID *string
	if claims, ok := auth.GetClaims(ctx); ok {
		akteur = "USER"
		bearbeiterID = &claims.UserID
	}
	if err := repository.ProtokolliereDSGVOAuskunft(ctx, t.DB.Pool, id, bearbeiterID, akteur); err != nil {
		log.Printf("dsgvo-auskunft: Audit-Protokollierung fehlgeschlagen: %v", err)
	}
}

// DSGVOAuskunft stellt die vollständige Betroffenenauskunft nach
// Art. 15 DSGVO für einen Leser zusammen, gleich welcher Art (pkg/leserart). Die Erteilung
// selbst steht im Protokoll (Rechenschaftspflicht, Art. 5 Abs. 2 DSGVO).
//
// Die Anmerkungen stehen direkt über der Tür: swag ordnet sie der folgenden Erklärung
// zu und übergeht sie an anderer Stelle still.
// @Summary      DSGVO-Betroffenenauskunft (Art. 15) für einen Leser
// @Tags         students
// @Produce      json
// @Param        id   path      string  true  "Kennung des Lesers (UUID)"
// @Success      200  {object}  auskunft.Antwort
// @Failure      404  {object}  map[string]string
// @Router       /schueler/{id}/dsgvo-auskunft [get]
func (t *Tueren) DSGVOAuskunft() http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("missing student ID parameter", nil)
		}
		ctx := r.Context()

		daten, err := t.sammleDSGVODaten(ctx, id, t.HatRecht(r, dsgvoKontoRecht))
		if err != nil {
			return err
		}

		// Rechenschaftspflicht: Die Auskunftserteilung selbst wird protokolliert.
		t.protokolliereDSGVOAuskunft(ctx, id)

		tuer.AntworteJSON(w, http.StatusOK, dsgvoAntwort(daten, time.Now()))
		return nil
	})
}

// dsgvoAntwort ist die Auskunft als ein Objekt: Die JSON-Antwort und das PDF entstehen beide
// daraus, damit das Blatt keinen Teil auslässt, den die Antwort nennt. Gate:
// TestBlatt_DrucktJedeAngabeDerAuskunft (internal/auskunft).
func dsgvoAntwort(daten *dsgvoDaten, erstelltAm time.Time) auskunft.Antwort {
	return auskunft.Antwort{
		Art:                   "Auskunft nach Art. 15 DSGVO",
		ErstelltAm:            erstelltAm,
		Stammdaten:            *daten.stammdaten,
		Foto:                  daten.foto,
		Ausleihen:             daten.ausleihen,
		Schadensfaelle:        daten.schaeden,
		Vormerkungen:          daten.vormerkungen,
		Bescheide:             daten.bescheide,
		NachbuchMeldungen:     daten.nachbuchMeldungen,
		ProtokollEintraege:    daten.protokollEintraege,
		Verwaltung:            daten.verwaltung,
		Zugangskonto:          daten.zugangskonto,
		FruehereZugangskonten: daten.fruehereKonten,
		Verarbeitungsangaben:  daten.verarbeitung,
	}
}

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
func (s *Tueren) dsgvoFristen(ctx context.Context) auskunft.FristWerte {
	einst, err := repository.NewEinstellungsAbfragen(s.DB.Pool).Lade(ctx)
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

func (s *Tueren) dsgvoQueryStammdaten(ctx context.Context, id string) (*auskunft.Stammdaten, error) {
	zeile, err := repository.LadeDSGVOStammdaten(ctx, s.DB.Pool, id)
	if err != nil || zeile == nil {
		return nil, err
	}
	st := auskunft.Stammdaten(*zeile)
	return &st, nil
}

func (s *Tueren) dsgvoQueryFoto(ctx context.Context, id string) (auskunft.Foto, error) {
	foto := auskunft.Foto{Hinweis: "Foto wird verschlüsselt gespeichert; Kopie über die Akte in der Leserdatei abrufbar"}
	aktualisiert, err := repository.LadeDSGVOFotoStand(ctx, s.DB.Pool, id)
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

func (s *Tueren) dsgvoQueryAusleihen(ctx context.Context, id string) ([]auskunft.Ausleihe, error) {
	zeilen, err := repository.LadeDSGVOAusleihen(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOAusleiheZeile) auskunft.Ausleihe {
		return auskunft.Ausleihe(z)
	})
}

func (s *Tueren) dsgvoQuerySchadensfaelle(ctx context.Context, id string) ([]auskunft.Schadensfall, error) {
	zeilen, err := repository.LadeDSGVOSchadensfaelle(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOSchadensfallZeile) auskunft.Schadensfall {
		return auskunft.Schadensfall(z)
	})
}

func (s *Tueren) dsgvoQueryVormerkungen(ctx context.Context, id string) ([]auskunft.Vormerkung, error) {
	zeilen, err := repository.LadeDSGVOVormerkungen(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOVormerkungZeile) auskunft.Vormerkung {
		return auskunft.Vormerkung(z)
	})
}

func (s *Tueren) dsgvoQueryNachbuchMeldungen(ctx context.Context, id string) ([]auskunft.NachbuchMeldung, error) {
	zeilen, err := repository.LadeDSGVONachbuchMeldungen(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVONachbuchZeile) auskunft.NachbuchMeldung {
		return auskunft.NachbuchMeldung(z)
	})
}

func (s *Tueren) dsgvoQueryBescheide(ctx context.Context, id string) ([]auskunft.Bescheid, error) {
	zeilen, err := repository.LadeDSGVOBescheide(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOBescheidZeile) auskunft.Bescheid {
		return auskunft.Bescheid(z)
	})
}

func (s *Tueren) dsgvoQueryAuditEintraege(ctx context.Context, id string) ([]auskunft.ProtokollEintrag, error) {
	zeilen, err := repository.LadeDSGVOProtokollEintraege(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOProtokollZeile) auskunft.ProtokollEintrag {
		return auskunft.ProtokollEintrag(z)
	})
}

func (s *Tueren) dsgvoQueryVerwaltungsEintraege(ctx context.Context, id string) ([]auskunft.VerwaltungsEintrag, error) {
	zeilen, err := repository.LadeDSGVOVerwaltungsEintraege(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DSGVOVerwaltungZeile) auskunft.VerwaltungsEintrag {
		return auskunft.VerwaltungsEintrag(z)
	})
}

// dsgvoDaten bündelt alle personenbezogenen Daten eines Lesers für die Auskunft.
type dsgvoDaten struct {
	stammdaten        *auskunft.Stammdaten
	foto              auskunft.Foto
	ausleihen         []auskunft.Ausleihe
	schaeden          []auskunft.Schadensfall
	vormerkungen      []auskunft.Vormerkung
	bescheide         []auskunft.Bescheid
	nachbuchMeldungen []auskunft.NachbuchMeldung
	auditEintraege    []auskunft.ProtokollEintrag
	verwaltung        []auskunft.VerwaltungsEintrag
	zugangskonto      *repository.DSGVOZugangskonto
	fruehereKonten    []repository.DSGVOFrueheresZugangskonto
	verarbeitung      auskunft.Verarbeitungsangaben
}

// dsgvoKontoRecht verlangt die Auskunft zusätzlich zum Recht der Route, sobald auf den Leser
// ein Zugangskonto zeigt oder einmal gezeigt hat. Dann nennt sie das Konto, auch ein
// gelöschtes, seine Einträge im Verwaltungsprotokoll und jeden Vorgang, den die Person selbst
// bearbeitet hat, bei Verwaltungseingriffen mit IP-Adresse. Konten und dieses Protokoll zeigt
// die Anwendung sonst nur mit manage_users (GET /api/benutzer, GET /api/admin/auditlog). Das
// Recht der Route, manage_students_admin, hat ab Werk auch die Leitung, und es ist zum
// Delegieren ans Sekretariat gedacht (db/seed.go, RechteOptional). Gate:
// TestDsgvoAuskunft_KontoVerlangtKontenrecht.
const dsgvoKontoRecht = "manage_users"

// sammleDsgvoDaten lädt alle personenbezogenen Daten eines Lesers — Schüler, Lehrkraft
// oder LiV — für die Art.-15-Auskunft. Fehler sind bereits als HTTP-Fehler (apierrors)
// verpackt. darfKonto sagt, ob der Aufrufer dsgvoKontoRecht hat; ohne das Recht endet die
// Auskunft über einen Leser mit Zugangskonto mit 403, bevor sie protokolliert wird.
func (s *Tueren) sammleDsgvoDaten(ctx context.Context, id string, darfKonto bool) (*dsgvoDaten, error) {
	stammdaten, err := s.dsgvoQueryStammdaten(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Stammdaten", err)
	}
	if stammdaten == nil {
		return nil, apierrors.NotFound("reader record not found", nil)
	}

	foto, err := s.dsgvoQueryFoto(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Prüfen des Fotos", err)
	}
	ausleihen, err := s.dsgvoQueryAusleihen(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Ausleihhistorie", err)
	}
	schaeden, err := s.dsgvoQuerySchadensfaelle(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Schadensfälle", err)
	}
	vormerkungen, err := s.dsgvoQueryVormerkungen(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Vormerkungen", err)
	}
	bescheide, err := s.dsgvoQueryBescheide(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Schadensersatz-Bescheide", err)
	}
	nachbuchMeldungen, err := s.dsgvoQueryNachbuchMeldungen(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Nachbuch-Meldungen", err)
	}
	auditEintraege, err := s.dsgvoQueryAuditEintraege(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Protokolleinträge", err)
	}
	verwaltung, err := s.dsgvoQueryVerwaltungsEintraege(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Verwaltungsprotokolle", err)
	}
	zugangskonto, err := repository.LadeDSGVOZugangskonto(ctx, s.DB.Pool, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden des Zugangskontos", err)
	}
	fruehereKonten, err := repository.LadeDSGVOFruehereZugangskonten(ctx, s.DB.Pool, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der früheren Zugangskonten", err)
	}
	// Geprüft an den gelesenen Konten, nicht an stammdaten.HatZugangskonto: Entscheidend ist,
	// was die Antwort enthalten würde — auch ein gelöschtes Konto samt seinen Einträgen.
	if (zugangskonto != nil || len(fruehereKonten) > 0) && !darfKonto {
		return nil, apierrors.New(http.StatusForbidden,
			"Die Auskunft über einen Leser mit Zugangskonto verlangt das Recht „Benutzer & Rechte verwalten“.", nil)
	}
	verarbeitung := auskunft.Pflichtangaben(stammdaten.Art, s.dsgvoFristen(ctx))

	return &dsgvoDaten{
		verarbeitung:      verarbeitung,
		zugangskonto:      zugangskonto,
		fruehereKonten:    fruehereKonten,
		stammdaten:        stammdaten,
		foto:              foto,
		ausleihen:         ausleihen,
		schaeden:          schaeden,
		vormerkungen:      vormerkungen,
		bescheide:         bescheide,
		nachbuchMeldungen: nachbuchMeldungen,
		auditEintraege:    auditEintraege,
		verwaltung:        verwaltung,
	}, nil
}

// protokolliereDsgvoAuskunft schreibt den Rechenschafts-Audit-Eintrag der Auskunft.
// Ein Fehler wird nur protokolliert, nicht weitergereicht (die Auskunft geht vor).
func (s *Tueren) protokolliereDsgvoAuskunft(ctx context.Context, id string) {
	akteur := "SYSTEM"
	var bearbeiterID *string
	if claims, ok := auth.GetClaims(ctx); ok {
		akteur = "USER"
		bearbeiterID = &claims.UserID
	}
	if err := repository.ProtokolliereDSGVOAuskunft(ctx, s.DB.Pool, id, bearbeiterID, akteur); err != nil {
		log.Printf("dsgvo-auskunft: Audit-Protokollierung fehlgeschlagen: %v", err)
	}
}

// DsgvoAuskunftHandler stellt die vollständige Betroffenenauskunft nach
// Art. 15 DSGVO für einen Leser zusammen, gleich welcher Art (pkg/leserart). Die Erteilung
// selbst wird im Audit-Log protokolliert (Rechenschaftspflicht, Art. 5 Abs. 2 DSGVO).
//
// Die Annotationen stehen direkt über dem Handler: swag ordnet sie der folgenden Deklaration
// zu und übergeht sie an anderer Stelle still.
// @Summary      DSGVO-Betroffenenauskunft (Art. 15) für einen Leser
// @Tags         students
// @Produce      json
// @Param        id   path      string  true  "Reader ID (UUID)"
// @Success      200  {object}  auskunft.Antwort
// @Failure      404  {object}  map[string]string
// @Router       /schueler/{id}/dsgvo-auskunft [get]
func (s *Tueren) DsgvoAuskunftHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("missing student ID parameter", nil)
		}
		ctx := r.Context()

		daten, err := s.sammleDsgvoDaten(ctx, id, s.HatRecht(r, dsgvoKontoRecht))
		if err != nil {
			return err
		}

		// Rechenschaftspflicht: Die Auskunftserteilung selbst wird protokolliert.
		s.protokolliereDsgvoAuskunft(ctx, id)

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
		ProtokollEintraege:    daten.auditEintraege,
		Verwaltung:            daten.verwaltung,
		Zugangskonto:          daten.zugangskonto,
		FruehereZugangskonten: daten.fruehereKonten,
		Verarbeitungsangaben:  daten.verarbeitung,
	}
}

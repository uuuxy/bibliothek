package api

import (
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
func (s *Server) dsgvoFristen(ctx context.Context) auskunft.DsgvoFristWerte {
	einst, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil || einst == nil {
		return auskunft.DsgvoFristWerte{
			LesehistorieTage: repository.StandardLesehistorieTage,
			LernmittelTage:   repository.StandardLesehistorieLernmittelTage,
			KarenzTage:       repository.StandardAbgaengerKarenzTage,
			AuditMonate:      repository.StandardAuditAufbewahrungMonate,
			AnliegenTage:     repository.StandardAnliegenTage,
		}
	}
	return auskunft.DsgvoFristWerte{
		LesehistorieTage: repository.TageOderStandard(einst.LesehistorieTage, repository.StandardLesehistorieTage),
		LernmittelTage:   repository.TageOderStandard(einst.LesehistorieLernmittelTage, repository.StandardLesehistorieLernmittelTage),
		KarenzTage:       repository.AbgaengerKarenzTageOderStandard(einst),
		AuditMonate:      repository.AufbewahrungMonateOderStandard(einst.AuditAufbewahrungMonate),
		AnliegenTage:     repository.TageOderStandard(einst.AnliegenTage, repository.StandardAnliegenTage),
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

func (s *Server) dsgvoQueryStammdaten(ctx context.Context, id string) (*auskunft.DsgvoStammdaten, error) {
	zeile, err := repository.LeseDsgvoStammdaten(ctx, s.DB.Pool, id)
	if err != nil || zeile == nil {
		return nil, err
	}
	st := auskunft.DsgvoStammdaten(*zeile)
	return &st, nil
}

func (s *Server) dsgvoQueryFoto(ctx context.Context, id string) (auskunft.DsgvoFoto, error) {
	foto := auskunft.DsgvoFoto{Hinweis: "Foto wird verschlüsselt gespeichert; Kopie über die Akte in der Leserdatei abrufbar"}
	aktualisiert, err := repository.LeseDsgvoFotoStand(ctx, s.DB.Pool, id)
	if err != nil {
		return foto, err
	}
	if aktualisiert == nil {
		return auskunft.DsgvoFoto{Vorhanden: false, Hinweis: "Kein Foto gespeichert"}, nil
	}
	foto.Vorhanden = true
	foto.AktualisiertAm = aktualisiert
	return foto, nil
}

func (s *Server) dsgvoQueryAusleihen(ctx context.Context, id string) ([]auskunft.DsgvoAusleihe, error) {
	zeilen, err := repository.LeseDsgvoAusleihen(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DsgvoAusleiheZeile) auskunft.DsgvoAusleihe {
		return auskunft.DsgvoAusleihe(z)
	})
}

func (s *Server) dsgvoQuerySchadensfaelle(ctx context.Context, id string) ([]auskunft.DsgvoSchadensfall, error) {
	zeilen, err := repository.LeseDsgvoSchadensfaelle(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DsgvoSchadensfallZeile) auskunft.DsgvoSchadensfall {
		return auskunft.DsgvoSchadensfall(z)
	})
}

func (s *Server) dsgvoQueryVormerkungen(ctx context.Context, id string) ([]auskunft.DsgvoVormerkung, error) {
	zeilen, err := repository.LeseDsgvoVormerkungen(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DsgvoVormerkungZeile) auskunft.DsgvoVormerkung {
		return auskunft.DsgvoVormerkung(z)
	})
}

func (s *Server) dsgvoQueryNachbuchMeldungen(ctx context.Context, id string) ([]auskunft.DsgvoNachbuchMeldung, error) {
	zeilen, err := repository.LeseDsgvoNachbuchMeldungen(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DsgvoNachbuchZeile) auskunft.DsgvoNachbuchMeldung {
		return auskunft.DsgvoNachbuchMeldung(z)
	})
}

func (s *Server) dsgvoQueryBescheide(ctx context.Context, id string) ([]auskunft.DsgvoBescheid, error) {
	zeilen, err := repository.LeseDsgvoBescheide(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DsgvoBescheidZeile) auskunft.DsgvoBescheid {
		return auskunft.DsgvoBescheid(z)
	})
}

func (s *Server) dsgvoQueryAuditEintraege(ctx context.Context, id string) ([]auskunft.DsgvoAuditEintrag, error) {
	zeilen, err := repository.LeseDsgvoAuditEintraege(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DsgvoAuditZeile) auskunft.DsgvoAuditEintrag {
		return auskunft.DsgvoAuditEintrag(z)
	})
}

func (s *Server) dsgvoQueryVerwaltungsEintraege(ctx context.Context, id string) ([]auskunft.DsgvoVerwaltungsEintrag, error) {
	zeilen, err := repository.LeseDsgvoVerwaltungsEintraege(ctx, s.DB.Pool, id)
	return alsAntwort(zeilen, err, func(z repository.DsgvoVerwaltungZeile) auskunft.DsgvoVerwaltungsEintrag {
		return auskunft.DsgvoVerwaltungsEintrag(z)
	})
}

// dsgvoDaten bündelt alle personenbezogenen Daten eines Lesers für die Auskunft.
type dsgvoDaten struct {
	stammdaten        *auskunft.DsgvoStammdaten
	foto              auskunft.DsgvoFoto
	ausleihen         []auskunft.DsgvoAusleihe
	schaeden          []auskunft.DsgvoSchadensfall
	vormerkungen      []auskunft.DsgvoVormerkung
	bescheide         []auskunft.DsgvoBescheid
	nachbuchMeldungen []auskunft.DsgvoNachbuchMeldung
	auditEintraege    []auskunft.DsgvoAuditEintrag
	verwaltung        []auskunft.DsgvoVerwaltungsEintrag
	zugangskonto      *repository.DsgvoZugangskonto
	fruehereKonten    []repository.DsgvoFrueheresZugangskonto
	verarbeitung      auskunft.DsgvoVerarbeitungsangaben
}

// dsgvoKontoRecht verlangt die Auskunft zusätzlich zum Recht der Route, sobald auf den Leser
// ein Zugangskonto zeigt (seit 28.09.2026) oder einmal gezeigt hat (seit 29.09.2026: Die
// Auskunft nennt dann das gelöschte Konto samt seinen Einträgen, dieselbe Art Daten). Dann nennt sie das Konto, seine Einträge im
// Verwaltungsprotokoll und jeden Vorgang, den die Person selbst bearbeitet hat, bei
// Verwaltungseingriffen mit IP-Adresse. Konten und dieses Protokoll zeigt die Anwendung sonst
// nur mit manage_users (GET /api/benutzer, GET /api/admin/auditlog). Das Recht der Route,
// manage_students_admin, hat ab Werk auch die Leitung, und es ist zum Delegieren ans
// Sekretariat gedacht (db/seed.go, RechteOptional). Gate:
// TestDsgvoAuskunft_KontoVerlangtKontenrecht.
const dsgvoKontoRecht = "manage_users"

// sammleDsgvoDaten lädt alle personenbezogenen Daten eines Lesers — Schüler, Lehrkraft
// oder LiV — für die Art.-15-Auskunft. Fehler sind bereits als HTTP-Fehler (apierrors)
// verpackt. darfKonto sagt, ob der Aufrufer dsgvoKontoRecht hat; ohne das Recht endet die
// Auskunft über einen Leser mit Zugangskonto mit 403, bevor sie protokolliert wird.
func (s *Server) sammleDsgvoDaten(ctx context.Context, id string, darfKonto bool) (*dsgvoDaten, error) {
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
	zugangskonto, err := repository.LeseDsgvoZugangskonto(ctx, s.DB.Pool, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden des Zugangskontos", err)
	}
	fruehereKonten, err := repository.LeseDsgvoFruehereZugangskonten(ctx, s.DB.Pool, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der früheren Zugangskonten", err)
	}
	// Geprüft an den gelesenen Konten, nicht an stammdaten.HatZugangskonto: Entscheidend ist,
	// was die Antwort enthalten würde — auch ein gelöschtes Konto samt seinen Einträgen.
	if (zugangskonto != nil || len(fruehereKonten) > 0) && !darfKonto {
		return nil, apierrors.New(http.StatusForbidden,
			"Die Auskunft über einen Leser mit Zugangskonto verlangt das Recht „Benutzer & Rechte verwalten“.", nil)
	}
	verarbeitung := auskunft.DsgvoPflichtangaben(stammdaten.Art, s.dsgvoFristen(ctx))

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
func (s *Server) protokolliereDsgvoAuskunft(ctx context.Context, id string) {
	akteur := "SYSTEM"
	var bearbeiterID *string
	if claims, ok := auth.GetClaims(ctx); ok {
		akteur = "USER"
		bearbeiterID = &claims.UserID
	}
	if err := repository.SchreibeDsgvoAuskunftProtokoll(ctx, s.DB.Pool, id, bearbeiterID, akteur); err != nil {
		log.Printf("dsgvo-auskunft: Audit-Protokollierung fehlgeschlagen: %v", err)
	}
}

// DsgvoAuskunftHandler stellt die vollständige Betroffenenauskunft nach
// Art. 15 DSGVO für einen Leser zusammen, gleich welcher Art (pkg/leserart). Die Erteilung
// selbst wird im Audit-Log protokolliert (Rechenschaftspflicht, Art. 5 Abs. 2 DSGVO).
//
// Der Annotationsblock stand bis zum 05.08.2026 rund 70 Zeilen weiter oben — über einem
// Struct statt über diesem Handler. swag ordnet Annotationen der FOLGENDEN Deklaration
// zu, hat den Block deshalb übergangen, und der Endpunkt fehlte in der Swagger-Datei.
// @Summary      DSGVO-Betroffenenauskunft (Art. 15) für einen Leser
// @Tags         students
// @Produce      json
// @Param        id   path      string  true  "Reader ID (UUID)"
// @Success      200  {object}  auskunft.DsgvoAuskunftResponse
// @Failure      404  {object}  map[string]string
// @Router       /schueler/{id}/dsgvo-auskunft [get]
func (s *Server) DsgvoAuskunftHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("missing student ID parameter", nil)
		}
		ctx := r.Context()

		daten, err := s.sammleDsgvoDaten(ctx, id, s.BesitztRecht(r, dsgvoKontoRecht))
		if err != nil {
			return err
		}

		// Rechenschaftspflicht: Die Auskunftserteilung selbst wird protokolliert.
		s.protokolliereDsgvoAuskunft(ctx, id)

		RespondJSON(w, http.StatusOK, dsgvoAntwort(daten, time.Now()))
		return nil
	})
}

// dsgvoAntwort ist die Auskunft als EIN Objekt: Die JSON-Antwort und das PDF entstehen
// beide daraus. Bis zum 24.09.2026 setzte der JSON-Handler sein Objekt selbst zusammen und
// das PDF las die Einzelteile — das PDF ließ dabei die Nachbuch-Meldungen aus, und niemand
// merkte es. Gate: TestDsgvoPDF_DrucktJedeAngabeDerAuskunft.
func dsgvoAntwort(daten *dsgvoDaten, erstelltAm time.Time) auskunft.DsgvoAuskunftResponse {
	return auskunft.DsgvoAuskunftResponse{
		Art:                   "Auskunft nach Art. 15 DSGVO",
		ErstelltAm:            erstelltAm,
		Stammdaten:            *daten.stammdaten,
		Foto:                  daten.foto,
		Ausleihen:             daten.ausleihen,
		Schadensfaelle:        daten.schaeden,
		Vormerkungen:          daten.vormerkungen,
		Bescheide:             daten.bescheide,
		NachbuchMeldungen:     daten.nachbuchMeldungen,
		AuditEintraege:        daten.auditEintraege,
		Verwaltung:            daten.verwaltung,
		Zugangskonto:          daten.zugangskonto,
		FruehereZugangskonten: daten.fruehereKonten,
		Verarbeitungsangaben:  daten.verarbeitung,
	}
}

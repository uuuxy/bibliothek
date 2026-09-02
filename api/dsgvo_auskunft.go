package api

import (
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/repository"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// DsgvoStammdaten umfasst sämtliche im Schülerdatensatz gespeicherten
// Stammdaten — bewusst inklusive Soft-Delete-Zeitpunkt und Sperrgrund,
// denn die Auskunft nach Art. 15 DSGVO deckt alles ab, was gespeichert ist.
// „Sämtliche" ist seit 02.09.2026 gemessen: TestDsgvoAuskunft_KenntJedeSchuelerSpalte
// hält jede Spalte von schueler gegen dsgvoStammdatenSQL.
type DsgvoStammdaten struct {
	ID                string     `json:"id"`
	BarcodeID         string     `json:"barcode_id"`
	Vorname           string     `json:"vorname"`
	Nachname          string     `json:"nachname"`
	Klasse            string     `json:"klasse"`
	Geburtsdatum      *string    `json:"geburtsdatum"`
	AbgaengerJahr     int        `json:"abgaenger_jahr"`
	IstGesperrt       bool       `json:"ist_gesperrt"`
	IstAbgaenger      bool       `json:"ist_abgaenger"`
	LusdID            *string    `json:"lusd_id"`
	Strasse           string     `json:"strasse"`
	Hausnummer        string     `json:"hausnummer"`
	Plz               string     `json:"plz"`
	Ort               string     `json:"ort"`
	ElternEmail       string     `json:"eltern_email"`
	IsManuallyBlocked bool       `json:"manuell_gesperrt"`
	BlockReason       *string    `json:"sperrgrund"`
	ErstelltAm        time.Time  `json:"erfasst_am"`
	AktualisiertAm    time.Time  `json:"zuletzt_aktualisiert_am"`
	GeloeschtAm       *time.Time `json:"geloescht_am"`
	// Seit Migration 084/094 (nachgetragen 02.09.2026 — die Auskunft war um vier Spalten
	// unvollständig; Gate: TestDsgvoAuskunft_KenntJedeSchuelerSpalte).
	SchulEintrittAm  *string    `json:"schul_eintritt_am"`
	AbgaengerSeit    *time.Time `json:"abgaenger_seit"`
	LusdBestaetigtAm *time.Time `json:"lusd_bestaetigt_am"`
	AnonymisiertAm   *time.Time `json:"anonymisiert_am"`
}

// dsgvoStammdatenSQL ist die eine Spaltenliste der Auskunft. Sie steht außerhalb der
// Funktion, damit das Spalten-Gate sie gegen information_schema.columns halten kann.
//
// Adress-/Kontaktfelder sind in der DB nullbar (VARCHAR ohne NOT NULL), werden aber
// in nicht-nullbare Go-strings gescannt. Ohne COALESCE scheitert Scan(NULL → *string)
// mit 500 — das traf jeden Schüler ohne erfasste Adresse (nicht nur Demo-Daten).
const dsgvoStammdatenSQL = `
		SELECT id, barcode_id, vorname, nachname, klasse, geburtsdatum::text,
		       abgaenger_jahr, ist_gesperrt, ist_abgaenger, lusd_id,
		       COALESCE(strasse, '') AS strasse, COALESCE(hausnummer, '') AS hausnummer,
		       COALESCE(plz, '') AS plz, COALESCE(ort, '') AS ort,
		       COALESCE(eltern_email, '') AS eltern_email,
		       is_manually_blocked, block_reason,
		       erstellt_am, aktualisiert_am, deleted_at,
		       schul_eintritt_am::text, abgaenger_seit, lusd_bestaetigt_am, anonymized_at
		FROM schueler
		WHERE id = $1`

// DsgvoFoto beschreibt das (verschlüsselt gespeicherte) Ausweisfoto.
type DsgvoFoto struct {
	Vorhanden      bool       `json:"vorhanden"`
	AktualisiertAm *time.Time `json:"aktualisiert_am"`
	Hinweis        string     `json:"hinweis"`
}

// DsgvoAusleihe ist ein Eintrag der vollständigen Ausleihhistorie.
type DsgvoAusleihe struct {
	Gegenstand     string     `json:"gegenstand"`
	Barcode        string     `json:"barcode"`
	AusgeliehenAm  time.Time  `json:"ausgeliehen_am"`
	RueckgabeFrist time.Time  `json:"rueckgabe_frist"`
	RueckgabeAm    *time.Time `json:"rueckgabe_am"`
	IstHandapparat bool       `json:"ist_handapparat"`
}

// DsgvoSchadensfall ist ein gemeldeter Schadens-/Verlustfall des Schülers.
type DsgvoSchadensfall struct {
	Beschreibung      string     `json:"beschreibung"`
	Betrag            string     `json:"betrag_eur"`
	IstBezahlt        bool       `json:"ist_bezahlt"`
	ErstelltAm        time.Time  `json:"erstellt_am"`
	StorniertAm       *time.Time `json:"storniert_am"`
	Stornierungsgrund *string    `json:"stornierungsgrund"`
}

// DsgvoVormerkung ist eine Vormerkung auf einen Buchtitel.
type DsgvoVormerkung struct {
	Titel      string    `json:"titel"`
	Status     string    `json:"status"`
	Notiz      *string   `json:"notiz"`
	ErstelltAm time.Time `json:"erstellt_am"`
}

// DsgvoAuditEintrag ist ein Protokolleintrag, der den Schülerdatensatz betrifft.
type DsgvoAuditEintrag struct {
	Aktion    string    `json:"aktion"`
	Akteur    string    `json:"akteur"`
	Zeitpunkt time.Time `json:"zeitpunkt"`
	Kontext   *string   `json:"kontext"`
	// swaggertype: json.RawMessage ist ein []byte-Alias aus der Standardbibliothek, das
	// swag ohne --parseDependency nicht auflösen kann. Ohne diesen Hinweis bricht die
	// Generierung für DIESEN Endpunkt still ab — die DSGVO-Auskunft fehlte deshalb
	// komplett in der Swagger-Datei, obwohl sie annotiert war (gefunden 05.08.2026).
	Details json.RawMessage `json:"details" swaggertype:"object"`
}

// DsgvoVerwaltungsEintrag ist ein Eintrag des Verwaltungsprotokolls (audit_logs):
// Admin-Eingriffe wie DELETE/RESTORE/PURGE_STUDENT oder LUSD_ID_NACHGETRAGEN, die den
// Schüler über details->>'schueler_id' referenzieren. admin_id und ip_adresse sind
// Daten des BEARBEITERS, nicht des Schülers — sie gehören nicht in dessen Auskunft.
type DsgvoVerwaltungsEintrag struct {
	Aktion    string          `json:"aktion"`
	Zeitpunkt time.Time       `json:"zeitpunkt"`
	Details   json.RawMessage `json:"details" swaggertype:"object"`
}

// DsgvoVerarbeitungsangaben sind die Pflichtangaben nach Art. 15 Abs. 1 lit. a–d, g DSGVO.
type DsgvoVerarbeitungsangaben struct {
	Zwecke            []string `json:"zwecke"`
	Rechtsgrundlage   string   `json:"rechtsgrundlage"`
	Empfaenger        string   `json:"empfaenger"`
	Speicherdauer     string   `json:"speicherdauer"`
	Herkunft          string   `json:"herkunft_der_daten"`
	Betroffenenrechte string   `json:"betroffenenrechte"`
}

// DsgvoAuskunftResponse ist die vollständige Betroffenenauskunft nach Art. 15 DSGVO.
type DsgvoAuskunftResponse struct {
	Art                  string                    `json:"art"`
	ErstelltAm           time.Time                 `json:"auskunft_erstellt_am"`
	Stammdaten           DsgvoStammdaten           `json:"stammdaten"`
	Foto                 DsgvoFoto                 `json:"ausweisfoto"`
	Ausleihen            []DsgvoAusleihe           `json:"ausleihhistorie"`
	Schadensfaelle       []DsgvoSchadensfall       `json:"schadensfaelle"`
	Vormerkungen         []DsgvoVormerkung         `json:"vormerkungen"`
	AuditEintraege       []DsgvoAuditEintrag       `json:"protokolleintraege"`
	Verwaltung           []DsgvoVerwaltungsEintrag `json:"verwaltungsprotokolle"`
	Verarbeitungsangaben DsgvoVerarbeitungsangaben `json:"verarbeitungsangaben"`
}

// dsgvoVerarbeitungsangaben formuliert die Pflichtangaben nach Art. 15 Abs. 1 DSGVO —
// aus DENSELBEN Fristen, mit denen das System arbeitet (Einstellungen „Datenschutz &
// Sitzung"). Bis 22.08.2026 stand hier pauschal lit. e ohne Fristen, während SECURITY.md,
// VVT- und Art.-13-Entwurf längst zwei Tätigkeiten mit zwei Grundlagen und 90/730 Tagen
// beschrieben — eine Betroffenenauskunft mit falscher Pflichtangabe (Prüfung 22.08., B).
func dsgvoVerarbeitungsangaben(lesehistorieTage, lernmittelTage int) DsgvoVerarbeitungsangaben {
	frist := func(tage int) string {
		if tage <= 0 {
			return "bis zur Löschung des Schülerdatensatzes (Befristung in dieser Installation abgeschaltet)"
		}
		return fmt.Sprintf("%d Tage nach Rückgabe", tage)
	}
	return DsgvoVerarbeitungsangaben{
		Zwecke: []string{
			"Verwaltung der Lernmittelausleihe im Rahmen der Lernmittelfreiheit (Nachweis von Ausleihe und Rücklauf, Mahnung, Schadensersatz)",
			"Betrieb der Schülerbücherei (Ausleihe, Vormerkung, Rückgabeerinnerung)",
			"Abwicklung von Schadens- und Verlustfällen",
		},
		Rechtsgrundlage: "Lernmittelausleihe: Art. 6 Abs. 1 lit. e DSGVO i. V. m. § 83 und § 153 HSchG (Lernmittelfreiheit) und SchDSV — öffentlich-rechtliches Nutzungsverhältnis. " +
			"Schülerbücherei: Einwilligung, Art. 6 Abs. 1 lit. a DSGVO / § 3 SchDSV (freiwillige Nutzung; bei Minderjährigen durch die Erziehungsberechtigten), sofern die Schule sie nicht als schulische Aufgabe nach Art. 6 Abs. 1 lit. e führt — maßgeblich ist das Verzeichnis von Verarbeitungstätigkeiten der Schule.",
		Empfaenger: "Keine Übermittlung an Dritte; Verarbeitung durch das Bibliothekspersonal der Schule. Klassenleitungen erhalten die Liste überfälliger Medien ihrer Klasse. Helfer an der Theke sehen nur Name, Klasse und Sperrstatus.",
		Speicherdauer: "Ausleihvorgänge bleiben der Person zugeordnet: Schülerbücherei " + frist(lesehistorieTage) + ", Lernmittel " + frist(lernmittelTage) + "; danach automatisch getrennt. " +
			"Bearbeitende Person einer Ausleihe nach 14 Tagen entfernt. Schülerdatensatz: nach dem Abgang (ab 30. Januar des Folgejahres) gelöscht, sofern keine offene Ausleihe und kein unbezahlter Schadensfall; Altfälle nach 360 Tagen, Papierkorb nach 180 Tagen anonymisiert. Protokolle 24 Monate. Verschlüsselte Backups 14 Tage.",
		Herkunft:          "Stammdaten aus der Landesschülerdatenbank LUSD (Export/Import) bzw. manuelle Erfassung durch das Bibliotheksteam",
		Betroffenenrechte: "Recht auf Berichtigung (Art. 16), Löschung (Art. 17), Einschränkung (Art. 18) und Widerspruch (Art. 21) sowie Widerruf einer Einwilligung; Beschwerderecht beim Hessischen Beauftragten für Datenschutz und Informationsfreiheit (HBDI)",
	}
}

// dsgvoLesehistorieFristen liest die Befristung aus den Einstellungen; bei Fehlern gelten
// die Vorgaben (so arbeitet auch der Job).
func (s *Server) dsgvoLesehistorieFristen(ctx context.Context) (int, int) {
	einst, err := repository.NewSystemSettingsRepository(s.DB.Pool).GetSettings(ctx)
	if err != nil || einst == nil {
		return repository.StandardLesehistorieTage, repository.StandardLesehistorieLernmittelTage
	}
	return repository.TageOderStandard(einst.LesehistorieTage, repository.StandardLesehistorieTage),
		repository.TageOderStandard(einst.LesehistorieLernmittelTage, repository.StandardLesehistorieLernmittelTage)
}

func (s *Server) dsgvoQueryStammdaten(ctx context.Context, id string) (*DsgvoStammdaten, error) {
	var st DsgvoStammdaten
	err := s.DB.Pool.QueryRow(ctx, dsgvoStammdatenSQL, id).Scan(
		&st.ID, &st.BarcodeID, &st.Vorname, &st.Nachname, &st.Klasse, &st.Geburtsdatum,
		&st.AbgaengerJahr, &st.IstGesperrt, &st.IstAbgaenger, &st.LusdID,
		&st.Strasse, &st.Hausnummer, &st.Plz, &st.Ort, &st.ElternEmail,
		&st.IsManuallyBlocked, &st.BlockReason,
		&st.ErstelltAm, &st.AktualisiertAm, &st.GeloeschtAm,
		&st.SchulEintrittAm, &st.AbgaengerSeit, &st.LusdBestaetigtAm, &st.AnonymisiertAm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &st, nil
}

func (s *Server) dsgvoQueryFoto(ctx context.Context, id string) (DsgvoFoto, error) {
	foto := DsgvoFoto{Hinweis: "Foto wird verschlüsselt gespeichert; Kopie über das Schülerprofil abrufbar"}
	var aktualisiert time.Time
	err := s.DB.Pool.QueryRow(ctx,
		`SELECT aktualisiert_am FROM schueler_fotos WHERE schueler_id = $1`, id,
	).Scan(&aktualisiert)
	if errors.Is(err, pgx.ErrNoRows) {
		return DsgvoFoto{Vorhanden: false, Hinweis: "Kein Foto gespeichert"}, nil
	}
	if err != nil {
		return foto, err
	}
	foto.Vorhanden = true
	foto.AktualisiertAm = &aktualisiert
	return foto, nil
}

func (s *Server) dsgvoQueryAusleihen(ctx context.Context, id string) ([]DsgvoAusleihe, error) {
	const q = `
		SELECT COALESCE(t.titel, g.modellname, 'Unbekannt') AS gegenstand,
		       COALESCE(e.barcode_id, g.barcode_id, '') AS barcode,
		       a.ausgeliehen_am, a.rueckgabe_frist, a.rueckgabe_am, a.ist_handapparat
		FROM ausleihen a
		LEFT JOIN buecher_exemplare e ON e.id = a.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		LEFT JOIN geraete g ON g.id = a.geraet_id
		WHERE a.schueler_id = $1
		ORDER BY a.ausgeliehen_am DESC`
	rows, err := s.DB.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []DsgvoAusleihe{}
	for rows.Next() {
		var a DsgvoAusleihe
		if err := rows.Scan(&a.Gegenstand, &a.Barcode, &a.AusgeliehenAm, &a.RueckgabeFrist, &a.RueckgabeAm, &a.IstHandapparat); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Server) dsgvoQuerySchadensfaelle(ctx context.Context, id string) ([]DsgvoSchadensfall, error) {
	const q = `
		SELECT beschreibung, betrag::text, ist_bezahlt, erstellt_am, storniert_am, stornierungsgrund
		FROM schadensfaelle
		WHERE schueler_id = $1
		ORDER BY erstellt_am DESC`
	rows, err := s.DB.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []DsgvoSchadensfall{}
	for rows.Next() {
		var f DsgvoSchadensfall
		if err := rows.Scan(&f.Beschreibung, &f.Betrag, &f.IstBezahlt, &f.ErstelltAm, &f.StorniertAm, &f.Stornierungsgrund); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Server) dsgvoQueryVormerkungen(ctx context.Context, id string) ([]DsgvoVormerkung, error) {
	const q = `
		SELECT t.titel, v.status, v.notiz, v.erstellt_am
		FROM vormerkungen v
		JOIN buecher_titel t ON t.id = v.titel_id
		WHERE v.schueler_id = $1
		ORDER BY v.erstellt_am DESC`
	rows, err := s.DB.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []DsgvoVormerkung{}
	for rows.Next() {
		var v DsgvoVormerkung
		if err := rows.Scan(&v.Titel, &v.Status, &v.Notiz, &v.ErstelltAm); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Server) dsgvoQueryAuditEintraege(ctx context.Context, id string) ([]DsgvoAuditEintrag, error) {
	// Auch die Ausleih-Protokolle (CHECKOUT/RETURN) gehören zur Auskunft: Dort steht der
	// Schüler in details.schueler_id, datensatz_id ist das Exemplar. Ohne diesen Zweig
	// fehlte die Lesehistorie im Audit-Teil der Art.-15-Auskunft (Prüfung 22.08.2026, A5).
	const q = `
		SELECT aktion, akteur, timestamp, kontext, COALESCE(details, 'null'::jsonb)
		FROM audit_log
		WHERE (tabelle = 'schueler' AND datensatz_id = $1::uuid)
		   OR (tabelle = 'ausleihen' AND details->>'schueler_id' = $1::text)
		ORDER BY timestamp DESC`
	rows, err := s.DB.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []DsgvoAuditEintrag{}
	for rows.Next() {
		var e DsgvoAuditEintrag
		if err := rows.Scan(&e.Aktion, &e.Akteur, &e.Zeitpunkt, &e.Kontext, &e.Details); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// dsgvoQueryVerwaltungsEintraege liest die Verwaltungs-Protokolle (audit_logs). Bis zum
// 31.08.2026 fehlte diese Tabelle in der Auskunft komplett — sie war die eine Quelle mit
// Schülerbezug, die sammleDsgvoDaten nicht las (Gate: dsgvo_paar_vollstaendigkeit_test.go).
func (s *Server) dsgvoQueryVerwaltungsEintraege(ctx context.Context, id string) ([]DsgvoVerwaltungsEintrag, error) {
	const q = `
		SELECT aktion, zeitstempel, COALESCE(details, '{}'::jsonb)
		FROM audit_logs
		WHERE details->>'schueler_id' = $1
		ORDER BY zeitstempel DESC`
	rows, err := s.DB.Pool.Query(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []DsgvoVerwaltungsEintrag{}
	for rows.Next() {
		var e DsgvoVerwaltungsEintrag
		if err := rows.Scan(&e.Aktion, &e.Zeitpunkt, &e.Details); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// dsgvoDaten bündelt alle personenbezogenen Daten eines Schülers für die Auskunft.
type dsgvoDaten struct {
	stammdaten     *DsgvoStammdaten
	foto           DsgvoFoto
	ausleihen      []DsgvoAusleihe
	schaeden       []DsgvoSchadensfall
	vormerkungen   []DsgvoVormerkung
	auditEintraege []DsgvoAuditEintrag
	verwaltung     []DsgvoVerwaltungsEintrag
	verarbeitung   DsgvoVerarbeitungsangaben
}

// sammleDsgvoDaten lädt alle personenbezogenen Daten eines Schülers für die
// Art.-15-Auskunft. Fehler sind bereits als HTTP-Fehler (apierrors) verpackt.
func (s *Server) sammleDsgvoDaten(ctx context.Context, id string) (*dsgvoDaten, error) {
	stammdaten, err := s.dsgvoQueryStammdaten(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Stammdaten", err)
	}
	if stammdaten == nil {
		return nil, apierrors.NotFound("student record not found", nil)
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
	auditEintraege, err := s.dsgvoQueryAuditEintraege(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Protokolleinträge", err)
	}
	verwaltung, err := s.dsgvoQueryVerwaltungsEintraege(ctx, id)
	if err != nil {
		return nil, apierrors.Internal("Fehler beim Laden der Verwaltungsprotokolle", err)
	}
	verarbeitung := dsgvoVerarbeitungsangaben(s.dsgvoLesehistorieFristen(ctx))

	return &dsgvoDaten{
		verarbeitung:   verarbeitung,
		stammdaten:     stammdaten,
		foto:           foto,
		ausleihen:      ausleihen,
		schaeden:       schaeden,
		vormerkungen:   vormerkungen,
		auditEintraege: auditEintraege,
		verwaltung:     verwaltung,
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
	if _, err := s.DB.Pool.Exec(ctx,
		`INSERT INTO audit_log (tabelle, aktion, datensatz_id, bearbeiter_id, akteur)
		 VALUES ('schueler', 'dsgvo_auskunft', $1::uuid, $2, $3)`,
		id, bearbeiterID, akteur,
	); err != nil {
		log.Printf("dsgvo-auskunft: Audit-Protokollierung fehlgeschlagen: %v", err)
	}
}

// DsgvoAuskunftHandler stellt die vollständige Betroffenenauskunft nach
// Art. 15 DSGVO für einen Schüler zusammen. Die Erteilung selbst wird im
// Audit-Log protokolliert (Rechenschaftspflicht, Art. 5 Abs. 2 DSGVO).
//
// Der Annotationsblock stand bis zum 05.08.2026 rund 70 Zeilen weiter oben — über einem
// Struct statt über diesem Handler. swag ordnet Annotationen der FOLGENDEN Deklaration
// zu, hat den Block deshalb übergangen, und der Endpunkt fehlte in der Swagger-Datei.
// @Summary      DSGVO-Betroffenenauskunft (Art. 15) für einen Schüler
// @Tags         students
// @Produce      json
// @Param        id   path      string  true  "Student ID (UUID)"
// @Success      200  {object}  DsgvoAuskunftResponse
// @Failure      404  {object}  map[string]string
// @Router       /schueler/{id}/dsgvo-auskunft [get]
func (s *Server) DsgvoAuskunftHandler() http.HandlerFunc {
	return apierrors.Wrap(func(w http.ResponseWriter, r *http.Request) error {
		id := r.PathValue("id")
		if id == "" {
			return apierrors.BadRequest("missing student ID parameter", nil)
		}
		ctx := r.Context()

		daten, err := s.sammleDsgvoDaten(ctx, id)
		if err != nil {
			return err
		}

		// Rechenschaftspflicht: Die Auskunftserteilung selbst wird protokolliert.
		s.protokolliereDsgvoAuskunft(ctx, id)

		RespondJSON(w, http.StatusOK, DsgvoAuskunftResponse{
			Art:                  "Auskunft nach Art. 15 DSGVO",
			ErstelltAm:           time.Now(),
			Stammdaten:           *daten.stammdaten,
			Foto:                 daten.foto,
			Ausleihen:            daten.ausleihen,
			Schadensfaelle:       daten.schaeden,
			Vormerkungen:         daten.vormerkungen,
			AuditEintraege:       daten.auditEintraege,
			Verwaltung:           daten.verwaltung,
			Verarbeitungsangaben: daten.verarbeitung,
		})
		return nil
	})
}

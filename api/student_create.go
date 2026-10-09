package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// meldungSchuelerDuplikat nennt die Regel, an der die Anlage scheitert — auch die
// Schreibweise, denn genau die ist der Fall, den man vor sich hat, wenn man den Namen
// in der Liste nicht findet.
const meldungSchuelerDuplikat = "achtung: Ein Schüler mit diesem Namen (auch in anderer Schreibweise: Müller/Mueller, Groß/Klein) und Geburtsdatum existiert bereits im System"

// CreateStudentRequest defines the payload for creating a new reader.
//
// Der Name des Typs ist geblieben, der Inhalt ist mehr: Seit dem 16.09.2026 fragt „Neuen
// Leser anlegen" ZUERST nach der Art. Klasse und Geburtsdatum sind deshalb nicht mehr am
// Feld als Pflicht markiert, sondern im Handler AN DIE ART GEPAART (pruefeLeserAngaben) —
// genau so, wie es die Datenbank tut (chk_leser_schueler_pflichtfelder). Zwei Pflichten,
// die für alle gelten, wären hier dasselbe wie gar keine: Ein Kollege hat keine Klasse.
type CreateStudentRequest struct {
	Vorname  string `json:"vorname" validate:"required"`
	Nachname string `json:"nachname" validate:"required"`
	// Art: eine aus leserArten (api/leser_art.go). Leer heißt „schueler" — die Vorgabe der
	// Spalte und das Verhalten jedes Aufrufers, den es vor dem 16.09.2026 gab.
	Art string `json:"art"`
	// Email ist die Schuladresse und PFLICHT, wo ein Zugang zu „Mein Portal" dazugehört:
	// Lehrkraft, LiV, Sekretariat, U-plus (repository.ArtMitKonto; Absprache vom 16.09.2026).
	// Bei einem Schüler, einem Praktikum und einem Fachbereich bleibt sie leer — sie
	// bekommen kein Konto (Entscheidung vom 30.09.2026).
	//
	// Sie ist nicht Kontaktangabe, sondern SCHLÜSSEL: An ihr erkennt die Anmeldung eine
	// Person (IMAP), und über sie greift `benutzer_email_unique`. Genau das verhindert den
	// Doppeleintrag, um den es hier geht — meldet sich die Lehrkraft später über „Mein
	// Portal" selbst an, findet die Selbstanmeldung ihr Konto und legt keine zweite
	// Leserzeile an.
	Email     string `json:"email"`
	Klasse    string `json:"klasse"`
	BarcodeID string `json:"barcode_id"`
	// Geburtsdatum (YYYY-MM-DD) ist Pflicht für einen SCHÜLER — geprüft im Handler mit
	// eigener Meldung, weil der Grund erklärt werden muss (LUSD-Wiedererkennung), siehe
	// errGeburtsdatumPflicht.
	Geburtsdatum *string `json:"geburtsdatum"`
}

// meldungLeserNamensdublette warnt vor dem häufigsten Fall: Der Kollege hat sich längst
// selbst angemeldet und steht deshalb schon in der Leserdatei. Ein zweiter Eintrag teilt
// seine Ausleihen auf zwei Akten, ohne dass es jemand merkt — und anders als bei einem
// Schüler gibt es kein Geburtsdatum, an dem die Doppelprüfung greifen könnte.
const meldungLeserNamensdublette = "achtung: Unter diesem Namen steht bereits ein Leser in der Leserdatei. Hat sich die Person über Mein Portal schon selbst angemeldet? Ein zweiter Eintrag teilt ihre Ausleihen auf zwei Akten."

// pruefeKollegiumEmail prüft die Schuladresse eines Kollegen mit Zugang (repository.ArtMitKonto).
//
// PFLICHT, und zwar aus einem Grund, der nichts mit Erreichbarkeit zu tun hat: Ohne sie
// entsteht kein Konto, und ohne Konto steht die Person zweimal in der Leserdatei, sobald
// sie sich über „Mein Portal" selbst anmeldet — einmal von Hand, einmal vom Wächter
// trg_benutzer_hat_leserzeile. Ausweis und Ausleihen hängen dann am ersten Eintrag, die
// Anmeldung am zweiten, und niemand merkt es.
//
// Die Domain wird geprüft, WENN eine freigegeben ist (SELBSTANMELDUNG_DOMAIN). Das ist
// keine Förmlichkeit: Angemeldet wird über IMAP gegen den Schulserver. Eine fremde
// Adresse ergäbe ein Konto, an dem sich niemand anmelden kann — besser jetzt eine klare
// Meldung als später eine unerklärliche Abweisung.
func pruefeKollegiumEmail(roh string) error {
	email := strings.TrimSpace(roh)
	if email == "" {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
		return errors.New("Die Schul-E-Mail-Adresse ist Pflicht: An ihr erkennt die Anmeldung die Person, und sie verhindert einen zweiten Eintrag, wenn sie sich später selbst anmeldet.")
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || strings.ContainsAny(email, " \t") {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
		return fmt.Errorf("%q ist keine E-Mail-Adresse.", email)
	}
	// Vergleich über das LETZTE „@" und die vollständige Domain — dieselbe Regel wie in
	// auth.darfSichSelbstAnmelden. Ein Suffix-Vergleich ließe „boesephilipp-reis-schule.de"
	// durch.
	if freigegeben := auth.SelbstanmeldeDomain(); freigegeben != "" {
		if !strings.EqualFold(strings.TrimSpace(email[at+1:]), freigegeben) {
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
			return fmt.Errorf("Die Adresse muss auf @%s enden — angemeldet wird über den Schulserver.", freigegeben)
		}
	}
	return nil
}

// pruefeLeserAngaben paart die Pflichtfelder an die Art — dieselbe Paarung, die
// chk_leser_schueler_pflichtfelder in der Datenbank hält.
//
// Die Paarung ist der Punkt und nicht der Wert: Am 14.09.2026 hat genau diese Bugklasse
// die LUSD-Klasse getroffen (ein Gate prüfte den Wert, nicht die Paarung). Ein Schüler
// ohne Klasse fällt in jeder Klassenliste und jeder Mahnung lautlos hinten runter; ein
// Kollege MIT Klasse stünde umgekehrt in den Klassenlisten und im LUSD-Abgleich.
func pruefeLeserAngaben(req *CreateStudentRequest) error {
	if !leserArten[req.Art] {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
		return fmt.Errorf("Unbekannte Art %q. Möglich sind: %s.", req.Art, moeglicheArten())
	}
	if err := pruefeEmailZurArt(req.Art, req.Email); err != nil {
		return err
	}
	if istSchuelerArt(req.Art) {
		if req.Klasse == "" {
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
			return errors.New("Klasse fehlt. Ein Schüler ohne Klasse fällt aus jeder Klassenliste und jeder Mahnung.")
		}
		return pruefeKlassenname(req.Klasse)
	}
	if req.Klasse != "" {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
		return errors.New("Nur ein Schüler hat eine Klasse. Mit einer Klasse stünde die Person in den Klassenlisten und im LUSD-Abgleich.")
	}
	return nil
}

// meldungKeinZugang weist eine Schul-E-Mail bei Praktikum und Fachbereich ab. Aus der Adresse
// entstünde ein Zugang zu „Mein Portal", und den bekommen sie nicht (repository.ArtMitKonto).
const meldungKeinZugang = "Praktikum und Fachbereich bekommen keinen Zugang zu „Mein Portal“ und deshalb keine Schul-E-Mail."

// pruefeEmailZurArt paart die Schul-E-Mail an die Art: Pflicht, wo ein Zugang dazugehört;
// sonst muss sie leer bleiben. Dieselbe Regel gilt beim Nachtragen in der Akte
// (pruefeSchulEmail).
func pruefeEmailZurArt(art, email string) error {
	switch {
	case repository.ArtMitKonto(art):
		return pruefeKollegiumEmail(email)
	case strings.TrimSpace(email) == "":
		return nil
	case istSchuelerArt(art):
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
		return errors.New("Ein Schüler bekommt kein Konto und keine E-Mail-Adresse.")
	}
	//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
	return errors.New(meldungKeinZugang)
}

// errGeburtsdatumPflicht erklärt dem Sekretariat, WARUM das Datum nicht fehlen darf —
// eine nackte Validierungsmeldung würde als Schikane gelesen und umgangen.
var errGeburtsdatumPflicht = errors.New("Geburtsdatum fehlt. Es ist der einzige Schlüssel, über den der LUSD-Import diesen Schüler später wiedererkennt — ohne Geburtsdatum würde er beim nächsten Import doppelt angelegt.") //nolint:staticcheck // ST1005: nutzer-sichtbarer Text im Anlege-Dialog

// CreateStudentHandler inserts a new student record into the database.
// @Summary      Create student
// @Description  Creates a new student profile in the library database.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        student  body      CreateStudentRequest  true  "Student creation payload"
// @Success      200      {object}  map[string]any
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /schueler [post]
func (s *Server) CreateStudentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateStudentRequest
		if !DecodeAndValidate(w, r, &req) {
			return
		}

		req.Vorname = strings.TrimSpace(req.Vorname)
		req.Nachname = strings.TrimSpace(req.Nachname)
		req.Klasse = strings.TrimSpace(req.Klasse)
		req.BarcodeID = strings.TrimSpace(req.BarcodeID)
		req.Art = strings.TrimSpace(req.Art)
		if req.Art == "" {
			req.Art = "schueler"
		}

		if err := pruefeLeserAngaben(&req); err != nil {
			apierrors.SendHTTPError(w, http.StatusBadRequest, err)
			return
		}

		ctx := r.Context()

		// Geburtsdatum ist Pflicht für einen SCHÜLER — nicht als Stammdatum, sondern als
		// SCHLÜSSEL: Der LUSD-Export der Schule hat keine Schüler-ID; der Import erkennt
		// einen von Hand angelegten Schüler ausschließlich über Name + Geburtsdatum wieder
		// (Adoption bzw. Namensmodus). Ohne Datum entsteht beim nächsten Import zwangsläufig
		// ein Duplikat.
		//
		// Ein Kollege kommt nie aus der LUSD. Von ihm ein Geburtsdatum zu verlangen, wäre
		// eine Angabe ohne Zweck — und damit eine, die nicht erhoben gehört.
		if istSchuelerArt(req.Art) && (req.Geburtsdatum == nil || strings.TrimSpace(*req.Geburtsdatum) == "") {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errGeburtsdatumPflicht)
			return
		}
		parsedGebdatum, ok := parseCreateGeburtsdatum(w, req.Geburtsdatum)
		if !ok {
			return
		}

		// Freischalten ist eine Entscheidung, kein technischer Schritt: Angemeldet wird über
		// IMAP, ein Passwort gibt es bei uns nicht. Wer ein Konto freischalten darf, sagt
		// `manage_users` — und nur wer das Recht hat, legt hier ein AKTIVES Konto an.
		//
		// Das Konto entsteht in beiden Fällen. Es muss auch entstehen: Seine eindeutige
		// E-Mail-Adresse ist das, was den zweiten Eintrag bei der Selbstanmeldung
		// verhindert. Ob es aktiv ist, ändert daran nichts — wer nicht freischalten darf,
		// erzeugt eine Zugangsanfrage, wie sie die Selbstanmeldung auch erzeugt.
		studentID, barcodeID, ok := s.legeSchuelerAn(ctx, w, req, parsedGebdatum, s.BesitztRecht(r, "manage_users"))
		if !ok {
			return
		}

		RespondJSON(w, http.StatusCreated, map[string]any{
			"status":     "success",
			"id":         studentID,
			"barcode_id": barcodeID,
		})
	}
}

// legeSchuelerAn wickelt die Neuanlage in einer Transaktion ab (Duplikatsprüfung,
// Barcode-Auflösung/-Generierung, Insert, Commit) und liefert die neue Schüler- und
// Barcode-ID. ok=false: die Fehlerantwort wurde bereits geschrieben.
func (s *Server) legeSchuelerAn(ctx context.Context, w http.ResponseWriter, req CreateStudentRequest, parsedGebdatum *time.Time, darfFreischalten bool) (studentID, barcodeID string, ok bool) {
	tx, err := s.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	defer db.SafeRollback(ctx, tx)

	// 1. Doppelprüfung, bevor irgendetwas geschrieben wird.
	if !antworteAufLeserDublette(ctx, tx, w, req, parsedGebdatum) {
		return "", "", false
	}

	// 2. Resolve/generate barcode_id if not provided
	barcodeID, ok = resolveNeueBarcodeID(ctx, tx, w, req.BarcodeID)
	if !ok {
		return "", "", false
	}

	// 3. Die Leserzeile anlegen.
	//
	// Geschrieben wird die TABELLE `leser` und nicht die Sicht `schueler`: Durch die Sicht
	// könnte ein Kollege gar nicht entstehen (WITH CHECK OPTION). Die Schüler-Pflichten
	// hält weiter die Datenbank — chk_leser_schueler_pflichtfelder prüft die PAARUNG von
	// Art und Klasse/Abgangsjahr/Ausweis, nicht die einzelnen Werte.
	//
	// Klasse und Abgangsjahr sind bei einem Kollegen NULL, nicht ”: Ein leerer String wäre
	// eine Klasse namens „nichts", und die Klassenlisten fragen auf NULL.
	var klasse *string
	var abgaengerJahr *int
	if istSchuelerArt(req.Art) {
		jahr := repository.AbgaengerJahr(req.Klasse)
		klasse, abgaengerJahr = &req.Klasse, &jahr
	}
	studentID, err = repository.LegeLeserAn(ctx, tx, repository.LeserNeu{
		Barcode:       barcodeID,
		Vorname:       req.Vorname,
		Nachname:      req.Nachname,
		Klasse:        klasse,
		Geburtsdatum:  parsedGebdatum,
		AbgaengerJahr: abgaengerJahr,
		Art:           req.Art,
	})
	if err != nil {
		antworteAufLeserInsertFehler(w, err, barcodeID)
		return "", "", false
	}

	// 4. Das Konto einer Lehrkraft — in DERSELBEN Transaktion. Praktikum und Fachbereich
	// bekommen keines (repository.ArtMitKonto), ein Schüler ohnehin nicht.
	//
	// `leser_id` wird ausdrücklich mitgegeben, damit der Wächter trg_benutzer_hat_leserzeile
	// NICHT anspringt: Er legt zu jedem Konto ohne Leserzeile eine frische an, und das wäre
	// hier die zweite — genau der Doppeleintrag, den diese Änderung abschafft.
	var kontoID string
	if repository.ArtMitKonto(req.Art) {
		// Das Konto entsteht in DERSELBEN Transaktion wie die Leserzeile — scheitert es,
		// darf auch die Zeile nicht stehen bleiben (die belegte Adresse ist der häufige
		// Fall und heisst: Die Person steht schon da).
		params := repository.LegeKollegiumskontoParams{
			Vorname:  req.Vorname,
			Nachname: req.Nachname,
			Email:    req.Email,
			LeserID:  studentID,
			Aktiv:    darfFreischalten,
		}
		if kontoID, err = repository.LegeKollegiumskonto(ctx, tx, params); err != nil {
			antworteAufKontoFehler(w, err, strings.ToLower(strings.TrimSpace(req.Email)))
			return "", "", false
		}
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	// Dieselbe Spur wie über Benutzer & Rechte (docs/OFFEN.md 5.19): Bis zum 29.09.2026
	// hinterließ ein Konto, das über die Leserdatei entstand, keinen Eintrag.
	if kontoID != "" {
		s.protokolliereKontoAnlage(ctx, kontoID, strings.ToLower(strings.TrimSpace(req.Email)), "kollegium", req.Vorname, req.Nachname)
	}
	return studentID, barcodeID, true
}

// antworteAufLeserDublette lehnt einen Leser ab, den es schon gibt: einen Schüler über Name
// und Geburtsdatum (dieselbe Regel wie der LUSD-Schlüssel), einen Kollegen über den Namen
// allein. Er hat kein Geburtsdatum, und der häufigste Fall ist der Kollege, der sich über
// „Mein Portal" längst selbst angemeldet hat. false heißt: Die Antwort ist geschrieben.
func antworteAufLeserDublette(ctx context.Context, tx pgx.Tx, w http.ResponseWriter, req CreateStudentRequest, parsedGebdatum *time.Time) bool {
	if istSchuelerArt(req.Art) {
		isDuplicate, err := repository.SchuelerDubletteVorhanden(ctx, tx, req.Vorname, req.Nachname, parsedGebdatum)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return false
		}
		if isDuplicate {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(meldungSchuelerDuplikat))
			return false
		}
		return true
	}
	belegt, err := repository.LeserNamensdubletteVorhanden(ctx, tx, req.Vorname, req.Nachname)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return false
	}
	if belegt {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
		apierrors.SendHTTPError(w, http.StatusConflict, errors.New(meldungLeserNamensdublette))
		return false
	}
	return true
}

// antworteAufLeserInsertFehler ordnet den Fehler beim Anlegen der Leserzeile ein. Die
// Indizes sind die letzte Instanz bei zwei Arbeitsplätzen gleichzeitig; ihre Verletzung ist
// ein Bedienfall mit Erklärung, kein 500.
func antworteAufLeserInsertFehler(w http.ResponseWriter, err error, barcodeID string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "unique_schueler_name_gebdatum" {
		apierrors.SendHTTPError(w, http.StatusConflict, errors.New(meldungSchuelerDuplikat))
		return
	}
	if repository.IstAusweisKollision(err) {
		apierrors.SendHTTPError(w, http.StatusConflict,
			fmt.Errorf("die Ausweisnummer '%s' wird bereits von einer anderen Person verwendet", barcodeID))
		return
	}
	if repository.IstNummerBuchOderAusweisKollision(err) {
		apierrors.SendHTTPError(w, http.StatusConflict,
			fmt.Errorf("die Nummer '%s' ist der Barcode eines Buchs und kann kein Ausweis sein", barcodeID))
		return
	}
	apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
}

// parseCreateGeburtsdatum parst das Geburtsdatum (YYYY-MM-DD) aus dem Anlage-Request.
// Der nil-/Leer-Zweig ist seit der Pflicht (errGeburtsdatumPflicht) nur noch Absicherung.
// ok=false bedeutet: die Fehlerantwort wurde bereits geschrieben.
func parseCreateGeburtsdatum(w http.ResponseWriter, raw *string) (*time.Time, bool) {
	if raw == nil || *raw == "" {
		return nil, true
	}
	t, err := time.Parse(dateFormatISO, *raw)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("ungültiges Format für Geburtsdatum, erwartet YYYY-MM-DD"))
		return nil, false
	}
	return &t, true
}

// resolveNeueBarcodeID liefert die zu verwendende Barcode-ID: entweder die vom Client
// gewünschte (nach Eindeutigkeitsprüfung) oder eine neu generierte S-Nummer aus der
// zentralen Sequenz. ok=false bedeutet: die Fehlerantwort wurde bereits geschrieben.
func resolveNeueBarcodeID(ctx context.Context, tx pgx.Tx, w http.ResponseWriter, requested string) (string, bool) {
	if requested == "" {
		// Use central repository for sequence generation
		seqRepo := repository.NewSequenceRepository(tx)
		// `leser`, nicht die Sicht `schueler`: Die Sicht zeigt nur Schüler, und die
		// höchste Nummer kann seit Migration 125 an einem KOLLEGEN hängen. Über die Sicht
		// gerechnet gäbe der Generator sie ein zweites Mal aus — und der eindeutige Index
		// quittierte das als 500 statt mit einer Auskunft. Ein Nummernkreis, zwei
		// Generatoren: genau der Fehler aus Migration 068, nur eine Tabelle weiter.
		startNum, err := seqRepo.NaechsteAusweisnummer(ctx)
		if err != nil {
			db.SafeRollback(ctx, tx)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return "", false
		}
		return repository.AusweisNummer(startNum), true
	}

	// Die Ausweisnummer gehört genau einer Person — Schüler wie Kollegium. Gefragt wird die
	// TABELLE `leser` und nicht die Sicht `schueler`: Sonst sähe die Prüfung einen Kollegen
	// nicht, ließe die Nummer durch, und der eindeutige Index quittierte es als 500 statt
	// mit einer Auskunft an die Bibliothek (Migration 125).
	//
	// Gelöschte Leser geben ihre Nummer frei — dieselbe Bedingung wie
	// uniq_schueler_barcode_active, sonst wiese die Prüfung eine Nummer ab, die die
	// Datenbank vergeben würde.
	exists, err := repository.AusweisnummerVergeben(ctx, tx, requested)
	if err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", false
	}
	if exists {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("Barcode-ID '%s' wird bereits verwendet", requested))
		return "", false
	}
	return requested, true
}

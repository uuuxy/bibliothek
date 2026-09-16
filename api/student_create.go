package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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

// abschlussJahrgang liest aus der Klassenbezeichnung den aktuellen Jahrgang und den
// Jahrgang, mit dem der Bildungsgang endet. Es ist der Go-Zwilling von
// repository.AbschlussklasseSQL — DIESELBE Regel (H ab 9, R ab 10, alles andere ab 13),
// belegt durch das Paar-Gate abgaenger_jahr_paar_pg_test.go, das beide über den ganzen
// Formenraum vergleicht (Zwilling mit Vollprobe, sweeps.md).
//
// Bis zum 07.09.2026 rechnete diese Funktion „R, G und unmarkiert → 10“: Ein 10G stand in
// der Akte mit dem Abgang des laufenden Jahres, obwohl der Gymnasialzweig an dieser Schule
// in die Oberstufe weiterläuft (Register B, 05.09.2026). Die Oberstufe selbst heißt hier
// ET/12T/13T: „E“ ohne Ziffer ist die Einführungsphase, Jahrgang 11.
//
// ok=false: keine lesbare Jahrgangszahl (ABG, Q4, leer) — der Aufrufer nimmt einen
// Rückfallwert.
func abschlussJahrgang(klasse string) (jahrgang, abschluss int, ok bool) {
	klasse = strings.ToLower(strings.TrimSpace(klasse))
	if strings.HasPrefix(klasse, "e") {
		return 11, 13, true // Einführungsphase: ET, E1, E2 — Jahrgang 11
	}

	gradeStr := ""
	suffix := ""
	for i, c := range klasse {
		if c >= '0' && c <= '9' {
			gradeStr += string(c)
		} else {
			suffix = strings.TrimSpace(klasse[i:])
			break
		}
	}
	grade, err := strconv.Atoi(gradeStr)
	if err != nil || grade < 1 {
		return 0, 0, false
	}

	switch {
	case strings.HasPrefix(suffix, "h"):
		return grade, 9, true
	case strings.HasPrefix(suffix, "r"):
		return grade, 10, true
	default:
		return grade, 13, true
	}
}

// calculateAbgaengerJahr errechnet das voraussichtliche Abgangsjahr eines Schülers aus
// der Klasse (abschlussJahrgang) und dem laufenden Schuljahr.
//
// Das Schuljahr endet im Juli; ab August läuft das neue Schuljahr, daher wird das
// Basisjahr um 1 erhöht, wenn wir uns ab August befinden.
func calculateAbgaengerJahr(klasse string) int {
	return abgaengerJahrAm(klasse, time.Now())
}

func abgaengerJahrAm(klasse string, jetzt time.Time) int {
	jahrgang, abschluss, ok := abschlussJahrgang(klasse)
	if !ok {
		return jetzt.Year() + 5 // Fallback
	}
	yearsLeft := abschluss - jahrgang
	if yearsLeft < 0 {
		yearsLeft = 0
	}
	baseYear := jetzt.Year()
	if jetzt.Month() >= time.August {
		baseYear++
	}
	return baseYear + yearsLeft
}

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
	// Art: schueler | lehrkraft | liv. Leer heißt „schueler" — die Vorgabe der Spalte
	// und das Verhalten jedes Aufrufers, den es vor dem 16.09.2026 gab.
	Art string `json:"art"`
	// Email ist die Schuladresse einer Lehrkraft oder LiV und dort PFLICHT (Peter,
	// 16.09.2026). Bei einem Schüler bleibt sie leer — er hat kein Konto.
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

// leserArten sind die drei Arten aus chk_leser_art (Migration 123). Eine vierte ist ein
// Tippfehler und kein neuer Personenkreis: Die Datenbank wiese sie ab, aber als 500
// „interner Datenbankfehler" statt mit einer Auskunft.
var leserArten = map[string]bool{"schueler": true, "lehrkraft": true, "liv": true}

// istSchuelerArt sagt, ob für diese Art die Schüler-Pflichten gelten.
func istSchuelerArt(art string) bool { return art == "schueler" }

// meldungLeserNamensdublette warnt vor dem häufigsten Fall: Der Kollege hat sich längst
// selbst angemeldet und steht deshalb schon in der Leserdatei. Ein zweiter Eintrag teilt
// seine Ausleihen auf zwei Akten, ohne dass es jemand merkt — und anders als bei einem
// Schüler gibt es kein Geburtsdatum, an dem die Doppelprüfung greifen könnte.
const meldungLeserNamensdublette = "achtung: Unter diesem Namen steht bereits ein Leser in der Leserdatei. Hat sich die Person über Mein Portal schon selbst angemeldet? Ein zweiter Eintrag teilt ihre Ausleihen auf zwei Akten."

// pruefeKollegiumEmail prüft die Schuladresse einer Lehrkraft oder LiV.
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
		return fmt.Errorf("Unbekannte Art %q. Möglich sind: Schüler, Lehrkraft, LiV.", req.Art)
	}
	if !istSchuelerArt(req.Art) {
		if err := pruefeKollegiumEmail(req.Email); err != nil {
			return err
		}
	} else if strings.TrimSpace(req.Email) != "" {
		//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
		return errors.New("Ein Schüler bekommt kein Konto und keine E-Mail-Adresse.")
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
		return errors.New("Eine Lehrkraft hat keine Klasse. Mit einer Klasse stünde sie in den Klassenlisten und im LUSD-Abgleich.")
	}
	return nil
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

	// 1. Notfall-Wachhund: Doppelprüfung.
	//
	// Bei einem Schüler über Name + Geburtsdatum (dieselbe Regel wie der LUSD-Schlüssel),
	// bei einem Kollegen über den NAMEN allein — er hat kein Geburtsdatum, an dem die
	// Schüler-Prüfung greifen könnte, und der häufigste Fall ist der Kollege, der sich
	// über „Mein Portal" längst selbst angemeldet hat.
	if istSchuelerArt(req.Art) {
		isDuplicate, err := pruefeSchuelerDuplikat(ctx, tx, req.Vorname, req.Nachname, parsedGebdatum)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return "", "", false
		}
		if isDuplicate {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(meldungSchuelerDuplikat))
			return "", "", false
		}
	} else {
		belegt, err := pruefeLeserNamensdublette(ctx, tx, req.Vorname, req.Nachname)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return "", "", false
		}
		if belegt {
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(meldungLeserNamensdublette))
			return "", "", false
		}
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
		jahr := calculateAbgaengerJahr(req.Klasse)
		klasse, abgaengerJahr = &req.Klasse, &jahr
	}
	qInsert := `
		INSERT INTO leser (barcode_id, vorname, nachname, klasse, geburtsdatum, abgaenger_jahr, art)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	if err := tx.QueryRow(ctx, qInsert, barcodeID, req.Vorname, req.Nachname, klasse, parsedGebdatum, abgaengerJahr, req.Art).Scan(&studentID); err != nil {
		// Der Index ist die letzte Instanz (zwei Arbeitsplätze gleichzeitig): seine
		// Verletzung ist ein Bedienfall mit Erklärung, kein 500.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "unique_schueler_name_gebdatum" {
			apierrors.SendHTTPError(w, http.StatusConflict, errors.New(meldungSchuelerDuplikat))
			return "", "", false
		}
		if repository.IstAusweisKollision(err) {
			apierrors.SendHTTPError(w, http.StatusConflict,
				fmt.Errorf("die Ausweisnummer '%s' wird bereits von einer anderen Person verwendet", barcodeID))
			return "", "", false
		}
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}

	// 4. Das Konto einer Lehrkraft — in DERSELBEN Transaktion.
	//
	// `leser_id` wird ausdrücklich mitgegeben, damit der Wächter trg_benutzer_hat_leserzeile
	// NICHT anspringt: Er legt zu jedem Konto ohne Leserzeile eine frische an, und das wäre
	// hier die zweite — genau der Doppeleintrag, den diese Änderung abschafft.
	if !istSchuelerArt(req.Art) {
		if !s.legeKollegiumskontoAn(ctx, w, tx, req, studentID, darfFreischalten) {
			return "", "", false
		}
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	return studentID, barcodeID, true
}

// legeKollegiumskontoAn hängt an die frische Leserzeile das Anmeldekonto.
//
// ok=false: Die Fehlerantwort steht bereits. Der wichtigste Fall ist die BELEGTE Adresse
// — sie heisst fast immer, dass die Person längst im System steht (etwa über die
// Selbstanmeldung). Das ist eine Auskunft und kein Fehler: Wer sie liest, soll den
// vorhandenen Eintrag suchen und nicht einen zweiten bauen.
func (s *Server) legeKollegiumskontoAn(ctx context.Context, w http.ResponseWriter, tx pgx.Tx,
	req CreateStudentRequest, leserID string, aktiv bool) bool {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	tag, err := tx.Exec(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, leser_id, zugang_beantragt_am)
		VALUES ($1, $2, $3, 'kollegium', $4, $5, CASE WHEN $4 THEN NULL ELSE CURRENT_TIMESTAMP END)
	`, req.Vorname, req.Nachname, email, aktiv, leserID)
	if err == nil {
		// Der CommandTag wird geprüft und nicht verworfen: Bliebe die Zeile aus, hätte die
		// Lehrkraft eine Leserzeile ohne Konto — sie stünde in der Leserdatei, käme aber
		// nie ins Portal, und der Dialog hätte „angelegt" gemeldet. Genau der Phantom-
		// Erfolg, den die Ratsche in phantom_erfolg_test.go abfängt.
		if tag.RowsAffected() != 1 {
			apierrors.SendHTTPError(w, http.StatusInternalServerError,
				fmt.Errorf("das Konto zu %s ist nicht entstanden", email))
			return false
		}
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		apierrors.SendHTTPError(w, http.StatusConflict,
			//nolint:staticcheck // ST1005: nutzer-sichtbare Meldung im Anlege-Dialog
			fmt.Errorf("Unter %s steht bereits ein Zugang. Die Person ist schon in der Leserdatei — bitte dort suchen, statt einen zweiten Eintrag anzulegen.", email))
		return false
	}
	apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
	return false
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

// pruefeSchuelerDuplikat erkennt einen bereits existierenden Schüler anhand von
// Vor-/Nachname und Geburtsdatum (case-insensitive, ohne soft-gelöschte Datensätze).
//
// Ein fehlendes Geburtsdatum ist bewusst KEIN Duplikat-Kriterium: Zwei namensgleiche
// Schüler ohne (noch nicht aus der LUSD übernommenes) Geburtsdatum sind nicht
// automatisch dieselbe Person. Nur bei beidseitig bekanntem, identischem Geburtsdatum
// liegt ein echtes Duplikat vor. `geburtsdatum = $3` liefert genau das: Ist eine der
// beiden Seiten NULL, ist der Vergleich SQL-NULL ("nicht gleich"), die Zeile zählt nicht
// als Treffer. Vorher stülpte coalesce beiden Seiten '1900-01-01' über und machte damit
// namensgleiche Schüler OHNE Geburtsdatum fälschlich zu Duplikaten (Zwillings-Blockade):
// der zweite "Leon Müller" ohne Geburtsdatum konnte gar nicht angelegt werden.
//
// Seit Migration 108 vergleicht die Prüfung in der Normalform suchnorm — derselben, in
// der der Unique-Index und der LUSD-Schlüssel rechnen: „Müller" und „Mueller" sind ein
// Mensch. Vorher nur lower(): Die Schreibvariante rutschte an der Prüfung vorbei und
// stand als zweite Zeile in der Datenbank.
func pruefeSchuelerDuplikat(ctx context.Context, tx pgx.Tx, vorname, nachname string, gebdatum *time.Time) (bool, error) {
	var isDuplicate bool
	q := `SELECT EXISTS(SELECT 1 FROM schueler WHERE suchnorm(vorname) = suchnorm($1) AND suchnorm(nachname) = suchnorm($2) AND geburtsdatum = $3::DATE AND deleted_at IS NULL)`
	err := tx.QueryRow(ctx, q, vorname, nachname, gebdatum).Scan(&isDuplicate)
	return isDuplicate, err
}

// pruefeLeserNamensdublette sucht einen aktiven Leser mit demselben Namen — in der
// Normalform suchnorm, also „Müller" wie „Mueller".
//
// Gefragt wird die TABELLE `leser` und über ALLE Arten: Ein Kollege, der schon als
// Schülerzeile aus dem Altbestand steht, ist derselbe Mensch. Die Prüfung ist bewusst
// grob — sie weist auch zwei echte Namensvettern ab. Das ist der seltenere Fall, und er
// meldet sich sofort; ein stiller zweiter Eintrag meldet sich nie.
func pruefeLeserNamensdublette(ctx context.Context, tx pgx.Tx, vorname, nachname string) (bool, error) {
	var belegt bool
	q := `SELECT EXISTS(SELECT 1 FROM leser
	       WHERE suchnorm(vorname) = suchnorm($1) AND suchnorm(nachname) = suchnorm($2)
	         AND deleted_at IS NULL)`
	err := tx.QueryRow(ctx, q, vorname, nachname).Scan(&belegt)
	return belegt, err
}

// AusweisPraefix steht auf JEDER Ausweisnummer, die dieses System vergibt — für einen
// Schüler wie für einen Kollegen.
//
// „A" wie Ausweis (Peter, 16.09.2026). Vorher gab es zwei: „S-" aus der Handanlage und
// dem LUSD-Import, „L-" aus dem Littera-Personenlauf. Beide Buchstaben behaupteten etwas
// über die PERSON — Schüler, Lehrer —, und das ist seit der Leserdatei falsch: Wer jemand
// ist, steht in den Stammdaten, nicht auf seinem Ausweis. Ein Nummernkreis, ein Buchstabe.
//
// Die Vorsilbe bleibt, sie ist kein Schmuck: OHNE NETZ ist sie die einzige Information,
// an der die Theke einen Buchscan von einem Ausweisscan unterscheiden kann — offline gibt
// es niemanden zu fragen (frontend/src/lib/stores/omnibox.svelte.js). Littera braucht sie
// nicht, weil dort Nummer und Scanwert zwei verschiedene Felder sind und der Scanwert vom
// Kartenhersteller kommt.
//
// Die alten Vorsilben versteht der Scanner weiterhin (internal/service/omnibox_service.go);
// vergeben werden sie nicht mehr.
const AusweisPraefix = "A-"

// AusweisNummer setzt eine laufende Zahl in die Form, die auf den Ausweis gedruckt wird.
// Fünfstellig mit führenden Nullen, damit die Nummern gleich lang bleiben und sich
// lexikografisch wie numerisch gleich sortieren.
func AusweisNummer(n int) string { return fmt.Sprintf("%s%05d", AusweisPraefix, n) }

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
		startNum, err := seqRepo.GetNextSequence(ctx, "leser", "barcode_id", AusweisPraefix)
		if err != nil {
			db.SafeRollback(ctx, tx)
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return "", false
		}
		return AusweisNummer(startNum), true
	}

	// Die Ausweisnummer gehört genau einer Person — Schüler wie Kollegium. Gefragt wird die
	// TABELLE `leser` und nicht die Sicht `schueler`: Sonst sähe die Prüfung einen Kollegen
	// nicht, ließe die Nummer durch, und der eindeutige Index quittierte es als 500 statt
	// mit einer Auskunft an die Bibliothek (Migration 125).
	//
	// Gelöschte Leser geben ihre Nummer frei — dieselbe Bedingung wie
	// uniq_schueler_barcode_active, sonst wiese die Prüfung eine Nummer ab, die die
	// Datenbank vergeben würde.
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM leser WHERE barcode_id = $1 AND deleted_at IS NULL)`,
		requested).Scan(&exists); err != nil {
		apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
		return "", false
	}
	if exists {
		apierrors.SendHTTPError(w, http.StatusBadRequest, fmt.Errorf("Barcode-ID '%s' wird bereits verwendet", requested))
		return "", false
	}
	return requested, true
}

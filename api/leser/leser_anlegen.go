package leser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/leserart"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// meldungSchuelerDuplikat nennt die Regel, an der die Anlage scheitert — auch die
// Schreibweise, denn genau die ist der Fall, den man vor sich hat, wenn man den Namen
// in der Liste nicht findet.
const meldungSchuelerDuplikat = "Achtung: Ein Schüler mit diesem Namen (auch in anderer Schreibweise: Müller/Mueller, Groß/Klein) und Geburtsdatum existiert bereits im System."

// leserAnlegenAnfrage ist ein neuer Leser, gleich welcher Art. „Neuen Leser anlegen" fragt
// zuerst nach der Art; Klasse und Geburtsdatum sind deshalb nicht am Feld als Pflicht
// markiert, sondern in der Tür an die Art gepaart (pruefeLeserAngaben), so wie es die
// Datenbank tut (chk_leser_schueler_pflichtfelder). Zwei Pflichten, die für alle gelten,
// wären hier dasselbe wie gar keine: Ein Kollege hat keine Klasse.
type leserAnlegenAnfrage struct {
	Vorname  string `json:"vorname" validate:"required"`
	Nachname string `json:"nachname" validate:"required"`
	// Art: eine aus pkg/leserart. Leer heißt „schueler", die Vorgabe der Spalte; ein
	// Aufrufer, der die Art nicht kennt, legt so weiter Schüler an.
	Art string `json:"art"`
	// Email ist die Schuladresse und Pflicht, wo ein Zugang zu „Mein Portal" dazugehört:
	// Lehrkraft, LiV, Sekretariat, U-plus (leserart.HatKonto). Bei einem Schüler, einem
	// Praktikum und einem Fachbereich bleibt sie leer; sie bekommen kein Konto.
	//
	// Sie ist nicht Kontaktangabe, sondern Schlüssel: An ihr erkennt die Anmeldung eine
	// Person (IMAP), und über sie greift `benutzer_email_unique`. Genau das verhindert den
	// Doppeleintrag, um den es hier geht — meldet sich die Lehrkraft später über „Mein
	// Portal" selbst an, findet die Selbstanmeldung ihr Konto und legt keine zweite
	// Leserzeile an.
	Email     string `json:"email"`
	Klasse    string `json:"klasse"`
	BarcodeID string `json:"barcode_id"`
	// Geburtsdatum (JJJJ-MM-TT) ist Pflicht für einen Schüler, geprüft in der Tür mit
	// eigener Meldung, weil der Grund erklärt werden muss (LUSD-Wiedererkennung), siehe
	// meldungGeburtsdatumPflicht.
	Geburtsdatum *string `json:"geburtsdatum"`
}

// meldungLeserNamensdublette warnt vor dem häufigsten Fall: Der Kollege hat sich längst
// selbst angemeldet und steht deshalb schon in der Leserdatei. Ein zweiter Eintrag teilt
// seine Ausleihen auf zwei Akten, ohne dass es jemand merkt — und anders als bei einem
// Schüler gibt es kein Geburtsdatum, an dem die Doppelprüfung greifen könnte.
const meldungLeserNamensdublette = "Achtung: Unter diesem Namen steht bereits ein Leser in der Leserdatei. Hat sich die Person über Mein Portal schon selbst angemeldet? Ein zweiter Eintrag teilt ihre Ausleihen auf zwei Akten."

// pruefeKollegiumEmail prüft die Schuladresse eines Kollegen mit Zugang (leserart.HatKonto).
//
// Pflicht, und zwar aus einem Grund, der nichts mit Erreichbarkeit zu tun hat: Ohne sie
// entsteht kein Konto, und ohne Konto steht die Person zweimal in der Leserdatei, sobald
// sie sich über „Mein Portal" selbst anmeldet — einmal von Hand, einmal vom Wächter
// trg_benutzer_hat_leserzeile. Ausweis und Ausleihen hängen dann am ersten Eintrag, die
// Anmeldung am zweiten, und niemand merkt es.
//
// Die Domain wird geprüft, wenn eine freigegeben ist (SELBSTANMELDUNG_DOMAIN). Das ist
// keine Förmlichkeit: Angemeldet wird über IMAP gegen den Schulserver. Eine fremde
// Adresse ergäbe ein Konto, an dem sich niemand anmelden kann — besser jetzt eine klare
// Meldung als später eine unerklärliche Abweisung.
func pruefeKollegiumEmail(roh string) error {
	email := strings.TrimSpace(roh)
	if email == "" {
		return apierrors.BadRequest("Die Schul-E-Mail-Adresse ist Pflicht: An ihr erkennt die Anmeldung die Person, und sie verhindert einen zweiten Eintrag, wenn sie sich später selbst anmeldet.", nil)
	}
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || strings.ContainsAny(email, " \t") {
		return apierrors.BadRequest(fmt.Sprintf("„%s“ ist keine E-Mail-Adresse", email), nil)
	}
	// Vergleich über das letzte „@" und die vollständige Domain — dieselbe Regel wie in
	// auth.darfSichSelbstAnmelden. Ein Suffix-Vergleich ließe „boesephilipp-reis-schule.de"
	// durch.
	if freigegeben := auth.SelbstanmeldeDomain(); freigegeben != "" {
		if !strings.EqualFold(strings.TrimSpace(email[at+1:]), freigegeben) {
			return apierrors.BadRequest(fmt.Sprintf("Die Adresse muss auf @%s enden. Angemeldet wird über den Schulserver.", freigegeben), nil)
		}
	}
	return nil
}

// pruefeLeserAngaben paart die Pflichtfelder an die Art — dieselbe Paarung, die
// chk_leser_schueler_pflichtfelder in der Datenbank hält.
//
// Die Paarung ist der Punkt und nicht der Wert: Ein Gate, das den Wert prüft und nicht die
// Paarung, übersieht genau diese Bugklasse. Ein Schüler ohne Klasse fällt in jeder
// Klassenliste und jeder Mahnung lautlos hinten runter; ein Kollege mit Klasse stünde
// umgekehrt in den Klassenlisten und im LUSD-Abgleich.
func pruefeLeserAngaben(req *leserAnlegenAnfrage) error {
	if !leserart.IstBekannt(req.Art) {
		return apierrors.BadRequest(meldungUnbekannteArt(req.Art), nil)
	}
	if err := pruefeEmailZurArt(req.Art, req.Email); err != nil {
		return err
	}
	if leserart.IstSchueler(req.Art) {
		if req.Klasse == "" {
			return apierrors.BadRequest("Klasse fehlt. Ein Schüler ohne Klasse fällt aus jeder Klassenliste und jeder Mahnung.", nil)
		}
		return pruefeKlassenname(req.Klasse)
	}
	if req.Klasse != "" {
		return apierrors.BadRequest("Nur ein Schüler hat eine Klasse. Mit einer Klasse stünde die Person in den Klassenlisten und im LUSD-Abgleich.", nil)
	}
	return nil
}

// meldungKeinZugang weist eine Schul-E-Mail bei Praktikum und Fachbereich ab. Aus der Adresse
// entstünde ein Zugang zu „Mein Portal", und den bekommen sie nicht (leserart.HatKonto).
const meldungKeinZugang = "Praktikum und Fachbereich bekommen keinen Zugang zu „Mein Portal“ und deshalb keine Schul-E-Mail"

// pruefeEmailZurArt paart die Schul-E-Mail an die Art: Pflicht, wo ein Zugang dazugehört;
// sonst muss sie leer bleiben. Dieselbe Regel gilt beim Nachtragen in der Akte
// (pruefeSchulEmail).
func pruefeEmailZurArt(art, email string) error {
	switch {
	case leserart.HatKonto(art):
		return pruefeKollegiumEmail(email)
	case strings.TrimSpace(email) == "":
		return nil
	case leserart.IstSchueler(art):
		return apierrors.BadRequest(meldungSchuelerOhneZugang, nil)
	}
	return apierrors.BadRequest(meldungKeinZugang, nil)
}

// meldungSchuelerOhneZugang weist eine Schul-E-Mail bei einem Schüler ab, beim Anlegen wie in der
// Akte.
const meldungSchuelerOhneZugang = "Ein Schüler bekommt kein Konto und keine E-Mail-Adresse"

// meldungUnbekannteArt nennt zu einer Art, die es nicht gibt, die möglichen, beim Anlegen wie in
// der Akte.
func meldungUnbekannteArt(art string) string {
	return fmt.Sprintf("Unbekannte Art %q. Möglich sind: %s.", art, leserart.Moegliche())
}

// meldungGeburtsdatumPflicht erklärt dem Sekretariat, warum das Datum nicht fehlen darf —
// eine nackte Validierungsmeldung würde als Schikane gelesen und umgangen.
const meldungGeburtsdatumPflicht = "Geburtsdatum fehlt. Es ist der einzige Schlüssel, über den der LUSD-Import diesen Schüler später wiedererkennt. Ohne Geburtsdatum würde er beim nächsten Import doppelt angelegt."

// LeserAnlegen legt einen Leser an, mit Konto, wo seine Art eines hat.
// @Summary      Leser anlegen
// @Description  Legt einen Leser an; bei Lehrkraft, LiV, Sekretariat und U-plus mit Konto zur Schul-E-Mail.
// @Tags         students
// @Accept       json
// @Produce      json
// @Param        student  body      leserAnlegenAnfrage  true  "Der neue Leser"
// @Success      200      {object}  map[string]any
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /schueler [post]
func (t *Tueren) LeserAnlegen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req leserAnlegenAnfrage
		if !tuer.LeseAnfrage(w, r, &req) {
			return
		}

		req.Vorname = strings.TrimSpace(req.Vorname)
		req.Nachname = strings.TrimSpace(req.Nachname)
		req.Klasse = strings.TrimSpace(req.Klasse)
		req.BarcodeID = strings.TrimSpace(req.BarcodeID)
		req.Art = strings.TrimSpace(req.Art)
		if req.Art == "" {
			req.Art = leserart.Schueler
		}

		if err := pruefeLeserAngaben(&req); err != nil {
			apierrors.AntworteFehler(w, http.StatusBadRequest, err)
			return
		}

		ctx := r.Context()

		// Geburtsdatum ist Pflicht für einen Schüler — nicht als Stammdatum, sondern als
		// Schlüssel: Der LUSD-Export der Schule hat keine Schüler-ID; der Import erkennt
		// einen von Hand angelegten Schüler ausschließlich über Name + Geburtsdatum wieder
		// (Adoption bzw. Namensmodus). Ohne Datum entsteht beim nächsten Import zwangsläufig
		// ein Duplikat.
		//
		// Ein Kollege kommt nie aus der LUSD. Von ihm ein Geburtsdatum zu verlangen, wäre
		// eine Angabe ohne Zweck — und damit eine, die nicht erhoben gehört.
		if leserart.IstSchueler(req.Art) && (req.Geburtsdatum == nil || strings.TrimSpace(*req.Geburtsdatum) == "") {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, meldungGeburtsdatumPflicht, nil)
			return
		}
		geburtsdatum, ok := leseAnlageGeburtsdatum(w, req.Geburtsdatum)
		if !ok {
			return
		}

		// Freischalten ist eine Entscheidung, kein technischer Schritt: Angemeldet wird über
		// IMAP, ein Passwort gibt es bei uns nicht. Wer ein Konto freischalten darf, sagt
		// `manage_users` — und nur wer das Recht hat, legt hier ein aktives Konto an.
		//
		// Das Konto entsteht in beiden Fällen. Es muss auch entstehen: Seine eindeutige
		// E-Mail-Adresse ist das, was den zweiten Eintrag bei der Selbstanmeldung
		// verhindert. Ob es aktiv ist, ändert daran nichts — wer nicht freischalten darf,
		// erzeugt eine Zugangsanfrage, wie sie die Selbstanmeldung auch erzeugt.
		leserID, barcodeID, ok := t.legeLeserAn(ctx, w, req, geburtsdatum, t.HatRecht(r, "manage_users"))
		if !ok {
			return
		}

		tuer.AntworteJSON(w, http.StatusCreated, map[string]any{
			"status":     "success",
			"id":         leserID,
			"barcode_id": barcodeID,
		})
	}
}

// legeLeserAn wickelt die Neuanlage in einer Transaktion ab (Duplikatsprüfung,
// Barcode-Auflösung/-Erzeugung, Einfügen, Commit) und liefert die neue Schüler- und
// Barcode-ID. ok=false: die Fehlerantwort wurde bereits geschrieben.
func (t *Tueren) legeLeserAn(ctx context.Context, w http.ResponseWriter, req leserAnlegenAnfrage, geburtsdatum *time.Time, darfFreischalten bool) (leserID, barcodeID string, ok bool) {
	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	defer db.Verwerfe(ctx, tx)

	// 1. Doppelprüfung, bevor irgendetwas geschrieben wird.
	if !antworteAufLeserDublette(ctx, tx, w, req, geburtsdatum) {
		return "", "", false
	}

	// 2. Die Ausweisnummer: die gewünschte oder eine neue aus dem Generator.
	barcodeID, ok = bestimmeNeueBarcodeID(ctx, tx, w, req.BarcodeID)
	if !ok {
		return "", "", false
	}

	// 3. Die Leserzeile anlegen.
	//
	// Geschrieben wird die Tabelle `leser` und nicht die Sicht `schueler`: Durch die Sicht
	// könnte ein Kollege gar nicht entstehen (WITH CHECK OPTION). Die Schüler-Pflichten
	// hält weiter die Datenbank — chk_leser_schueler_pflichtfelder prüft die Paarung von
	// Art und Klasse/Abgangsjahr/Ausweis, nicht die einzelnen Werte.
	//
	// Klasse und Abgangsjahr sind bei einem Kollegen NULL, nicht '': Ein leerer String wäre
	// eine Klasse namens „nichts", und die Klassenlisten fragen auf NULL.
	var klasse *string
	var abgaengerJahr *int
	if leserart.IstSchueler(req.Art) {
		jahr := repository.AbgaengerJahr(req.Klasse)
		klasse, abgaengerJahr = &req.Klasse, &jahr
	}
	leserID, err = repository.LegeLeserAn(ctx, tx, repository.LeserNeu{
		Barcode:       barcodeID,
		Vorname:       req.Vorname,
		Nachname:      req.Nachname,
		Klasse:        klasse,
		Geburtsdatum:  geburtsdatum,
		AbgaengerJahr: abgaengerJahr,
		Art:           req.Art,
	})
	if err != nil {
		antworteAufAnlegeFehler(w, err, barcodeID)
		return "", "", false
	}

	// 4. Das Konto einer Lehrkraft — in derselben Transaktion. Praktikum und Fachbereich
	// bekommen keines (leserart.HatKonto), ein Schüler ohnehin nicht.
	//
	// `leser_id` wird ausdrücklich mitgegeben, damit der Wächter trg_benutzer_hat_leserzeile
	// nicht anspringt: Er legt zu jedem Konto ohne Leserzeile eine frische an, und das wäre
	// hier die zweite — genau der Doppeleintrag, den diese Änderung abschafft.
	var kontoID string
	if leserart.HatKonto(req.Art) {
		// Das Konto entsteht in derselben Transaktion wie die Leserzeile — scheitert es,
		// darf auch die Zeile nicht stehen bleiben (die belegte Adresse ist der häufige
		// Fall und heißt: Die Person steht schon da).
		params := repository.KollegiumskontoNeu{
			Vorname:  req.Vorname,
			Nachname: req.Nachname,
			Email:    req.Email,
			LeserID:  leserID,
			Aktiv:    darfFreischalten,
		}
		if kontoID, err = repository.LegeKollegiumskontoAn(ctx, tx, params); err != nil {
			antworteAufKontoFehler(w, err, strings.ToLower(strings.TrimSpace(req.Email)))
			return "", "", false
		}
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return "", "", false
	}
	// Dieselbe Spur wie über Benutzer & Rechte: Auch ein Konto, das über die Leserdatei
	// entsteht, hinterlässt einen Eintrag im Protokoll.
	if kontoID != "" {
		t.ProtokolliereKontoAnlage(ctx, kontoID, strings.ToLower(strings.TrimSpace(req.Email)), "kollegium", req.Vorname, req.Nachname)
	}
	return leserID, barcodeID, true
}

// antworteAufLeserDublette lehnt einen Leser ab, den es schon gibt: einen Schüler über Name
// und Geburtsdatum (dieselbe Regel wie der LUSD-Schlüssel), einen Kollegen über den Namen
// allein. Er hat kein Geburtsdatum, und der häufigste Fall ist der Kollege, der sich über
// „Mein Portal" längst selbst angemeldet hat. false heißt: Die Antwort ist geschrieben.
func antworteAufLeserDublette(ctx context.Context, tx pgx.Tx, w http.ResponseWriter, req leserAnlegenAnfrage, geburtsdatum *time.Time) bool {
	if leserart.IstSchueler(req.Art) {
		istDublette, err := repository.IstSchuelerDubletteVorhanden(ctx, tx, req.Vorname, req.Nachname, geburtsdatum)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return false
		}
		if istDublette {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, meldungSchuelerDuplikat, nil)
			return false
		}
		return true
	}
	belegt, err := repository.IstLeserNamensdubletteVorhanden(ctx, tx, req.Vorname, req.Nachname)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if belegt {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, meldungLeserNamensdublette, nil)
		return false
	}
	return true
}

// antworteAufAnlegeFehler ordnet den Fehler beim Anlegen der Leserzeile ein. Die
// Indizes sind die letzte Instanz bei zwei Arbeitsplätzen gleichzeitig; ihre Verletzung ist
// ein Bedienfall mit Erklärung, kein 500.
func antworteAufAnlegeFehler(w http.ResponseWriter, err error, barcodeID string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "unique_schueler_name_gebdatum" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, meldungSchuelerDuplikat, err)
		return
	}
	if repository.IstAusweisKollision(err) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, fmt.Sprintf("Die Ausweisnummer %s trägt bereits eine andere Person", barcodeID), nil)
		return
	}
	if repository.IstNummerBuchOderAusweisKollision(err) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, fmt.Sprintf("Die Nummer %s ist der Barcode eines Buchs und kann kein Ausweis sein", barcodeID), nil)
		return
	}
	apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
}

// leseAnlageGeburtsdatum liest das Geburtsdatum (JJJJ-MM-TT) aus der Anfrage. Der nil- und
// Leer-Zweig ist wegen der Pflicht (meldungGeburtsdatumPflicht) nur Absicherung.
// ok=false bedeutet: die Fehlerantwort wurde bereits geschrieben.
func leseAnlageGeburtsdatum(w http.ResponseWriter, roh *string) (*time.Time, bool) {
	if roh == nil || *roh == "" {
		return nil, true
	}
	t, err := time.Parse(tuer.DatumsformatISO, *roh)
	if err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Das Geburtsdatum muss die Form JJJJ-MM-TT haben", nil)
		return nil, false
	}
	return &t, true
}

// bestimmeNeueBarcodeID liefert die zu verwendende Barcode-ID: entweder die von der Oberfläche
// gewünschte (nach Eindeutigkeitsprüfung) oder eine neu generierte S-Nummer aus der
// zentralen Sequenz. ok=false bedeutet: die Fehlerantwort wurde bereits geschrieben.
func bestimmeNeueBarcodeID(ctx context.Context, tx pgx.Tx, w http.ResponseWriter, gewuenscht string) (string, bool) {
	if gewuenscht == "" {
		// Die Nummer zieht der Generator der Datenbank.
		nummernAbfragen := repository.NewAusweisnummernAbfragen(tx)
		// `leser`, nicht die Sicht `schueler`: Die Sicht zeigt nur Schüler, und die
		// höchste Nummer kann seit Migration 125 an einem Kollegen hängen. Über die Sicht
		// gerechnet gäbe der Generator sie ein zweites Mal aus — und der eindeutige Index
		// quittierte das als 500 statt mit einer Auskunft. Ein Nummernkreis, zwei
		// Generatoren: genau der Fehler aus Migration 068, nur eine Tabelle weiter.
		startNummer, err := nummernAbfragen.ZieheNaechste(ctx)
		if err != nil {
			db.Verwerfe(ctx, tx)
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return "", false
		}
		return repository.AusweisNummer(startNummer), true
	}

	// Die Ausweisnummer gehört genau einer Person — Schüler wie Kollegium. Gefragt wird die
	// Tabelle `leser` und nicht die Sicht `schueler`: Sonst sähe die Prüfung einen Kollegen
	// nicht, ließe die Nummer durch, und der eindeutige Index quittierte es als 500 statt
	// mit einer Auskunft an die Bibliothek (Migration 125).
	//
	// Gelöschte Leser geben ihre Nummer frei — dieselbe Bedingung wie
	// uniq_schueler_barcode_active, sonst wiese die Prüfung eine Nummer ab, die die
	// Datenbank vergeben würde.
	vorhanden, err := repository.IstAusweisnummerVergeben(ctx, tx, gewuenscht)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return "", false
	}
	if vorhanden {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, fmt.Sprintf("Barcode-ID '%s' wird bereits verwendet", gewuenscht), nil)
		return "", false
	}
	return gewuenscht, true
}

package theke

import (
	"bibliothek/internal/apierrors"
	"bibliothek/internal/auth"
	"bibliothek/internal/db"
	"errors"
	"fmt"
	"log/slog"

	"context"
	"net/http"
	"time"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/pkg/lmfplan"
	"bibliothek/internal/pkg/schulzeit"
	"bibliothek/internal/repository"
	"bibliothek/internal/service"

	"github.com/jackc/pgx/v5"
)

// meldungAusleiheNichtGefunden antwortet auf eine Ausleihe, die es nicht gibt oder die schon
// zurückgegeben ist; beides sieht die Theke gleich.
const meldungAusleiheNichtGefunden = "Ausleihe nicht gefunden oder schon zurückgegeben"

// fristAnfrage ist die Frist, die von Hand gesetzt wird.
type fristAnfrage struct {
	FaelligAm string `json:"faellig_am" validate:"required"` // Zeitstempel nach RFC 3339 oder nur der Tag nach ISO 8601
}

// verlaengerungsAnfrage nennt freiwillig die Frist, die die Oberfläche beim Klick zeigte.
type verlaengerungsAnfrage struct {
	FristBisher string `json:"frist_bisher"` // wie faellig_am; leer: verlängert ohne Abgleich
}

// einstellungsAbruf liest die Einstellungen (repository.EinstellungsAbfragen); ohne Fehler ist
// das Ergebnis nie nil.
type einstellungsAbruf interface {
	Lade(ctx context.Context) (*repository.SystemEinstellungen, error)
}

// AusleiheVerlaengern verlängert eine Ausleihe um die Frist für Bücher aus den Einstellungen
// (ohne Einstellung 28 Tage).
func (t *Tueren) AusleiheVerlaengern(einstellungsAbfragen einstellungsAbruf) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.verlaengereAusleihe(w, r, einstellungsAbfragen)
	}
}

// verlaengereAusleihe verlängert eine einzelne Ausleihe. Als eigene Methode ausgelagert,
// damit die frühen Rücksprünge nicht zusätzlich als Verschachtelung in der Closure zählen
// (Sonar S3776).
func (t *Tueren) verlaengereAusleihe(w http.ResponseWriter, r *http.Request, einstellungsAbfragen einstellungsAbruf) {
	ausleiheID := r.PathValue("ausleihe_id")
	if ausleiheID == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung der Ausleihe fehlt", nil)
		return
	}
	gezeigt, ok := leseGezeigteFrist(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	// Sanktions-Konsistenz: Ist das Buch an einen gesperrten Schüler verliehen, darf die
	// Frist nicht verlängert werden — die Sperre soll zur Rückgabe zwingen, nicht durch
	// eine Verlängerung ausgehebelt werden. Welche Sperre zählt, sagt die Regel der Theke
	// (pruefeAusleihSperre): Beim Schulbuch nur die von Hand, beim Kollegen keine.
	gesperrt, grund, geprueft := t.sperreDerAusleihe(ctx, w, ausleiheID, "Einzel-Verlängerung")
	if !geprueft {
		return
	}
	if gesperrt {
		meldung := meldungVerlaengerungGesperrt
		// Der Sperr-Freitext ist Verwaltungsinformation (PII-Matrix Stufe 2) —
		// dieselbe view_students-Grenze wie bei /api/action (ohneSperrgrund).
		if grund != "" && t.HatRecht(r, "view_students") {
			meldung += " (" + grund + ")"
		}
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, meldung, nil)
		return
	}

	tage, sommerferien := verlaengerungsRegel(ctx, einstellungsAbfragen)

	// Gerechnet wird ab dem späteren von altem Fristende und heute: Wer vor Fristablauf
	// verlängert, verliert die verbleibenden Tage nicht, und eine lange überfällige Ausleihe
	// bekommt keine Frist in der Vergangenheit. Die neue Frist ist eine Tagesfrist wie die der
	// Ausleihe (lmfplan.Ferientabelle.Tagesfrist), und die Mahnfolge beginnt neu — sonst
	// überspränge dasselbe Buch beim nächsten Überziehen die erste Mahnstufe. Die Zeile bleibt
	// gesperrt, bis die neue Frist steht, damit zwei Verlängerungen zugleich nacheinander rechnen.
	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("transaktion der einzel-verlängerung beginnen: %w", err))
		return
	}
	defer db.Verwerfe(ctx, tx)

	alteFrist, err := repository.SperreOffeneAusleiheMitFrist(ctx, tx, ausleiheID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungAusleiheNichtGefunden, nil)
		return
	}
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("einzel-verlängerung: %w", err))
		return
	}
	if gezeigt != nil && !gleicherTag(*gezeigt, alteFrist) {
		antworteFristGeaendert(w, alteFrist)
		return
	}
	jetzt := t.Jetzt()
	neueFrist, err := ladeVerlaengerteFrist(ctx, tx, ausleiheID, alteFrist, jetzt, tage, lmfplan.FerientabelleAus(sommerferien))
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("einzel-verlängerung: %w", err))
		return
	}

	_, gesetzteFrist, err := repository.VerlaengereAusleihe(ctx, tx, ausleiheID, neueFrist)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("einzel-verlängerung: %w", err))
		return
	}
	if err := tx.Commit(ctx); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("abschluss der einzel-verlängerung: %w", err))
		return
	}
	// War die Frist schon vorbei, beginnen Mahnfolge und Überfällig-Sperre neu: ein Eingriff in
	// eine Sanktion wie die Frist von Hand, also ins Protokoll. Vor Ablauf der Frist ist es der
	// gewöhnliche Fall der Leserakte und steht nicht da.
	if alteFrist.Before(jetzt) {
		t.ProtokolliereVerwaltung(ctx, tuer.AktionUeberfaelligVerlaengert, map[string]any{
			"ausleihe_id":     ausleiheID,
			"bisherige_frist": alteFrist.Format(time.RFC3339),
			"neue_frist":      gesetzteFrist.Format(time.RFC3339),
		})
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]any{
		"success":               true,
		"neues_rueckgabe_datum": gesetzteFrist,
	})
}

// ladeVerlaengerteFrist rechnet die neue Frist: die Tage ab dem späteren von altem Fristende und
// jetzt, beim Gerät die der Geräte wie bei seiner Ausleihe. Ein Abgänger verlängert außer beim
// Schulbuch nicht über den letzten Schultag vor den Sommerferien hinaus (service.AbgaengerFrist);
// kürzer als bisher wird die Frist dadurch nicht.
func ladeVerlaengerteFrist(ctx context.Context, tx pgx.Tx, ausleiheID string, alteFrist, jetzt time.Time, tage verlaengerungsTage, kalender lmfplan.Ferientabelle) (time.Time, error) {
	klasse, lernmittel, geraet, err := repository.LadeFristRegelDerAusleihe(ctx, tx, ausleiheID)
	if err != nil {
		return time.Time{}, err
	}
	basis := jetzt
	if alteFrist.After(basis) {
		basis = alteFrist
	}
	n := tage.buch
	if geraet {
		n = tage.geraet
	}
	neueFrist := kalender.Tagesfrist(basis, n)
	if lernmittel {
		return neueFrist, nil
	}
	neueFrist = service.AbgaengerFrist(neueFrist, klasse, kalender, jetzt)
	if neueFrist.Before(alteFrist) {
		return alteFrist, nil
	}
	return neueFrist, nil
}

// leseGezeigteFrist liest die Frist, die die Oberfläche beim Klick zeigte; nil ohne Angabe.
// ok=false: Die Fehlerantwort ist geschrieben.
func leseGezeigteFrist(w http.ResponseWriter, r *http.Request) (*time.Time, bool) {
	var req verlaengerungsAnfrage
	if !tuer.LeseFreiwilligeAnfrage(w, r, &req) {
		return nil, false
	}
	if req.FristBisher == "" {
		return nil, true
	}
	// Ohne die Grenze der Frist von Hand: Die gezeigte Frist ist die des Servers und dient nur dem
	// Abgleich, und eine Frist aus den Altdaten darf dahinter liegen.
	gezeigt, err := leseFristText(req.FristBisher)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		return nil, false
	}
	return &gezeigt, true
}

// gleicherTag vergleicht zwei Fristen am Kalendertag der Schule.
func gleicherTag(a, b time.Time) bool {
	zone := schulzeit.Zone()
	return a.In(zone).Format(time.DateOnly) == b.In(zone).Format(time.DateOnly)
}

// antworteFristGeaendert lehnt eine zweite Verlängerung ab: Die Frist am Server ist nicht mehr
// die gezeigte, ein früherer Auftrag hat schon gewirkt, etwa einer, dessen Antwort verloren
// ging. Die Antwort nennt die jetzige Frist, damit die Oberfläche sie zeigt.
func antworteFristGeaendert(w http.ResponseWriter, frist time.Time) {
	tuer.AntworteJSON(w, http.StatusConflict, map[string]any{
		"error":           fmt.Sprintf("Die Frist steht schon auf dem %s.", frist.In(schulzeit.Zone()).Format(tuer.DatumsformatDE)),
		"rueckgabe_frist": frist,
	})
}

// AusleihFristSetzen setzt die Frist einer laufenden Ausleihe von Hand
// (PATCH /api/admin/ausleihen/{id}/faelligkeit).
func (t *Tueren) AusleihFristSetzen(protokollAbfragen *repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.setzeAusleihFrist(w, r, protokollAbfragen)
	}
}

// setzeAusleihFrist setzt die Frist einer laufenden Ausleihe von Hand.
func (t *Tueren) setzeAusleihFrist(w http.ResponseWriter, r *http.Request, protokollAbfragen *repository.ProtokollAbfragen) {
	claims, ok := auth.ClaimsAusKontext(r.Context())
	if !ok {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
		return
	}

	ausleiheID := r.PathValue("id")
	if ausleiheID == "" {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung der Ausleihe fehlt", nil)
		return
	}

	var req fristAnfrage
	if !tuer.LeseAnfrage(w, r, &req) {
		return
	}

	neuesDatum, err := leseFaelligAm(req.FaelligAm)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusBadRequest, err)
		return
	}

	ctx := r.Context()

	// Wie bei den beiden Verlängerungen: Eine Frist in der Zukunft machte die Ausleihe eines
	// gesperrten Lesers wieder „nicht überfällig", setzte die Mahnfolge zurück und hebelte so
	// die Sperre aus, die aus überfälligen Ausleihen berechnet wird. Ein vorgezogenes Datum
	// (Rückruf) bleibt auch bei Sperre erlaubt, auch wenn es noch in der Zukunft liegt: Die Frist
	// rückt nur näher und hebt die Sanktion nicht auf.
	if !t.fristTrotzSperreErlaubt(ctx, w, ausleiheID, neuesDatum) {
		return
	}

	id, gesetzteFrist, err := repository.SetzeAusleihFrist(ctx, t.DB.Pool, ausleiheID, neuesDatum)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungAusleiheNichtGefunden, nil)
			return
		}
		var vorDerAusleihe *repository.FristVorDerAusleihe
		if errors.As(err, &vorDerAusleihe) {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, fmt.Sprintf("Das Rückgabedatum liegt vor dem Tag der Ausleihe, dem %s",
				vorDerAusleihe.Ausgeliehen.In(schulzeit.Zone()).Format(tuer.DatumsformatDE)), err)
			return
		}
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("manuelle frist-überschreibung: %w", err))
		return
	}

	// Eine von Hand gesetzte Frist ist ein Eingriff in eine Sanktion (Mahnfristen, Sperre durch
	// das Programm); sie gehört ins Protokoll wie das Übergehen beim Ausleihen (OVERRIDE_BLOCK).
	// Der Eingriff gilt auch, wenn das Protokoll klemmt; der Fehlversuch steht dann im Log.
	if protokollFehler := protokollAbfragen.ProtokolliereAdminAktion(ctx, claims.KontoID, "FRIST_OVERRIDE", "", map[string]any{
		"ausleihe_id": id,
		"neue_frist":  gesetzteFrist.Format(time.RFC3339),
	}); protokollFehler != nil {
		slog.Error("Audit für Frist-Überschreibung fehlgeschlagen", "fehler", protokollFehler)
	}

	tuer.AntworteJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"faellig_am": gesetzteFrist,
	})
}

// leseFristText liest eine Frist als Zeitstempel (ISO 8601) oder als Datum. Ein Datum gilt bis
// zum Tagesende in der Schulzeitzone, nach derselben Definition wie die berechnete Frist
// (schulzeit.TagesEnde): Roh in UTC wäre die gesetzte Frist ein bis zwei Stunden später fällig
// als eine berechnete zum selben Datum.
func leseFristText(text string) (time.Time, error) {
	if frist, err := time.Parse(time.RFC3339, text); err == nil {
		return frist, nil
	}
	datum, err := time.Parse("2006-01-02", text)
	if err != nil {
		return time.Time{}, apierrors.BadRequest("Das Fälligkeitsdatum muss ein Datum der Form JJJJ-MM-TT sein", err)
	}
	return schulzeit.TagesEnde(datum), nil
}

// leseFaelligAm liest eine Frist von Hand wie leseFristText und weist ein Datum hinter der
// längsten Frist des Programms ab. Die Untergrenze, den Tag der Ausleihe, prüft
// repository.SetzeAusleihFrist.
func leseFaelligAm(text string) (time.Time, error) {
	frist, err := leseFristText(text)
	if err != nil {
		return time.Time{}, err
	}
	if err := tuer.PruefeFristObergrenze(frist, schulzeit.Jetzt()); err != nil {
		return time.Time{}, err
	}
	return frist, nil
}

// verlaengerungsTage sind die Tage einer Verlängerung: die Leihfrist der Bücher und die der Geräte.
type verlaengerungsTage struct {
	buch, geraet int
}

// verlaengerungsRegel liefert die Tage einer Verlängerung und die Sommerferien aus den
// Einstellungen. Sind sie nicht lesbar, gelten die Vorgaben der Leihfristen, mit denen auch die
// Ausleihe dann rechnet, und die Ferientabelle des Programms.
func verlaengerungsRegel(ctx context.Context, einstellungsAbfragen einstellungsAbruf) (tage verlaengerungsTage, sommerferien string) {
	tage = verlaengerungsTage{buch: repository.FristBuchTageVorgabe, geraet: repository.FristGeraeteTageVorgabe}
	einstellungen, err := einstellungsAbfragen.Lade(ctx)
	if err != nil {
		return tage, ""
	}
	if einstellungen.FristBuchTage > 0 {
		tage.buch = einstellungen.FristBuchTage
	}
	if einstellungen.FristGeraeteTage > 0 {
		tage.geraet = einstellungen.FristGeraeteTage
	}
	return tage, einstellungen.Sommerferien
}

// fristTrotzSperreErlaubt sagt, ob die Frist von Hand gesetzt werden darf: Ein Datum in der
// Vergangenheit immer, eines in der Zukunft bei einer Sperre nur, wenn es die Frist näher rückt.
// Bei false ist die Anfrage beantwortet.
func (t *Tueren) fristTrotzSperreErlaubt(ctx context.Context, w http.ResponseWriter, ausleiheID string, neuesDatum time.Time) bool {
	if !neuesDatum.After(time.Now()) {
		return true
	}
	gesperrt, _, geprueft := t.sperreDerAusleihe(ctx, w, ausleiheID, "Frist-Überschreibung")
	if !geprueft {
		return false
	}
	return !gesperrt || t.ziehtFristVor(ctx, w, ausleiheID, neuesDatum)
}

// ziehtFristVor lässt die Frist einer gesperrten Ausleihe nur näher rücken: Ein Datum nach der
// bisherigen Frist lehnt es ab. Bei false ist die Anfrage beantwortet.
func (t *Tueren) ziehtFristVor(ctx context.Context, w http.ResponseWriter, ausleiheID string, neuesDatum time.Time) bool {
	alteFrist, err := repository.LadeFristDerOffenenAusleihe(ctx, t.DB.Pool, ausleiheID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungAusleiheNichtGefunden, nil)
		return false
	}
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("frist der gesperrten ausleihe lesen: %w", err))
		return false
	}
	if neuesDatum.After(alteFrist) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusForbidden, "Die Ausleihe ist gesperrt. Ihre Frist lässt sich nicht verlängern, die Sperre soll zur Rückgabe führen.", nil)
		return false
	}
	return true
}

// sperreDerAusleihe fragt die Regel der Theke (pruefeAusleihSperre) für eine Verlängerung oder
// eine Frist von Hand. Fehlt die Ausleihe oder scheitert die Prüfung, ist die Anfrage hier
// beantwortet und geprueft false.
func (t *Tueren) sperreDerAusleihe(ctx context.Context, w http.ResponseWriter, ausleiheID, vorgang string) (gesperrt bool, grund string, geprueft bool) {
	gesperrt, grund, err := t.pruefeAusleihSperre(ctx, ausleiheID)
	if errors.Is(err, pgx.ErrNoRows) {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungAusleiheNichtGefunden, nil)
		return false, "", false
	}
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, fmt.Errorf("sperr-prüfung (%s): %w", vorgang, err))
		return false, "", false
	}
	return gesperrt, grund, true
}

// pruefeAusleihSperre prüft, ob eine Sperre am Leser die Verlängerung dieser Ausleihe anhält,
// und liefert den Grund. Es entscheidet die Regel der Theke (service.PruefeSperreAmLeser):
// beim Schulbuch nur die Sperre von Hand, beim Kollegen keine; ein anonymisierter Datensatz
// und der Papierkorb bekommen nichts. Eine eigene Abfrage, die jede Sperre zählte, hielte
// auch die Ehemaligen beim Schulbuch an, das die Theke demselben Kind ausgibt.
func (t *Tueren) pruefeAusleihSperre(ctx context.Context, ausleiheID string) (bool, string, error) {
	leserID, lernmittel, err := repository.LadeLeserUndLernmittelDerAusleihe(ctx, t.DB.Pool, ausleiheID)
	if err != nil || leserID == nil {
		return false, "", err
	}
	leser, err := repository.NewLeserAbfragen(t.DB.Pool).Finde(ctx, *leserID)
	if err != nil {
		return false, "", err
	}
	if leser == nil {
		// Im Papierkorb: Die Theke findet ihn nicht und gibt ihm nichts aus.
		return true, "", nil
	}
	sperre := service.PruefeSperreAmLeser(leser, lernmittel)
	if sperre == nil {
		return false, "", nil
	}
	var mitGrund *service.SperrgrundFehler
	if errors.As(sperre, &mitGrund) {
		return true, mitGrund.Grund, nil
	}
	return true, "", nil
}

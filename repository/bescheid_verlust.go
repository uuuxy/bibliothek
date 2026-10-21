package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// UeberfaelligeAusleihe ist ein überfälliges Buch eines Kindes, das noch keine Forderung trägt:
// ein Kandidat für den Brief.
type UeberfaelligeAusleihe struct {
	AusleiheID string
	ExemplarID string
	Titel      string
	ISBN       string
	Kaufpreis  float64
	// Listenpreis und ZustandAbschlag sind dieselben Größen wie in OffeneForderung, damit beide
	// Wege denselben Betrag vorschlagen.
	Listenpreis     float64
	ZustandAbschlag int
	// Topf ist das Eigentum des Exemplars (ExemplarTopfSQL), wie bei OffeneForderung.
	Topf        string
	FaelligSeit time.Time
	// Mahnstufe und LetztesMahndatum sagen, wie oft und wann zuletzt ein Mahnbrief zu diesem
	// Buch gedruckt wurde. Eine aus Littera übernommene Mahnstufe hat kein Datum.
	Mahnstufe        int
	LetztesMahndatum *time.Time
	// SchuljahreMitAusleihe und SchuljahreImBestand sind dieselben beiden Größen wie bei
	// OffeneForderung, für die Staffel.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
}

// BescheidVerlustEingabe ist ein Buch, das mit dem Brief als Verlust gebucht wird.
type BescheidVerlustEingabe struct {
	AusleiheID string
	Betrag     float64
}

// Die Abweisungen sind Bedienfälle (409), keine Serverfehler: Der Dialog stand offen, während
// sich die Lage änderte, und neu öffnen genügt.
var (
	ErrAusleiheInzwischenZurueck = errors.New("ausleihe inzwischen zurückgegeben")
	ErrAusleiheSchonGemeldet     = errors.New("ausleihe hat schon eine forderung")
	ErrAusleiheFremd             = errors.New("ausleihe gehört nicht zu diesem schüler")
)

// LadeUeberfaelligeAusleihen liest die überfälligen Bücher eines Kindes ohne offene Forderung,
// die Kandidaten für den Brief.
// Dieselbe Bedingung wie die Mahnliste (rueckgabe_frist < jetzt); was schon eine Forderung
// trägt, steht bei LadeOffeneForderungen.
func (a *BescheidAbfragen) LadeUeberfaelligeAusleihen(ctx context.Context, schuelerID string) ([]UeberfaelligeAusleihe, error) {
	// Ein Zeitpunkt für beide Rechnungen dieser Funktion: Filterte die Abfrage mit der Uhr der
	// Datenbank, während die Staffel das Schuljahr aus schulzeit.Jetzt() bestimmt, könnte am
	// Schuljahreswechsel die eine schon im neuen Jahr sein und die andere noch im alten. Das
	// Buch stünde in der Liste, und die Staffel nennte eine Stufe daneben.
	jetzt := schulzeit.Jetzt()
	rows, err := a.db.Query(ctx, `
		SELECT a.id, e.id, t.titel, coalesce(t.isbn, ''), coalesce(e.einkaufspreis, 0)::float8,
		       coalesce(t.listenpreis, 0)::float8, coalesce(e.zustand_abwertung_prozent, 0),
		       -- Zugang statt Bestelltag (Begründung an ersatzwert_groessen.go).
		       `+ExemplarTopfSQL+`, a.rueckgabe_frist, a.mahnstufe, a.letztes_mahndatum,
		       COALESCE(e.zugang_am, e.erworben_am)
		FROM ausleihen a
		JOIN buecher_exemplare e ON e.id = a.exemplar_id
		JOIN buecher_titel t ON t.id = e.titel_id
		`+ExemplarTopfJoin+`
		WHERE a.schueler_id = $1
		  AND a.rueckgabe_am IS NULL
		  AND a.rueckgabe_frist < $2
		  -- Dauerleihen bleiben aussen vor: Sie werden nicht überfällig (dieselbe Regel wie
		  -- in der Sperr-Automatik und in der Leserliste), und eine Forderung „wegen
		  -- Überschreitung der Frist" gegen jemanden, der keine hat, wäre unbegründet.
		  -- Ob ein Kollege für ein verlorenes Buch zahlen soll, ist davon unberührt — das
		  -- ist eine Betriebsfrage und keine Nebenwirkung der Frist.
		  AND a.ist_handapparat = false
		  AND NOT EXISTS (SELECT 1 FROM schadensfaelle f WHERE f.ausleihe_id = a.id AND f.storniert_am IS NULL)
		ORDER BY a.rueckgabe_frist, t.titel COLLATE deutsch`, schuelerID, jetzt)
	if err != nil {
		return nil, fmt.Errorf("überfällige ausleihen lesen: %w", err)
	}
	defer rows.Close()

	heute := schuljahrVon(jetzt)
	out := []UeberfaelligeAusleihe{}
	exemplare := []string{}
	for rows.Next() {
		var u UeberfaelligeAusleihe
		var zugang *time.Time
		if err := rows.Scan(&u.AusleiheID, &u.ExemplarID, &u.Titel, &u.ISBN, &u.Kaufpreis,
			&u.Listenpreis, &u.ZustandAbschlag,
			&u.Topf, &u.FaelligSeit, &u.Mahnstufe, &u.LetztesMahndatum, &zugang); err != nil {
			return nil, err
		}
		if zugang != nil {
			u.SchuljahreImBestand = heute - schuljahrVon(*zugang)
		}
		out = append(out, u)
		exemplare = append(exemplare, u.ExemplarID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	schuljahre, err := a.zaehleSchuljahreMitAusleihe(ctx, exemplare)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].SchuljahreMitAusleihe = schuljahre[out[i].ExemplarID]
	}
	return out, nil
}

// bucheVerluste bucht je gewähltem Buch den Verlust in der Transaktion des Briefs (die Ausleihe
// endet, das Exemplar wird als VERLUST ausgesondert, die Forderung „nicht zurückgegeben"
// entsteht) und liefert die Forderungen als Positionen. So gibt es keinen Brief über ein Buch,
// dessen Ausleihe weiterläuft, und keine beendete Ausleihe ohne Brief. Kommt das Buch zurück,
// storniert der Scan an der Theke die Forderung (verbucheRueckkehr).
//
// Geprüft wird in der Transaktion, an der gesperrten Zeile der Ausleihe: Sie gehört diesem
// Kind, läuft noch und trägt keine Forderung. Der Dialog kann minutenlang offen stehen, während
// das Buch an der Theke zurückkommt oder jemand in der Akte den Verlust meldet; ohne die
// Prüfung entstünde ein Brief über ein Buch, das im Regal steht, oder eine zweite Forderung für
// dasselbe Buch.
func bucheVerluste(ctx context.Context, tx pgx.Tx, e BescheidEingabe, referenznummer string) ([]BescheidPositionEingabe, error) {
	beschreibung := "Nicht zurückgegeben, Schadensersatz-Bescheid " + referenznummer
	positionen := make([]BescheidPositionEingabe, 0, len(e.Verluste))
	for _, v := range e.Verluste {
		var exemplarID string
		var schuelerID *string
		var zurueck, gemeldet bool
		err := tx.QueryRow(ctx, `
			SELECT a.exemplar_id, a.schueler_id, a.rueckgabe_am IS NOT NULL,
			       EXISTS (SELECT 1 FROM schadensfaelle f WHERE f.ausleihe_id = a.id AND f.storniert_am IS NULL)
			FROM ausleihen a WHERE a.id = $1 FOR UPDATE`, v.AusleiheID,
		).Scan(&exemplarID, &schuelerID, &zurueck, &gemeldet)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAusleiheFremd
		}
		if err != nil {
			return nil, fmt.Errorf("ausleihe %s lesen: %w", v.AusleiheID, err)
		}
		switch {
		case schuelerID == nil || *schuelerID != e.SchuelerID:
			return nil, ErrAusleiheFremd
		case gemeldet:
			return nil, ErrAusleiheSchonGemeldet
		case zurueck:
			return nil, ErrAusleiheInzwischenZurueck
		}
		schadensfallID, err := meldeSchaden(ctx, tx, schadensMeldung{
			exemplarID:   exemplarID,
			ausleiheID:   v.AusleiheID,
			bearbeiterID: e.ErstelltVon,
			beschreibung: beschreibung,
			art:          SchadensArtNichtZurueck,
			betrag:       v.Betrag,
		})
		if errors.Is(err, ErrExemplarNeuVerliehen) {
			return nil, err // Bedienfall: Die Tür antwortet mit 409, ein Zusatz brächte nichts
		}
		if err != nil {
			return nil, fmt.Errorf("verlust für ausleihe %s buchen: %w", v.AusleiheID, err)
		}
		positionen = append(positionen, BescheidPositionEingabe{SchadensfallID: schadensfallID, Betrag: v.Betrag})
	}
	return positionen, nil
}

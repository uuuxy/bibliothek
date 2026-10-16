package repository

import (
	"context"
	"fmt"
	"time"

	"bibliothek/pkg/schulzeit"
)

// BescheidEmpfaengerDaten sind die Angaben, aus denen der Snapshot des Empfängers entsteht.
type BescheidEmpfaengerDaten struct {
	Vorname     string
	Nachname    string
	Klasse      string
	Strasse     string
	Hausnummer  string
	PLZ         string
	Ort         string
	Volljaehrig bool
}

// OffeneForderung ist eine Forderung, die auf einen Brief kann, mit den Größen, aus denen der
// Vorschlag für den Betrag entsteht.
type OffeneForderung struct {
	SchadensfallID string
	Art            string
	Titel          string
	ISBN           string
	Kaufpreis      float64
	// Listenpreis und ZustandAbschlag: Die Arbeitshilfe rechnet ab dem zweiten Verleihjahr auf
	// den Neupreis, und der Zustand dieses Exemplars zieht davon ab. Ein Listenpreis von 0
	// heißt „nicht erfasst".
	Listenpreis     float64
	ZustandAbschlag int
	// Topf ist das Eigentum des Exemplars (ExemplarTopfSQL): wem die Forderung zusteht und nach
	// welcher Regel ihr Betrag vorgeschlagen wird. Leer bei einer Forderung ohne Exemplar
	// (Geräteschaden); sie gehört auf keinen der beiden Briefe.
	Topf string
	// SchuljahreMitAusleihe und SchuljahreImBestand sind die beiden Größen für das Verleihjahr,
	// beide in Schuljahren: in wie vielen das Exemplar ausgeliehen war und wie viele seit
	// seiner Beschaffung liegen. Beide sind unvollständig (der Altbestand kam ohne
	// Ausleihhistorie); ersatzwert.Verleihjahr rechnet deshalb mit dem Maximum. Mit der Zahl
	// der Ausleihen statt der Schuljahre ergäben sechs Ausleihen in einem Schuljahr das sechste
	// Verleihjahr.
	SchuljahreMitAusleihe int
	SchuljahreImBestand   int
}

// LadeEmpfaenger liest die Angaben des Schülers für Anrede und Anschriftfeld. Volljaehrig
// entscheidet über die Anrede: Bei Minderjährigen geht der Brief an die Erziehungsberechtigten.
// Ohne Geburtsdatum gilt minderjährig; das ist an einer Schule der Regelfall und der
// schonendere Fehler.
func (a *BescheidAbfragen) LadeEmpfaenger(ctx context.Context, schuelerID string) (BescheidEmpfaengerDaten, error) {
	var d BescheidEmpfaengerDaten
	err := a.db.QueryRow(ctx, `
		SELECT vorname, nachname, coalesce(klasse, ''), coalesce(strasse, ''),
		       coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''),
		       -- Volljährigkeit am Kalendertag der Schule, nicht dem der Sitzung (im Image
		       -- UTC): Am 18. Geburtstag galt das Kind bis 2 Uhr Berliner Zeit noch als
		       -- minderjährig, und der Bescheid ging an die Eltern statt an die Person.
		       -- Dieselbe Rechnung wie die Frist (sqlSchulHeute).
		       coalesce(geburtsdatum <= `+sqlSchulHeute+` - INTERVAL '18 years', false)
		FROM schueler WHERE id = $1`, schuelerID).
		Scan(&d.Vorname, &d.Nachname, &d.Klasse, &d.Strasse, &d.Hausnummer, &d.PLZ, &d.Ort, &d.Volljaehrig)
	return d, err
}

// LadeOffeneForderungen liest die Forderungen eines Schülers, die auf keinem Brief stehen.
func (a *BescheidAbfragen) LadeOffeneForderungen(ctx context.Context, schuelerID string) ([]OffeneForderung, error) {
	// Die Abfrage liefert Datumswerte, keine Schuljahre: Wann das Schuljahr wechselt, steht in
	// SchuljahrBeginn und nicht ein zweites Mal in einer Abfrage; eine abweichende Auslegung
	// verschöbe jeden Betrag um eine Stufe der Staffel.
	rows, err := a.db.Query(ctx, `
		SELECT f.id, f.art, coalesce(t.titel, f.beschreibung), coalesce(t.isbn, ''),
		       coalesce(e.einkaufspreis, 0)::float8,
		       coalesce(t.listenpreis, 0)::float8,
		       coalesce(e.zustand_abwertung_prozent, 0),
		       CASE WHEN e.id IS NULL THEN '' ELSE `+ExemplarTopfSQL+` END,
		       -- Zugang statt Bestelltag (Begründung an ersatzwert_groessen.go).
		       f.exemplar_id, COALESCE(e.zugang_am, e.erworben_am)
		FROM schadensfaelle f
		LEFT JOIN buecher_exemplare e ON e.id = f.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		`+ExemplarTopfJoin+`
		WHERE f.schueler_id = $1
		  AND f.bescheid_id IS NULL
		  AND f.ist_bezahlt = false
		  AND f.storniert_am IS NULL
		ORDER BY f.art, coalesce(t.titel, f.beschreibung)`, schuelerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	heute := schuljahrVon(schulzeit.Jetzt())
	out := []OffeneForderung{}
	// exemplarJeForderung läuft im Gleichschritt mit out: Ein Geräteschaden hat kein Exemplar
	// und steht mit leerem Text darin.
	exemplarJeForderung := []string{}
	for rows.Next() {
		var f OffeneForderung
		var exemplarID *string
		var zugang *time.Time
		if err := rows.Scan(&f.SchadensfallID, &f.Art, &f.Titel, &f.ISBN, &f.Kaufpreis,
			&f.Listenpreis, &f.ZustandAbschlag, &f.Topf, &exemplarID, &zugang); err != nil {
			return nil, err
		}
		if zugang != nil {
			f.SchuljahreImBestand = heute - schuljahrVon(*zugang)
		}
		out = append(out, f)
		exemplarJeForderung = append(exemplarJeForderung, zeigerText(exemplarID))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	schuljahre, err := a.zaehleSchuljahreMitAusleihe(ctx, exemplarJeForderung)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].SchuljahreMitAusleihe = schuljahre[exemplarJeForderung[i]]
	}
	return out, nil
}

// zaehleSchuljahreMitAusleihe zählt je Exemplar, in wie vielen Schuljahren es ausgeliehen war.
// Die Zeitpunkte kommen roh aus der Datenbank, gezählt wird in Go mit schuljahrVon, derselben
// Grenze wie überall: Ein count(DISTINCT …) in der Abfrage müsste den 1. August ein zweites Mal
// kennen.
func (a *BescheidAbfragen) zaehleSchuljahreMitAusleihe(ctx context.Context, exemplarIDs []string) (map[string]int, error) {
	je := map[string]int{}
	gefragt := []string{}
	gesehen := map[string]bool{}
	for _, id := range exemplarIDs {
		if id == "" || gesehen[id] {
			continue
		}
		gesehen[id] = true
		gefragt = append(gefragt, id)
	}
	if len(gefragt) == 0 {
		return je, nil
	}

	rows, err := a.db.Query(ctx,
		`SELECT exemplar_id, ausgeliehen_am FROM ausleihen WHERE exemplar_id = ANY($1)`, gefragt)
	if err != nil {
		return nil, fmt.Errorf("ausleihzeitpunkte des exemplars lesen: %w", err)
	}
	defer rows.Close()

	jahreJeExemplar := map[string]map[int]bool{}
	for rows.Next() {
		var exemplarID string
		var ausgeliehenAm time.Time
		if err := rows.Scan(&exemplarID, &ausgeliehenAm); err != nil {
			return nil, err
		}
		if jahreJeExemplar[exemplarID] == nil {
			jahreJeExemplar[exemplarID] = map[int]bool{}
		}
		jahreJeExemplar[exemplarID][schuljahrVon(ausgeliehenAm)] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for id, jahre := range jahreJeExemplar {
		je[id] = len(jahre)
	}
	return je, nil
}

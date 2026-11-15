package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Bestellbedarf ist eine Zeile der Bedarfsliste: ein Schulbuch, dessen Bestand nicht für die
// Kinder reicht, die es im nächsten Schuljahr brauchen.
type Bestellbedarf struct {
	ID     string `json:"id"`
	Titel  string `json:"titel"`
	Autor  string `json:"autor"`
	ISBN   string `json:"isbn"`
	Verlag string `json:"verlag"`
	// Signatur und Erscheinungsjahr helfen, beim Nachbestellen die richtige Ausgabe zu treffen:
	// Bei Lernmitteln unterscheiden sich die Ausgaben oft nur darin.
	Signatur         string `json:"signatur,omitempty"`
	Erscheinungsjahr int    `json:"erscheinungsjahr,omitempty"`
	CoverURL         string `json:"cover_url,omitempty"`
	// GesamtBestand zählt alle nicht ausgesonderten Exemplare, auch die verliehenen und die als
	// bestellt vermerkten: Verliehene kommen zurück, und ein Buch fällt aus der Liste, sobald
	// genug bestellt ist.
	GesamtBestand int `json:"gesamt_bestand"`
	// Bedarf zählt die Kinder, die das Buch im nächsten Schuljahr brauchen (LadeBestellbedarf),
	// Fehlen den Unterschied zum Bestand: die Menge der Bestellung.
	Bedarf int `json:"bedarf"`
	Fehlen int `json:"fehlen"`
	// IstLernmittel ist der Vorschlag für den Topf der Bestellung im Warenkorb.
	IstLernmittel bool `json:"ist_lernmittel"`
	// Auflagen: Gehört der Titel zu einem Buch mit mehreren Auflagen, ist die Zeile das Buch.
	// Die Zahlen oben gelten für das Buch; Titel, ISBN und Kennung sind die der neuesten
	// Auflage, denn die wird bestellt. Hier steht der Bestand je Auflage, die neueste zuerst.
	// Ein Buch mit einer Auflage hat das Feld nicht.
	Auflagen []BestellbedarfAuflage `json:"auflagen,omitempty"`
}

// BestellbedarfAuflage ist eine Auflage in der Aufschlüsselung einer Zeile, mit ihrem Bestand
// und ihrer ISBN: Wer in der Liste nach der alten Auflage sucht (oder sie scannt), soll die
// Zeile des Buchs finden.
type BestellbedarfAuflage struct {
	ID               string `json:"id"`
	Auflage          string `json:"auflage"`
	ISBN             string `json:"isbn"`
	Erscheinungsjahr int    `json:"erscheinungsjahr"`
	GesamtBestand    int    `json:"gesamt_bestand"`
}

// sqlBedarfKinder sind die aktiven Kinder mit Jahrgang und je Jahrgang die Zahl im nächsten
// Schuljahr: die Kinder, die aus dem Jahrgang darunter nachrücken. Wer in einer Abschlussklasse
// ist, rückt nicht nach (AbschlussklasseSQL, wie bei der Versetzung). Hat der Jahrgang darunter
// kein Kind, wie Jahrgang 5, gilt die Zahl von heute.
var sqlBedarfKinder = `
	kinder AS (
		SELECT s.id, ` + JahrgangSQL("s.klasse") + ` AS jahrgang, ` + AbschlussklasseSQL("s.klasse") + ` AS geht_ab
		FROM schueler s
		WHERE s.deleted_at IS NULL AND s.ist_abgaenger = false AND ` + JahrgangSQL("s.klasse") + ` IS NOT NULL
	),
	jahrgaenge AS (
		SELECT jahrgang, count(*) AS kinder, count(*) FILTER (WHERE NOT geht_ab) AS bleiben
		FROM kinder GROUP BY jahrgang
	),
	nachruecken AS (
		SELECT j.jahrgang, j.kinder, coalesce(v.bleiben, j.kinder) AS naechstes
		FROM jahrgaenge j LEFT JOIN jahrgaenge v ON v.jahrgang = j.jahrgang - 1
	)`

// sqlSchulbuchImSchuljahr wählt die Ausleihen (a) von Lernmitteln (t) an Kinder (k), die zum
// Schuljahr [$1, $2) gehören, offen oder schon zurückgegeben. Das Schuljahr einer Ausleihe ist
// das ihrer Frist (wie in buch_in_klasse.go): So zählt der Büchertausch im Juni die alten
// Bücher weiter, und die neuen mit der Frist des nächsten Schuljahres nicht. Ein Band über
// mehrere Jahrgänge zählt auch in den Jahren vor seiner Frist, solange das Kind in seiner
// Spanne ist.
const sqlSchulbuchImSchuljahr = `t.ist_lernmittel AND (
	(a.rueckgabe_frist >= $1 AND a.rueckgabe_frist < $2)
	OR (t.mehrjahresband AND a.rueckgabe_am IS NULL AND a.rueckgabe_frist >= $2
	    AND k.jahrgang BETWEEN coalesce(t.jahrgang_von, 1) AND t.jahrgang_bis))`

// LadeBestellbedarf liefert die Schulbücher, deren Bestand nicht für die Kinder reicht, die sie
// im nächsten Schuljahr brauchen, den größten Fehlbestand zuerst; heute bestimmt das Schuljahr.
//
// Welche Kinder ein Buch brauchen, zeigen die Ausleihen dieses Schuljahres
// (sqlSchulbuchImSchuljahr): Die Bücher bleiben beim Jahrgang, die Kinder rücken weiter. Je
// Jahrgang gilt im nächsten Schuljahr derselbe Anteil wie heute, gerechnet auf die Kinder, die
// nachrücken (sqlBedarfKinder). So zählt ein Buch für einen Zweig oder ein Wahlfach mit seinem
// Anteil, ein Buch für alle mit dem ganzen Jahrgang.
//
// Gezählt wird am Buch: Auflagen mit derselben werk_id bilden eine Zeile, und gezeigt wird die
// neueste Auflage (sqlNeuesteAuflageZuerst), denn die wird bestellt. Ein Titel ohne Werk ist
// über COALESCE(werk_id, id) sein eigenes Buch.
func LadeBestellbedarf(ctx context.Context, db Datenbank, heute time.Time) ([]Bestellbedarf, error) {
	neueste := sqlNeuesteAuflageZuerst("t")
	beginn := SchuljahrBeginn(heute)
	query := fmt.Sprintf(`
		WITH titel AS (
			SELECT t.id, COALESCE(t.werk_id, t.id) AS buch, t.titel, t.autor, t.isbn, t.verlag,
			       t.signatur, t.erscheinungsjahr, t.erstellt_am, t.auflage, t.cover_url,
			       t.ist_lernmittel, t.mehrjahresband, t.jahrgang_von, t.jahrgang_bis
			FROM buecher_titel t
			WHERE t.ist_lernmittel
		),
		%s,
		nutzung AS (
			SELECT t.buch, k.jahrgang, count(DISTINCT k.id) AS kinder
			FROM ausleihen a
			JOIN kinder k ON k.id = a.schueler_id
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN titel t ON t.id = e.titel_id
			WHERE %s
			GROUP BY t.buch, k.jahrgang
		),
		bedarf AS (
			SELECT n.buch, round(sum(n.kinder::numeric * r.naechstes / r.kinder))::int AS kinder
			FROM nutzung n JOIN nachruecken r ON r.jahrgang = n.jahrgang
			GROUP BY n.buch
		),
		bestand AS (
			SELECT t.id, t.buch, count(e.id) FILTER (WHERE e.ist_ausgesondert = false)::int AS gesamt
			FROM titel t
			LEFT JOIN buecher_exemplare e ON e.titel_id = t.id
			WHERE t.buch IN (SELECT buch FROM bedarf)
			GROUP BY t.id, t.buch
		),
		buch AS (
			SELECT b.buch, sum(b.gesamt)::int AS gesamt, count(*) AS anzahl,
			       json_agg(json_build_object('id', t.id, 'auflage', coalesce(t.auflage, ''), 'isbn', coalesce(t.isbn, ''),
			           'erscheinungsjahr', coalesce(t.erscheinungsjahr, 0), 'gesamt_bestand', b.gesamt)
			           ORDER BY %s) AS auflagen
			FROM bestand b JOIN titel t ON t.id = b.id
			GROUP BY b.buch
		),
		neueste AS (
			SELECT DISTINCT ON (t.buch) t.* FROM titel t
			WHERE t.buch IN (SELECT buch FROM bedarf)
			ORDER BY t.buch, %s
		)
		SELECT n.id, n.titel, coalesce(n.autor, ''), coalesce(n.isbn, ''), coalesce(n.verlag, ''),
		       coalesce(n.signatur, ''), coalesce(n.erscheinungsjahr, 0),
		       %s,
		       b.gesamt, d.kinder, n.ist_lernmittel,
		       CASE WHEN b.anzahl > 1 THEN b.auflagen END
		FROM bedarf d
		JOIN buch b ON b.buch = d.buch
		JOIN neueste n ON n.buch = d.buch
		WHERE d.kinder > b.gesamt
		ORDER BY (d.kinder - b.gesamt) DESC, n.titel COLLATE deutsch ASC
	`, sqlBedarfKinder, sqlSchulbuchImSchuljahr, neueste, neueste, sqlCoverOderDNB("n"))

	rows, err := db.Query(ctx, query, beginn, beginn.AddDate(1, 0, 0))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := make([]Bestellbedarf, 0)
	for rows.Next() {
		var t Bestellbedarf
		var auflagen []byte
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.Verlag, &t.Signatur,
			&t.Erscheinungsjahr, &t.CoverURL, &t.GesamtBestand, &t.Bedarf, &t.IstLernmittel,
			&auflagen); err != nil {
			return nil, err
		}
		t.Fehlen = t.Bedarf - t.GesamtBestand
		if len(auflagen) > 0 {
			if err := json.Unmarshal(auflagen, &t.Auflagen); err != nil {
				return nil, fmt.Errorf("auflagen der zeile %s: %w", t.ID, err)
			}
		}
		zeilen = append(zeilen, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return zeilen, nil
}

// LadeBestellbedarfGrundlage zählt die aktiven Kinder mit Jahrgang und die davon, die in diesem
// Schuljahr ein Schulbuch bekommen haben: die Grundlage von LadeBestellbedarf. Ohne sie hieße
// eine leere Liste „es fehlt nichts", auch wenn noch kein Schulbuch ausgegeben ist.
func LadeBestellbedarfGrundlage(ctx context.Context, db Datenbank, heute time.Time) (kinder, mitSchulbuch int, err error) {
	beginn := SchuljahrBeginn(heute)
	err = db.QueryRow(ctx, `
		WITH `+sqlBedarfKinder+`
		SELECT (SELECT count(*) FROM kinder)::int,
		       (SELECT count(DISTINCT k.id)
		        FROM ausleihen a
		        JOIN kinder k ON k.id = a.schueler_id
		        JOIN buecher_exemplare e ON e.id = a.exemplar_id
		        JOIN buecher_titel t ON t.id = e.titel_id
		        WHERE `+sqlSchulbuchImSchuljahr+`)::int`,
		beginn, beginn.AddDate(1, 0, 0)).Scan(&kinder, &mitSchulbuch)
	if err != nil {
		return 0, 0, fmt.Errorf("grundlage des bestellbedarfs: %w", err)
	}
	return kinder, mitSchulbuch, nil
}

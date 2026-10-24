package repository

import (
	"context"
	"encoding/json"
	"fmt"
)

// Bestellbedarf ist eine Zeile der Bedarfsliste: ein Titel, dessen Gesamtbestand unter der
// Schwelle des Bestellbedarfs liegt.
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
	Meldebestand     int    `json:"meldebestand"`
	// VerfuegbarBestand zählt, was im Regal steht (nicht ausgeliehen, nicht ausgesondert). Die
	// Zahl dient der Anzeige; der Bedarf misst sich an GesamtBestand.
	VerfuegbarBestand int `json:"verfuegbarer_bestand"`
	// GesamtBestand zählt alle nicht ausgesonderten Exemplare, auch die als bestellt
	// vermerkten. Daran misst sich der Bedarf: Gemeint ist, wie viele Exemplare die Schule
	// besitzen will, nicht wie viele gerade im Regal stehen. Ein Klassensatz ist das ganze
	// Schuljahr verliehen; „0 verfügbar" bei 30 vorhandenen ist kein Grund zu bestellen.
	GesamtBestand int `json:"gesamt_bestand"`
	// IstLernmittel ist der Vorschlag für den Topf der Bestellung im Warenkorb. Es steht auch
	// bei der Vorgabe ?type=lmf dabei: Bei ?type=alle stünden sonst Titel der Bücherei ohne
	// Kennzeichen im Warenkorb.
	IstLernmittel bool `json:"ist_lernmittel"`
	// Auflagen: Gehört der Titel zu einem Buch mit mehreren Auflagen, ist die Zeile das Buch.
	// Die beiden Bestandszahlen oben sind die Summe; Titel, ISBN und Kennung sind die der
	// neuesten Auflage, denn die wird bestellt. Hier steht die Aufschlüsselung, die neueste
	// zuerst. Ein Buch mit einer Auflage hat das Feld nicht.
	Auflagen []BestellbedarfAuflage `json:"auflagen,omitempty"`
}

// BestellbedarfAuflage ist eine Auflage in der Aufschlüsselung einer Zeile, mit denselben zwei
// Zahlen, deren Summe die Zeile trägt, und mit ihrer ISBN: Wer in der Liste nach der alten
// Auflage sucht (oder sie scannt), soll die Zeile des Buchs finden.
type BestellbedarfAuflage struct {
	ID                string `json:"id"`
	Auflage           string `json:"auflage"`
	ISBN              string `json:"isbn"`
	Erscheinungsjahr  int    `json:"erscheinungsjahr"`
	VerfuegbarBestand int    `json:"verfuegbarer_bestand"`
	GesamtBestand     int    `json:"gesamt_bestand"`
}

// LadeBestellbedarf liefert die Titel, deren Gesamtbestand (nicht ausgesondert) unter der
// eingestellten Schwelle liegt, den größten Fehlbestand zuerst. Gesamt und nicht verfügbar:
// Sonst gälte jeder verliehene Klassensatz, im Schuljahr der Normalfall, als Bedarf. Weil der
// Gesamtbestand die als bestellt vermerkten Exemplare mitzählt, fällt ein Titel aus der Liste,
// sobald genug bestellt ist. t.meldebestand steht nur zur Anzeige dabei.
//
// Gezählt wird am Buch: Auflagen mit derselben werk_id bilden eine Zeile, ihre Summe steht
// gegen die Schwelle, und gezeigt wird die neueste Auflage (sqlNeuesteAuflageZuerst), denn die
// wird bestellt. Ein Titel ohne Werk ist über COALESCE(werk_id, id) sein eigenes Buch. Die
// Bedingung des Topfs (topfBedingung) gilt je Auflage, bevor gezählt wird.
func LadeBestellbedarf(ctx context.Context, db Datenbank, topfBedingung string, schwelle int) ([]Bestellbedarf, error) {
	neueste := sqlNeuesteAuflageZuerst("t")
	// Ein LATERAL je Titel liefert beide Bestandszahlen in einem Durchgang; danach wird je Buch
	// summiert, und DISTINCT ON wählt die neueste Auflage je Buch. Ein zweites LATERAL je Zeile
	// liefe für jede Zeile einmal über alle Titel: gemessen über 6.048 Titel 810 bis 825 ms
	// gegen 66 bis 71 ms in dieser Form.
	query := fmt.Sprintf(`
		WITH titel AS (
			SELECT t.id, COALESCE(t.werk_id, t.id) AS buch, t.titel, t.autor, t.isbn, t.verlag,
			       t.signatur, t.erscheinungsjahr, t.erstellt_am, t.auflage, t.cover_url,
			       t.meldebestand, t.ist_lernmittel, v.verfuegbar, v.gesamt
			FROM buecher_titel t
			JOIN LATERAL (
				SELECT
					COUNT(*) FILTER (
						WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false
						  AND NOT EXISTS (SELECT 1 FROM ausleihen a
						                  WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL)
					)::int AS verfuegbar,
					COUNT(*) FILTER (WHERE e.ist_ausgesondert = false)::int AS gesamt
				FROM buecher_exemplare e
				WHERE e.titel_id = t.id
			) v ON true
			WHERE true %s
		),
		buch AS (
			SELECT t.buch, sum(t.verfuegbar)::int AS verfuegbar, sum(t.gesamt)::int AS gesamt,
			       count(*) AS anzahl,
			       json_agg(json_build_object('id', t.id, 'auflage', coalesce(t.auflage, ''), 'isbn', coalesce(t.isbn, ''),
			           'erscheinungsjahr', coalesce(t.erscheinungsjahr, 0),
			           'verfuegbarer_bestand', t.verfuegbar, 'gesamt_bestand', t.gesamt)
			           ORDER BY %s) AS auflagen
			FROM titel t
			GROUP BY t.buch
			HAVING sum(t.gesamt) < $1
		),
		neueste AS (
			SELECT DISTINCT ON (t.buch) t.* FROM titel t ORDER BY t.buch, %s
		)
		SELECT n.id, n.titel, coalesce(n.autor, ''), coalesce(n.isbn, ''), coalesce(n.verlag, ''),
		       coalesce(n.signatur, ''), coalesce(n.erscheinungsjahr, 0),
		       %s,
		       n.meldebestand, b.verfuegbar, b.gesamt, n.ist_lernmittel,
		       CASE WHEN b.anzahl > 1 THEN b.auflagen END
		FROM buch b
		JOIN neueste n ON n.buch = b.buch
		ORDER BY ($1 - b.gesamt) DESC, n.titel COLLATE deutsch ASC
	`, topfBedingung, neueste, neueste, sqlCoverOderDNB("n"))

	rows, err := db.Query(ctx, query, schwelle)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := make([]Bestellbedarf, 0)
	for rows.Next() {
		var t Bestellbedarf
		var auflagen []byte
		if err := rows.Scan(&t.ID, &t.Titel, &t.Autor, &t.ISBN, &t.Verlag, &t.Signatur,
			&t.Erscheinungsjahr, &t.CoverURL, &t.Meldebestand, &t.VerfuegbarBestand,
			&t.GesamtBestand, &t.IstLernmittel, &auflagen); err != nil {
			return nil, err
		}
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

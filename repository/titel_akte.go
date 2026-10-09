package repository

import (
	"context"
	"time"
)

// TitelExemplar ist ein Exemplar in der Akte seines Titels. ImBestand heißt weder ausgesondert
// noch bestellt (SQLExemplarImBestand), IstVerfuegbar heißt nicht verliehen. Eigentum und
// seine Herkunft folgen der einen Regel (ExemplarTopfSQL, ExemplarTopfHerkunftSQL);
// LitteraEigentumsvermerk ist der Wortlaut aus Littera, wenn es einen gab.
type TitelExemplar struct {
	ID                      string
	BarcodeID               string
	ZustandNotiz            string
	IstAusleihbar           bool
	IstAusgesondert         bool
	ImBestand               bool
	ZustandAbwertungProzent int
	IstVerfuegbar           bool
	Eigentum                string
	EigentumHerkunft        string
	LitteraEigentumsvermerk string
	Standort                string
}

// ListeExemplareDesTitels liefert alle Exemplare eines Titels, die ausgesonderten zuletzt,
// sonst nach Barcode. Eine unlesbare Zeile ist ein Fehler, kein Exemplar, das still aus der
// Akte verschwindet.
func ListeExemplareDesTitels(ctx context.Context, db DBQueryer, titelID string) ([]TitelExemplar, error) {
	query := `
		SELECT e.id, e.barcode_id, coalesce(e.zustand_notiz, ''), e.ist_ausleihbar, e.ist_ausgesondert,
		       (` + SQLExemplarImBestand + `) AS im_bestand,
		       coalesce(e.zustand_abwertung_prozent, 0),
		       NOT EXISTS (SELECT 1 FROM ausleihen a WHERE a.exemplar_id = e.id AND a.rueckgabe_am IS NULL) AS ist_verfuegbar,
		       ` + ExemplarTopfSQL + `, ` + ExemplarTopfHerkunftSQL + `,
		       coalesce(e.erweiterte_eigenschaften->>'littera_eigentumsvermerk', ''),
		       coalesce(e.standort, '')
		FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		` + ExemplarTopfJoin + `
		WHERE e.titel_id = $1
		ORDER BY e.ist_ausgesondert ASC, e.barcode_id
	`
	rows, err := db.Query(ctx, query, titelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	exemplare := []TitelExemplar{}
	for rows.Next() {
		var e TitelExemplar
		if err := rows.Scan(&e.ID, &e.BarcodeID, &e.ZustandNotiz, &e.IstAusleihbar,
			&e.IstAusgesondert, &e.ImBestand, &e.ZustandAbwertungProzent, &e.IstVerfuegbar,
			&e.Eigentum, &e.EigentumHerkunft, &e.LitteraEigentumsvermerk, &e.Standort); err != nil {
			return nil, err
		}
		exemplare = append(exemplare, e)
	}
	return exemplare, rows.Err()
}

// TitelAusleiher ist eine offene Ausleihe eines Titels mit ihrem Ausleiher. Art ist die Art
// des Lesers (leer bei einer getrennten Ausleihe); IstDauerleihe ist die Ausleihe an jemanden,
// der kein Schüler ist.
type TitelAusleiher struct {
	Vorname          string
	Nachname         string
	Klasse           string
	Art              string
	AusleiherBarcode string
	ExemplarBarcode  string
	AusgeliehenAm    time.Time
	RueckgabeFrist   time.Time
	IstDauerleihe    bool
}

// ListeAusleiherDesTitels liefert die offenen Ausleihen eines Titels, die früheste Frist
// zuerst.
//
// LEFT JOIN auf BEIDE Ausleiher-Arten: Eine Ausleihe an eine Lehrkraft trägt keine
// schueler_id, und der frühere INNER JOIN auf schueler ließ sie damit verschwinden
// — der Reiter zeigte weniger Ausleiher, als der Titel hat, und wer das Exemplar
// suchte, suchte im Regal. Die Art reist mit: Beim Kollegen zeigt die Tür statt der Klasse
// das Wort seiner Art (leserart.KlasseOderArt), dieselbe Auskunft wie in der Titel-Historie; der
// Klassenfilter des Reiters liest dieses Feld. COALESCE auf 'Anonym' deckt die getrennte Ausleihe
// ab — laufende trifft die Lesehistorie-Befristung zwar nicht, aber die Antwort soll
// auch dann keinen leeren Namen tragen.
func ListeAusleiherDesTitels(ctx context.Context, db DBQueryer, titelID string) ([]TitelAusleiher, error) {
	query := `
			SELECT
			  COALESCE(l.vorname, 'Anonym') AS vorname,
			  COALESCE(l.nachname, '') AS nachname,
			  COALESCE(l.klasse, '') AS klasse,
			  COALESCE(l.art, '') AS art,
			  COALESCE(l.barcode_id, '') AS ausleiher_barcode,
			  e.barcode_id, a.ausgeliehen_am, a.rueckgabe_frist, a.ist_handapparat
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			LEFT JOIN leser l ON a.schueler_id = l.id
			WHERE e.titel_id = $1 AND a.rueckgabe_am IS NULL
			ORDER BY a.rueckgabe_frist ASC
		`
	rows, err := db.Query(ctx, query, titelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ausleiher := []TitelAusleiher{}
	for rows.Next() {
		var a TitelAusleiher
		if err := rows.Scan(&a.Vorname, &a.Nachname, &a.Klasse, &a.Art, &a.AusleiherBarcode, &a.ExemplarBarcode, &a.AusgeliehenAm, &a.RueckgabeFrist, &a.IstDauerleihe); err != nil {
			return nil, err
		}
		ausleiher = append(ausleiher, a)
	}
	return ausleiher, rows.Err()
}

// TitelAusleihe ist ein Vorgang der Ausleihhistorie eines Titels. Name, Klasse und Art sind
// nil, wenn die Ausleihe von ihrem Leser getrennt ist.
type TitelAusleihe struct {
	Vorname         *string
	Nachname        *string
	Klasse          *string
	Art             *string
	ExemplarBarcode string
	AusgeliehenAm   time.Time
	RueckgabeAm     *time.Time
}

// ListeAusleihhistorieDesTitels liefert die letzten 200 Ausleihen eines Titels, die jüngste
// zuerst.
func ListeAusleihhistorieDesTitels(ctx context.Context, db DBQueryer, titelID string) ([]TitelAusleihe, error) {
	query := `
			SELECT 
			  l.vorname AS vorname,
			  l.nachname AS nachname,
			  l.klasse AS klasse,
			  l.art AS art,
			  e.barcode_id, a.ausgeliehen_am, a.rueckgabe_am
			FROM ausleihen a
			JOIN buecher_exemplare e ON a.exemplar_id = e.id
			LEFT JOIN leser l ON a.schueler_id = l.id
			WHERE e.titel_id = $1
			ORDER BY a.ausgeliehen_am DESC
			LIMIT 200
		`
	rows, err := db.Query(ctx, query, titelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	historie := []TitelAusleihe{}
	for rows.Next() {
		var h TitelAusleihe
		if err := rows.Scan(&h.Vorname, &h.Nachname, &h.Klasse, &h.Art, &h.ExemplarBarcode, &h.AusgeliehenAm, &h.RueckgabeAm); err != nil {
			return nil, err
		}
		historie = append(historie, h)
	}
	return historie, rows.Err()
}

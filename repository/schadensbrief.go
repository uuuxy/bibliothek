package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SchadensfallBrief sind die Angaben eines Schadensfalls für den Elternbrief: Forderung, Leser
// mit Anschrift fürs Fensterkuvert, Buch. Land sagt, ob das Exemplar dem Land gehört
// (ExemplarTopfSQL); daran hängt der Zahlungsweg des Briefs.
type SchadensfallBrief struct {
	Beschreibung     string
	Betrag           float64
	ErstelltAm       time.Time
	SchuelerVorname  string
	SchuelerNachname string
	SchuelerKlasse   string
	Strasse          string
	Hausnummer       string
	PLZ              string
	Ort              string
	BuchTitel        string
	ExemplarBarcode  string
	Land             bool
}

// LadeSchadensfallBrief lädt den Fall für den Elternbrief und sagt dazu, ob die Forderung
// schon auf einem Schadensersatz-Bescheid steht. pgx.ErrNoRows, wenn es den Fall nicht gibt.
func LadeSchadensfallBrief(ctx context.Context, db DBQueryer, id string) (SchadensfallBrief, bool, error) {
	var f SchadensfallBrief
	var aufBescheid bool

	// COALESCE auf den Adressspalten: nullbar in der DB, nicht-nullbar in Go
	// (NULL-Scan-Bugklasse). Anschrift fürs Fensterkuvert.
	query := `
		SELECT
			sf.beschreibung, sf.betrag, sf.erstellt_am,
			s.vorname, s.nachname, s.klasse,
			COALESCE(s.strasse, ''), COALESCE(s.hausnummer, ''),
			COALESCE(s.plz, ''), COALESCE(s.ort, ''),
			t.titel, e.barcode_id,
			-- Der Topf und damit der Zahlungsweg des Briefs (pdf/zahlungsweg.go): das Eigentum
			-- des Exemplars, dieselbe Regel wie am Etikett und im Bescheid.
			(` + ExemplarTopfSQL + ` = 'land'),
			sf.bescheid_id IS NOT NULL
		FROM schadensfaelle sf
		JOIN schueler s ON sf.schueler_id = s.id
		JOIN buecher_exemplare e ON sf.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		` + ExemplarTopfJoin + `
		WHERE sf.id = $1
	`

	err := db.QueryRow(ctx, query, id).Scan(
		&f.Beschreibung, &f.Betrag, &f.ErstelltAm,
		&f.SchuelerVorname, &f.SchuelerNachname, &f.SchuelerKlasse,
		&f.Strasse, &f.Hausnummer, &f.PLZ, &f.Ort,
		&f.BuchTitel, &f.ExemplarBarcode, &f.Land, &aufBescheid,
	)
	if err != nil {
		return SchadensfallBrief{}, false, err
	}
	return f, aufBescheid, nil
}

// MerkeElternbriefErzeugt vermerkt am Schadensfall, dass der Elternbrief erzeugt wurde.
func MerkeElternbriefErzeugt(ctx context.Context, db DBQueryer, id string) error {
	updateQuery := `
		UPDATE schadensfaelle
		SET elternbrief_generiert = true,
		    elternbrief_generiert_am = CURRENT_TIMESTAMP,
		    aktualisiert_am = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := db.Exec(ctx, updateQuery, id)
	return err
}

// RechnungsPosition ist eine offene Forderung auf der Ersatzforderung. Land sagt, ob die
// Position ein Buch des Landes betrifft; ein Geräteschaden ist es nie.
type RechnungsPosition struct {
	Titel        string
	Barcode      string
	Ausleihdatum time.Time
	Ersatzpreis  float64
	Land         bool
}

// ListeOffeneForderungenOhneBescheid lädt die offenen (unbezahlten) Schadensfälle eines
// Schülers als Rechnungspositionen.
//
// LEFT JOIN, nicht INNER (Bestands-Durchgang 06.09.2026, Frage 3): `exemplar_id` und
// `ausleihe_id` sind beide nullbar, `ausleihe_id` steht sogar auf ON DELETE SET NULL, und
// `geraet_id` wartet als dritte Bezugsspalte in derselben Tabelle. Mit INNER JOINs fiele
// eine Forderung ohne diese Bezüge lautlos aus dem Brief — und wären ALLE betroffen,
// antwortete der Weg mit „keine offenen Schadensfälle", während die Akte offene Beträge
// zeigt und der Schüler gesperrt bleibt. Heute ist der Fall nicht erreichbar (die drei
// Löschpfade räumen die Forderungen VOR den Ausleihen), aber das ist eine Zusicherung,
// die nur zufällig hält: Geld darf nicht davon abhängen, ob ein Buch noch existiert.
//
// Die Datenbank verlangt sogar GENAU EINES von beidem (CHECK check_damage_item:
// exemplar_id XOR geraet_id) — der Geräteschaden ist im Schema also vorgesehen, ihm fehlt
// nur der Schreiber. Deshalb steht das Gerät hier schon als Position: Modellname statt
// Titel, Geräte-Barcode statt Exemplar-Barcode. Bleibt beides leer, trägt die Beschreibung
// des Schadensfalls die Zeile; Ausleihdatum ersatzweise das Datum der Forderung.
//
// Nicht hierher gehört eine Forderung, die schon auf einem Schadensersatz-Bescheid steht
// (`bescheid_id`, 14.09.2026): Dieselbe Forderung auf beiden Briefen wäre zweimal dieselbe
// Zahlungsaufforderung, mit zwei Fristen und zwei Nummern. Seit dem 17.09.2026 nennen beide
// Briefe für ein Lernmittel dasselbe Konto (pdf/zahlungsweg.go) — bis dahin verlangte dieser
// hier „bar in der Bibliothek", was die Arbeitshilfe untersagt.
//
// Eine unlesbare Zeile bricht ab: Eine Rechnung, der still eine Position fehlt, nennt einen zu
// kleinen Betrag. Lieber gar kein Brief als ein falscher.
func ListeOffeneForderungenOhneBescheid(ctx context.Context, db DBQueryer, schuelerID uuid.UUID) ([]RechnungsPosition, error) {
	query := `
		SELECT COALESCE(t.titel, g.modellname, sf.beschreibung),
		       COALESCE(e.barcode_id, g.barcode_id, ''),
		       COALESCE(a.ausgeliehen_am, sf.erstellt_am),
		       sf.betrag,
		       -- Der Topf der Position und damit ihr Zahlungsweg: das Eigentum des Exemplars
		       -- (repository.ExemplarTopfSQL), dieselbe Regel wie am Etikett und im Bescheid.
		       -- Ohne Exemplar (Geräteschaden) ist es kein Buch des Landes: Die Bedingung auf
		       -- e.id liefert dann false statt NULL (NULL-Scan-Bugklasse).
		       (e.id IS NOT NULL AND ` + ExemplarTopfSQL + ` = 'land')
		FROM schadensfaelle sf
		LEFT JOIN buecher_exemplare e ON sf.exemplar_id = e.id
		LEFT JOIN buecher_titel t ON e.titel_id = t.id
		` + ExemplarTopfJoin + `
		LEFT JOIN geraete g ON sf.geraet_id = g.id
		LEFT JOIN ausleihen a ON sf.ausleihe_id = a.id
		WHERE sf.schueler_id = $1 AND sf.ist_bezahlt = false
		  AND sf.bescheid_id IS NULL
	`
	rows, err := db.Query(ctx, query, schuelerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RechnungsPosition
	for rows.Next() {
		var item RechnungsPosition
		if err := rows.Scan(&item.Titel, &item.Barcode, &item.Ausleihdatum, &item.Ersatzpreis, &item.Land); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// HatOffeneForderungAufBescheid sagt, ob eine unbezahlte Forderung des Lesers auf einem
// Schadensersatz-Bescheid steht.
func HatOffeneForderungAufBescheid(ctx context.Context, db DBQueryer, schuelerID uuid.UUID) (bool, error) {
	var aufBescheid bool
	err := db.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM schadensfaelle
			               WHERE schueler_id = $1 AND ist_bezahlt = false AND bescheid_id IS NOT NULL)
		`, schuelerID).Scan(&aufBescheid)
	return aufBescheid, err
}

// BriefAnschrift sind Name und Anschrift eines Schülers für einen Brief im Fensterkuvert.
type BriefAnschrift struct {
	Vorname    string
	Nachname   string
	Strasse    string
	Hausnummer string
	PLZ        string
	Ort        string
}

// LadeBriefAnschrift liest Name und Anschrift eines Schülers, der nicht im Papierkorb liegt;
// pgx.ErrNoRows sonst. Leere Anschriftfelder kommen als leerer Text.
func LadeBriefAnschrift(ctx context.Context, db DBQueryer, schuelerID uuid.UUID) (BriefAnschrift, error) {
	var a BriefAnschrift
	err := db.QueryRow(ctx, `
		SELECT vorname, nachname,
		       COALESCE(strasse, ''), COALESCE(hausnummer, ''),
		       COALESCE(plz, ''), COALESCE(ort, '')
		FROM schueler WHERE id = $1 AND deleted_at IS NULL
	`, schuelerID).Scan(&a.Vorname, &a.Nachname, &a.Strasse, &a.Hausnummer, &a.PLZ, &a.Ort)
	return a, err
}

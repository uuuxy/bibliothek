package inventur

import (
	"bibliothek/repository"
	"context"
	"encoding/json"
	"fmt"

	"bibliothek/db"
)

// LegeTitelAn legt einen Titel an. Trägt ein anderer die ISBN, lehnt sie ab (DubletteISBN);
// heißt ohne ISBN ein anderer gleich, nennt sie ihn (DubletteTitel).
func (a *TitelAbfragen) LegeTitelAn(ctx context.Context, eintrag Titel) (string, error) {
	return a.legeTitelZeileAn(ctx, eintrag, false)
}

// LegeTitelAlsAnderesMediumAn legt einen Titel ohne ISBN auch neben einem gleichnamigen an: Die
// Maske hat gefragt, und es ist ein anderes Heft, ein anderer Band oder eine andere Ausgabe.
func (a *TitelAbfragen) LegeTitelAlsAnderesMediumAn(ctx context.Context, eintrag Titel) (string, error) {
	return a.legeTitelZeileAn(ctx, eintrag, true)
}

func (a *TitelAbfragen) legeTitelZeileAn(ctx context.Context, eintrag Titel, anderesMedium bool) (string, error) {
	// subject ist FK auf die Systematik: unbekannte Fächer erst
	// registrieren, die kanonische Schreibweise schreiben, Leerwert wird NULL.
	kanonisch, err := StelleFaecherSicher(ctx, a.pool, []string{eintrag.Fach})
	if err != nil {
		return "", err
	}

	abfrage := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, track, last_counted, medientyp, erweiterte_eigenschaften, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, signatur, ist_lernmittel, auflage, listenpreis, mehrjahresband)
		VALUES (NULLIF($1, ''), $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7::text, '')::date, $8, $9, NULLIF($10, 0), NULLIF($11, 0), $12, $13, $14, NULLIF($15, ''), $16, NULLIF($17, ''), $18, $19)
		RETURNING id`

	medientyp := eintrag.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

	eigenschaften := eintrag.ErweiterteEigenschaften
	if eigenschaften == nil {
		eigenschaften = make(map[string]any)
	}

	// Anlegen des Titels und Abgleich des Bestands in einer Transaktion: Scheitert das Anlegen der
	// Exemplare, darf kein Titel mit 0 Exemplaren zurückbleiben, den die Tür als „erstellt" meldet.
	// Der Fehler des Abgleichs geht deshalb zurück und steht nicht nur im Log.
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht erstellt werden: %w", err)
	}
	defer db.Verwerfe(ctx, tx)

	// Die Dublettenkontrolle läuft vor dem Schreiben in derselben Transaktion.
	if err := pruefeNeuenTitel(ctx, tx, eintrag, anderesMedium); err != nil {
		return "", err
	}

	var id string
	err = tx.QueryRow(
		ctx,
		abfrage,
		eintrag.ISBN,
		eintrag.Titel,
		eintrag.Autor,
		eintrag.CoverURL,
		kanonisch[eintrag.Fach],
		eintrag.Zweig,
		eintrag.ZuletztGezaehlt,
		medientyp,
		eigenschaften,
		eintrag.JahrgangVon,
		eintrag.JahrgangBis,
		eintrag.Untertitel,
		eintrag.Verlag,
		eintrag.Erscheinungsjahr,
		eintrag.Signatur,
		eintrag.IstLernmittel,
		eintrag.Auflage,
		eintrag.Listenpreis,
		eintrag.Mehrjahresband,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht erstellt werden: %w", uebersetzeDatenbankfehler(err))
	}

	// Schlagworte in derselben Transaktion: Scheitern sie, entsteht auch
	// der Titel nicht. nil (Scanner-Neuanlage, Aufrufer ohne das Feld) legt keine an.
	if eintrag.Schlagworte != nil {
		if _, err := repository.SetzeSchlagworte(ctx, tx, id, eintrag.Schlagworte); err != nil {
			return "", fmt.Errorf("schlagworte konnten nicht gespeichert werden: %w", err)
		}
	}

	if eintrag.Bestand > 0 {
		if err := a.gleicheBestandAb(ctx, tx, id, eintrag.Bestand, ""); err != nil {
			return "", fmt.Errorf("exemplare konnten nicht angelegt werden: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("buch konnte nicht erstellt werden: %w", err)
	}
	return id, nil
}

// titelBeiKonflikt ist die Regel des Listenimports für eine ISBN, die schon im Katalog steht,
// eine Fassung für den Batch und den Rückfall auf einzelne Titel: Der Bestand gewinnt, der Import
// füllt nur Lücken. Der Listenimport ist für Sammelkäufe und Spenden gedacht (weitere Exemplare
// eines vorhandenen Titels); die Stammdaten hat dort ein Mensch gepflegt. Setzte die Klausel
// jede Spalte auf EXCLUDED.*, leerte eine Datei `isbn;bestand` Titel, Autor und Verlag, setzte
// Jahr und Jahrgang auf 0 und den Medientyp auf „Buch", würfe ein von Hand hochgeladenes Cover
// weg und meldete „1 Titel importiert".
//
// Zwei Ausnahmen mit Absicht: signatur, ein gesetzter Wert aus der Datei gewinnt (die Liste der
// Signaturen ist der Zweck mancher Importe), ein leerer überschreibt nie; und ist_lernmittel,
// wird nur gesetzt, nie gelöscht. Bei den Eigenschaften werden beide Seiten gemischt, bei
// gleichem Schlüssel gewinnt der Bestand.
const titelBeiKonflikt = `ON CONFLICT (isbn) DO UPDATE SET
			titel = COALESCE(NULLIF(buecher_titel.titel, ''), EXCLUDED.titel),
			autor = COALESCE(NULLIF(buecher_titel.autor, ''), EXCLUDED.autor),
			cover_url = COALESCE(NULLIF(buecher_titel.cover_url, ''), EXCLUDED.cover_url),
			subject = COALESCE(buecher_titel.subject, EXCLUDED.subject),
			track = COALESCE(NULLIF(buecher_titel.track, ''), EXCLUDED.track),
			last_counted = COALESCE(buecher_titel.last_counted, EXCLUDED.last_counted),
			medientyp = COALESCE(NULLIF(buecher_titel.medientyp, ''), EXCLUDED.medientyp),
			-- Eine gepflegte Spanne bleibt, eine fehlende kommt aus der Datei. Beide Spalten
			-- hängen an derselben Bedingung: Die Spanne ist ganz gesetzt oder gar nicht.
			jahrgang_von = CASE WHEN buecher_titel.jahrgang_von IS NULL THEN EXCLUDED.jahrgang_von ELSE buecher_titel.jahrgang_von END,
			jahrgang_bis = CASE WHEN buecher_titel.jahrgang_von IS NULL THEN EXCLUDED.jahrgang_bis ELSE buecher_titel.jahrgang_bis END,
			untertitel = COALESCE(NULLIF(buecher_titel.untertitel, ''), EXCLUDED.untertitel),
			verlag = COALESCE(NULLIF(buecher_titel.verlag, ''), EXCLUDED.verlag),
			erscheinungsjahr = COALESCE(NULLIF(buecher_titel.erscheinungsjahr, 0), EXCLUDED.erscheinungsjahr),
			erweiterte_eigenschaften = COALESCE(EXCLUDED.erweiterte_eigenschaften, '{}'::jsonb)
				|| COALESCE(buecher_titel.erweiterte_eigenschaften, '{}'::jsonb),
			signatur = COALESCE(NULLIF(EXCLUDED.signatur, ''), buecher_titel.signatur),
			auflage = COALESCE(NULLIF(buecher_titel.auflage, ''), EXCLUDED.auflage),
			-- Ein bereits erfasster Listenpreis gewinnt: Die Datei eines Händlers ist
			-- keine Korrektur dessen, was die Bücherei nachgeschlagen hat. Fehlt er,
			-- wird er ergänzt — nie geleert (Upsert-Blanking, Register 10.09.2026).
			listenpreis = COALESCE(buecher_titel.listenpreis, EXCLUDED.listenpreis),
			ist_lernmittel = buecher_titel.ist_lernmittel OR EXCLUDED.ist_lernmittel`

type titelStapel struct {
	isbns             []string
	titel             []string
	autoren           []string
	coverURLs         []string
	faecher           []string
	zweige            []string
	bestaende         []int32
	zuletztGezaehlt   []*string
	medientypen       []string
	jahrgaengeVon     []int
	jahrgaengeBis     []int
	untertitel        []string
	verlage           []string
	erscheinungsjahre []int
	signaturen        []string
	auflagen          []string
	// Zeiger, weil NULL („kein Listenpreis erfasst") und 0 verschiedene Aussagen sind.
	listenpreise            []*float64
	lernmittel              []bool
	erweiterteEigenschaften [][]byte
}

func baueTitelStapel(eintraege []Titel) titelStapel {
	daten := titelStapel{
		isbns:                   make([]string, len(eintraege)),
		titel:                   make([]string, len(eintraege)),
		autoren:                 make([]string, len(eintraege)),
		coverURLs:               make([]string, len(eintraege)),
		faecher:                 make([]string, len(eintraege)),
		zweige:                  make([]string, len(eintraege)),
		bestaende:               make([]int32, len(eintraege)),
		zuletztGezaehlt:         make([]*string, len(eintraege)),
		medientypen:             make([]string, len(eintraege)),
		jahrgaengeVon:           make([]int, len(eintraege)),
		jahrgaengeBis:           make([]int, len(eintraege)),
		untertitel:              make([]string, len(eintraege)),
		verlage:                 make([]string, len(eintraege)),
		erscheinungsjahre:       make([]int, len(eintraege)),
		signaturen:              make([]string, len(eintraege)),
		auflagen:                make([]string, len(eintraege)),
		listenpreise:            make([]*float64, len(eintraege)),
		lernmittel:              make([]bool, len(eintraege)),
		erweiterteEigenschaften: make([][]byte, len(eintraege)),
	}

	for i, b := range eintraege {
		daten.isbns[i] = b.ISBN
		daten.titel[i] = b.Titel
		daten.autoren[i] = b.Autor
		daten.coverURLs[i] = b.CoverURL
		daten.faecher[i] = b.Fach
		daten.zweige[i] = b.Zweig
		// #nosec G115 - leseBestand begrenzt den Bestand beim Import auf [0, MaxInt32]
		daten.bestaende[i] = int32(b.Bestand)
		daten.zuletztGezaehlt[i] = b.ZuletztGezaehlt
		daten.medientypen[i] = b.Medientyp
		if daten.medientypen[i] == "" {
			daten.medientypen[i] = "Buch"
		}
		daten.jahrgaengeVon[i] = b.JahrgangVon
		daten.jahrgaengeBis[i] = b.JahrgangBis
		daten.untertitel[i] = b.Untertitel
		daten.verlage[i] = b.Verlag
		daten.erscheinungsjahre[i] = b.Erscheinungsjahr
		daten.signaturen[i] = b.Signatur
		daten.auflagen[i] = b.Auflage
		daten.listenpreise[i] = b.Listenpreis
		daten.lernmittel[i] = b.IstLernmittel

		// Optimization: Avoid json.Marshal allocation for empty properties
		if len(b.ErweiterteEigenschaften) == 0 {
			daten.erweiterteEigenschaften[i] = []byte("{}")
		} else {
			// In JSON umwandeln für pgx JSONB-Array Kompatibilität
			eigenschaftenJSON, _ := json.Marshal(b.ErweiterteEigenschaften) //nolint:errcheck
			daten.erweiterteEigenschaften[i] = eigenschaftenJSON
		}
	}
	return daten
}

func (a *TitelAbfragen) fuehreStapelAbfrageAus(ctx context.Context, q dbSchreiber, daten titelStapel) (int64, error) {
	// signatur: NULLIF beim Insert + COALESCE beim Konflikt — Import-Läufe
	// dürfen eine physisch verklebte Signatur nie mit Leerwerten überschreiben.
	// ist_lernmittel wird per OR nur gesetzt, nie gelöscht: Eine Excel-Liste ohne
	// diese Spalte darf ein markiertes Schulbuch nicht zum Bibliotheksbuch machen.
	abfrage := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, track, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, erweiterte_eigenschaften, signatur, ist_lernmittel, auflage, listenpreis)
		SELECT t.isbn, t.titel, t.autor, t.cover_url, NULLIF(t.subject, ''), t.track, NULLIF(t.last_counted_text, '')::date, t.medientyp, NULLIF(t.jahrgang_von, 0), NULLIF(t.jahrgang_bis, 0), t.untertitel, t.verlag, t.erscheinungsjahr, t.erweiterte_eigenschaften, NULLIF(t.signatur, ''), t.ist_lernmittel, NULLIF(t.auflage, ''), t.listenpreis
		FROM UNNEST($1::text[], $2::text[], $3::text[], $4::text[], $5::text[], $6::text[], $7::text[], $8::text[], $9::int[], $10::int[], $11::text[], $12::text[], $13::int[], $14::jsonb[], $15::text[], $16::boolean[], $17::text[], $18::numeric[])
		AS t(isbn, titel, autor, cover_url, subject, track, last_counted_text, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, erweiterte_eigenschaften, signatur, ist_lernmittel, auflage, listenpreis)
		` + titelBeiKonflikt + `
	`

	ergebnis, err := q.Exec(
		ctx,
		abfrage,
		daten.isbns,
		daten.titel,
		daten.autoren,
		daten.coverURLs,
		daten.faecher,
		daten.zweige,
		daten.zuletztGezaehlt,
		daten.medientypen,
		daten.jahrgaengeVon,
		daten.jahrgaengeBis,
		daten.untertitel,
		daten.verlage,
		daten.erscheinungsjahre,
		daten.erweiterteEigenschaften,
		daten.signaturen,
		daten.lernmittel,
		daten.auflagen,
		daten.listenpreise,
	)
	if err != nil {
		return 0, fmt.Errorf("bücher konnten nicht im batch importiert werden: %w", err)
	}

	return ergebnis.RowsAffected(), nil
}

// SchreibeTitelStapel schreibt einen Stapel Titel aus dem Listenimport (titelBeiKonflikt) und legt
// ihre Exemplare an.
func (a *TitelAbfragen) SchreibeTitelStapel(ctx context.Context, eintraege []Titel) (int64, error) {
	if len(eintraege) == 0 {
		return 0, nil
	}

	daten := baueTitelStapel(eintraege)
	// subject ist FK auf die Systematik: unbekannte Fächer der
	// Import-Zeilen registrieren und jede auf die kanonische Schreibweise ziehen.
	kanonisch, err := StelleFaecherSicher(ctx, a.pool, daten.faecher)
	if err != nil {
		return 0, err
	}
	for i, s := range daten.faecher {
		daten.faecher[i] = kanonisch[s]
	}

	// Schreiben der Titel und Anlegen der Exemplare in einer Transaktion: legeImportExemplareAn ist
	// additiv und nicht idempotent (jede Zeile legt ihre Stückzahl an). Ohne gemeinsame Transaktion
	// hinterließe ein Fehler danach Titel ohne Exemplare, und ein wiederholter Lauf des Imports legte
	// die Exemplare der schon gelungenen Titel ein zweites Mal an. So rollt ein Fehlschlag den ganzen
	// Batch zurück.
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("import konnte nicht begonnen werden: %w", err)
	}
	defer db.Verwerfe(ctx, tx)

	betroffen, err := a.fuehreStapelAbfrageAus(ctx, tx, daten)
	if err != nil {
		return 0, err
	}
	// Stückzahlen werden zu echten Exemplaren; eine bloß aufsummierte Zahl stünde nirgends mehr
	// sichtbar, und es entstünden Titel ohne Exemplare. Additiv: Jede Zeile des Imports legt ihre
	// Stückzahl an.
	if err := a.legeImportExemplareAn(ctx, tx, daten.isbns, daten.bestaende); err != nil {
		return 0, fmt.Errorf("exemplare zum import konnten nicht angelegt werden: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("import konnte nicht abgeschlossen werden: %w", err)
	}
	return betroffen, nil
}

// legeImportExemplareAn legt je Zeile des Imports so viele Exemplare an, wie sie nennt (dieselbe
// Mechanik wie gleicheBestandAb, aber additiv je Zeile). Die Nummern kommen aus barcode_seq, der
// einen Quelle aller Exemplarnummern.
//
// Gezogen wird die Summe aller Stückzahlen, zugeordnet über die Zeilennummer. Eine ISBN, die
// keinen Titel findet, lässt ihre Nummern ungenutzt: Eine Lücke im Nummernkreis ist harmlos,
// eine doppelt vergebene Nummer nicht (repository/barcode_vergabe.go).
func (a *TitelAbfragen) legeImportExemplareAn(ctx context.Context, q repository.Datenbank, isbns []string, stueck []int32) error {
	gesamt := 0
	for _, n := range stueck {
		if n > 0 {
			gesamt += int(n)
		}
	}
	if gesamt == 0 {
		return nil
	}
	barcodes, err := repository.ZieheFreieExemplarBarcodes(ctx, q, gesamt)
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `
		WITH ziel AS (
			SELECT t.id AS titel_id, row_number() OVER () AS nr
			FROM UNNEST($1::text[], $2::int[]) AS u(isbn, stueck)
			-- Normalform beider Seiten: der neue Titel steht schon normalisiert in der Tabelle,
			-- die Importzeile kommt roh; der Bestand kann noch anders geschrieben sein.
			JOIN buecher_titel t ON `+repository.SQLTitelTraegtISBN("t", "u.isbn")+`
			CROSS JOIN generate_series(1, u.stueck)
			WHERE u.stueck > 0 AND u.isbn <> ''
		)
		INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz, standort)
		SELECT z.titel_id, c.code, true, 'Automatisch generiert (Sammelimport)', `+repository.SQLGeerbterStandort("z.titel_id")+`
		FROM ziel z
		JOIN UNNEST($3::text[]) WITH ORDINALITY AS c(code, nr) ON c.nr = z.nr
	`, isbns, stueck, barcodes)
	return err
}

// SchreibeTitel schreibt einen einzelnen Titel aus dem Listenimport, neu oder nach
// titelBeiKonflikt, und legt seine Exemplare an.
func (a *TitelAbfragen) SchreibeTitel(ctx context.Context, eintrag Titel) (string, error) {
	// subject ist ein Fremdschlüssel auf die Systematik, siehe LegeTitelAn.
	kanonisch, err := StelleFaecherSicher(ctx, a.pool, []string{eintrag.Fach})
	if err != nil {
		return "", err
	}

	abfrage := `
		INSERT INTO buecher_titel (isbn, titel, autor, cover_url, subject, track, last_counted, medientyp, jahrgang_von, jahrgang_bis, untertitel, verlag, erscheinungsjahr, erweiterte_eigenschaften, signatur, ist_lernmittel, auflage, listenpreis)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7::text, '')::date, $8, NULLIF($9, 0), NULLIF($10, 0), $11, $12, $13, $14, NULLIF($15, ''), $16, NULLIF($17, ''), $18)
		` + titelBeiKonflikt + `
		RETURNING id`

	medientyp := eintrag.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

	eigenschaften := eintrag.ErweiterteEigenschaften
	if eigenschaften == nil {
		eigenschaften = make(map[string]any)
	}

	// Schreiben des Titels und (additives) Anlegen der Exemplare in einer Transaktion, wie im Batch:
	// gegen Titel ohne Exemplare und doppelten Bestand bei einem wiederholten Import.
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht importiert werden: %w", err)
	}
	defer db.Verwerfe(ctx, tx)

	var id string
	err = tx.QueryRow(
		ctx,
		abfrage,
		eintrag.ISBN,
		eintrag.Titel,
		eintrag.Autor,
		eintrag.CoverURL,
		kanonisch[eintrag.Fach],
		eintrag.Zweig,
		eintrag.ZuletztGezaehlt,
		medientyp,
		eintrag.JahrgangVon,
		eintrag.JahrgangBis,
		eintrag.Untertitel,
		eintrag.Verlag,
		eintrag.Erscheinungsjahr,
		eigenschaften,
		eintrag.Signatur,
		eintrag.IstLernmittel,
		eintrag.Auflage,
		eintrag.Listenpreis,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("buch konnte nicht importiert werden: %w", err)
	}

	// Additiv wie der Batch-Weg: die Stückzahl der Import-Zeile wird zu Exemplaren.
	if eintrag.Bestand > 0 {
		if err := a.legeImportExemplareAn(ctx, tx, []string{eintrag.ISBN}, []int32{int32(eintrag.Bestand)}); err != nil { // #nosec G115 -- leseBestand begrenzt den Bestand
			return "", fmt.Errorf("exemplare zum import konnten nicht angelegt werden: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("buch konnte nicht importiert werden: %w", err)
	}
	return id, nil
}

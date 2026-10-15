package inventur

import (
	"context"
	"fmt"
	"log"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// buchWerkSpalten: das Buch, zu dem ein Titel als Auflage gehört (Migration 148), und sein
// Rang darin — 1 ist die neueste nach repository.SQLNeuesteAuflageZuerst, gezählt über ALLE
// Auflagen des Buchs, auch die ohne Exemplar. Ohne Werk "" und 0; die Unterabfrage läuft nur
// für Titel mit Werk. Der Medienkatalog fasst damit die Auflagen zu einer Kachel zusammen
// (docs/OFFEN.md 4.18, Stufe 6). Eine Stelle für beide SELECTs, damit scanBuchZeilen passt.
var buchWerkSpalten = `,
		COALESCE(bt.werk_id::text, '') AS werk_id,
		CASE WHEN bt.werk_id IS NULL THEN 0 ELSE (
			SELECT x.rang FROM (
				SELECT w.id, row_number() OVER (ORDER BY ` + repository.SQLNeuesteAuflageZuerst("w") + `) AS rang
				FROM buecher_titel w WHERE w.werk_id = bt.werk_id) x
			WHERE x.id = bt.id) END AS werk_rang`

// buchListenSelect ist der gemeinsame SELECT für Buchlisten und Einzel-Reads:
// Stammdaten (inkl. Signatur) plus Verfügbarkeits-/Bestandszählung über die
// Exemplare. Verwender hängen WHERE an; buchListenGroupBy muss folgen.
var buchListenSelect = `
	SELECT
		bt.id, COALESCE(bt.isbn, '') AS isbn, bt.titel AS title, COALESCE(bt.autor, '') AS author,
		COALESCE(bt.signatur, '') AS signatur,
		COALESCE(bt.cover_url, '') AS cover_url, COALESCE(bt.subject, '') AS subject,
		COALESCE(bt.track, '') AS track, bt.ist_lernmittel,
		COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
		COUNT(e.id) FILTER (WHERE ` + repository.SQLExemplarImBestand + `) AS gesamt,
		` + repository.SQLFilterImZulauf + ` AS im_zulauf,
		TO_CHAR(bt.last_counted, 'YYYY-MM-DD') as last_counted, bt.sort_order, COALESCE(bt.medientyp, 'Buch') AS medientyp,
		COALESCE(bt.jahrgang_von, 0) AS jahrgang_von, COALESCE(bt.jahrgang_bis, 0) AS jahrgang_bis,
		COALESCE(bt.untertitel, '') AS untertitel, COALESCE(bt.verlag, '') AS verlag,
		COALESCE(bt.erscheinungsjahr, 0) AS erscheinungsjahr,
		bt.erweiterte_eigenschaften, COALESCE(bt.auflage, '') AS auflage, bt.listenpreis, bt.mehrjahresband` + buchWerkSpalten + `
	FROM buecher_titel bt
	LEFT JOIN buecher_exemplare e ON e.titel_id = bt.id
	LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
`

const buchListenGroupBy = `
	GROUP BY bt.id, bt.titel, bt.autor, bt.isbn, bt.signatur, bt.cover_url, bt.subject, bt.track, bt.ist_lernmittel, bt.last_counted, bt.sort_order, bt.medientyp, bt.jahrgang_von, bt.jahrgang_bis, bt.untertitel, bt.verlag, bt.erscheinungsjahr, bt.erweiterte_eigenschaften, bt.auflage, bt.listenpreis, bt.mehrjahresband, bt.werk_id
`

// buchListenSelectSchlank ist die Listen-Variante: gleiche Spaltenzahl und -reihenfolge
// (scanBuchZeilen bleibt unverändert), aber die schwerste Spalte, erweiterte_eigenschaften
// (JSONB), kommt als leere Konstante statt vom Server. Die Listen- und Katalogansicht und
// ihre Suche im Browser nutzen sie nicht; das Detail holt sie per Einzel-Read
// (ListBooksByIDs) mit dem vollen buchListenSelect nach.
var buchListenSelectSchlank = `
	SELECT
		bt.id, COALESCE(bt.isbn, '') AS isbn, bt.titel AS title, COALESCE(bt.autor, '') AS author,
		COALESCE(bt.signatur, '') AS signatur,
		COALESCE(bt.cover_url, '') AS cover_url, COALESCE(bt.subject, '') AS subject,
		COALESCE(bt.track, '') AS track, bt.ist_lernmittel,
		COUNT(e.id) FILTER (WHERE e.ist_ausleihbar = true AND e.ist_ausgesondert = false AND a.id IS NULL) AS verfuegbar,
		COUNT(e.id) FILTER (WHERE ` + repository.SQLExemplarImBestand + `) AS gesamt,
		` + repository.SQLFilterImZulauf + ` AS im_zulauf,
		TO_CHAR(bt.last_counted, 'YYYY-MM-DD') as last_counted, bt.sort_order, COALESCE(bt.medientyp, 'Buch') AS medientyp,
		COALESCE(bt.jahrgang_von, 0) AS jahrgang_von, COALESCE(bt.jahrgang_bis, 0) AS jahrgang_bis,
		COALESCE(bt.untertitel, '') AS untertitel, COALESCE(bt.verlag, '') AS verlag,
		COALESCE(bt.erscheinungsjahr, 0) AS erscheinungsjahr,
		'{}'::jsonb AS erweiterte_eigenschaften, COALESCE(bt.auflage, '') AS auflage, bt.listenpreis, bt.mehrjahresband` + buchWerkSpalten + `
	FROM buecher_titel bt
	LEFT JOIN buecher_exemplare e ON e.titel_id = bt.id
	LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
`

// buchListenGroupBySchlank lässt die Konstanten-Spalte aus der Gruppierung weg: Eine
// Konstante muss nicht gruppiert werden, das spart dem Server das Hashen großer Werte.
const buchListenGroupBySchlank = `
	GROUP BY bt.id, bt.titel, bt.autor, bt.isbn, bt.signatur, bt.cover_url, bt.subject, bt.track, bt.ist_lernmittel, bt.last_counted, bt.sort_order, bt.medientyp, bt.jahrgang_von, bt.jahrgang_bis, bt.untertitel, bt.verlag, bt.erscheinungsjahr, bt.auflage, bt.listenpreis, bt.mehrjahresband, bt.werk_id
`

// titelListeGrenze kappt die Katalogliste als reine Runaway-/Speicher-Bremse.
// Bewusst weit über jedem realistischen Katalog (~14k Titel), damit die clientseitige
// Sofort-Suche (lädt den ganzen gefilterten Satz) im Normalbetrieb NIE etwas verliert.
// Wird die Grenze je erreicht, ist das das Signal, auf echte serverseitige Pagination
// umzustellen — und es wird geloggt, statt still Titel zu verschlucken.
const titelListeGrenze = 50000

// leseTitelZeilen liest alle Zeilen eines buchListenSelect-Ergebnisses ein.
func leseTitelZeilen(zeilen pgx.Rows) ([]Titel, error) {
	eintraege := make([]Titel, 0)
	for zeilen.Next() {
		var eintrag Titel
		err := zeilen.Scan(
			&eintrag.ID,
			&eintrag.ISBN,
			&eintrag.Titel,
			&eintrag.Autor,
			&eintrag.Signatur,
			&eintrag.CoverURL,
			&eintrag.Fach,
			&eintrag.Zweig,
			&eintrag.IstLernmittel,
			&eintrag.Verfuegbar,
			&eintrag.Gesamt,
			&eintrag.ImZulauf,
			&eintrag.ZuletztGezaehlt,
			&eintrag.Reihenfolge,
			&eintrag.Medientyp,
			&eintrag.JahrgangVon,
			&eintrag.JahrgangBis,
			&eintrag.Untertitel,
			&eintrag.Verlag,
			&eintrag.Erscheinungsjahr,
			&eintrag.ErweiterteEigenschaften,
			&eintrag.Auflage,
			&eintrag.Listenpreis,
			&eintrag.Mehrjahresband,
			&eintrag.WerkID,
			&eintrag.WerkRang,
		)
		if err != nil {
			return nil, fmt.Errorf("daten konnten nicht gelesen werden: %w", err)
		}
		eintrag.Bestand = eintrag.Gesamt
		eintraege = append(eintraege, eintrag)
	}
	if err := zeilen.Err(); err != nil {
		return nil, fmt.Errorf("fehler beim iterieren: %w", err)
	}
	return eintraege, nil
}

// LadeTitelListe liefert die Titel zu Fach und Suchtext.
//
// Nutzt die schlanke Listen-Variante (ohne erweiterte_eigenschaften) und
// eine Sicherheits-Kappung gegen unbegrenztes Wachstum — siehe buchListenSelectSchlank
// bzw. titelListeGrenze.
//
// nurOhneExemplare wählt die Sicht: false ist der Katalog (nur Titel mit mindestens einem
// nicht ausgesonderten Exemplar, repository.SQLTitelHatExemplar), true die Aufräumsicht
// der Verwaltung mit genau den Titeln, die der Katalog nicht zeigt. Beide Sichten sind
// dasselbe Prädikat mit und ohne NOT — eine dritte Zahl über denselben Titel gibt es nicht.
//
// Der Suchtext trifft seit dem 23.09.2026 auch die Schlagworte und die Verweise darauf
// (repository.SQLTitelUeberSchlagwort), und jeder Titel bringt seine Suchwörter mit: Der
// Reiter „Suche & Filter" lädt die ganze Liste und sucht im Browser — ohne die Wörter fände
// er über ein Schlagwort nichts, während die Titel-Verwaltung daneben am Server sucht und
// es fände.
//
// Die Reihenfolge der Abfrage hält die Kappung und gleiche Titel fest; nach dem Titel ordnet
// der Aufrufer (sortiereBuecherNachTitel), weil die Datenbank keine deutsche Sortierregel hat.
//
// Der Suchtext wird als Wortlaut verglichen und geht deshalb in die Form, in der Titeltexte
// und Schlagworte gespeichert sind (repository.TiteltextNormalform).
func (a *TitelAbfragen) LadeTitelListe(ctx context.Context, fach string, suchanfrage string, nurOhneExemplare bool) ([]Titel, error) {
	suchanfrage = repository.TiteltextNormalform(suchanfrage)
	sicht := repository.SQLTitelHatExemplar("bt")
	if nurOhneExemplare {
		sicht = "NOT " + sicht
	}
	abfrage := buchListenSelectSchlank + `
		WHERE ` + sicht + `
		  AND ($1 = '' OR bt.subject = $1)
		  AND ($2 = '' OR bt.titel ILIKE '%' || $2 || '%' OR bt.autor ILIKE '%' || $2 || '%' OR regexp_replace(coalesce(bt.isbn, ''), '[- ]', '', 'g') ILIKE '%' || regexp_replace($2, '[- ]', '', 'g') || '%' OR bt.subject ILIKE '%' || $2 || '%' OR CAST(bt.id AS TEXT) ILIKE '%' || $2 || '%'
		       OR ` + repository.SQLSuchtextIstISBN("bt", "$2") + `
		       OR ` + repository.SQLTitelUeberSchlagwort("bt", "$2") + `)
	` + buchListenGroupBySchlank + `
		ORDER BY bt.titel ASC, bt.id ASC
		LIMIT $3`

	zeilen, err := a.pool.Query(ctx, abfrage, fach, suchanfrage, titelListeGrenze)
	if err != nil {
		return nil, fmt.Errorf("bücher konnten nicht geladen werden: %w", err)
	}
	defer zeilen.Close()

	eintraege, err := leseTitelZeilen(zeilen)
	if err != nil {
		return nil, err
	}
	if len(eintraege) >= titelListeGrenze {
		log.Printf("WARNUNG: /api/books hat die Sicherheits-Kappung von %d Titeln erreicht — "+
			"die Katalogliste ist unvollständig. Jetzt auf serverseitige Pagination umstellen.",
			titelListeGrenze)
	}

	ids := make([]string, len(eintraege))
	for i := range eintraege {
		ids[i] = eintraege[i].ID
	}
	suchwoerter, err := repository.LadeSuchwoerterDerTitel(ctx, a.pool, ids)
	if err != nil {
		return nil, err
	}
	standorte, err := repository.LadeStandorteDerTitel(ctx, a.pool, ids)
	if err != nil {
		return nil, err
	}
	for i := range eintraege {
		eintraege[i].Suchwoerter = suchwoerter[eintraege[i].ID]
		eintraege[i].Standorte = standorte[eintraege[i].ID]
	}
	return eintraege, nil
}

// LadeTitelMitFremdemCover lists books having external cover URLs.
func (a *TitelAbfragen) LadeTitelMitFremdemCover(ctx context.Context, grenze int) ([]Titel, error) {
	if grenze <= 0 {
		grenze = 100
	}

	abfrage := `
		SELECT id, COALESCE(isbn, '') AS isbn, titel AS title, COALESCE(cover_url, '') AS cover_url
		FROM buecher_titel
		WHERE cover_url LIKE 'http%'
		ORDER BY id ASC
		LIMIT $1`

	zeilen, err := a.pool.Query(ctx, abfrage, grenze)
	if err != nil {
		return nil, fmt.Errorf("bücher mit externen covern konnten nicht geladen werden: %w", err)
	}
	defer zeilen.Close()

	eintraege := make([]Titel, 0)
	for zeilen.Next() {
		var eintrag Titel
		if leseFehler := zeilen.Scan(&eintrag.ID, &eintrag.ISBN, &eintrag.Titel, &eintrag.CoverURL); leseFehler != nil {
			return nil, fmt.Errorf("bücher mit externen covern konnten nicht gelesen werden: %w", leseFehler)
		}
		eintraege = append(eintraege, eintrag)
	}
	if zeilenFehler := zeilen.Err(); zeilenFehler != nil {
		return nil, fmt.Errorf("fehler beim iterieren externer cover-bücher: %w", zeilenFehler)
	}

	return eintraege, nil
}

// LadeSchlagworteDesTitels lädt die Schlagworte eines Titels (Migration 138) für den
// Einzel-Read. Nicht in buchListenSelect: Die Katalogliste bleibt schlank, und ein Feld,
// das dort als nil steht, sagt „nicht geladen" — Speichern lässt es dann unangetastet.
func (a *TitelAbfragen) LadeSchlagworteDesTitels(ctx context.Context, id string) ([]string, error) {
	return repository.LadeSchlagworteDesTitels(ctx, a.pool, id)
}

// LadeTitelNachIDs retrieves list of books for provided IDs.
// Liefert den VOLLEN Datensatz inkl. Signatur und Bestandszählung — der
// Einzel-Read GET /api/books/{id} (Buch-Akte, „Titel bearbeiten") hängt daran.
// Das frühere Minimal-SELECT (id, isbn, titel, cover_url) ließ Signatur und
// Exemplar-Zahlen im Frontend leer erscheinen.
func (a *TitelAbfragen) LadeTitelNachIDs(ctx context.Context, ids []string) ([]Titel, error) {
	if len(ids) == 0 {
		return []Titel{}, nil
	}

	abfrage := buchListenSelect + `
		WHERE bt.id = ANY($1::uuid[])
	` + buchListenGroupBy + `
		ORDER BY bt.id ASC`

	zeilen, err := a.pool.Query(ctx, abfrage, ids)
	if err != nil {
		return nil, fmt.Errorf("bücher nach ids konnten nicht geladen werden: %w", err)
	}
	defer zeilen.Close()

	return leseTitelZeilen(zeilen)
}

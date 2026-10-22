package repository

import (
	"context"
	"errors"
	"slices"
	"testing"

	"bibliothek/internal/testdaten"

	"github.com/jackc/pgx/v5/pgxpool"
)

// zusammenfuehrenProbe ist der Fall vom Testserver: dasselbe Buch zweimal, der große Titel aus
// einer Liste ohne ISBN, der kleine aus dem Bestands-Import mit ISBN, beide als Auflagen
// verbunden.
type zusammenfuehrenProbe struct {
	bearbeiter, gross, klein, werk string
	exemplarGross, exemplarKlein   string
	exemplarBestellt               string
}

func legeZusammenfuehrenProbeAn(t *testing.T, pool *pgxpool.Pool) zusammenfuehrenProbe {
	t.Helper()
	ctx := context.Background()
	testdaten.LeereBestandsdaten(t, pool)
	var p zusammenfuehrenProbe
	p.bearbeiter = testdaten.LegeBearbeiterAn(t, pool)
	if err := pool.QueryRow(ctx, `INSERT INTO werke DEFAULT VALUES RETURNING id::text`).Scan(&p.werk); err != nil {
		t.Fatalf("Werk: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, signatur, ist_lernmittel, jahrgang_von, jahrgang_bis, werk_id, erscheinungsjahr)
		VALUES ('Natur und Technik - Biologie 7/10', 'Ulrike Austenfeld', 'LMF-Bio 7', true, 7, 10, $1, 2018)
		RETURNING id::text`, p.werk).Scan(&p.gross); err != nil {
		t.Fatalf("großer Titel: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO buecher_titel (titel, autor, isbn, verlag, erscheinungsjahr, listenpreis, cover_url, cover_status,
		                           ist_lernmittel, werk_id)
		VALUES ('Natur und Technik - Biologie 7/10', 'Austenfeld, Ulrike', '9783060158423', 'Cornelsen', 2018, 28.99,
		        'https://cover.example/nt.jpg', 'FOUND', true, $1)
		RETURNING id::text`, p.werk).Scan(&p.klein); err != nil {
		t.Fatalf("kleiner Titel: %v", err)
	}
	p.exemplarGross = testdaten.LegeExemplarAn(t, pool, p.gross, "101602", true, "")
	p.exemplarKlein = testdaten.LegeExemplarAn(t, pool, p.klein, "LMF-9783060158423-001", true, "")
	p.exemplarBestellt = testdaten.LegeExemplarImZulaufAn(t, pool, p.klein, "B-10950")
	return p
}

// Alles, was am kleinen Titel hängt, wandert zum großen, der kleine fällt, und der große bekommt
// die ISBN und die übrigen leeren Felder. Was er selbst trägt, bleibt.
func TestFuehreTitelZusammen_AllesWandertUndDieQuelleFaellt(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	p := legeZusammenfuehrenProbeAn(t, pool)

	// Ein Leser hat beide Titel vorgemerkt, am kleinen schon abholbereit: Diese bleibt. Ein
	// zweiter nur den kleinen.
	r1 := testdaten.LegeSchuelerAn(t, pool, "S-ZF-1", "Ada", "07A")
	r2 := testdaten.LegeSchuelerAn(t, pool, "S-ZF-2", "Ben", "07A")
	var vGross, vKlein, vZwei string
	for _, v := range []struct {
		titel, leser, status string
		id                   *string
	}{{p.gross, r1, "wartend", &vGross}, {p.klein, r1, "abholbereit", &vKlein}, {p.klein, r2, "wartend", &vZwei}} {
		if err := pool.QueryRow(ctx, `
			INSERT INTO vormerkungen (titel_id, schueler_id, status, bereitgestellt_bis)
			VALUES ($1, $2, $3::text, CASE WHEN $3::text = 'abholbereit' THEN now() + interval '3 days' END)
			RETURNING id::text`, v.titel, v.leser, v.status).Scan(v.id); err != nil {
			t.Fatalf("Vormerkung: %v", err)
		}
	}
	for _, sql := range []string{
		`INSERT INTO klassensatz_reservierungen (titel_id, klasse, anzahl) VALUES ($1, '07A', 28)`,
		`INSERT INTO lehrer_anliegen (art, titel_text, titel_id) VALUES ('wunsch', 'Natur und Technik', $1)`,
		`INSERT INTO class_books (class_name, book_id) VALUES ('07A', $1)`,
	} {
		if _, err := pool.Exec(ctx, sql, p.klein); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	testdaten.LegeBestellungOhneTopfAn(t, pool, "Lieferant", []string{p.klein})
	var s1, s2 string
	for _, s := range []struct {
		wort string
		id   *string
	}{{"Biologie-ZF", &s1}, {"Ökologie-ZF", &s2}} {
		if err := pool.QueryRow(ctx, `INSERT INTO schlagworte (wort) VALUES ($1)
			ON CONFLICT (lower(wort)) DO UPDATE SET wort = EXCLUDED.wort RETURNING id::text`, s.wort).Scan(s.id); err != nil {
			t.Fatalf("Schlagwort: %v", err)
		}
	}
	for _, ts := range [][2]string{{p.gross, s1}, {p.klein, s1}, {p.klein, s2}} {
		if _, err := pool.Exec(ctx, `INSERT INTO titel_schlagworte (titel_id, schlagwort_id) VALUES ($1, $2)`, ts[0], ts[1]); err != nil {
			t.Fatalf("Schlagwort am Titel: %v", err)
		}
	}
	var zugangVorher string
	if err := pool.QueryRow(ctx, `SELECT zugang_am::text FROM buecher_exemplare WHERE id = $1`, p.exemplarKlein).Scan(&zugangVorher); err != nil {
		t.Fatalf("Zugang lesen: %v", err)
	}

	erg, err := FuehreTitelZusammen(ctx, pool, TitelZusammenfuehrenAuftrag{ZielID: p.gross, QuelleID: p.klein, BearbeiterID: p.bearbeiter})
	if err != nil {
		t.Fatalf("FuehreTitelZusammen: %v", err)
	}

	if erg.Exemplare != 2 || erg.Vormerkungen != 2 || erg.VormerkungenDoppelt != 1 ||
		erg.Klassensaetze != 1 || erg.Bestellpositionen != 1 || erg.Anliegen != 1 {
		t.Errorf("Ergebnis %+v, erwartet 2 Exemplare, 2 Vormerkungen, 1 doppelte, je 1 Klassensatz, Position, Anliegen", *erg)
	}
	var uebernommen []string
	for _, f := range erg.Uebernommen {
		uebernommen = append(uebernommen, f.Bezeichnung+"="+f.Wert)
	}
	if want := []string{"ISBN=9783060158423", "Verlag=Cornelsen", "Listenpreis=28,99 €", "Cover="}; !slices.Equal(uebernommen, want) {
		t.Errorf("übernommen %q, erwartet %q", uebernommen, want)
	}
	if len(erg.Abweichend) != 1 || erg.Abweichend[0] != (TitelAbweichung{"Autor", "Ulrike Austenfeld", "Austenfeld, Ulrike"}) {
		t.Errorf("abweichend %+v, erwartet nur den Autor", erg.Abweichend)
	}

	var quelleDa bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM buecher_titel WHERE id = $1)`, p.klein).Scan(&quelleDa); err != nil || quelleDa {
		t.Fatalf("der kleine Titel steht noch da (%v)", err)
	}
	var isbn, verlag, autor, signatur, cover string
	var jahrVon, jahrBis int
	var werk *string
	if err := pool.QueryRow(ctx, `
		SELECT isbn, verlag, autor, signatur, cover_url, jahrgang_von, jahrgang_bis, werk_id::text
		FROM buecher_titel WHERE id = $1`, p.gross).Scan(&isbn, &verlag, &autor, &signatur, &cover, &jahrVon, &jahrBis, &werk); err != nil {
		t.Fatalf("großen Titel lesen: %v", err)
	}
	if isbn != "9783060158423" || verlag != "Cornelsen" || cover != "https://cover.example/nt.jpg" {
		t.Errorf("aufgefüllt: ISBN %q, Verlag %q, Cover %q", isbn, verlag, cover)
	}
	if autor != "Ulrike Austenfeld" || signatur != "LMF-Bio 7" || jahrVon != 7 || jahrBis != 10 {
		t.Errorf("eigene Felder geändert: Autor %q, Signatur %q, Jahrgang %d–%d", autor, signatur, jahrVon, jahrBis)
	}
	// Ein Werk mit nur noch einem Titel fällt, wie beim Lösen einer Auflage.
	if werk != nil {
		t.Errorf("der große Titel hängt noch am Werk %s", *werk)
	}

	for _, ex := range []string{p.exemplarGross, p.exemplarKlein, p.exemplarBestellt} {
		var titel string
		if err := pool.QueryRow(ctx, `SELECT titel_id::text FROM buecher_exemplare WHERE id = $1`, ex).Scan(&titel); err != nil || titel != p.gross {
			t.Errorf("Exemplar %s hängt an %q (%v)", ex, titel, err)
		}
	}
	// Das Umhängen ist kein Zugang: Das Zugangsdatum bleibt, auch im Zulauf leer.
	var zugangNachher, zugangBestellt *string
	if err := pool.QueryRow(ctx, `SELECT (SELECT zugang_am::text FROM buecher_exemplare WHERE id = $1),
		(SELECT zugang_am::text FROM buecher_exemplare WHERE id = $2)`, p.exemplarKlein, p.exemplarBestellt).Scan(&zugangNachher, &zugangBestellt); err != nil {
		t.Fatalf("Zugang lesen: %v", err)
	}
	if zugangNachher == nil || *zugangNachher != zugangVorher || zugangBestellt != nil {
		t.Errorf("Zugang nach dem Umhängen %v (vorher %s), im Zulauf %v", zugangNachher, zugangVorher, zugangBestellt)
	}

	var vormerkungen []string
	rows, err := pool.Query(ctx, `SELECT id::text FROM vormerkungen WHERE titel_id = $1 ORDER BY id`, p.gross)
	if err != nil {
		t.Fatalf("Vormerkungen lesen: %v", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("Vormerkung lesen: %v", err)
		}
		vormerkungen = append(vormerkungen, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Vormerkungen lesen: %v", err)
	}
	rows.Close()
	want := []string{vKlein, vZwei}
	slices.Sort(want)
	if !slices.Equal(vormerkungen, want) {
		t.Errorf("Vormerkungen am großen Titel %q, erwartet die abholbereite und die des zweiten Lesers %q (die wartende %s fällt)", vormerkungen, want, vGross)
	}

	zaehlen := map[string]string{
		"Klassensatz":            `SELECT count(*) FROM klassensatz_reservierungen WHERE titel_id = $1`,
		"Anliegen":               `SELECT count(*) FROM lehrer_anliegen WHERE titel_id = $1`,
		"Position":               `SELECT count(*) FROM bestellungen_positionen WHERE titel_id = $1`,
		"Klasse":                 `SELECT count(*) FROM class_books WHERE book_id = $1`,
		"Schlagworte":            `SELECT count(*) FROM titel_schlagworte WHERE titel_id = $1`,
		"Protokoll":              `SELECT count(*) FROM audit_log WHERE datensatz_id = $1 AND details->>'action' = 'titel_zusammengefuehrt'`,
		"Exemplare im Protokoll": `SELECT jsonb_array_length(details->'gewandert'->'Exemplare') FROM audit_log WHERE datensatz_id = $1 AND details->>'action' = 'titel_zusammengefuehrt'`,
	}
	for was, want := range map[string]int{"Klassensatz": 1, "Anliegen": 1, "Position": 1, "Klasse": 1, "Schlagworte": 2, "Protokoll": 1, "Exemplare im Protokoll": 2} {
		var n int
		if err := pool.QueryRow(ctx, zaehlen[was], p.gross).Scan(&n); err != nil || n != want {
			t.Errorf("%s am großen Titel: %d (%v), erwartet %d", was, n, err, want)
		}
	}
}

// Tragen beide Titel eine ISBN, sind es zwei Ausgaben: Das Zusammenführen lehnt ab und ändert
// nichts.
func TestFuehreTitelZusammen_ZweiISBNSindZweiAusgaben(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	p := legeZusammenfuehrenProbeAn(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET isbn = '9783060130764' WHERE id = $1`, p.gross); err != nil {
		t.Fatalf("ISBN setzen: %v", err)
	}

	_, err := FuehreTitelZusammen(ctx, pool, TitelZusammenfuehrenAuftrag{ZielID: p.gross, QuelleID: p.klein, BearbeiterID: p.bearbeiter})
	if !errors.Is(err, ErrTitelZusammenfuehrenZweiISBN) {
		t.Fatalf("Fehler %v, erwartet ErrTitelZusammenfuehrenZweiISBN", err)
	}
	var titel string
	if err := pool.QueryRow(ctx, `SELECT titel_id::text FROM buecher_exemplare WHERE id = $1`, p.exemplarKlein).Scan(&titel); err != nil || titel != p.klein {
		t.Errorf("das Exemplar des kleinen Titels hängt an %q (%v)", titel, err)
	}
}

// Die Vorschau sagt, was geschähe, und lässt alles, wie es ist.
func TestFuehreTitelZusammen_VorschauAendertNichts(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	p := legeZusammenfuehrenProbeAn(t, pool)

	erg, err := FuehreTitelZusammen(ctx, pool, TitelZusammenfuehrenAuftrag{
		ZielID: p.gross, QuelleID: p.klein, BearbeiterID: p.bearbeiter, NurVorschau: true,
	})
	if err != nil {
		t.Fatalf("Vorschau: %v", err)
	}
	if erg.Exemplare != 2 || erg.QuelleISBN != "9783060158423" || len(erg.Uebernommen) == 0 {
		t.Errorf("Vorschau %+v, erwartet 2 Exemplare, die ISBN und übernommene Felder", *erg)
	}
	var titel, isbn *string
	var protokoll int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT titel_id::text FROM buecher_exemplare WHERE id = $1),
		(SELECT isbn FROM buecher_titel WHERE id = $2),
		(SELECT count(*) FROM audit_log WHERE details->>'action' = 'titel_zusammengefuehrt')::int`,
		p.exemplarKlein, p.gross).Scan(&titel, &isbn, &protokoll); err != nil {
		t.Fatalf("Stand lesen: %v", err)
	}
	if wert(titel) != p.klein || isbn != nil || protokoll != 0 {
		t.Errorf("Vorschau hat geschrieben: Exemplar an %q, ISBN am großen %q, %d Protokolleinträge", wert(titel), wert(isbn), protokoll)
	}
}

// War der kleine Titel Auflage eines Buchs mit einer weiteren Ausgabe, tritt der große an seine
// Stelle; hatte der große ein eigenes Buch, werden beide eins.
func TestFuehreTitelZusammen_ZielTrittInsWerkDerQuelle(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	p := legeZusammenfuehrenProbeAn(t, pool)
	// Der große Titel gehört zu einem eigenen Buch mit einer alten Ausgabe; am Buch des kleinen
	// hängt eine neue.
	var werkGross, alt, neu string
	if err := pool.QueryRow(ctx, `INSERT INTO werke DEFAULT VALUES RETURNING id::text`).Scan(&werkGross); err != nil {
		t.Fatalf("Werk: %v", err)
	}
	for _, x := range []struct {
		titel, werk string
		id          *string
	}{{"Natur und Technik 2010", werkGross, &alt}, {"Natur und Technik 2024", p.werk, &neu}} {
		if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, ist_lernmittel, werk_id) VALUES ($1, true, $2) RETURNING id::text`,
			x.titel, x.werk).Scan(x.id); err != nil {
			t.Fatalf("Ausgabe: %v", err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET werk_id = $1 WHERE id = $2`, werkGross, p.gross); err != nil {
		t.Fatalf("großen Titel umhängen: %v", err)
	}

	if _, err := FuehreTitelZusammen(ctx, pool, TitelZusammenfuehrenAuftrag{ZielID: p.gross, QuelleID: p.klein, BearbeiterID: p.bearbeiter}); err != nil {
		t.Fatalf("FuehreTitelZusammen: %v", err)
	}
	var werke int
	if err := pool.QueryRow(ctx, `SELECT count(DISTINCT werk_id) FROM buecher_titel WHERE id IN ($1, $2, $3)`, p.gross, alt, neu).Scan(&werke); err != nil {
		t.Fatalf("Werke zählen: %v", err)
	}
	var ohneWerk int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM buecher_titel WHERE id IN ($1, $2, $3) AND werk_id IS NULL`, p.gross, alt, neu).Scan(&ohneWerk); err != nil {
		t.Fatalf("Werke zählen: %v", err)
	}
	if werke != 1 || ohneWerk != 0 {
		t.Errorf("%d Werke, %d Titel ohne Werk; erwartet ein Buch mit drei Ausgaben", werke, ohneWerk)
	}
}

// Hat nur die Quelle ein Buch mit einer weiteren Ausgabe, tritt das Ziel an ihre Stelle: Das
// Buch bleibt mit zwei Ausgaben stehen.
func TestFuehreTitelZusammen_ZielOhneWerkUebernimmtDasDerQuelle(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	p := legeZusammenfuehrenProbeAn(t, pool)
	var neu string
	if err := pool.QueryRow(ctx, `INSERT INTO buecher_titel (titel, ist_lernmittel, werk_id) VALUES ('Natur und Technik 2024', true, $1)
		RETURNING id::text`, p.werk).Scan(&neu); err != nil {
		t.Fatalf("Ausgabe: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET werk_id = NULL WHERE id = $1`, p.gross); err != nil {
		t.Fatalf("großen Titel lösen: %v", err)
	}

	if _, err := FuehreTitelZusammen(ctx, pool, TitelZusammenfuehrenAuftrag{ZielID: p.gross, QuelleID: p.klein, BearbeiterID: p.bearbeiter}); err != nil {
		t.Fatalf("FuehreTitelZusammen: %v", err)
	}
	var werkGross, werkNeu *string
	if err := pool.QueryRow(ctx, `SELECT (SELECT werk_id::text FROM buecher_titel WHERE id = $1),
		(SELECT werk_id::text FROM buecher_titel WHERE id = $2)`, p.gross, neu).Scan(&werkGross, &werkNeu); err != nil {
		t.Fatalf("Werke lesen: %v", err)
	}
	if wert(werkGross) != p.werk || wert(werkNeu) != p.werk {
		t.Errorf("Werk des großen Titels %q, der neuen Ausgabe %q; erwartet beide %s", wert(werkGross), wert(werkNeu), p.werk)
	}
}

// Derselbe Titel zweimal und ein unbekannter Titel sind kein Zusammenführen.
func TestFuehreTitelZusammen_GleichOderUnbekannt(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	p := legeZusammenfuehrenProbeAn(t, pool)

	if _, err := FuehreTitelZusammen(ctx, pool, TitelZusammenfuehrenAuftrag{ZielID: p.gross, QuelleID: p.gross, BearbeiterID: p.bearbeiter}); !errors.Is(err, ErrTitelZusammenfuehrenGleich) {
		t.Errorf("derselbe Titel: Fehler %v, erwartet ErrTitelZusammenfuehrenGleich", err)
	}
	if _, err := FuehreTitelZusammen(ctx, pool, TitelZusammenfuehrenAuftrag{ZielID: p.gross, QuelleID: "00000000-0000-0000-0000-000000000001", BearbeiterID: p.bearbeiter}); !errors.Is(err, ErrTitelNichtGefunden) {
		t.Errorf("unbekannter Titel: Fehler %v, erwartet ErrTitelNichtGefunden", err)
	}
}

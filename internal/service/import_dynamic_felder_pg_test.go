package service

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"

	"github.com/jackc/pgx/v5/pgxpool"
)

// importTitelZeile ist, was der Bestands-Import an einem Titel schreibt.
type importTitelZeile struct {
	Autor, Verlag, ISBN, Fach, Signatur string
	Jahr, Von, Bis                      int
	Lernmittel                          bool
}

func liesImportTitel(t *testing.T, pool *pgxpool.Pool, titel string) importTitelZeile {
	t.Helper()
	var z importTitelZeile
	if err := pool.QueryRow(context.Background(), `
		SELECT coalesce(autor, ''), coalesce(verlag, ''), coalesce(isbn, ''), coalesce(subject, ''),
		       coalesce(signatur, ''), coalesce(erscheinungsjahr, 0), coalesce(jahrgang_von, 0),
		       coalesce(jahrgang_bis, 0), ist_lernmittel
		FROM buecher_titel WHERE titel = $1`, titel).
		Scan(&z.Autor, &z.Verlag, &z.ISBN, &z.Fach, &z.Signatur, &z.Jahr, &z.Von, &z.Bis, &z.Lernmittel); err != nil {
		t.Fatalf("Titel %q lesen: %v", titel, err)
	}
	return z
}

// raeumeImportProbe entfernt die Exemplare und Titel einer Probe, vor dem Test und danach.
func raeumeImportProbe(t *testing.T, pool *pgxpool.Pool, barcodes, titel []string) {
	t.Helper()
	raeume := func() {
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_exemplare WHERE barcode_id = ANY($1)`, barcodes); err != nil {
			t.Errorf("aufräumen: Probe-Exemplare löschen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = ANY($1)`, titel); err != nil {
			t.Errorf("aufräumen: Probe-Titel löschen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)
}

// Der Bestands-Import schreibt jede Spalte der Datei an ihre Stelle, am Titel wie am Exemplar.
// Jede Spalte trägt einen eigenen Wert: Mit gleichen fiele nicht auf, wenn zwei vertauscht
// ankämen. Die Datei nennt drei neue Titel, damit auffällt, wenn ein Exemplar am falschen hängt.
func TestImportDynamic_JedeSpalteKommtAnIhrerStelleAn(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	barcodes := []string{"FELDER-PROBE-562-1", "FELDER-PROBE-562-2", "FELDER-PROBE-562-3", "FELDER-PROBE-562-4"}
	const lernbuch, sachbuch, rechenbuch = "Felder-Probe Biologie heute", "Felder-Probe Zahlenbuch", "Felder-Probe Rechenbuch"
	const isbn = "9780000562012"
	raeumeImportProbe(t, pool, barcodes, []string{lernbuch, sachbuch, rechenbuch})

	kopf := map[string]int{"titel": 0, "autor": 1, "verlag": 2, "isbn": 3, "jahr": 4, "kategorie": 5,
		"signatur": 6, "barcode": 7, "zustand": 8}
	datei := [][]string{
		{"Titel", "Autor", "Verlag", "ISBN", "Jahr", "Kategorie", "Signatur", "Barcode", "Zustand"},
		{lernbuch, "Autorin des Lernbuchs", "Verlag des Lernbuchs", isbn, "2019", "", "LMF Bio 7-9", barcodes[0], "verliehen"},
		{lernbuch, "", "", isbn, "", "", "", barcodes[1], "Ecke geknickt"},
		{sachbuch, "Autor des Sachbuchs", "Verlag des Sachbuchs", "", "2001", "Mathematik", "Ma 12", barcodes[2], ""},
		{rechenbuch, "", "", "", "", "Buch LMF Ma 6/Gri", "Ma 6 Rech", barcodes[3], ""},
	}
	neueTitel, neueExemplare, err := NewImportService(nil, pool).ImportDynamic(ctx, datei, kopf)
	if err != nil {
		t.Fatalf("ImportDynamic: %v", err)
	}
	if neueTitel != 3 || neueExemplare != 4 {
		t.Errorf("%d neue Titel und %d neue Exemplare gemeldet, erwartet 3 und 4", neueTitel, neueExemplare)
	}

	titel := map[string]importTitelZeile{
		lernbuch: {Autor: "Autorin des Lernbuchs", Verlag: "Verlag des Lernbuchs", ISBN: isbn, Fach: "Biologie",
			Signatur: "LMF Bio 7-9", Jahr: 2019, Von: 7, Bis: 9, Lernmittel: true},
		// Kein Lernmittel: Das Fach kommt aus der Kategorie, der Jahrgang bleibt unbekannt.
		sachbuch: {Autor: "Autor des Sachbuchs", Verlag: "Verlag des Sachbuchs", Fach: "Mathematik",
			Signatur: "Ma 12", Jahr: 2001},
		// Die Kennung der Lernmittel steht nur in der Kategorie: Kennzeichen, Fach und Jahrgang
		// kommen von dort, die Signatur bleibt, wie die Datei sie nennt.
		rechenbuch: {Fach: "Mathematik", Signatur: "Ma 6 Rech", Von: 6, Bis: 6, Lernmittel: true},
	}
	for name, soll := range titel {
		if ist := liesImportTitel(t, pool, name); ist != soll {
			t.Errorf("Titel %q:\n  ist  %+v\n  soll %+v", name, ist, soll)
		}
	}

	type exemplarZeile struct {
		Titel, Notiz        string
		Ausleihbar, Etikett bool
	}
	exemplare := map[string]exemplarZeile{
		// „verliehen" in der Spalte Zustand sperrt das Exemplar und bleibt als Notiz stehen.
		barcodes[0]: {Titel: lernbuch, Notiz: "verliehen", Ausleihbar: false, Etikett: true},
		barcodes[1]: {Titel: lernbuch, Notiz: "Ecke geknickt", Ausleihbar: true, Etikett: true},
		barcodes[2]: {Titel: sachbuch, Ausleihbar: true, Etikett: true},
		barcodes[3]: {Titel: rechenbuch, Ausleihbar: true, Etikett: true},
	}
	for barcode, soll := range exemplare {
		var ist exemplarZeile
		if err := pool.QueryRow(ctx, `
			SELECT t.titel, coalesce(e.zustand_notiz, ''), e.ist_ausleihbar, e.etikett_gedruckt
			FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
			WHERE e.barcode_id = $1`, barcode).Scan(&ist.Titel, &ist.Notiz, &ist.Ausleihbar, &ist.Etikett); err != nil {
			t.Fatalf("Exemplar %s lesen: %v", barcode, err)
		}
		if ist != soll {
			t.Errorf("Exemplar %s:\n  ist  %+v\n  soll %+v", barcode, ist, soll)
		}
	}

	// Dieselbe Datei ein zweites Mal: Jede Nummer gibt es schon. Kein Exemplar entsteht doppelt,
	// und der Import meldet keines als neu.
	neueTitel, neueExemplare, err = NewImportService(nil, pool).ImportDynamic(ctx, datei, kopf)
	if err != nil {
		t.Fatalf("zweiter Lauf: %v", err)
	}
	if neueTitel != 0 || neueExemplare != 0 {
		t.Errorf("zweiter Lauf: %d neue Titel und %d neue Exemplare gemeldet, erwartet 0 und 0", neueTitel, neueExemplare)
	}
	var vorhanden int
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM buecher_exemplare WHERE barcode_id = ANY($1)`, barcodes).
		Scan(&vorhanden); err != nil {
		t.Fatal(err)
	}
	if vorhanden != 4 {
		t.Errorf("nach dem zweiten Lauf %d Exemplare unter den vier Nummern, erwartet 4", vorhanden)
	}
}

// Führt die Systematik ein Fach in anderer Schreibweise als die Datei, trägt der neue Titel die
// Schreibweise der Systematik: subject ist ein Fremdschlüssel auf ihre Bezeichnung, mit der
// Schreibweise der Datei scheiterte der ganze Import.
func TestImportDynamic_FachInDerSchreibweiseDerSystematik(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	barcodes := []string{"FACH-PROBE-562-1"}
	const titel = "Fach-Probe Werken und Technik"
	// Die Sachgruppe fällt nach den Titeln: Solange einer sie trägt, lässt sie sich nicht löschen.
	raeumeSachgruppe := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM systematik_kategorien WHERE kuerzel = 'ARBL562'`); err != nil {
			t.Errorf("aufräumen: Sachgruppe löschen: %v", err)
		}
	}
	t.Cleanup(raeumeSachgruppe)
	raeumeImportProbe(t, pool, barcodes, []string{titel})
	raeumeSachgruppe()

	tag, err := pool.Exec(ctx, `INSERT INTO systematik_kategorien (kuerzel, bezeichnung)
		VALUES ('ARBL562', 'ARBEITSLEHRE') ON CONFLICT (lower(bezeichnung)) DO NOTHING`)
	if err != nil {
		t.Fatalf("Sachgruppe anlegen: %v", err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatal("die Systematik führt das Fach schon; die Probe braucht es in eigener Schreibweise")
	}

	neueTitel, _, err := NewImportService(nil, pool).ImportDynamic(ctx, [][]string{
		{"Titel", "Kategorie", "Barcode"},
		{titel, "Arbeitslehre", barcodes[0]},
	}, map[string]int{"titel": 0, "kategorie": 1, "barcode": 2})
	if err != nil {
		t.Fatalf("ImportDynamic: %v", err)
	}
	if fach := liesImportTitel(t, pool, titel).Fach; neueTitel != 1 || fach != "ARBEITSLEHRE" {
		t.Errorf("%d neue Titel, Fach %q — erwartet 1 und die Schreibweise der Systematik, ARBEITSLEHRE", neueTitel, fach)
	}
}

// Die Signatur aus der Datei kommt auch an einen Titel, den es schon gibt. Trägt sie die
// Kennung der Lernmittel, wird der Titel als Lernmittel markiert; ohne Kennung bleibt ein
// gesetztes Kennzeichen stehen, und eine leere Angabe lässt die Signatur, wie sie ist.
func TestImportDynamic_SignaturAmVorhandenenTitel(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	barcodes := []string{"SIGNATUR-PROBE-562-1", "SIGNATUR-PROBE-562-2", "SIGNATUR-PROBE-562-3"}
	const wirdLernmittel, bleibtLernmittel, ohneAngabe = "Signatur-Probe wird Lernmittel",
		"Signatur-Probe bleibt Lernmittel", "Signatur-Probe ohne Angabe"
	raeumeImportProbe(t, pool, barcodes, []string{wirdLernmittel, bleibtLernmittel, ohneAngabe})

	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_titel (titel, signatur, ist_lernmittel) VALUES
			($1, NULL, false), ($2, 'LMF Eng 6', true), ($3, 'Bleibt 1', false)`,
		wirdLernmittel, bleibtLernmittel, ohneAngabe); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	neueTitel, neueExemplare, err := NewImportService(nil, pool).ImportDynamic(ctx, [][]string{
		{"Titel", "Signatur", "Barcode"},
		{wirdLernmittel, "LMF Deu 5", barcodes[0]},
		{bleibtLernmittel, "Eng 6 a", barcodes[1]},
		{ohneAngabe, "", barcodes[2]},
	}, map[string]int{"titel": 0, "signatur": 1, "barcode": 2})
	if err != nil {
		t.Fatalf("ImportDynamic: %v", err)
	}
	if neueTitel != 0 || neueExemplare != 3 {
		t.Errorf("%d neue Titel und %d neue Exemplare gemeldet, erwartet 0 und 3", neueTitel, neueExemplare)
	}

	soll := map[string]importTitelZeile{
		wirdLernmittel:   {Signatur: "LMF Deu 5", Lernmittel: true},
		bleibtLernmittel: {Signatur: "Eng 6 a", Lernmittel: true},
		ohneAngabe:       {Signatur: "Bleibt 1"},
	}
	for name, s := range soll {
		if ist := liesImportTitel(t, pool, name); ist.Signatur != s.Signatur || ist.Lernmittel != s.Lernmittel {
			t.Errorf("Titel %q: Signatur %q, Lernmittel %v — erwartet %q und %v",
				name, ist.Signatur, ist.Lernmittel, s.Signatur, s.Lernmittel)
		}
	}
}

// Tragen zwei Titel dieselbe ISBN in verschiedener Schreibweise, ordnet der Bestands-Import
// eine Zeile dem zu, der sie in der Normalform trägt, wie der Katalog-Import
// (repository.LadeTitelBestand). Solche Paare ließen die Migrationen 140 und 157 stehen; der
// zweite Titel kommt deshalb am Trigger vorbei in die Tabelle. Beide Reihenfolgen, weil die
// Regel nicht davon abhängen darf, welchen der zwei die Datenbank zuerst liefert.
func TestImportDynamic_DoppelteISBNTrifftDenTitelInNormalform(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	faelle := []struct {
		name, normal, alt, isbnDerDatei, barcode string
		normalZuerst                             bool
	}{
		{"Normalform zuerst angelegt", "9780804429573", "080442957X", "0-8044-2957-X", "DOPPEL-PROBE-562-1", true},
		{"Altform zuerst angelegt", "9783161484100", "978-3-16-148410-0", "978-3-16-148410-0", "DOPPEL-PROBE-562-2", false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			titelNormal, titelAlt := "Doppel-Probe Normalform "+f.barcode, "Doppel-Probe Altform "+f.barcode
			inDerDatei := "Doppel-Probe unter drittem Namen " + f.barcode
			raeumeImportProbe(t, pool, []string{f.barcode}, []string{titelNormal, titelAlt, inDerDatei})

			lege := func(titel, isbn string) {
				t.Helper()
				tx, err := pool.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				// Ohne Trigger, nur in dieser Transaktion: So blieb die Altform in der Tabelle.
				_, err = tx.Exec(ctx, `SET LOCAL session_replication_role = replica`)
				if err == nil {
					_, err = tx.Exec(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2)`, titel, isbn)
				}
				if err == nil {
					err = tx.Commit(ctx)
				}
				if err != nil {
					if rbErr := tx.Rollback(ctx); rbErr != nil {
						t.Logf("zurückrollen: %v", rbErr)
					}
					t.Fatalf("Titel %q anlegen: %v", titel, err)
				}
			}
			if f.normalZuerst {
				lege(titelNormal, f.normal)
				lege(titelAlt, f.alt)
			} else {
				lege(titelAlt, f.alt)
				lege(titelNormal, f.normal)
			}

			neueTitel, neueExemplare, err := NewImportService(nil, pool).ImportDynamic(ctx, [][]string{
				{"Titel", "ISBN", "Barcode"},
				{inDerDatei, f.isbnDerDatei, f.barcode},
			}, map[string]int{"titel": 0, "isbn": 1, "barcode": 2})
			if err != nil {
				t.Fatalf("ImportDynamic: %v", err)
			}
			var amTitel string
			if err := pool.QueryRow(ctx, `
				SELECT t.titel FROM buecher_exemplare e JOIN buecher_titel t ON t.id = e.titel_id
				WHERE e.barcode_id = $1`, f.barcode).Scan(&amTitel); err != nil {
				t.Fatalf("Probe-Exemplar lesen: %v", err)
			}
			if neueTitel != 0 || neueExemplare != 1 || amTitel != titelNormal {
				t.Errorf("%d neue Titel, %d neue Exemplare, Exemplar am Titel %q — erwartet 0, 1 und %q",
					neueTitel, neueExemplare, amTitel, titelNormal)
			}
		})
	}
}

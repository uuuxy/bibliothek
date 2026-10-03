package repository

import (
	"context"
	"testing"
)

// Regressionstest gegen den echten Juni-2026-MAB-Export (siehe
// verify_katalogisat_test.go in internal/service): 4.747 von 13.708 Titeln haben
// dort ein leeres Autor-Feld, 779 ein Jahr von 0. Ein erneuter Import über einen
// bereits vorhandenen Bestand darf solche fehlenden Felder NICHT nachträglich
// leeren — genau das tat BulkUpsertBookTitles vor dieser Korrektur: qUpdate schrieb
// autor/verlag/erscheinungsjahr ungeschützt, während signatur/isbn längst per
// COALESCE(NULLIF(...)) geschützt waren. Ein Reimport mit einer schlechteren
// Datenquelle hätte den bestehenden, besseren Bestand stillschweigend verarmt.
func TestBulkUpsertBookTitles_LeereFelderUeberschreibenBestandNicht(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	repo := NewBookRepository(pool)

	// Bestand: vollständiger Titel mit Autor, Verlag und Jahr.
	if _, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{{
		Titel: "Effi Briest", Autor: "Fontane, Theodor",
		ISBN: "9783150001", Verlag: "Reclam", Erscheinungsjahr: 2001, Signatur: "Pa",
	}}); err != nil {
		t.Fatalf("Erstimport: %v", err)
	}

	// Reimport derselben ISBN aus einer Quelle ohne Autor/Verlag/Jahr (wie es im
	// echten MAB-Export häufig vorkommt), aber mit neuer Signatur.
	if _, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{{
		Titel: "Effi Briest", ISBN: "9783150001", Signatur: "Pg",
	}}); err != nil {
		t.Fatalf("Reimport: %v", err)
	}

	var autor, verlag, signatur string
	var jahr *int
	if err := pool.QueryRow(ctx,
		`SELECT autor, verlag, erscheinungsjahr, signatur FROM buecher_titel WHERE isbn = $1`,
		"9783150001").Scan(&autor, &verlag, &jahr, &signatur); err != nil {
		t.Fatalf("Titel nach Reimport nicht lesbar: %v", err)
	}

	if autor != "Fontane, Theodor" {
		t.Errorf("autor = %q, will von leerem Reimport nicht überschrieben werden — want %q", autor, "Fontane, Theodor")
	}
	if verlag != "Reclam" {
		t.Errorf("verlag = %q, will von leerem Reimport nicht überschrieben werden — want %q", verlag, "Reclam")
	}
	if jahr == nil || *jahr != 2001 {
		t.Errorf("erscheinungsjahr = %v, will von jahr=0 im Reimport nicht auf NULL gesetzt werden — want 2001", jahr)
	}
	// Die Signatur DARF sich ändern — Reimport bringt hier bewusst einen neuen Wert.
	if signatur != "Pg" {
		t.Errorf("signatur = %q, want %q (neuer Wert aus dem Reimport)", signatur, "Pg")
	}
}

// Ein Reimport MIT neuen Werten darf die alten weiterhin ersetzen (Enrichment) —
// der Fix darf kein Deadlock in Richtung "nie mehr ändern" werden.
func TestBulkUpsertBookTitles_NichtLeereFelderUeberschreibenBestand(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	repo := NewBookRepository(pool)

	if _, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{{
		Titel: "Faust", Autor: "Unbekannt", ISBN: "9783150002",
		Verlag: "Alter Verlag", Erscheinungsjahr: 1990, Signatur: "De",
	}}); err != nil {
		t.Fatalf("Erstimport: %v", err)
	}

	if _, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{{
		Titel: "Faust", Autor: "Goethe, Johann Wolfgang von", ISBN: "9783150002",
		Verlag: "Reclam", Erscheinungsjahr: 2020, Signatur: "Deu",
	}}); err != nil {
		t.Fatalf("Reimport: %v", err)
	}

	var autor, verlag, signatur string
	var jahr *int
	if err := pool.QueryRow(ctx,
		`SELECT autor, verlag, erscheinungsjahr, signatur FROM buecher_titel WHERE isbn = $1`,
		"9783150002").Scan(&autor, &verlag, &jahr, &signatur); err != nil {
		t.Fatalf("Titel nach Reimport nicht lesbar: %v", err)
	}
	if autor != "Goethe, Johann Wolfgang von" || verlag != "Reclam" || jahr == nil || *jahr != 2020 || signatur != "Deu" {
		t.Errorf("Reimport mit echten Werten wurde nicht übernommen: autor=%q verlag=%q jahr=%v signatur=%q",
			autor, verlag, jahr, signatur)
	}
}

// Ein Mehrjahresband behält seine Spanne (Migration 134, Rasterdurchgang 23.09.2026, K1).
//
// Der Katalogisat-Import übernimmt die Jahrgangsspanne aus Litteras Signatur, und die nennt
// oft nur einen Jahrgang („LMF Bio 7"). Bei einem Mehrjahresband verletzte das
// chk_mehrjahresband_spanne — und weil der Import alles oder nichts schreibt, fiel mit dem
// einen Titel der ganze Import (am alten Code: „violates check constraint
// chk_mehrjahresband_spanne"). Den Schalter gibt es nur in der Titel-Verwaltung; wer ihn
// setzt, setzt die Spanne mit, und an ihrem „bis" hängt die Frist.
func TestBulkUpsertBookTitles_MehrjahresbandBehaeltSeineSpanne(t *testing.T) {
	pool := pgTestPool(t)
	ctx := context.Background()
	repo := NewBookRepository(pool)
	const isbnBand, isbnNormal = "9780000134017", "9780000134024"
	t.Cleanup(func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn IN ($1, $2)`, isbnBand, isbnNormal); err != nil {
			t.Errorf("aufräumen: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `
		INSERT INTO buecher_titel (titel, isbn, ist_lernmittel, jahrgang_von, jahrgang_bis, mehrjahresband)
		VALUES ('Band Bio 7-9', $1, true, 7, 9, true),
		       ('Normal Bio 7', $2, true, 5, 10, false)`, isbnBand, isbnNormal); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}

	// Gegenprobe: Ohne Schalter folgt die Spanne weiter der Quelle.
	if _, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{{
		Titel: "Normal Bio 7", ISBN: isbnNormal, Signatur: "LMF Bio 7",
		IstLernmittel: true, JahrgangVon: 7, JahrgangBis: 7,
	}}); err != nil {
		t.Fatalf("Import über einen gewöhnlichen Titel: %v", err)
	}
	var von, bis int
	if err := pool.QueryRow(ctx, `SELECT jahrgang_von, jahrgang_bis FROM buecher_titel WHERE isbn = $1`, isbnNormal).Scan(&von, &bis); err != nil {
		t.Fatal(err)
	}
	if von != 7 || bis != 7 {
		t.Errorf("gewöhnlicher Titel: Spanne %d–%d, erwartet 7–7 aus der Signatur", von, bis)
	}

	if _, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{{
		Titel: "Band Bio 7-9", ISBN: isbnBand, Signatur: "LMF Bio 7",
		IstLernmittel: true, JahrgangVon: 7, JahrgangBis: 7,
	}}); err != nil {
		t.Fatalf("Import über einen Mehrjahresband: %v", err)
	}
	var band bool
	if err := pool.QueryRow(ctx, `SELECT jahrgang_von, jahrgang_bis, mehrjahresband FROM buecher_titel WHERE isbn = $1`, isbnBand).Scan(&von, &bis, &band); err != nil {
		t.Fatal(err)
	}
	if von != 7 || bis != 9 || !band {
		t.Errorf("Mehrjahresband nach dem Import: Spanne %d–%d, Schalter %v — erwartet 7–9 und an", von, bis, band)
	}
}

// Der Katalog-Import ordnet einen Datensatz über die ISBN zu und vergleicht dabei Zeichen für
// Zeichen. Die Datenbank speichert eine zehnstellige ISBN dreizehnstellig (Migration 157);
// die Zuordnung muss dieselbe Form führen. Sonst findet der Import den vorhandenen Titel nicht,
// legt ihn neu an und scheitert am UNIQUE-Index — der ganze Import, er läuft in einer
// Transaktion.
func TestBulkUpsertBookTitles_BeideLaengenDerISBNSindEinTitel(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewBookRepository(pool)

	titelZur := func(isbn string) (anzahl int, signatur string) {
		t.Helper()
		if err := pool.QueryRow(ctx, `SELECT count(*)::int, coalesce(max(signatur), '')
			FROM buecher_titel WHERE isbn = $1`, isbn).Scan(&anzahl, &signatur); err != nil {
			t.Fatal(err)
		}
		return anzahl, signatur
	}

	// Der Bestand trägt die dreizehnstellige; die Datei nennt das Buch zehnstellig und unter
	// einem anderen Titeltext, damit nur die ISBN zuordnen kann.
	if _, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{{
		Titel: "Harry Potter und der Stein der Weisen", ISBN: "9783551551672", Signatur: "Jf",
	}}); err != nil {
		t.Fatalf("Erstimport: %v", err)
	}
	if _, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{{
		Titel: "Harry Potter 1", ISBN: "3551551677", Signatur: "JF Row",
	}}); err != nil {
		t.Fatalf("Import mit der zehnstelligen ISBN: %v", err)
	}
	if anzahl, signatur := titelZur("9783551551672"); anzahl != 1 || signatur != "JF Row" {
		t.Errorf("nach dem Import: %d Titel mit Signatur %q, erwartet 1 mit „JF Row“", anzahl, signatur)
	}

	// Eine Datei nennt dasselbe Buch in beiden Längen: ein Titel, der zweite Datensatz ist
	// eine Dublette innerhalb der Datei.
	eingereiht, err := repo.BulkUpsertBookTitles(ctx, []BookTitle{
		{Titel: "Advanced Organic Chemistry", ISBN: "0306406152"},
		{Titel: "Advanced Organic Chemistry, Part A", ISBN: "9780306406157"},
	})
	if err != nil {
		t.Fatalf("Import mit beiden Längen: %v", err)
	}
	if anzahl, _ := titelZur("9780306406157"); anzahl != 1 || eingereiht != 1 {
		t.Errorf("beide Längen in einer Datei: %d Titel, %d Datensätze eingereiht, erwartet je 1", anzahl, eingereiht)
	}

	// Der Aufrufer behält seine Liste, wie er sie übergeben hat.
	datei := []BookTitle{{Titel: "Harry Potter 1", ISBN: "3551551677"}}
	if _, err := repo.BulkUpsertBookTitles(ctx, datei); err != nil {
		t.Fatal(err)
	}
	if datei[0].ISBN != "3551551677" {
		t.Errorf("die Liste des Aufrufers trägt danach %q, übergeben war 3551551677", datei[0].ISBN)
	}
}

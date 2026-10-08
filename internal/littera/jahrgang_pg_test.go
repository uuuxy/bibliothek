package littera

import (
	"context"
	"strings"
	"testing"
)

// Sagt Littera nichts über den Jahrgang (keine Lernmittel-Signatur, kein Interessenkreis),
// bleibt er am übernommenen Titel unbekannt: NULL in beiden Spalten (Migration 162). Mit einer
// Vorgabe wären diese Titel von denen mit dem Interessenkreis „Sekundarstufe 1" (5 bis 10)
// nicht zu unterscheiden.
func TestJahrgangOhneAngabeBleibtUnbekannt(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)
	ctx := context.Background()

	gelesen, err := LeseTitel(strings.NewReader(
		"Buchungsnummer,Haupttitel,Verlag,Medienart\n" +
			`1,"Faust",1,1` + "\n" +
			`2,"Die Welle",1,1` + "\n"))
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	bericht, err := s.SchreibeBestand(ctx, bestand(gelesen...))
	if err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}
	if bericht.Titel != 2 {
		t.Fatalf("geschrieben %d Titel, erwartet 2", bericht.Titel)
	}

	var titel, mitSpanne int
	if err := pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE jahrgang_von IS NOT NULL OR jahrgang_bis IS NOT NULL)
		FROM buecher_titel`).Scan(&titel, &mitSpanne); err != nil {
		t.Fatal(err)
	}
	if titel != 2 || mitSpanne != 0 {
		t.Errorf("%d von %d übernommenen Titeln tragen eine Jahrgangsspanne, erwartet keiner von 2", mitSpanne, titel)
	}
}

// Nennt die Lernmittel-Signatur mehrere Jahrgänge, steht die Spanne als „von" und „bis" am
// übernommenen Titel; ein einzelner Jahrgang steht in beiden. Mit einem einzigen Jahrgang
// allein fiele nicht auf, wenn die zwei Werte vertauscht ankämen.
func TestJahrgangAusDerSignatur_SpanneUndEinJahr(t *testing.T) {
	pool := pgTestPool(t)
	leereAlles(t, pool)
	s, _ := testSchreiber(t, pool, nil)
	ctx := context.Background()

	gelesen, err := LeseTitel(strings.NewReader(
		"Buchungsnummer,Haupttitel,Verlag,Medienart\n" +
			`1,"Natur und Technik",1,1` + "\n" +
			`2,"Deutschbuch",1,1` + "\n"))
	if err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	ab := bestand(gelesen...)
	ab.Signaturen["1"] = "LMF Bio 7-9 / Nat"
	ab.Signaturen["2"] = "LMF Deu 7 / Bie"
	if _, err := s.SchreibeBestand(ctx, ab); err != nil {
		t.Fatalf("SchreibeBestand: %v", err)
	}

	for titel, soll := range map[string][2]int{"Natur und Technik": {7, 9}, "Deutschbuch": {7, 7}} {
		var von, bis int
		if err := pool.QueryRow(ctx, `
			SELECT coalesce(jahrgang_von, 0), coalesce(jahrgang_bis, 0) FROM buecher_titel WHERE titel = $1`, titel).
			Scan(&von, &bis); err != nil {
			t.Fatalf("%s: %v", titel, err)
		}
		if von != soll[0] || bis != soll[1] {
			t.Errorf("%s: übernommen mit %d bis %d, erwartet %d bis %d", titel, von, bis, soll[0], soll[1])
		}
	}
}

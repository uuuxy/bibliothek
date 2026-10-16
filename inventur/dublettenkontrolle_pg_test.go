package inventur

import (
	"context"
	"errors"
	"testing"

	"bibliothek/internal/pgtest"
)

// Dublettenkontrolle beim Anlegen nach dem Vorbild von Littera: Dort wird bei der Aufnahme eines
// Mediums über die ISBN geprüft (ohne ISBN über Verfasser und Haupttitel), und statt eines
// zweiten Titels bietet das Programm an, ein weiteres Exemplar anzulegen.
//
// Der UNIQUE-Index auf isbn fängt nur zeichengleiche Dubletten: „978-3-12-345678-9" und
// „9783123456789" sind dieselbe Nummer und stünden sonst als zwei Titel nebeneinander, mit zwei
// Beständen, zwei Meldebeständen und zwei Zeilen in der Nachbestellung.
//
// Ohne ISBN entscheidet das Paar aus Titel und Autor. Eine zweite Auflage ist erlaubt: Dafür
// gibt es ein Feld, und wer es füllt, sagt damit, dass er ein anderes Buch meint.
func TestDublettenkontrolle_BeimAnlegen(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	abfragen := NewTitelAbfragen(pool)

	aufraeumen := func(t *testing.T, titel string) {
		t.Helper()
		t.Cleanup(func() {
			if _, err := pool.Exec(context.Background(), `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
				t.Logf("Aufräumen: %v", err)
			}
		})
	}

	t.Run("dieselbe ISBN in anderer Schreibweise", func(t *testing.T) {
		const titel = "Dubletten-Probe ISBN"
		aufraeumen(t, titel)
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
			t.Fatal(err)
		}
		if _, err := abfragen.LegeTitelAn(ctx, Titel{ISBN: "978-3-12-345678-9", Titel: titel}); err != nil {
			t.Fatalf("erster Titel: %v", err)
		}
		_, err := abfragen.LegeTitelAn(ctx, Titel{ISBN: "9783123456789", Titel: titel + " (zweite Schreibweise)"})
		if !errors.Is(err, errISBNDoppelt) {
			t.Errorf("zweite Schreibweise: Fehler %v, erwartet errISBNDoppelt — es ist dieselbe Nummer", err)
		}
	})

	t.Run("ohne ISBN entscheidet Titel und Autor", func(t *testing.T) {
		const titel = "Dubletten-Probe ohne Nummer"
		aufraeumen(t, titel)
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
			t.Fatal(err)
		}
		if _, err := abfragen.LegeTitelAn(ctx, Titel{Titel: titel, Autor: "Meyer"}); err != nil {
			t.Fatalf("erster Titel: %v", err)
		}
		_, err := abfragen.LegeTitelAn(ctx, Titel{Titel: titel, Autor: "Meyer"})
		if !errors.Is(err, errDubletteTitel) {
			t.Errorf("zweiter gleicher Titel: Fehler %v, erwartet errDubletteTitel", err)
		}

		// Eine andere Auflage ist ein anderes Buch und darf angelegt werden.
		if _, err := abfragen.LegeTitelAn(ctx, Titel{Titel: titel, Autor: "Meyer", Auflage: "2. Aufl."}); err != nil {
			t.Errorf("zweite Auflage: %v — mit eigener Auflage ist es ein anderes Buch", err)
		}
	})

	// Aus Littera kommt die Auflage mit an den Titel. Wer einen gleichnamigen Titel aufnimmt und
	// keine Auflage nennt, hat nicht gesagt, dass er ein anderes Buch meint: Die Tür fragt.
	// Die Auflage steht, wie sie getippt wurde; dieselbe Angabe mit anderem Leerraum oder in
	// anderer Großschreibung ist dieselbe Auflage.
	t.Run("ohne ISBN: ein Titel mit Auflage zählt, wenn die Eingabe keine nennt", func(t *testing.T) {
		const titel = "Dubletten-Probe mit Auflage"
		aufraeumen(t, titel)
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
			t.Fatal(err)
		}
		if _, err := abfragen.LegeTitelAn(ctx, Titel{Titel: titel, Autor: "Meyer", Auflage: "3.  Aufl."}); err != nil {
			t.Fatalf("erster Titel: %v", err)
		}
		for _, auflage := range []string{"", "3. aufl. ", "3.   Aufl."} {
			_, err := abfragen.LegeTitelAn(ctx, Titel{Titel: titel, Autor: "Meyer", Auflage: auflage})
			if !errors.Is(err, errDubletteTitel) {
				t.Errorf("Auflage %q: Fehler %v, erwartet errDubletteTitel", auflage, err)
			}
		}
		if _, err := abfragen.LegeTitelAn(ctx, Titel{Titel: titel, Autor: "Meyer", Auflage: "4. Aufl."}); err != nil {
			t.Errorf("andere Auflage: %v — mit eigener Auflage ist es ein anderes Buch", err)
		}
	})

	// Die Datenbank speichert Titel und Autor mit einem Leerzeichen zwischen den Wörtern.
	// Die Kontrolle vergleicht die Eingabe in derselben Form: Sonst legte,
	// wer denselben Wortlaut ein zweites Mal tippt, ohne Rückfrage einen zweiten Titel an.
	t.Run("ohne ISBN: Leerraum in Folge ist derselbe Titel", func(t *testing.T) {
		const titel = "Dubletten-Probe La Peste"
		geschuetzt := string(rune(0x00A0))
		aufraeumen(t, titel)
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE titel = $1`, titel); err != nil {
			t.Fatal(err)
		}
		id, err := abfragen.LegeTitelAn(ctx, Titel{Titel: "Dubletten-Probe La  Peste", Autor: "Camus,  Albert"})
		if err != nil {
			t.Fatalf("erster Titel: %v", err)
		}
		var gespeichert, autor string
		if err := pool.QueryRow(ctx, `SELECT titel, autor FROM buecher_titel WHERE id = $1`, id).
			Scan(&gespeichert, &autor); err != nil {
			t.Fatal(err)
		}
		if gespeichert != titel || autor != "Camus, Albert" {
			t.Fatalf("gespeichert %q von %q, erwartet %q von %q", gespeichert, autor, titel, "Camus, Albert")
		}

		for _, eingabe := range []Titel{
			{Titel: "Dubletten-Probe La  Peste", Autor: "Camus,  Albert"},
			{Titel: " Dubletten-Probe La" + geschuetzt + "Peste ", Autor: "Camus," + geschuetzt + "Albert"},
			{Titel: "Dubletten-Probe La Peste", Autor: "Camus, Albert"},
		} {
			_, err := abfragen.LegeTitelAn(ctx, eingabe)
			if !errors.Is(err, errDubletteTitel) {
				t.Errorf("%+q von %+q: Fehler %v, erwartet errDubletteTitel", eingabe.Titel, eingabe.Autor, err)
			}
		}
	})
}

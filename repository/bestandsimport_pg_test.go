package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Tragen zwei Titel dieselbe ISBN in verschiedener Schreibweise, führt der Titelbestand unter
// ihrer Normalform den Titel, der sie so trägt; damit ordnen Katalog-Import und Bestands-Import
// zu. Solche Paare ließen die Migrationen 140 und 157 stehen, die Altform kommt deshalb am
// Trigger vorbei in die Tabelle. Beide Reihenfolgen: Die Regel darf nicht davon abhängen,
// welchen der zwei die Datenbank zuerst liefert.
func TestLadeTitelBestand_BeiDoppelterISBNGiltDieNormalform(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()

	faelle := []struct {
		name, normal, alt string
		normalZuerst      bool
	}{
		{"Normalform zuerst angelegt", "9780804429573", "080442957X", true},
		{"Altform zuerst angelegt", "9783161484100", "978-3-16-148410-0", false},
	}
	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			tx := beginne(t, pool)
			defer func() {
				if err := tx.Rollback(ctx); err != nil {
					t.Errorf("zurückrollen: %v", err)
				}
			}()
			lege := func(titel, isbn string) (id string) {
				t.Helper()
				if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, isbn) VALUES ($1, $2) RETURNING id::text`,
					titel, isbn).Scan(&id); err != nil {
					t.Fatalf("Titel %q anlegen: %v", titel, err)
				}
				return id
			}
			if _, err := tx.Exec(ctx, `ALTER TABLE buecher_titel DISABLE TRIGGER trg_titel_isbn_normalform`); err != nil {
				t.Fatal(err)
			}
			var idNormal string
			if f.normalZuerst {
				idNormal = lege("Bestand-Probe Normalform", f.normal)
				lege("Bestand-Probe Altform", f.alt)
			} else {
				lege("Bestand-Probe Altform", f.alt)
				idNormal = lege("Bestand-Probe Normalform", f.normal)
			}

			isbnZuID, titelZuID, err := LadeTitelBestand(ctx, tx)
			if err != nil {
				t.Fatalf("LadeTitelBestand: %v", err)
			}
			if ist := isbnZuID[f.normal]; ist != idNormal {
				t.Errorf("unter %s steht der Titel %s, erwartet der Titel in Normalform %s", f.normal, ist, idNormal)
			}
			if ist := titelZuID[NormalisiereTitelKey("Bestand-Probe Normalform")]; ist != idNormal {
				t.Errorf("unter dem Titeltext steht %s, erwartet %s", ist, idNormal)
			}
			if _, da := isbnZuID[f.alt]; da {
				t.Errorf("der Bestand führt die Altform %s als eigenen Schlüssel", f.alt)
			}
		})
	}
}

// Der Katalog-Import legt einen neuen Titel mit jedem Feld des Datensatzes an. Jedes Feld trägt
// einen eigenen Wert: Mit gleichen fiele nicht auf, wenn zwei vertauscht ankämen.
func TestBulkUpsertBookTitles_JedesFeldEinesNeuenTitelsKommtAn(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const isbn, fach = "9780000562029", "Importfach Felderkunde"
	raeume := func() {
		if _, err := pool.Exec(ctx, `DELETE FROM buecher_titel WHERE isbn = $1 OR subject = $2`, isbn, fach); err != nil {
			t.Errorf("aufräumen: Probe-Titel löschen: %v", err)
		}
		if _, err := pool.Exec(ctx, `DELETE FROM systematik_kategorien WHERE bezeichnung = $1`, fach); err != nil {
			t.Errorf("aufräumen: Sachgruppe löschen: %v", err)
		}
	}
	raeume()
	t.Cleanup(raeume)

	neu := BookTitle{
		Titel: "Felder-Probe Katalog", Autor: "Autorin im Katalog", ISBN: isbn, Verlag: "Verlag im Katalog",
		Erscheinungsjahr: 2017, Signatur: "LMF Fel 5-6", IstLernmittel: true, Fach: fach,
		JahrgangVon: 5, JahrgangBis: 6,
	}
	if _, err := NewBookRepository(pool).BulkUpsertBookTitles(ctx, []BookTitle{neu}); err != nil {
		t.Fatalf("Import: %v", err)
	}

	var ist BookTitle
	if err := pool.QueryRow(ctx, `
		SELECT titel, coalesce(autor, ''), coalesce(isbn, ''), coalesce(verlag, ''), coalesce(erscheinungsjahr, 0),
		       coalesce(signatur, ''), ist_lernmittel, coalesce(subject, ''), coalesce(jahrgang_von, 0),
		       coalesce(jahrgang_bis, 0)
		FROM buecher_titel WHERE isbn = $1`, isbn).
		Scan(&ist.Titel, &ist.Autor, &ist.ISBN, &ist.Verlag, &ist.Erscheinungsjahr, &ist.Signatur,
			&ist.IstLernmittel, &ist.Fach, &ist.JahrgangVon, &ist.JahrgangBis); err != nil {
		t.Fatalf("Titel lesen: %v", err)
	}
	if ist.Titel != neu.Titel || ist.Autor != neu.Autor || ist.ISBN != neu.ISBN || ist.Verlag != neu.Verlag ||
		ist.Erscheinungsjahr != neu.Erscheinungsjahr || ist.Signatur != neu.Signatur ||
		ist.IstLernmittel != neu.IstLernmittel || ist.Fach != neu.Fach ||
		ist.JahrgangVon != neu.JahrgangVon || ist.JahrgangBis != neu.JahrgangBis {
		t.Errorf("neuer Titel:\n  ist  %+v\n  soll %+v", ist, neu)
	}
}

// Ein Eintrag ohne Signatur lässt die Signatur seines Titels, wie sie ist; der Eintrag daneben
// wird geschrieben.
func TestSetzeImportSignaturen_LeereSignaturUeberschreibtNicht(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()
	lege := func(titel, signatur string) (id string) {
		t.Helper()
		if err := tx.QueryRow(ctx, `INSERT INTO buecher_titel (titel, signatur) VALUES ($1, $2) RETURNING id::text`,
			titel, signatur).Scan(&id); err != nil {
			t.Fatalf("Titel %q anlegen: %v", titel, err)
		}
		return id
	}
	bleibt, wechselt := lege("Signatur-Probe bleibt", "Alt 1"), lege("Signatur-Probe wechselt", "Alt 2")

	if err := SetzeImportSignaturen(ctx, tx, []ImportSignatur{
		{TitelID: bleibt},
		{TitelID: wechselt, Signatur: "Neu 2"},
	}); err != nil {
		t.Fatalf("SetzeImportSignaturen: %v", err)
	}
	for id, soll := range map[string]string{bleibt: "Alt 1", wechselt: "Neu 2"} {
		var ist string
		if err := tx.QueryRow(ctx, `SELECT coalesce(signatur, '') FROM buecher_titel WHERE id = $1`, id).Scan(&ist); err != nil {
			t.Fatal(err)
		}
		if ist != soll {
			t.Errorf("Titel %s trägt die Signatur %q, erwartet %q", id, ist, soll)
		}
	}
}

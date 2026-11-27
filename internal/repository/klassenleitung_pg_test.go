package repository

import (
	"context"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Auslöser schreibt eine Klasse in der Schreibweise des Vokabulars. Setzen, Nachsehen und
// Löschen treffen sie deshalb auch, wenn die Anfrage anders schreibt („7z9“ für „07Z9“), über
// klassen_normkey wie Klassensätze und LMF-Plan. Vorher hielt Setzen die zweite Schreibweise für
// eine neue Klasse (Protokoll „eingetragen“), und Löschen traf keine Zeile.
func TestKlassenleitung_TrifftDieKlasseInJederSchreibweise(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	tx := beginne(t, pool)
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("zurückrollen: %v", err)
		}
	}()

	if neu, _, err := SetzeKlassenleitung(ctx, tx, "07Z9", "erste@schule.invalid"); err != nil || !neu {
		t.Fatalf("SetzeKlassenleitung(07Z9) = neu %v, %v; erwartet neu", neu, err)
	}
	neu, unveraendert, err := SetzeKlassenleitung(ctx, tx, "7z9", "zweite@schule.invalid")
	if err != nil || neu || unveraendert {
		t.Errorf("SetzeKlassenleitung(7z9) = neu %v, unveraendert %v, %v; erwartet geändert", neu, unveraendert, err)
	}
	if _, unveraendert, err := SetzeKlassenleitung(ctx, tx, "7 z9", "zweite@schule.invalid"); err != nil || !unveraendert {
		t.Errorf("SetzeKlassenleitung(7 z9) mit derselben Adresse = unveraendert %v, %v; erwartet unverändert", unveraendert, err)
	}
	var zeilen int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM klassen_lehrer_mapping
		WHERE klassen_normkey(klasse) = '7z9'`).Scan(&zeilen); err != nil || zeilen != 1 {
		t.Errorf("%d Zeilen für die Klasse (%v), erwartet eine", zeilen, err)
	}
	if vorhanden, err := IstKlassenleitungVorhanden(ctx, tx, "7Z9"); err != nil || !vorhanden {
		t.Errorf("IstKlassenleitungVorhanden(7Z9) = %v, %v; erwartet true", vorhanden, err)
	}
	if getroffen, err := LoescheKlassenleitung(ctx, tx, "7z9"); err != nil || getroffen != 1 {
		t.Errorf("LoescheKlassenleitung(7z9) = %d, %v; erwartet eine Zeile", getroffen, err)
	}
}

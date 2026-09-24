package api

import (
	"context"
	"errors"
	"testing"

	"bibliothek/repository"
)

// Zwei Akten desselben Kollegen zusammenführen — und die Grenze, die dabei zu halten ist.
//
// Der Anlass (16.09.2026): Ein Kollege, der von Hand in die Leserdatei eingetragen
// wurde und sich danach über "Mein Portal" selbst anmeldet, steht zweimal da. Der Wächter
// trg_benutzer_hat_leserzeile legt zu jedem neuen Konto eine frische Leserzeile an, ohne
// zu prüfen, ob die Person schon dasteht. Ausweis und Ausleihen hängen dann am ersten
// Eintrag, die Anmeldung am zweiten.
//
// Repariert werden konnte das bis heute von NIEMANDEM, auch nicht vom Administrator:
// Das Zusammenführen las und schrieb gegen die Sicht `schueler` (WHERE art='schueler'),
// ein Kollege stand nicht darin, und der Vorgang endete in "nicht gefunden".
//
// Mit dem Umhängen auf die Tabelle fällt ein Schutz weg, den vorher die Sicht zufällig
// gestellt hat: dass sich ein Kollege nicht mit einem SCHÜLER verschmelzen lässt. Das
// wäre nicht bloss falsch, sondern unrettbar — Zusammenführen löscht die Quelle. Deshalb
// steht die Grenze jetzt ausdrücklich im Code, und dieser Test hält sie fest.
func TestZusammenfuehrenKollegium(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	legeLeser := func(t *testing.T, vorname, nachname, art, barcode string) string {
		t.Helper()
		var id string
		var bc any
		if barcode != "" {
			bc = barcode
		}
		if err := pool.QueryRow(ctx,
			`INSERT INTO leser (vorname, nachname, art, barcode_id) VALUES ($1,$2,$3,$4) RETURNING id`,
			vorname, nachname, art, bc).Scan(&id); err != nil {
			t.Fatalf("Leser anlegen: %v", err)
		}
		return id
	}

	legeSchueler := func(t *testing.T, nachname, barcode string) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO leser (vorname, nachname, art, klasse, barcode_id, abgaenger_jahr)
			 VALUES ('Zf', $1, 'schueler', '07A', $2, 2030) RETURNING id`,
			nachname, barcode).Scan(&id); err != nil {
			t.Fatalf("Schüler anlegen: %v", err)
		}
		return id
	}

	// Der Fall, um den es geht: die Handanlage (mit Ausweis) und die Zeile aus der
	// Selbstanmeldung (ohne alles). Die Selbstanmeldung legt JEDEN als 'lehrkraft' an —
	// auch eine LiV. Genau deshalb müssen sich Lehrkraft und LiV treffen dürfen.
	t.Run("Handanlage und Selbstanmeldung werden eine Akte", func(t *testing.T) {
		ziel := legeLeser(t, "Katrin", "Wendlandt", "liv", "ZF-K-1")
		quelle := legeLeser(t, "Katrin", "Wendlandt", "lehrkraft", "")

		erg, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle))
		if err != nil {
			t.Fatalf("Zusammenführen: %v", err)
		}
		if erg.ZielID != ziel {
			t.Fatalf("Ziel ist %q, erwartet %q", erg.ZielID, ziel)
		}
		if n := zfZaehle(t, pool, `SELECT count(*) FROM leser WHERE id = $1`, quelle); n != 0 {
			t.Fatalf("die Quelle steht noch da (%d Zeilen)", n)
		}
		// Die Art des Ziels bleibt, was sie war: LiV ist die genauere Angabe, und die
		// Selbstanmeldung wusste es nicht besser.
		var art string
		if err := pool.QueryRow(ctx, `SELECT art FROM leser WHERE id = $1`, ziel).Scan(&art); err != nil {
			t.Fatal(err)
		}
		if art != "liv" {
			t.Fatalf("Art des Ziels ist %q, erwartet %q", art, "liv")
		}
	})

	// Der Kern: Die Grenze zum Schüler, in beide Richtungen.
	t.Run("ein Kollege lässt sich nicht mit einem Schüler verschmelzen", func(t *testing.T) {
		kollege := legeLeser(t, "Grenz", "Fall", "lehrkraft", "ZF-K-2")
		schueler := legeSchueler(t, "Fall", "ZF-S-2")

		for _, f := range []struct {
			name         string
			ziel, quelle string
		}{
			{"Schüler als Ziel", schueler, kollege},
			{"Kollege als Ziel", kollege, schueler},
		} {
			t.Run(f.name, func(t *testing.T) {
				_, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(f.ziel, f.quelle))
				if !errors.Is(err, repository.ErrZusammenfuehrenVerschiedeneArten) {
					t.Fatalf("Fehler war %v, erwartet ErrZusammenfuehrenVerschiedeneArten", err)
				}
				// Nichts darf passiert sein — Zusammenführen löscht die Quelle.
				for _, id := range []string{f.ziel, f.quelle} {
					if n := zfZaehle(t, pool, `SELECT count(*) FROM leser WHERE id = $1`, id); n != 1 {
						t.Fatalf("Datensatz %s wurde trotz Ablehnung angetastet", id)
					}
				}
			})
		}
	})

	// Die Trefferliste darf nichts anbieten, was das Zusammenführen danach ablehnt — und
	// sie muss einen Kollegen überhaupt erst lesen können: Ohne COALESCE scheitert der
	// Scan an der fehlenden Klasse und Ausweisnummer, als 500 statt als Liste.
	t.Run("die Suche bleibt auf derselben Seite der Grenze", func(t *testing.T) {
		kollege := legeLeser(t, "Such", "Gleichnam", "lehrkraft", "")
		zwilling := legeLeser(t, "Such", "Gleichnam", "liv", "")
		schueler := legeSchueler(t, "Gleichnam", "ZF-S-3")

		treffer, err := repository.SucheZusammenfuehrenKandidaten(ctx, pool, kollege, "Gleichnam", 20)
		if err != nil {
			t.Fatalf("Suche vom Kollegen aus: %v", err)
		}
		ids := map[string]bool{}
		for _, k := range treffer {
			ids[k.ID] = true
		}
		if !ids[zwilling] {
			t.Fatalf("der zweite Kollege fehlt in der Trefferliste (%d Treffer)", len(treffer))
		}
		if ids[schueler] {
			t.Fatal("ein Schüler steht in der Trefferliste eines Kollegen")
		}

		vomSchueler, err := repository.SucheZusammenfuehrenKandidaten(ctx, pool, schueler, "Gleichnam", 20)
		if err != nil {
			t.Fatalf("Suche vom Schüler aus: %v", err)
		}
		for _, k := range vomSchueler {
			if k.ID == kollege || k.ID == zwilling {
				t.Fatal("ein Kollege steht in der Trefferliste eines Schülers")
			}
		}
	})

	// Ein aktives Konto ist nie ohne Nummer (Migration 145, docs/OFFEN.md 5.16) — auch nicht,
	// wenn es beim Zusammenführen auf eine Zeile ohne Nummer wandert. Das Ziel behält seine
	// Nummer (siehe Kopf von repository/schueler_zusammenfuehren.go); hat es keine, kommt
	// eine aus dem Generator.
	t.Run("ein aktives Konto wandert nicht auf eine Zeile ohne Nummer", func(t *testing.T) {
		ziel := legeLeser(t, "Ohne", "Nummerziel", "lehrkraft", "")
		var quelle string
		if err := pool.QueryRow(ctx, `
			INSERT INTO benutzer (email, vorname, nachname, rolle)
			VALUES ('ohne.nummerziel@schule.invalid', 'Ohne', 'Nummerziel', 'kollegium')
			RETURNING leser_id`).Scan(&quelle); err != nil {
			t.Fatalf("Konto anlegen: %v", err)
		}
		if _, err := repository.ZusammenfuehrenSchueler(ctx, pool, zfAuftrag(ziel, quelle)); err != nil {
			t.Fatalf("Zusammenführen: %v", err)
		}
		var nummer *string
		var aktiv bool
		if err := pool.QueryRow(ctx, `
			SELECT l.barcode_id, b.aktiv FROM leser l JOIN benutzer b ON b.leser_id = l.id
			 WHERE l.id = $1`, ziel).Scan(&nummer, &aktiv); err != nil {
			t.Fatalf("Ziel lesen: %v", err)
		}
		if aktiv && nummer == nil {
			t.Fatal("ein aktives Konto hängt nach dem Zusammenführen an einer Zeile ohne Ausweisnummer")
		}
	})
}

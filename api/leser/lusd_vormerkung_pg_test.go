package leser

import (
	"context"
	"fmt"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/lusd"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Wer die Schule verlässt, holt kein vorgemerktes Buch mehr ab. Der Abgleich löscht deshalb die
// wartenden Vormerkungen der Abgänger; sonst hielten sie den Titel für die anderen besetzt. Die
// Vormerkung eines Schülers, der bleibt, bleibt.
func TestLUSDImport_AbgaengerGibtWartendeVormerkungFrei(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	var titelID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO buecher_titel (titel, autor) VALUES ('Begehrtes Buch', 'Test') RETURNING id`).Scan(&titelID); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	ids := map[string]string{}
	for i, name := range []string{"geht", "bleibt"} {
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
			 VALUES ($1, $2, 'Vormerkung', '9a', 2030, $3) RETURNING id`,
			fmt.Sprintf("VM-%d", i), name, "L-VM-"+name).Scan(&id); err != nil {
			t.Fatalf("Schüler %s anlegen: %v", name, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO vormerkungen (titel_id, schueler_id, status) VALUES ($1, $2, 'wartend')`, titelID, id); err != nil {
			t.Fatalf("Vormerkung von %s anlegen: %v", name, err)
		}
		ids[name] = id
	}

	// Der Export nennt nur noch „bleibt"; „geht" wird Abgänger.
	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	res, err := s.berechneLUSDAenderungen(ctx, []lusd.Zeile{
		{LUSDID: "L-VM-bleibt", Vorname: "bleibt", Nachname: "Vormerkung", Klasse: "10a", Nr: 1},
	}, true, true)
	if err != nil {
		t.Fatalf("s.berechneLUSDAenderungen(…) = %v", err)
	}
	if len(res.Abgaenger) != 1 {
		t.Fatalf("Testaufbau: %d Abgänger, erwartet 1 (geht)", len(res.Abgaenger))
	}

	for name, soll := range map[string]int{"geht": 0, "bleibt": 1} {
		if n := pgtest.ZaehleZeilen(t, pool,
			`SELECT count(*) FROM vormerkungen WHERE schueler_id = $1 AND status = 'wartend'`, ids[name]); n != soll {
			t.Errorf("%s: %d wartende Vormerkungen nach dem Abgleich, erwartet %d", name, n, soll)
		}
	}
}

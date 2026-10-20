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
	"bibliothek/repository"
)

// Ein Schüler im Papierkorb gehört nicht zum Bestand des Abgleichs. Nennt der Export ihn nicht,
// wird er kein Abgänger: Sperre und Anonymisierung des Abgangs erreichen die gelöschte Zeile
// nicht. Nennt der Export seine LUSD-ID, entsteht ein frischer Datensatz, und die Zeile im
// Papierkorb bleibt, wie sie ist; der eindeutige Index der LUSD-ID gilt nur außerhalb des
// Papierkorbs.
func TestLUSDImport_PapierkorbGehoertNichtZumBestand(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()
	leserAbfragen := repository.NewLeserAbfragen(pool)
	bearbeiter := testdaten.LegeBearbeiterAn(t, pool)

	ids := map[string]string{}
	for i, name := range []string{"aktiv", "zurueck", "weg"} {
		var id string
		if err := pool.QueryRow(ctx,
			`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, lusd_id)
			 VALUES ($1, $2, 'Papierkorb', '07A', 2032, $3) RETURNING id`,
			fmt.Sprintf("PK-%d", i), name, "L-PK-"+name).Scan(&id); err != nil {
			t.Fatalf("Schüler %s anlegen: %v", name, err)
		}
		ids[name] = id
	}
	for _, name := range []string{"zurueck", "weg"} {
		if err := leserAbfragen.LegeLeserInPapierkorb(ctx, ids[name], bearbeiter, "Testaufbau"); err != nil {
			t.Fatalf("Schüler %s in den Papierkorb legen: %v", name, err)
		}
	}

	s := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	res, err := s.berechneLUSDAenderungen(ctx, []lusd.Zeile{
		{LUSDID: "L-PK-aktiv", Vorname: "aktiv", Nachname: "Papierkorb", Klasse: "08A", Nr: 1},
		{LUSDID: "L-PK-zurueck", Vorname: "zurueck", Nachname: "Papierkorb", Klasse: "08B", Nr: 2},
	}, true, true)
	if err != nil {
		t.Fatalf("s.berechneLUSDAenderungen(…) = %v", err)
	}
	if res.AktiveImBestand != 1 || len(res.Abgaenger) != 0 || len(res.Neuzugaenge) != 1 {
		t.Errorf("Vorschau: %d aktive im Bestand, %d Abgänger, %d Neuzugänge — erwartet 1, 0 und 1 (zurueck)",
			res.AktiveImBestand, len(res.Abgaenger), len(res.Neuzugaenge))
	}

	// Beide Zeilen im Papierkorb sind unberührt: weiter gelöscht, kein Abgänger, alte Klasse.
	for _, name := range []string{"zurueck", "weg"} {
		var klasse string
		var geloescht, abgaenger bool
		if err := pool.QueryRow(ctx,
			`SELECT klasse, deleted_at IS NOT NULL, ist_abgaenger OR abgaenger_seit IS NOT NULL FROM leser WHERE id = $1`,
			ids[name]).Scan(&klasse, &geloescht, &abgaenger); err != nil {
			t.Fatalf("%s nach dem Abgleich lesen: %v", name, err)
		}
		if klasse != "07A" || !geloescht || abgaenger {
			t.Errorf("%s im Papierkorb: Klasse %q, gelöscht %v, Abgänger %v — erwartet 07A, true, false",
				name, klasse, geloescht, abgaenger)
		}
	}
	// Die LUSD-ID aus dem Export trägt jetzt eine neue Zeile außerhalb des Papierkorbs.
	var neueKlasse string
	if err := pool.QueryRow(ctx,
		`SELECT klasse FROM schueler WHERE lusd_id = 'L-PK-zurueck' AND deleted_at IS NULL AND id <> $1`,
		ids["zurueck"]).Scan(&neueKlasse); err != nil {
		t.Fatalf("frischer Datensatz zu L-PK-zurueck: %v", err)
	}
	if neueKlasse != "08B" {
		t.Errorf("frischer Datensatz zu L-PK-zurueck in Klasse %q, erwartet 08B", neueKlasse)
	}
}

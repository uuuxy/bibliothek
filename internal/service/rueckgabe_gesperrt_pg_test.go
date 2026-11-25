package service

import (
	"context"
	"testing"
)

// Ein Buch, das während der Ausleihe gesperrt wurde (etwa „Einband lose“), kommt bei der Rückgabe
// nicht ins Abholfach: Es bleibt gesperrt, und die Vormerkung wartet weiter wie beim Nachrücken
// (bedieneNaechstenWartenden). Sonst legte die Theke ein beschädigtes Buch für den Nächsten
// bereit, und der Scan beim Abholen höbe die Sperre still auf.
func TestRueckgabe_GesperrtesBuchKommtNichtInsAbholfach(t *testing.T) {
	for _, fall := range []struct {
		name      string
		amLeser   bool
		titelname string
	}{
		{"Rückgabe am geöffneten Leser", true, "GesperrtLeser"},
		{"Rückgabe ohne geöffneten Leser", false, "GesperrtOhne"},
	} {
		t.Run(fall.name, func(t *testing.T) {
			v := newVormerkTheke(t)
			ctx := context.Background()
			titelID, nummern := v.titel(t, fall.titelname, "Lose")
			nummer := nummern["Lose"]
			ausleiher := v.leser(t, "Ausleiher"+fall.titelname, "schueler")
			if res, err := v.leihe(t, nummer, ausleiher); err != nil || res.Art != "ausleihe" {
				t.Fatalf("Ausleihe: %+v, Fehler %v", res, err)
			}
			if _, err := v.pool.Exec(ctx, `UPDATE buecher_exemplare SET ist_ausleihbar = false, zustand_notiz = 'Einband lose'
				WHERE barcode_id = $1`, nummer); err != nil {
				t.Fatalf("Exemplar sperren: %v", err)
			}
			wartend := v.leser(t, "Wartend"+fall.titelname, "schueler")
			if _, err := v.pool.Exec(ctx, `INSERT INTO vormerkungen (titel_id, schueler_id) VALUES ($1, $2)`, titelID, wartend); err != nil {
				t.Fatalf("wartende Vormerkung: %v", err)
			}

			var res *AusleihErgebnis
			var err error
			if fall.amLeser {
				res, err = v.leihe(t, nummer, ausleiher)
			} else {
				exemplar, ladeErr := v.buecher.FindeExemplar(ctx, nummer)
				if ladeErr != nil || exemplar == nil {
					t.Fatalf("Exemplar laden: %v", ladeErr)
				}
				res, err = v.dienst.NimmZurueck(ctx, exemplar, v.mitarbeiter)
			}
			if err != nil {
				t.Fatalf("Rückgabe: %v", err)
			}
			if res.HatVormerkung || res.VorgemerktFuer != "" {
				t.Errorf("die Theke meldet das gesperrte Buch fürs Abholfach: %+v", res)
			}

			var status string
			var bereit *string
			if err := v.pool.QueryRow(ctx, `SELECT status, bereitgestellt_exemplar_id::text FROM vormerkungen
				WHERE titel_id = $1 AND schueler_id = $2`, titelID, wartend).Scan(&status, &bereit); err != nil {
				t.Fatalf("Vormerkung lesen: %v", err)
			}
			if status != "wartend" || bereit != nil {
				t.Errorf("Vormerkung %s, Exemplar bereitgelegt: %t; erwartet wartend ohne Exemplar", status, bereit != nil)
			}
			var ausleihbar bool
			if err := v.pool.QueryRow(ctx, `SELECT ist_ausleihbar FROM buecher_exemplare WHERE barcode_id = $1`, nummer).Scan(&ausleihbar); err != nil {
				t.Fatalf("Exemplar lesen: %v", err)
			}
			if ausleihbar {
				t.Error("die Rückgabe hat die Sperre aufgehoben")
			}
		})
	}
}

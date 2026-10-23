package service

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"bibliothek/internal/pgtest"
)

// Der Bestands-Import gibt einem Exemplar mit LMF-Nummer eine kurze Nummer und legt die
// LMF-Nummer nach lmf_barcode, mit großer Vorsilbe wie vom Scanner. Nennt die Datei dieselbe
// LMF-Nummer zweimal, entsteht ein Exemplar. Ein zweiter Lauf derselben Datei legt nichts an und
// zieht keine Nummer.
func TestImportiereZeilen_LMFNummerWirdAltesEtikett(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const titel = "LMF-Import-Probe Biologie"
	alteEtiketten := []string{"LMF-9780000562098-001", "LMF-9780000562098-002"}
	raeume := func() {
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE titel = $1)`,
			`DELETE FROM buecher_titel WHERE titel = $1`,
		} {
			if _, err := pool.Exec(context.Background(), sql, titel); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	}
	raeume()
	t.Cleanup(raeume)

	kopf := map[string]int{"titel": 0, "isbn": 1, "barcode": 2}
	datei := [][]string{
		{"Titel", "ISBN", "Barcode"},
		{titel, "9780000562098", alteEtiketten[0]},
		{titel, "9780000562098", "lmf-9780000562098-002"},
		{titel, "9780000562098", alteEtiketten[0]},
		{titel, "9780000562098", "IMPORT-LMF-PROBE-1"},
	}
	dienst := NewImportDienst(nil, pool)
	if _, neu, err := dienst.ImportiereZeilen(ctx, datei, kopf); err != nil || neu != 3 {
		t.Fatalf("erster Lauf: %d neue Exemplare, Fehler %v; erwartet 3", neu, err)
	}

	kurz := regexp.MustCompile(`^B-[0-9]+$`)
	for _, alt := range alteEtiketten {
		var nummer string
		var gedruckt bool
		if err := pool.QueryRow(ctx, `SELECT barcode_id, etikett_gedruckt FROM buecher_exemplare WHERE lmf_barcode = $1`, alt).
			Scan(&nummer, &gedruckt); err != nil {
			t.Fatalf("Exemplar mit altem Etikett %s: %v", alt, err)
		}
		if !kurz.MatchString(nummer) || !gedruckt {
			t.Errorf("Exemplar mit altem Etikett %s trägt %q, gedruckt %v; erwartet eine kurze Nummer und ein Etikett", alt, nummer, gedruckt)
		}
	}
	var mitNummer int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM buecher_exemplare WHERE barcode_id = 'IMPORT-LMF-PROBE-1' AND lmf_barcode IS NULL`).
		Scan(&mitNummer); err != nil || mitNummer != 1 {
		t.Errorf("Exemplar ohne LMF-Nummer: %d mit seiner Nummer (%v), erwartet 1", mitNummer, err)
	}

	var standVorher, standNachher int64
	if err := pool.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&standVorher); err != nil {
		t.Fatal(err)
	}
	if _, neu, err := dienst.ImportiereZeilen(ctx, datei, kopf); err != nil || neu != 0 {
		t.Errorf("zweiter Lauf: %d neue Exemplare, Fehler %v; erwartet keins", neu, err)
	}
	if err := pool.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&standNachher); err != nil {
		t.Fatal(err)
	}
	if standNachher != standVorher {
		t.Errorf("zweiter Lauf zog Nummern: barcode_seq von %d auf %d", standVorher, standNachher)
	}
}

// Nennt die Datei neben einem LMF-Exemplar ein Buch mit einer kurzen Nummer, die noch kein
// Exemplar trägt, bekommt das LMF-Exemplar nicht dieselbe: Sein Etikett führte sonst zum falschen
// Buch, und das andere fiele still als „vorhanden" weg oder der ganze Import ab. Geprüft in beiden
// Reihenfolgen der Zeilen, mit der Nummer, die barcode_seq als nächste ausgäbe.
func TestImportiereZeilen_KurzeNummerTrifftKeineNummerDerDatei(t *testing.T) {
	pool := pgtest.Pool(t)
	ctx := context.Background()
	const titel = "Import-Nummernprobe"
	raeume := func() {
		for _, sql := range []string{
			`DELETE FROM buecher_exemplare WHERE titel_id IN (SELECT id FROM buecher_titel WHERE titel LIKE $1 || '%')`,
			`DELETE FROM buecher_titel WHERE titel LIKE $1 || '%'`,
		} {
			if _, err := pool.Exec(context.Background(), sql, titel); err != nil {
				t.Errorf("aufräumen: %v", err)
			}
		}
	}
	raeume()
	t.Cleanup(raeume)

	kopf := map[string]int{"titel": 0, "isbn": 1, "barcode": 2}
	dienst := NewImportDienst(nil, pool)
	for _, fall := range []struct {
		name      string
		lmfZuerst bool
	}{{"LMF-Zeile zuerst", true}, {"LMF-Zeile danach", false}} {
		t.Run(fall.name, func(t *testing.T) {
			var stand int64
			if err := pool.QueryRow(ctx, `SELECT last_value FROM barcode_seq`).Scan(&stand); err != nil {
				t.Fatal(err)
			}
			naechste := fmt.Sprintf("B-%05d", stand+1)
			lmf := fmt.Sprintf("LMF-9780000562098-%03d", stand%1000)
			zeileLMF := []string{titel + " Lernmittel", "9780000562098", lmf}
			zeileBuch := []string{titel + " Roman " + naechste, "", naechste}
			datei := [][]string{{"Titel", "ISBN", "Barcode"}, zeileBuch, zeileLMF}
			if fall.lmfZuerst {
				datei = [][]string{{"Titel", "ISBN", "Barcode"}, zeileLMF, zeileBuch}
			}

			if _, neu, err := dienst.ImportiereZeilen(ctx, datei, kopf); err != nil || neu != 2 {
				t.Fatalf("%d neue Exemplare, Fehler %v; erwartet beide Bücher der Datei", neu, err)
			}
			var buchTitel, lmfNummer string
			if err := pool.QueryRow(ctx, `SELECT t.titel FROM buecher_exemplare e
				JOIN buecher_titel t ON t.id = e.titel_id WHERE e.barcode_id = $1`, naechste).Scan(&buchTitel); err != nil {
				t.Fatalf("Exemplar %s: %v", naechste, err)
			}
			if buchTitel != zeileBuch[0] {
				t.Errorf("das Etikett %s führt zu %q, erwartet %q", naechste, buchTitel, zeileBuch[0])
			}
			if err := pool.QueryRow(ctx, `SELECT barcode_id FROM buecher_exemplare WHERE lmf_barcode = $1`, lmf).
				Scan(&lmfNummer); err != nil {
				t.Fatalf("Exemplar mit altem Etikett %s: %v", lmf, err)
			}
			if lmfNummer == naechste {
				t.Errorf("das LMF-Exemplar trägt %s, die Nummer des Buchs aus der Datei", lmfNummer)
			}
		})
	}
}

package katalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Die Titel-Historie zeigt die befristete Lesehistorie am deutlichsten
// (jobs/cron_dsgvo_lesehistorie.go): Nach der Trennung steht dort kein Schüler mehr, und vor
// allem kein falscher. Eine Ausleihe ohne Schüler ist keine Ausleihe einer Lehrkraft; mit der
// Befristung wäre sonst jede getrennte Ausleihe eine. Erwartung: getrennte Ausleihe eines
// Schülers → „Anonym" ohne Klasse; Ausleihe einer Lehrkraft → das Wort ihrer Art, „Lehrkraft";
// zugeordnete Ausleihe eines Schülers → Name und Klasse.
func TestTitelHistorie_GetrennteAusleiheIstAnonymNichtLehrkraft(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	var titelID, exemplarID, schuelerID, lehrerID string
	titelID = testdaten.LegeTitelAn(t, pool, "Historie-Titel", 5)
	exemplarID = testdaten.LegeExemplarAn(t, pool, titelID, "B-HIST-1", true, "")
	schuelerID = testdaten.LegeSchuelerMitNamenAn(t, pool, "S-HIST", "Mia", "Muster", "7a", 2030)
	lehrerID = testdaten.LegeKontoAn(t, pool, "Lena", "Lehr", "lehr-hist@example.org", "kollegium", true)
	// Drei abgeschlossene Vorgänge desselben Exemplars, in dieser Reihenfolge (neueste zuerst
	// in der Antwort): zugeordnet (vor 3 Tagen), getrennt (vor 200 Tagen), Lehrer (vor 400).
	for _, v := range []struct {
		schueler, lehrer *string
		tage             int
	}{
		{&schuelerID, nil, 3},
		{nil, nil, 200},
		{nil, &lehrerID, 400},
	} {
		if _, err := pool.Exec(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist, rueckgabe_am)
			VALUES ($1, coalesce($2, (SELECT leser_id FROM benutzer WHERE id = $3)), now() - make_interval(days => $4 + 20), now() - make_interval(days => $4 + 1), now() - make_interval(days => $4))`,
			exemplarID, v.schueler, v.lehrer, v.tage); err != nil {
			t.Fatalf("Ausleihe (%d Tage): %v", v.tage, err)
		}
	}

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	req := httptest.NewRequest(http.MethodGet, "/api/buecher/titel/"+titelID+"/historie", nil)
	req.SetPathValue("id", titelID)
	rec := httptest.NewRecorder()
	srv.gibTitelHistorie(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, erwartet 200: %s", rec.Code, rec.Body.String())
	}
	var got []historieZeile
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Antwort: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Historie: %d Einträge, erwartet 3: %+v", len(got), got)
	}
	if got[0].Vorname != "Mia" || got[0].Klasse != "07A" {
		t.Errorf("zugeordnete Ausleihe: %+v", got[0])
	}
	if got[1].Vorname != "Anonym" || got[1].Nachname != "" || got[1].Klasse != "" {
		t.Errorf("getrennte Ausleihe muss anonym OHNE Klasse sein (nicht 'Lehrer'): %+v", got[1])
	}
	if got[2].Vorname != "Lena" || got[2].Klasse != "Lehrkraft" {
		t.Errorf("Lehrer-Ausleihe: %+v", got[2])
	}
}

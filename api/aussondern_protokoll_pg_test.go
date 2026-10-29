package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"bibliothek/internal/pgtest"
	"bibliothek/internal/routertest"
	"bibliothek/internal/testdaten"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Fünf Wege sondern ein Exemplar aus, ohne es zu löschen: der Status in der Buchakte, ein
// kleinerer Bestand in „Buch bearbeiten", der Abschluss einer Inventur, „Verlust oder Schaden
// melden" in der Leserakte und der Verlust im Schadensersatz-Bescheid. Das Abgangsbuch nennt
// Datum und Grund; wer es war, steht im Protokoll (docs/invarianten.md, Frage 19): Bearbeiter,
// Weg und Grund, ohne die Zustandsnotiz. Was nichts aussondert oder abgelehnt wird, schreibt
// keinen Eintrag. Jeder Weg läuft über den ganzen Router: Die Person kommt aus der Sitzung,
// nicht aus dem Test.

// aussonderungsEintrag ist ein Protokolleintrag zu einem ausgesonderten Exemplar.
type aussonderungsEintrag struct {
	bearbeiter, kontext, roh string
	details                  map[string]any
}

// aussonderungsSpur liest die Einträge, die das Aussondern zu einem Exemplar geschrieben hat.
func aussonderungsSpur(t *testing.T, pool *pgxpool.Pool, exemplarID string) []aussonderungsEintrag {
	t.Helper()
	rows, err := pool.Query(t.Context(), `
		SELECT coalesce(bearbeiter_id::text, ''), coalesce(kontext, ''), details::text
		FROM audit_log
		WHERE tabelle = 'buecher_exemplare' AND aktion = 'UPDATE' AND datensatz_id = $1
		  AND details->>'action' = $2
		ORDER BY timestamp`, exemplarID, repository.ProtokollAktionAusgesondert)
	if err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	defer rows.Close()
	var liste []aussonderungsEintrag
	for rows.Next() {
		var e aussonderungsEintrag
		if err := rows.Scan(&e.bearbeiter, &e.kontext, &e.roh); err != nil {
			t.Fatalf("Protokoll lesen: %v", err)
		}
		if err := json.Unmarshal([]byte(e.roh), &e.details); err != nil {
			t.Fatalf("Details unlesbar: %v", err)
		}
		liste = append(liste, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Protokoll lesen: %v", err)
	}
	return liste
}

// pruefeAussonderungsEintrag verlangt genau einen Eintrag mit Person, Weg und Grund.
func pruefeAussonderungsEintrag(t *testing.T, pool *pgxpool.Pool, exemplarID, bearbeiter, weg, grund string) aussonderungsEintrag {
	t.Helper()
	spur := aussonderungsSpur(t, pool, exemplarID)
	if len(spur) != 1 {
		t.Fatalf("%d Einträge im Protokoll, erwartet 1", len(spur))
	}
	e := spur[0]
	if e.bearbeiter != bearbeiter {
		t.Errorf("Bearbeiter %q, erwartet %q", e.bearbeiter, bearbeiter)
	}
	if e.kontext != weg {
		t.Errorf("Weg %q, erwartet %q", e.kontext, weg)
	}
	if e.details["grund"] != grund {
		t.Errorf("Grund %v, erwartet %q", e.details["grund"], grund)
	}
	return e
}

func TestAussondern_StehtMitDerPersonImProtokoll(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := t.Context()
	adminID, rufe := routertest.MitAufruf(t, pool)
	t.Cleanup(func() {
		pgtest.RaeumeAuf(t, pool, `DELETE FROM audit_log WHERE details->>'action' = $1`, repository.ProtokollAktionAusgesondert)
	})

	t.Run("Status in der Buchakte", func(t *testing.T) {
		titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Spur Status", "Spu 1", 0)
		ex := testdaten.LegeExemplarAn(t, pool, titelID, "SPUR-STATUS-1", true, "")
		status := func(rumpf string) int {
			return rufe(t, http.MethodPut, "/api/buecher/exemplare/"+ex+"/status", rumpf).Code
		}

		if code := status(`{"ist_ausleihbar":false,"ist_ausgesondert":false,"zustand_notiz":"zurückgelegt"}`); code != http.StatusOK {
			t.Fatalf("sperren: Status %d", code)
		}
		if spur := aussonderungsSpur(t, pool, ex); len(spur) != 0 {
			t.Fatalf("Sperren sondert nicht aus und schreibt %d Einträge", len(spur))
		}

		const notiz = "von Mia Beispiel im Bus verloren"
		if code := status(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"` + notiz + `"}`); code != http.StatusOK {
			t.Fatalf("aussondern: Status %d", code)
		}
		e := pruefeAussonderungsEintrag(t, pool, ex, adminID, repository.AussonderungsWegStatus, "VERLUST")
		if strings.Contains(e.roh, "Mia") {
			t.Errorf("der Eintrag trägt die Zustandsnotiz: %s", e.roh)
		}

		// Dasselbe Exemplar mit geänderter Notiz: kein zweiter Wechsel, kein zweiter Eintrag.
		if code := status(`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":"im Bus verloren"}`); code != http.StatusOK {
			t.Fatalf("Notiz ändern: Status %d", code)
		}
		pruefeAussonderungsEintrag(t, pool, ex, adminID, repository.AussonderungsWegStatus, "VERLUST")

		// Ein verliehenes Exemplar lehnt die Tür ab; die Ablehnung schreibt nichts.
		verliehen := testdaten.LegeExemplarAn(t, pool, titelID, "SPUR-STATUS-2", true, "")
		testdaten.LegeAusleiheVorZweiTagenAn(t, pool, verliehen, testdaten.LegeSchuelerAn(t, pool, "S-SPUR-1", "Mia", "5a"))
		rec := rufe(t, http.MethodPut, "/api/buecher/exemplare/"+verliehen+"/status",
			`{"ist_ausleihbar":false,"ist_ausgesondert":true,"zustand_notiz":""}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("verliehenes Exemplar aussondern: Status %d, erwartet 400: %s", rec.Code, rec.Body.String())
		}
		if spur := aussonderungsSpur(t, pool, verliehen); len(spur) != 0 {
			t.Errorf("eine abgelehnte Aussonderung schreibt %d Einträge", len(spur))
		}
	})

	t.Run("kleinerer Bestand in Buch bearbeiten", func(t *testing.T) {
		titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Spur Bestand", "Spu 2", 0)
		for i := 1; i <= 3; i++ {
			testdaten.LegeExemplarAn(t, pool, titelID, fmt.Sprintf("SPUR-BESTAND-%d", i), true, "")
		}
		bestand := func(rumpf string) {
			t.Helper()
			if rec := rufe(t, http.MethodPut, "/api/books/"+titelID, rumpf); rec.Code != http.StatusOK {
				t.Fatalf("%s: Status %d: %s", rumpf, rec.Code, rec.Body.String())
			}
		}
		// jeExemplar zählt die Einträge je Exemplar des Titels, getrennt nach ausgesondert.
		jeExemplar := func() (ausgesondertMitEintrag, ausgesondert, imBestandMitEintrag int) {
			t.Helper()
			rows, err := pool.Query(ctx,
				`SELECT id::text, ist_ausgesondert FROM buecher_exemplare WHERE titel_id = $1`, titelID)
			if err != nil {
				t.Fatalf("Exemplare lesen: %v", err)
			}
			lage := map[string]bool{}
			for rows.Next() {
				var id string
				var aus bool
				if err := rows.Scan(&id, &aus); err != nil {
					t.Fatalf("Exemplare lesen: %v", err)
				}
				lage[id] = aus
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				t.Fatalf("Exemplare lesen: %v", err)
			}
			for id, aus := range lage {
				switch {
				case aus:
					ausgesondert++
					pruefeAussonderungsEintrag(t, pool, id, adminID, repository.AussonderungsWegBestandskorrektur, "BESTANDSKORREKTUR")
					ausgesondertMitEintrag++
				case len(aussonderungsSpur(t, pool, id)) > 0:
					imBestandMitEintrag++
				}
			}
			return ausgesondertMitEintrag, ausgesondert, imBestandMitEintrag
		}

		bestand(`{"stock":1,"stockGesehen":3}`)
		if mit, aus, fremd := jeExemplar(); aus != 2 || mit != 2 || fremd != 0 {
			t.Fatalf("von 3 auf 1: %d ausgesondert, %d davon mit Eintrag, %d Einträge an Exemplaren im Bestand — erwartet 2, 2 und 0", aus, mit, fremd)
		}

		// Ein größerer Bestand legt Exemplare an und sondert nichts aus.
		bestand(`{"stock":2,"stockGesehen":1}`)
		if mit, aus, fremd := jeExemplar(); aus != 2 || mit != 2 || fremd != 0 {
			t.Errorf("von 1 auf 2: %d ausgesondert, %d davon mit Eintrag, %d Einträge an Exemplaren im Bestand — erwartet 2, 2 und 0", aus, mit, fremd)
		}
	})

	t.Run("Abschluss einer Inventur", func(t *testing.T) {
		titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Spur Inventur", "SpuInv 1", 0)
		gezaehlt := testdaten.LegeExemplarAn(t, pool, titelID, "SPUR-INV-1", true, "")
		fehlt := testdaten.LegeExemplarAn(t, pool, titelID, "SPUR-INV-2", true, "")
		signatur := "SpuInv"
		invRepo := repository.NewInventurAbfragen(pool)
		session, err := invRepo.LegeLaufAn(ctx, "signature",
			repository.InventurBereich{Signatur: &signatur}, "SpuInv", "")
		if err != nil {
			t.Fatalf("Inventur beginnen: %v", err)
		}
		if _, err := invRepo.VerbucheScan(ctx, session.ID, session.Bereich(), gezaehlt); err != nil {
			t.Fatalf("Exemplar zählen: %v", err)
		}

		rec := rufe(t, http.MethodPost, "/api/inventur/finish", `{"session_id":"`+session.ID+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("Inventur abschließen: Status %d: %s", rec.Code, rec.Body.String())
		}
		e := pruefeAussonderungsEintrag(t, pool, fehlt, adminID, repository.AussonderungsWegInventur, "VERLUST")
		if e.details["inventur_session_id"] != session.ID {
			t.Errorf("der Eintrag nennt die Inventur %v, erwartet %q", e.details["inventur_session_id"], session.ID)
		}
		if spur := aussonderungsSpur(t, pool, gezaehlt); len(spur) != 0 {
			t.Errorf("das gezählte Exemplar trägt %d Einträge", len(spur))
		}
	})

	// Die Ausleihe nennt, wer sie beendet hat, nur bis BearbeiterKennungTage nach der Rückgabe,
	// und nach einer Rückgabe an der Theke nennt sie die Theke, nicht die Meldung.
	t.Run("Verlust oder Schaden in der Leserakte", func(t *testing.T) {
		titelID := testdaten.LegeTitelMitSignaturAn(t, pool, "Spur Schaden", "Spu 3", 0)
		// melde verleiht ein neues Exemplar an den Leser und meldet den Schaden zweimal. vorher
		// läuft vor der Meldung an der Ausleihe, etwa eine Rückgabe an einem anderen Platz.
		melde := func(barcode, leserID string, vorher func(ausleiheID string)) (exemplarID, schadensID string) {
			t.Helper()
			exemplarID = testdaten.LegeExemplarAn(t, pool, titelID, barcode, true, "")
			testdaten.LegeAusleiheVorZweiTagenAn(t, pool, exemplarID, leserID)
			var ausleiheID string
			if err := pool.QueryRow(ctx, `SELECT id::text FROM ausleihen WHERE exemplar_id = $1`, exemplarID).Scan(&ausleiheID); err != nil {
				t.Fatalf("Ausleihe lesen: %v", err)
			}
			if vorher != nil {
				vorher(ausleiheID)
			}
			rumpf := `{"loan_id":"` + ausleiheID + `","art":"beschaedigt","beschreibung":"von Mia Beispiel zerrissen","betrag":12.50}`
			for range 2 {
				rec := rufe(t, http.MethodPost, "/api/damage/report", rumpf)
				if rec.Code != http.StatusOK {
					t.Fatalf("Schaden melden: Status %d: %s", rec.Code, rec.Body.String())
				}
				var antwort struct {
					SchadensID string `json:"schadens_id"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &antwort); err != nil {
					t.Fatalf("Antwort unlesbar: %v", err)
				}
				schadensID = antwort.SchadensID
			}
			return exemplarID, schadensID
		}
		// pruefe verlangt den einen Eintrag, auch nach dem zweiten Klick, mit der Forderung
		// und ohne die Beschreibung.
		pruefe := func(exemplarID, schadensID string) {
			t.Helper()
			e := pruefeAussonderungsEintrag(t, pool, exemplarID, adminID, repository.AussonderungsWegSchaden, "BESCHAEDIGUNG")
			var forderung any
			if schadensID != "" {
				forderung = schadensID
			}
			if e.details["schadensfall_id"] != forderung {
				t.Errorf("der Eintrag nennt die Forderung %v, erwartet %v", e.details["schadensfall_id"], forderung)
			}
			if strings.Contains(e.roh, "Mia") {
				t.Errorf("der Eintrag trägt die Beschreibung: %s", e.roh)
			}
		}

		schueler := testdaten.LegeSchuelerAn(t, pool, "S-SPUR-2", "Mia", "5a")
		ex, schaden := melde("SPUR-SCHADEN-1", schueler, nil)
		if schaden == "" {
			t.Fatal("die Meldung für einen Schüler legt keine Forderung an")
		}
		pruefe(ex, schaden)

		theke := testdaten.LegeBearbeiterAn(t, pool)
		ex, schaden = melde("SPUR-SCHADEN-2", schueler, func(ausleiheID string) {
			if _, err := pool.Exec(ctx, `
				UPDATE ausleihen SET rueckgabe_am = NOW(), rueckgabe_bearbeiter_id = $2 WHERE id = $1`,
				ausleiheID, theke); err != nil {
				t.Fatalf("Rückgabe an der Theke: %v", err)
			}
		})
		pruefe(ex, schaden)

		// Beim Kollegium entsteht keine Forderung; der Eintrag nennt dann auch keine.
		kollege := testdaten.LegeLeserAn(t, pool, "", "Karl", "Spur", "lehrkraft")
		ex, schaden = melde("SPUR-SCHADEN-3", kollege, nil)
		if schaden != "" {
			t.Fatalf("die Meldung für einen Kollegen legt die Forderung %q an", schaden)
		}
		pruefe(ex, "")
	})

	t.Run("Verlust im Schadensersatz-Bescheid", func(t *testing.T) {
		testdaten.LeereBescheide(t, pool)
		testdaten.SetzeBescheidAngaben(t, pool)
		schueler := testdaten.LegeSchuelerAn(t, pool, "S-SPUR-3", "Ben", "8a")
		ex := testdaten.LegeExemplarAn(t, pool, testdaten.LegeLernmittelAn(t, pool, "Spur Bescheid"), "SPUR-BESCHEID-1", true, "")
		var ausleiheID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO ausleihen (exemplar_id, schueler_id, ausgeliehen_am, rueckgabe_frist)
			VALUES ($1, $2, CURRENT_DATE - 60, CURRENT_DATE - 30) RETURNING id`, ex, schueler).Scan(&ausleiheID); err != nil {
			t.Fatalf("Ausleihe anlegen: %v", err)
		}

		rec := rufe(t, http.MethodPost, "/api/schueler/"+schueler+"/bescheide", fmt.Sprintf(
			`{"mittel":"land","frist_bis":%q,"positionen":[],"ausleihen":[{"ausleihe_id":%q,"betrag":12.00}]}`,
			testdaten.In28Tagen(), ausleiheID))
		if rec.Code != http.StatusCreated {
			t.Fatalf("Bescheid erstellen: Status %d: %s", rec.Code, rec.Body.String())
		}
		var forderung string
		if err := pool.QueryRow(ctx, `SELECT id::text FROM schadensfaelle WHERE ausleihe_id = $1`, ausleiheID).Scan(&forderung); err != nil {
			t.Fatalf("Forderung lesen: %v", err)
		}
		e := pruefeAussonderungsEintrag(t, pool, ex, adminID, repository.AussonderungsWegBescheid, "VERLUST")
		if e.details["schadensfall_id"] != forderung {
			t.Errorf("der Eintrag nennt die Forderung %v, erwartet %q", e.details["schadensfall_id"], forderung)
		}
	})
}

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Drei Wege sondern ein Exemplar aus, ohne es zu löschen: der Status in der Buchakte, ein
// kleinerer Bestand in „Buch bearbeiten" und der Abschluss einer Inventur. Das Abgangsbuch
// nennt Datum und Grund; wer es war, steht im Protokoll (docs/invarianten.md, Frage 19):
// Bearbeiter, Weg und Grund, ohne die Zustandsnotiz. Was nichts aussondert oder abgelehnt
// wird, schreibt keinen Eintrag. Jeder Weg läuft über den ganzen Router: Die Person kommt aus
// der Sitzung, nicht aus dem Test.

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
		ORDER BY timestamp`, exemplarID, repository.AuditAktionAusgesondert)
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
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := t.Context()
	adminID, rufe := protokollWelt(t, pool)
	t.Cleanup(func() {
		aufraeumen(t, pool, `DELETE FROM audit_log WHERE details->>'action' = $1`, repository.AuditAktionAusgesondert)
	})

	t.Run("Status in der Buchakte", func(t *testing.T) {
		titelID := titelMitSignatur(t, pool, "Spur Status", "Spu 1", 0)
		ex := exemplar(t, pool, titelID, "SPUR-STATUS-1", true, "")
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
		verliehen := exemplar(t, pool, titelID, "SPUR-STATUS-2", true, "")
		seedLeserAusleihe(t, pool, verliehen, seedSchueler(t, pool, "S-SPUR-1", "Mia", "5a"))
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
		titelID := titelMitSignatur(t, pool, "Spur Bestand", "Spu 2", 0)
		for i := 1; i <= 3; i++ {
			exemplar(t, pool, titelID, fmt.Sprintf("SPUR-BESTAND-%d", i), true, "")
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
		titelID := titelMitSignatur(t, pool, "Spur Inventur", "SpuInv 1", 0)
		gezaehlt := exemplar(t, pool, titelID, "SPUR-INV-1", true, "")
		fehlt := exemplar(t, pool, titelID, "SPUR-INV-2", true, "")
		signatur := "SpuInv"
		invRepo := repository.NewInventoryRepository(pool)
		session, err := invRepo.CreateInventurSession(ctx, "signature",
			repository.InventurScope{Signatur: &signatur}, "SpuInv", "")
		if err != nil {
			t.Fatalf("Inventur beginnen: %v", err)
		}
		if err := invRepo.RecordInventurScan(ctx, session.ID, gezaehlt); err != nil {
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
}

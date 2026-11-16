package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LadeAbgeschlosseneLaeufe speist die Liste, über die der Fehlbestandsbericht einer
// abgeschlossenen Inventur wieder aufrufbar wird. Ihre Abfrage zählt über CTE und LATERAL; ein
// vertauschtes Paar von Zählungen oder eine gedrehte Sortierung fiele sonst nicht auf: Beide
// Spalten sind Zahlen, beide Sortierrichtungen liefern Zeilen.
func TestLadeAbgeschlosseneLaeufe(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	// Drei Inventuren mit verschiedenen Zahlen für Erfassung und Verlust; sonst belegte der
	// Test nicht, dass die beiden Zählungen an der richtigen Spalte hängen. Je Bereich: n
	// Exemplare, davon k gescannt; der Rest wird beim Abschluss Verlust.
	faelle := []struct {
		signatur          string
		exemplare, gescan int
	}{
		{"Erdkunde", 6, 5}, // 5 erfasst, 1 Verlust
		{"Physik", 5, 2},   // 2 erfasst, 3 Verluste
		{"Chemie", 4, 4},   // 4 erfasst, 0 Verluste
	}

	for _, f := range faelle {
		ex := legeSignaturMitExemplarenAn(t, pool, f.signatur, f.exemplare)
		signatur := f.signatur
		sess, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &signatur}, f.signatur, "")
		if err != nil {
			t.Fatalf("Session %s anlegen: %v", f.signatur, err)
		}
		for _, id := range ex[:f.gescan] {
			if _, err := repo.VerbucheScan(ctx, sess.ID, sess.Bereich(), id); err != nil {
				t.Fatalf("Scan %s: %v", f.signatur, err)
			}
		}
		if _, err := repo.SchliesseLaufAb(ctx, sess.ID, InventurBereich{Signatur: &signatur}, bearbeiter); err != nil {
			t.Fatalf("Abschluss %s: %v", f.signatur, err)
		}
	}

	seite, err := repo.LadeAbgeschlosseneLaeufe(ctx, nil, 10)
	if err != nil {
		t.Fatalf("LadeAbgeschlosseneLaeufe: %v", err)
	}
	sessions := seite.Laeufe
	if len(sessions) != 3 {
		t.Fatalf("Anzahl abgeschlossener Sessions = %d, erwartet 3", len(sessions))
	}

	// Neueste zuerst: Chemie wurde zuletzt abgeschlossen.
	if sessions[0].Bezeichnung != "Chemie" {
		t.Errorf("erste Session = %q, erwartet Chemie (zuletzt abgeschlossen)", sessions[0].Bezeichnung)
	}

	erwartet := map[string]struct{ erfasst, verluste int }{
		"Erdkunde": {5, 1},
		"Physik":   {2, 3},
		"Chemie":   {4, 0},
	}
	for _, s := range sessions {
		will, bekannt := erwartet[s.Bezeichnung]
		if !bekannt {
			t.Errorf("unerwartete Session %q", s.Bezeichnung)
			continue
		}
		if s.Erfasst != will.erfasst {
			t.Errorf("%s: erfasst = %d, erwartet %d", s.Bezeichnung, s.Erfasst, will.erfasst)
		}
		if s.Verluste != will.verluste {
			t.Errorf("%s: verluste = %d, erwartet %d", s.Bezeichnung, s.Verluste, will.verluste)
		}
	}
}

// Eine Seite behält die neuesten Läufe und nennt, wie viele ältere danach kommen; die nächste
// schließt an ihren letzten Lauf an. Die Abfrage kappt vor dem Zählen; nähme die CTE die
// falschen Zeilen, fiele es ohne diesen Test nicht auf.
func TestLadeAbgeschlosseneLaeufe_Seiten(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	ids := schliesseLaeufeAb(t, pool, repo, "Alpha", "Beta", "Gamma", "Delta", "Epsilon")

	erste, err := repo.LadeAbgeschlosseneLaeufe(ctx, nil, 2)
	if err != nil {
		t.Fatalf("erste Seite: %v", err)
	}
	if got := bezeichnungen(erste.Laeufe); got != "Epsilon Delta" || erste.Weitere != 3 {
		t.Fatalf("erste Seite = [%s], %d weitere; erwartet [Epsilon Delta], 3 weitere", got, erste.Weitere)
	}
	if erste.Naechste == nil || erste.Naechste.ID != ids["Delta"] {
		t.Fatalf("Marke der ersten Seite = %+v, erwartet den Lauf Delta", erste.Naechste)
	}

	// Zwischen zwei Seiten verschwindet an einem anderen Arbeitsplatz der letzte Lauf der ersten:
	// Die zweite schließt trotzdem an ihn an, ohne Gamma zu überspringen.
	if _, err := pool.Exec(ctx, `DELETE FROM inventur_sessions WHERE id = $1`, ids["Delta"]); err != nil {
		t.Fatalf("Delta löschen: %v", err)
	}
	zweite, err := repo.LadeAbgeschlosseneLaeufe(ctx, erste.Naechste, 2)
	if err != nil {
		t.Fatalf("zweite Seite: %v", err)
	}
	if got := bezeichnungen(zweite.Laeufe); got != "Gamma Beta" || zweite.Weitere != 1 || zweite.Naechste == nil {
		t.Fatalf("zweite Seite = [%s], %d weitere, Marke %+v; erwartet [Gamma Beta], 1 weiterer", got, zweite.Weitere, zweite.Naechste)
	}

	letzte, err := repo.LadeAbgeschlosseneLaeufe(ctx, zweite.Naechste, 2)
	if err != nil {
		t.Fatalf("letzte Seite: %v", err)
	}
	if got := bezeichnungen(letzte.Laeufe); got != "Alpha" || letzte.Weitere != 0 || letzte.Naechste != nil {
		t.Errorf("letzte Seite = [%s], %d weitere, Marke %+v; erwartet [Alpha], keine weitere, keine Marke", got, letzte.Weitere, letzte.Naechste)
	}
}

// schliesseLaeufeAb legt je Name einen Lauf über eine eigene Signatur an und schließt ihn ab, in
// dieser Reihenfolge. Geliefert werden die Kennungen nach Namen.
func schliesseLaeufeAb(t *testing.T, pool *pgxpool.Pool, repo *InventurAbfragen, namen ...string) map[string]string {
	t.Helper()
	ctx := context.Background()
	bearbeiter := legeBearbeiterAn(t, pool)
	ids := map[string]string{}
	for _, name := range namen {
		legeSignaturMitExemplarenAn(t, pool, name, 2)
		signatur := name
		sess, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &signatur}, name, "")
		if err != nil {
			t.Fatalf("Lauf %s anlegen: %v", name, err)
		}
		if _, err := repo.SchliesseLaufAb(ctx, sess.ID, InventurBereich{Signatur: &signatur}, bearbeiter); err != nil {
			t.Fatalf("Abschluss %s: %v", name, err)
		}
		ids[name] = sess.ID
	}
	return ids
}

func bezeichnungen(laeufe []Inventurlauf) string {
	namen := make([]string, 0, len(laeufe))
	for _, l := range laeufe {
		namen = append(namen, l.Bezeichnung)
	}
	return strings.Join(namen, " ")
}

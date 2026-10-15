package repository

import (
	"context"
	"errors"
	"testing"
)

// Zwei Läufe in verschiedenen Bereichen beeinflussen sich weder im Fortschritt noch beim
// Abschluss: Jede Erfassung hängt an ihrem Lauf.
func TestInventurParallelbetrieb(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	mathe, deutsch := "Mathematik", "Deutsch"
	matheEx := legeSignaturMitExemplarenAn(t, pool, mathe, 5)
	deutschEx := legeSignaturMitExemplarenAn(t, pool, deutsch, 4)

	// Kollege A: Lauf für Mathematik, scannt 3 von 5.
	sessA, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &mathe}, "Mathematik", "")
	if err != nil {
		t.Fatalf("Session A anlegen: %v", err)
	}
	for _, ex := range matheEx[:3] {
		if err := repo.VerbucheScan(ctx, sessA.ID, ex); err != nil {
			t.Fatalf("Scan A: %v", err)
		}
	}

	// Kollege B startet daneben einen Lauf für Deutsch; der Fortschritt von A bleibt.
	sessB, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &deutsch}, "Deutsch", "")
	if err != nil {
		t.Fatalf("Session B anlegen: %v", err)
	}

	aErfasst := erfassungsZahl(t, pool, sessA.ID)
	if aErfasst != 3 {
		t.Errorf("A's Fortschritt nach B's Start: erwartet 3, war %d", aErfasst)
	}

	// A schließt ab: genau die 2 nicht gescannten Mathe-Exemplare gelten als Verlust.
	verloren, err := repo.SchliesseLaufAb(ctx, sessA.ID, InventurBereich{Signatur: &mathe}, bearbeiter)
	if err != nil {
		t.Fatalf("Finish A: %v", err)
	}
	if verloren != 2 {
		t.Errorf("Mathe-Verluste: erwartet 2, war %d", verloren)
	}

	// Der Bestand für Deutsch (der Bereich von B) bleibt unberührt.
	if ausgesondert := ausgesonderteZahl(t, pool, deutschEx); ausgesondert != 0 {
		t.Errorf("Deutsch-Exemplare fälschlich ausgesondert: %d (Session B lief parallel)", ausgesondert)
	}
	_ = sessB
}

// Ein verliehenes Buch ist beim Schüler und nicht verschwunden: Beim Abschluss wird es nicht
// als Verlust gebucht, obwohl es niemand scannen konnte.
func TestInventurAusgelieheneNichtVerloren(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	physik := "Physik"
	ex := legeSignaturMitExemplarenAn(t, pool, physik, 3)
	leiheAus(t, pool, ex[0]) // ein Exemplar verliehen

	// Der Bereich zählt nur die 2, die im Regal stehen.
	scope, err := repo.ZaehleBereich(ctx, InventurBereich{Signatur: &physik})
	if err != nil {
		t.Fatalf("ZaehleBereich: %v", err)
	}
	if scope != 2 {
		t.Errorf("Scope mit 1 verliehenen: erwartet 2, war %d", scope)
	}

	// Lauf starten, nichts scannen, abschließen: Nur die 2 im Regal fehlen. Das verliehene Buch
	// ist nicht darunter.
	sess, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &physik}, "Physik", "")
	if err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}
	verloren, err := repo.SchliesseLaufAb(ctx, sess.ID, InventurBereich{Signatur: &physik}, bearbeiter)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if verloren != 2 {
		t.Errorf("Verluste: erwartet 2 (nur anwesende), war %d — das verliehene Buch wurde fälschlich mitgezählt", verloren)
	}
	if ausgesondert := ausgesonderteZahl(t, pool, ex[:1]); ausgesondert != 0 {
		t.Error("das verliehene Buch wurde als Verlust ausgesondert")
	}
}

// Der partielle Unique-Index: Ein zweiter Start im selben Bereich wird als
// ErrInventurLaeuftBereits abgewiesen und übernimmt nicht den Fortschritt des ersten.
func TestInventurEinOffenerLaufJeBereich(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)

	chemie := "Chemie"
	legeSignaturMitExemplarenAn(t, pool, chemie, 2)

	if _, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &chemie}, "Chemie", ""); err != nil {
		t.Fatalf("erste Session: %v", err)
	}

	_, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &chemie}, "Chemie", "")
	if !errors.Is(err, ErrInventurLaeuftBereits) {
		t.Errorf("zweite Session im selben Scope: erwartet ErrInventurLaeuftBereits, war %v", err)
	}

	// Nach dem Verwerfen ist der Bereich wieder frei.
	offen, err := repo.LadeOffeneLaeufe(ctx)
	if err != nil {
		t.Fatalf("ListOffene: %v", err)
	}
	if len(offen) != 1 {
		t.Fatalf("erwartet 1 offene Session, waren %d", len(offen))
	}
	if err := repo.VerwerfeLauf(ctx, offen[0].ID); err != nil {
		t.Fatalf("Abort: %v", err)
	}
	if _, err := repo.LegeLaufAn(ctx, "signature", InventurBereich{Signatur: &chemie}, "Chemie", ""); err != nil {
		t.Errorf("nach Abbruch muss ein Neustart möglich sein, war: %v", err)
	}
}

// Eine Inventur nach Fach und Jahrgang erfasst genau die Titel mit passendem Fach und passendem
// Jahrgangsbereich, nicht die mit anderem Fach oder außerhalb des Jahrgangs.
func TestInventurBereichFachUndJahrgang(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventurAbfragen(pool)
	bearbeiter := legeBearbeiterAn(t, pool)

	matheKl5 := legeFachExemplarAn(t, pool, "Mathematik", 5, 6, "BC-MA5")  // im Scope
	matheKl9 := legeFachExemplarAn(t, pool, "Mathematik", 9, 10, "BC-MA9") // falsche Klasse
	deutschKl5 := legeFachExemplarAn(t, pool, "Deutsch", 5, 6, "BC-DE5")   // falsches Fach
	// Ein Titel ohne Jahrgangsangabe gehört zu keinem Jahrgang. Läge er in jeder Inventur der
	// Jahrgänge 5 bis 10, stünde er beim Abschluss als Verlust im Bericht.
	matheOhne := legeFachExemplarAn(t, pool, "Mathematik", 5, 6, "BC-MA0")
	if _, err := pool.Exec(ctx, `UPDATE buecher_titel SET jahrgang_von = NULL, jahrgang_bis = NULL
		WHERE id = (SELECT titel_id FROM buecher_exemplare WHERE id = $1)`, matheOhne); err != nil {
		t.Fatalf("Titel ohne Jahrgang: %v", err)
	}

	subject, grade := "Mathematik", 5
	scope := InventurBereich{Fach: &subject, Jahrgang: &grade}

	if n, err := repo.ZaehleBereich(ctx, scope); err != nil {
		t.Fatalf("ZaehleBereich: %v", err)
	} else if n != 1 {
		t.Errorf("Scope 'Mathe Kl.5': erwartet 1, war %d", n)
	}

	// Die Warnung beim Scan: Nur das Exemplar mit passendem Fach und Jahrgang gehört dazu.
	if in, err := repo.IstExemplarImBereich(ctx, matheKl5, scope); err != nil {
		t.Fatalf("IstExemplarImBereich matheKl5: %v", err)
	} else if !in {
		t.Error("Mathe Kl.5 sollte im Scope sein")
	}

	if in, err := repo.IstExemplarImBereich(ctx, deutschKl5, scope); err != nil {
		t.Fatalf("IstExemplarImBereich deutschKl5: %v", err)
	} else if in {
		t.Error("Deutsch Kl.5 sollte NICHT im Scope sein (falsches Fach)")
	}
	if in, err := repo.IstExemplarImBereich(ctx, matheOhne, scope); err != nil {
		t.Fatalf("IstExemplarImBereich matheOhne: %v", err)
	} else if in {
		t.Error("Mathe ohne Jahrgang sollte NICHT im Scope einer Klasse sein")
	}

	sess, err := repo.LegeLaufAn(ctx, "filter", scope, "Mathematik · Kl. 5", "")
	if err != nil {
		t.Fatalf("Filter-Session anlegen: %v", err)
	}

	// Nichts scannen, abschließen: Genau das eine Exemplar im Bereich fehlt.
	verloren, err := repo.SchliesseLaufAb(ctx, sess.ID, scope, bearbeiter)
	if err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if verloren != 1 {
		t.Errorf("Verluste: erwartet 1 (nur Mathe Kl.5), war %d", verloren)
	}
	if aus := ausgesonderteZahl(t, pool, []string{matheKl5}); aus != 1 {
		t.Errorf("In-Scope Mathe Kl.5 nicht als Verlust ausgesondert: %d", aus)
	}
	if aus := ausgesonderteZahl(t, pool, []string{matheKl9, deutschKl5, matheOhne}); aus != 0 {
		t.Errorf("Out-of-Scope-Exemplare fälschlich ausgesondert: %d", aus)
	}
}

package repository

import (
	"context"
	"errors"
	"testing"
)

// TestInventurParallelbetrieb ist der Kern-Regressionstest: Zwei Sessions in
// verschiedenen Scopes dürfen sich weder im Fortschritt noch beim Abschluss
// gegenseitig beeinflussen. Genau das war im alten globalen Modell die Ursache für
// stillen Datenverlust bei parallelem Scannen.
func TestInventurParallelbetrieb(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	matheID, matheEx := seedSignaturMitExemplaren(t, pool, "Mathematik", 5)
	_, deutschEx := seedSignaturMitExemplaren(t, pool, "Deutsch", 4)

	// Kollege A: Mathe-Session, scannt 3 von 5.
	sessA, err := repo.CreateInventurSession(ctx, "signature", &matheID, "Mathematik", "")
	if err != nil {
		t.Fatalf("Session A anlegen: %v", err)
	}
	for _, ex := range matheEx[:3] {
		if err := repo.RecordInventurScan(ctx, sessA.ID, ex); err != nil {
			t.Fatalf("Scan A: %v", err)
		}
	}

	// Kollege B startet PARALLEL eine Deutsch-Session — im alten Modell hätte das A's
	// Fortschritt global gelöscht.
	deutschID := deutschSignaturID(t, pool)
	sessB, err := repo.CreateInventurSession(ctx, "signature", &deutschID, "Deutsch", "")
	if err != nil {
		t.Fatalf("Session B anlegen: %v", err)
	}

	aErfasst := erfassungsZahl(t, pool, sessA.ID)
	if aErfasst != 3 {
		t.Errorf("A's Fortschritt nach B's Start: erwartet 3, war %d", aErfasst)
	}

	// A schließt ab: genau die 2 nicht gescannten Mathe-Exemplare gelten als Verlust.
	verloren, err := repo.FinishInventurSession(ctx, sessA.ID, &matheID)
	if err != nil {
		t.Fatalf("Finish A: %v", err)
	}
	if verloren != 2 {
		t.Errorf("Mathe-Verluste: erwartet 2, war %d", verloren)
	}

	// Deutsch-Bestand (B's Scope) muss völlig unberührt sein.
	if ausgesondert := ausgesonderteZahl(t, pool, deutschEx); ausgesondert != 0 {
		t.Errorf("Deutsch-Exemplare fälschlich ausgesondert: %d (Session B lief parallel)", ausgesondert)
	}
	_ = sessB
}

// TestInventurAusgelieheneNichtVerloren sichert den Zusatzfund ab: Ein aktuell
// verliehenes Buch ist beim Schüler, nicht verschwunden — es darf beim Abschluss
// niemals als Verlust markiert werden, obwohl es niemand scannen konnte.
func TestInventurAusgelieheneNichtVerloren(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	sigID, ex := seedSignaturMitExemplaren(t, pool, "Physik", 3)
	leiheAus(t, pool, ex[0]) // ein Exemplar verliehen

	// Scope zählt nur die 2 physisch anwesenden.
	scope, err := repo.ZaehleScope(ctx, &sigID)
	if err != nil {
		t.Fatalf("ZaehleScope: %v", err)
	}
	if scope != 2 {
		t.Errorf("Scope mit 1 verliehenen: erwartet 2, war %d", scope)
	}

	// Session starten, NICHTS scannen, abschließen: nur die 2 anwesenden fehlen.
	// Das verliehene Buch darf nicht darunter sein.
	sess, err := repo.CreateInventurSession(ctx, "signature", &sigID, "Physik", "")
	if err != nil {
		t.Fatalf("Session anlegen: %v", err)
	}
	verloren, err := repo.FinishInventurSession(ctx, sess.ID, &sigID)
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

// TestInventurEineOffeneSessionJeScope prüft den partiellen Unique-Index: Ein zweiter
// Start im selben Scope wird als ErrInventurLaeuftBereits abgewiesen (statt still den
// Fortschritt zu übernehmen/überschreiben).
func TestInventurEineOffeneSessionJeScope(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewInventoryRepository(pool)

	sigID, _ := seedSignaturMitExemplaren(t, pool, "Chemie", 2)

	if _, err := repo.CreateInventurSession(ctx, "signature", &sigID, "Chemie", ""); err != nil {
		t.Fatalf("erste Session: %v", err)
	}

	_, err := repo.CreateInventurSession(ctx, "signature", &sigID, "Chemie", "")
	if !errors.Is(err, ErrInventurLaeuftBereits) {
		t.Errorf("zweite Session im selben Scope: erwartet ErrInventurLaeuftBereits, war %v", err)
	}

	// Nach Abbruch ist der Scope wieder frei.
	offen, err := repo.ListOffeneInventurSessions(ctx)
	if err != nil {
		t.Fatalf("ListOffene: %v", err)
	}
	if len(offen) != 1 {
		t.Fatalf("erwartet 1 offene Session, waren %d", len(offen))
	}
	if err := repo.AbortInventurSession(ctx, offen[0].ID); err != nil {
		t.Fatalf("Abort: %v", err)
	}
	if _, err := repo.CreateInventurSession(ctx, "signature", &sigID, "Chemie", ""); err != nil {
		t.Errorf("nach Abbruch muss ein Neustart möglich sein, war: %v", err)
	}
}

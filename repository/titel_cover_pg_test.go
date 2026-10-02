package repository

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// Das lokal gespeicherte Cover fällt mit seinem Titel, an beiden Lösch-Türen. Die Maske
// „Buch bearbeiten" (DeleteTitle) ließ die Datei liegen, nur die Massenaktion der
// Bestandstabelle nahm sie mit. Eine Datei, die ein weiterer Titel trägt, bleibt.
func TestDeleteTitle_NimmtDasLokaleCoverMit(t *testing.T) {
	pool := pgTestPool(t)
	resetAuflagen(t, pool)
	ctx := context.Background()
	t.Chdir(t.TempDir())
	if err := os.Mkdir("uploads", 0o750); err != nil {
		t.Fatal(err)
	}
	datei := func(name string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join("uploads", name), []byte("bild"), 0o600); err != nil {
			t.Fatal(err)
		}
		return "/uploads/" + name
	}
	vorhanden := func(coverURL string) bool {
		_, err := os.Stat(filepath.Join(".", coverURL))
		return err == nil
	}
	titelMit := func(name, coverURL string) (id string) {
		t.Helper()
		if err := pool.QueryRow(ctx,
			`INSERT INTO buecher_titel (titel, cover_url) VALUES ($1, $2) RETURNING id::text`,
			name, coverURL).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}

	eigen, geteilt := datei("cover_auto_eigen.webp"), datei("cover_auto_geteilt.webp")
	allein := titelMit("Cover allein", eigen)
	erster := titelMit("Cover geteilt 1", geteilt)
	zweiter := titelMit("Cover geteilt 2", geteilt)
	fremd := titelMit("Cover im Netz", "https://portal.dnb.de/opac/mvb/cover?isbn=9783551551672")

	nur, err := LokaleCoverNurDieserTitel(ctx, pool, []string{allein, erster, fremd})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(nur, []string{eigen}) {
		t.Errorf("Cover nur dieser Titel: %v, erwartet nur %s — das geteilte trägt ein weiterer Titel", nur, eigen)
	}

	repo := NewAuditRepository(pool)
	bearbeiter := seedBearbeiter(t, pool)
	for _, id := range []string{allein, erster} {
		if err := repo.DeleteTitle(ctx, id, bearbeiter); err != nil {
			t.Fatalf("DeleteTitle: %v", err)
		}
	}
	if vorhanden(eigen) {
		t.Errorf("%s liegt noch da, sein Titel ist gelöscht", eigen)
	}
	if !vorhanden(geteilt) {
		t.Fatalf("%s ist entfernt, ein zweiter Titel trägt es noch", geteilt)
	}

	if err := repo.DeleteTitle(ctx, zweiter, bearbeiter); err != nil {
		t.Fatalf("DeleteTitle: %v", err)
	}
	if vorhanden(geteilt) {
		t.Errorf("%s liegt noch da, beide Titel sind gelöscht", geteilt)
	}
}

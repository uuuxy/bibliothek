package api

import (
	"context"
	"testing"

	"bibliothek/db"
)

// Regression: Die DSGVO-Auskunft (Art. 15) scannte die in der DB NULLBAREN Adress-/
// Kontaktfelder (strasse, hausnummer, plz, ort, eltern_email) in nicht-nullbare
// Go-strings. Ein Schüler OHNE erfasste Adresse ließ die Auskunft mit HTTP 500
// "Fehler beim Laden der Stammdaten" abstürzen — das betraf jeden realen Schüler
// ohne Adresse, nicht nur Demo-Daten. Der COALESCE(...,'')-Fix macht NULL → ''.
//
// pgxmock kann das nicht prüfen (es führt kein SQL aus, COALESCE würde umgangen) —
// daher ein echter PG-Integrationstest, gated auf TEST_DATABASE_URL.
func TestDsgvoStammdaten_OhneAdresseKein500(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	var id string
	if err := pool.QueryRow(ctx, `
		INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr)
		VALUES ('DSGVO-NULLADDR', 'Ohne', 'Adresse', '7z', 2030)
		RETURNING id`).Scan(&id); err != nil {
		t.Fatalf("Schüler ohne Adresse anlegen: %v", err)
	}

	srv := &Server{DB: &db.Database{Pool: pool}}
	st, err := srv.dsgvoQueryStammdaten(ctx, id)
	if err != nil {
		t.Fatalf("dsgvoQueryStammdaten darf bei fehlender Adresse nicht scheitern (500-Regression): %v", err)
	}
	if st == nil {
		t.Fatal("Stammdaten unerwartet nil")
	}
	if st.Strasse != "" || st.Hausnummer != "" || st.Plz != "" || st.Ort != "" || st.ElternEmail != "" {
		t.Fatalf("nullbare Adress-/Kontaktfelder müssen als leere Strings ankommen, war: %+v", st)
	}
}

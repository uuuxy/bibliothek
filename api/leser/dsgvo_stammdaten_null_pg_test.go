package leser

import (
	"context"
	"testing"

	"bibliothek/api/tuer"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/internal/testdaten"
)

// Die DSGVO-Auskunft (Art. 15) liest Adress- und Kontaktfelder (strasse, hausnummer, plz,
// ort, eltern_email), die in der Datenbank NULL sein dürfen. Ohne COALESCE auf den leeren
// Text in Go-Strings gelesen, ließe ein Schüler ohne erfasste Adresse die Auskunft mit HTTP
// 500 „Fehler beim Laden der Stammdaten" abstürzen, und das träfe jeden echten Schüler ohne
// Adresse.
//
// pgxmock kann das nicht prüfen (es führt kein SQL aus, COALESCE würde umgangen); daher ein
// echter Test an Postgres, der TEST_DATABASE_URL braucht.
func TestStammdaten_OhneAdresseKein500(t *testing.T) {
	pool := pgtest.Pool(t)
	testdaten.LeereBestandsdaten(t, pool)
	ctx := context.Background()

	id := testdaten.LegeSchuelerMitNamenAn(t, pool, "DSGVO-NULLADDR", "Ohne", "Adresse", "7z", 2030)

	srv := &Tueren{Basis: &tuer.Basis{DB: &db.Verbindung{Pool: pool}}}
	st, err := srv.dsgvoStammdaten(ctx, id)
	if err != nil {
		t.Fatalf("dsgvoStammdaten darf bei fehlender Adresse nicht scheitern (500-Regression): %v", err)
	}
	if st == nil {
		t.Fatal("Stammdaten unerwartet nil")
	}
	if st.Strasse != "" || st.Hausnummer != "" || st.Plz != "" || st.Ort != "" || st.ElternEmail != "" {
		t.Fatalf("nullbare Adress-/Kontaktfelder müssen als leere Strings ankommen, war: %+v", st)
	}
}

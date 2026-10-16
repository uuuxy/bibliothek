package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Leserzeile eines gelöschten Kontos geht mit, wenn sie unberührt ist, und bleibt stehen,
// wenn etwas an ihr hängt.
//
// Der Fall ist die abgelehnte Zugangsanfrage: Die Selbstanmeldung legt ein Konto ohne
// Leserzeile an, der Trigger trg_benutzer_hat_leserzeile hängt eine frische daran. Nach dem
// Ablehnen bliebe diese Zeile als Waise in der Lesertabelle stehen, ohne Ausweis, ohne
// Vorgänge, mit dem aus der Adresse geratenen Namen.
//
// Geprüft wird am Postgres, weil die Prüfung den Katalog der Fremdschlüssel liest: Eine
// Attrappe spielte genau das nach, was hier die Frage ist.
func TestLoescheKonto_LeserzeileGehtMitWennUnberuehrt(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	konto, leser := legeZugangsanfrageAn(t, pool, "anfrage.unberuehrt@schule.example")
	if err := NewKontoAbfragen(pool).LoescheKonto(ctx, konto, legeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("Konto löschen: %v", err)
	}

	if leserzeileExistiert(t, pool, leser) {
		t.Error("die Leserzeile blieb als Waise in der Leserdatei stehen")
	}
}

// Die Gegenprobe: Hängt etwas an der Zeile, gehört sie einem Menschen und nicht dem Konto. Hier
// eine abgeschlossene Ausleihe, ein Fremdschlüssel mit RESTRICT, der das Löschen ohnehin
// abwiese; die Prüfung sieht ihn vorher, damit der Vorgang nicht an einer Meldung der Datenbank
// endet.
func TestLoescheKonto_LeserzeileBleibtWennEtwasDranHaengt(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	konto, leser := legeZugangsanfrageAn(t, pool, "anfrage.mitbuch@schule.example")
	bearbeiter := legeBearbeiterAn(t, pool)
	copyID := legeSignaturMitExemplarenAn(t, pool, "LeserzeileBleibt", 1)[0]
	loan := legeAusleiheAn(t, pool, copyID, leser, bearbeiter)
	bucheRueckgabe(t, pool, loan)

	if err := NewKontoAbfragen(pool).LoescheKonto(ctx, konto, bearbeiter); err != nil {
		t.Fatalf("Konto löschen: %v", err)
	}

	if !leserzeileExistiert(t, pool, leser) {
		t.Error("die Leserzeile mit einer Ausleihe wurde mitgelöscht")
	}
}

// Ein Ausweis hält die Zeile ebenfalls. Ihn sieht keine Abfrage über Fremdschlüssel, er steht
// als Spalte in der Zeile; eine Nummer wird nie neu vergeben.
func TestLoescheKonto_LeserzeileBleibtMitAusweis(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()

	konto, leser := legeZugangsanfrageAn(t, pool, "anfrage.mitausweis@schule.example")
	if _, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = 'A-77001' WHERE id = $1`, leser); err != nil {
		t.Fatal(err)
	}

	if err := NewKontoAbfragen(pool).LoescheKonto(ctx, konto, legeBearbeiterAn(t, pool)); err != nil {
		t.Fatalf("Konto löschen: %v", err)
	}

	if !leserzeileExistiert(t, pool, leser) {
		t.Error("die Leserzeile mit Ausweisnummer wurde mitgelöscht")
	}
}

// Ein Konto aus der Selbstanmeldung: ohne leser_id angelegt, die Leserzeile hängt der Trigger
// an.
func legeZugangsanfrageAn(t *testing.T, pool *pgxpool.Pool, email string) (string, string) {
	t.Helper()
	var konto string
	var leser *string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv, zugang_beantragt_am)
		 VALUES ('Neue', 'Kraft', $1, 'kollegium', false, CURRENT_TIMESTAMP)
		 RETURNING id, leser_id`, email).Scan(&konto, &leser); err != nil {
		t.Fatalf("Zugangsanfrage anlegen: %v", err)
	}
	if leser == nil {
		t.Fatal("der Wächter hat keine Leserzeile angehängt — dann prüft dieser Test nichts")
	}
	return konto, *leser
}

func leserzeileExistiert(t *testing.T, pool *pgxpool.Pool, leserID string) bool {
	t.Helper()
	var da bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM leser WHERE id = $1)`, leserID).Scan(&da); err != nil {
		t.Fatalf("Leserzeile lesen: %v", err)
	}
	return da
}

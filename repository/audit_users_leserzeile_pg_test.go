package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Die Leserzeile eines gelöschten Kontos geht mit, wenn sie unberührt ist — und bleibt
// stehen, wenn etwas an ihr hängt.
//
// Anlass ist die abgelehnte Zugangsanfrage: Die Selbstanmeldung legt ein Konto ohne
// Leserzeile an, der Wächter trg_benutzer_hat_leserzeile hängt eine frische daran. Wurde
// die Anfrage abgelehnt, blieb diese Zeile als Waise in der Leserdatei stehen — ohne
// Ausweis, ohne Vorgänge, mit dem aus der Adresse geratenen Namen (OFFEN.md 5.18, Fund 1).
//
// Geprüft wird am ECHTEN Postgres, weil die Prüfung den Fremdschlüssel-Katalog der
// Datenbank liest: Ein Mock würde genau das nachspielen, was hier die Frage ist.
func TestDeleteUser_LeserzeileGehtMitWennUnberuehrt(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewAuditRepository(pool)

	konto, leser := seedZugangsanfrage(t, pool, "anfrage.unberuehrt@schule.example")
	if err := repo.DeleteUser(ctx, konto, seedBearbeiter(t, pool)); err != nil {
		t.Fatalf("Konto löschen: %v", err)
	}

	if leserzeileExistiert(t, pool, leser) {
		t.Error("die Leserzeile blieb als Waise in der Leserdatei stehen")
	}
}

// Die Gegenprobe, und die ist die wichtigere: Hängt etwas an der Zeile, gehört sie einem
// Menschen und nicht dem Konto. Hier eine abgeschlossene Ausleihe — ein Fremdschlüssel mit
// RESTRICT, der das Löschen ohnehin abwiese; die Prüfung muss ihn VORHER sehen, damit der
// Vorgang nicht an einer Constraint-Meldung endet.
func TestDeleteUser_LeserzeileBleibtWennEtwasDranHaengt(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewAuditRepository(pool)

	konto, leser := seedZugangsanfrage(t, pool, "anfrage.mitbuch@schule.example")
	bearbeiter := seedBearbeiter(t, pool)
	copyID := seedSignaturMitExemplaren(t, pool, "LeserzeileBleibt", 1)[0]
	loan := seedAusleihe(t, pool, copyID, leser, bearbeiter)
	returnLoan(t, pool, loan)

	if err := repo.DeleteUser(ctx, konto, bearbeiter); err != nil {
		t.Fatalf("Konto löschen: %v", err)
	}

	if !leserzeileExistiert(t, pool, leser) {
		t.Error("die Leserzeile mit einer Ausleihe wurde mitgelöscht")
	}
}

// Ein Ausweis hält die Zeile ebenfalls — ihn sieht keine Fremdschlüssel-Abfrage, er steht
// als Spalte in der Zeile. Eine Nummer ist vergeben und wird nie recycelt.
func TestDeleteUser_LeserzeileBleibtMitAusweis(t *testing.T) {
	pool := pgTestPool(t)
	resetInventurDaten(t, pool)
	ctx := context.Background()
	repo := NewAuditRepository(pool)

	konto, leser := seedZugangsanfrage(t, pool, "anfrage.mitausweis@schule.example")
	if _, err := pool.Exec(ctx, `UPDATE leser SET barcode_id = 'A-77001' WHERE id = $1`, leser); err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteUser(ctx, konto, seedBearbeiter(t, pool)); err != nil {
		t.Fatalf("Konto löschen: %v", err)
	}

	if !leserzeileExistiert(t, pool, leser) {
		t.Error("die Leserzeile mit Ausweisnummer wurde mitgelöscht")
	}
}

// Ein Konto aus der Selbstanmeldung: ohne leser_id angelegt, die Leserzeile hängt der
// Wächter an (Migration 123).
func seedZugangsanfrage(t *testing.T, pool *pgxpool.Pool, email string) (string, string) {
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

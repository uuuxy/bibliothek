package api

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestBlockReason_ConstraintLehntGesperrtOhneGrundAb sichert #3 ab: Die DB verbietet den
// Zombie-Zustand „gesperrt ohne Grund" (chk_schueler_block_reason). Ein gesperrter
// Schüler ohne block_reason darf gar nicht erst entstehen.
func TestBlockReason_ConstraintLehntGesperrtOhneGrundAb(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt)
		 VALUES ('BR-1', 'Ohne', 'Grund', '5a', 2030, true)`)
	if err == nil {
		t.Fatal("INSERT eines gesperrten Schülers ohne block_reason muss am CHECK scheitern")
	}
	if !strings.Contains(err.Error(), "chk_schueler_block_reason") {
		t.Errorf("erwartet Verletzung von chk_schueler_block_reason, war: %v", err)
	}

	// Leerer/whitespace-Grund zählt ebenfalls als „kein Grund".
	_, err = pool.Exec(ctx,
		`INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt, block_reason)
		 VALUES ('BR-2', 'Leer', 'Grund', '5a', 2030, true, '   ')`)
	if err == nil {
		t.Fatal("gesperrt mit reinem Whitespace-Grund muss ebenfalls scheitern")
	}
}

// TestSperreAbgaenger_SetztGrund: Ein Abgänger mit offenem Buch wird gesperrt — und trägt
// dabei einen aussagekräftigen Grund, damit das Personal im Profil sieht, WARUM.
func TestSperreAbgaenger_SetztGrund(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	sid := seedSchueler(t, pool, "BR-3", "Abgang", "10a")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := sperreAbgaenger(ctx, tx, sid); err != nil {
		t.Fatalf("sperreAbgaenger: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	gesperrt, grund := gesperrtState(t, pool, sid)
	if !gesperrt {
		t.Fatal("Abgänger mit offenem Buch soll gesperrt sein")
	}
	if strings.TrimSpace(grund) == "" {
		t.Error("sperreAbgaenger muss einen block_reason setzen (sonst Zombie-Sperre)")
	}
}

// TestAnonymisiereAbgaenger_SetztFestenGrund: Der anonymisierte Abgänger bekommt einen
// festen, PII-freien Grund (kein evtl. altbelasteter Grund bleibt stehen).
func TestAnonymisiereAbgaenger_SetztFestenGrund(t *testing.T) {
	pool := pgTestPool(t)
	resetBestandsdaten(t, pool)
	ctx := context.Background()

	sid := seedSchueler(t, pool, "BR-4", "Anon", "10a")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := anonymisiereAbgaenger(ctx, tx, sid); err != nil {
		t.Fatalf("anonymisiereAbgaenger: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	gesperrt, grund := gesperrtState(t, pool, sid)
	if !gesperrt || grund != "Abgänger anonymisiert" {
		t.Errorf("erwartet gesperrt mit festem Grund 'Abgänger anonymisiert', war gesperrt=%v grund=%q", gesperrt, grund)
	}
}

func gesperrtState(t *testing.T, pool *pgxpool.Pool, id string) (bool, string) {
	t.Helper()
	var gesperrt bool
	var grund *string
	if err := pool.QueryRow(context.Background(),
		`SELECT ist_gesperrt, block_reason FROM schueler WHERE id = $1`, id).Scan(&gesperrt, &grund); err != nil {
		t.Fatalf("Sperr-Status lesen: %v", err)
	}
	if grund == nil {
		return gesperrt, ""
	}
	return gesperrt, *grund
}

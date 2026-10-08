package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/internal/pgtest"
	"bibliothek/pkg/lmf"
	"bibliothek/sse"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PG-Integrationstests fürs api-Paket (gated auf TEST_DATABASE_URL, wie db/ und
// repository/). Nötig für die order-/graduates-Bugs, deren Kern in SQL-Filtern liegt
// (bereits bestellte Exemplare, numerische Barcode-Sortierung, Abgänger-Filter) —
// pgxmock würde nur nachgespielte Antworten prüfen, nicht die SQL-Korrektheit.
//
// Pool-Aufbau, Advisory-Lock und Notbremse liegen seit dem 31.08.2026 in
// internal/pgtest (vorher fünffach kopiert); hier stehen nur noch die Helfer
// dieses Pakets.
func pgTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return pgtest.Pool(t)
}

// resetBestandsdaten leert Bestands-, Bestell- und Personendaten zwischen Tests.
// klassen gehört mit in den Reset, obwohl dort keine Testdaten im üblichen Sinn stehen:
// Die Tabelle ist das Vokabular, das der Trigger trg_schueler_klasse_vokabular
// (Migration 079) beim Schreiben nachschlägt. Steht dort schon "7a", wird ein später
// eingefügtes "07a" stillschweigend als "7a" gespeichert — steht sie leer, bleibt "07a"
// stehen und wird selbst zur kanonischen Form.
//
// Damit hing die SCHREIBWEISE einer Klasse davon ab, welcher Test vorher gelaufen war.
// Am 23.08.2026 machte das einen neuen Test allein grün und in der vollen Suite rot; die
// erste Erklärung dafür ("ein anderer Test schreibt alle Schülerzeilen") war falsch — das
// Paket kennt kein t.Parallel(), und schueler wird ohnehin geleert. Die Kopplung lief über
// diese eine nicht zurückgesetzte Tabelle. Die gefährliche Richtung ist die umgekehrte:
// ein Test, den fremdes Vokabular still grün hält.
//
// Truncate mit CASCADE räumt die vier referenzierenden Tabellen mit (schueler,
// klassen_lehrer_mapping, class_books, klassensatz_reservierungen) — alles Testdaten.
// Befüllt wird klassen von keiner Migration, sie entsteht allein durch den Trigger.
//
// ausweisnummern_ausgeschieden gehört aus demselben Grund dazu: Der Generator zieht über sie
// (Migration 146), und eine Nummer, die ein früherer Test entfernt hat, verschöbe die festen
// Nummern in sequence_pg_test.go.
func resetBestandsdaten(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		TRUNCATE buecher_exemplare, buecher_titel, ausleihen, leser, benutzer, klassen,
		         ausweisnummern_ausgeschieden
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("Reset fehlgeschlagen: %v", err)
	}
}

// titelMitMeldebestand legt einen Titel mit gegebenem Meldebestand an. Ein Titel mit
// Litteras LMF-Kennung („LMF-Mathe 7") wird als Lernmittel angelegt — so, wie es
// Migration 093 mit dem Altbestand tat; die Tests dieses Pakets sprechen seit 2026 in
// dieser Konvention, und sie bleibt hier lesbar.
func titelMitMeldebestand(t *testing.T, pool *pgxpool.Pool, titel string, meldebestand int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, meldebestand, ist_lernmittel) VALUES ($1, $2, $3) RETURNING id`,
		titel, meldebestand, lmf.HatKennung(titel)).Scan(&id); err != nil {
		t.Fatalf("Titel anlegen: %v", err)
	}
	return id
}

// titelMitSignatur legt einen Titel mit expliziter Signatur an; Lernmittel, wenn Titel
// oder Signatur die LMF-Kennung tragen (siehe titelMitMeldebestand).
func titelMitSignatur(t *testing.T, pool *pgxpool.Pool, titel, signatur string, meldebestand int) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_titel (titel, signatur, meldebestand, ist_lernmittel) VALUES ($1, $2, $3, $4) RETURNING id`,
		titel, signatur, meldebestand, lmf.HatKennung(titel) || lmf.HatKennung(signatur)).Scan(&id); err != nil {
		t.Fatalf("Titel mit Signatur anlegen: %v", err)
	}
	return id
}

// exemplar legt ein Exemplar mit Verleih-/Aussonderungsstatus und Notiz an.
func exemplar(t *testing.T, pool *pgxpool.Pool, titelID, barcode string, ausleihbar bool, notiz string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		titelID, barcode, ausleihbar, notiz).Scan(&id); err != nil {
		t.Fatalf("Exemplar %q anlegen: %v", barcode, err)
	}
	return id
}

// inTx führt f in einer eigenen Transaktion aus und committet — für Testschreiber, die
// wie LogAusleihe seit dem 07.09.2026 eine Transaktion des Aufrufers verlangen.
func inTx(t *testing.T, pool *pgxpool.Pool, f func(tx pgx.Tx) error) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := f(tx); err != nil {
		_ = tx.Rollback(ctx) //nolint:errcheck
		t.Fatalf("in Transaktion: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// routerMitSitzung legt ein Admin-Konto an, erzeugt dafür eine Sitzung und baut den GANZEN
// Router. Jeder Test, der eine Tür am Live-Pfad prüfen will, nimmt ihn: Recht, CSRF und
// Middleware hängen am Router, nicht am Handler, und genau dort saßen in diesem Projekt
// schon Fehler, die der nackte Handler nicht zeigte.
//
// Die E-Mail gehört dem Test (ON CONFLICT hält den Aufruf wiederholbar), damit zwei Tests
// sich nicht dasselbe Konto teilen und seine Protokolleinträge zählen.
func routerMitSitzung(t *testing.T, pool *pgxpool.Pool, email, vorname, nachname string) (adminID, sitzung string, router http.Handler) {
	t.Helper()
	return routerMitSitzungUeber(t, pool, pool, email, vorname, nachname)
}

// routerMitSitzungUeber ist routerMitSitzung mit einem eigenen Pool für den Server. Ein Test
// wickelt damit den echten Pool ein und lässt eine einzelne Abfrage scheitern; Konto und
// Sitzung entstehen am echten.
func routerMitSitzungUeber(t *testing.T, pool *pgxpool.Pool, serverPool db.PgxPoolIface, email, vorname, nachname string) (adminID, sitzung string, router http.Handler) {
	t.Helper()
	authenticator, err := auth.NewAuthenticator("pg-test-sitzungsgeheimnis-32-zeichen!!", pool, time.Hour)
	if err != nil {
		t.Fatalf("Authenticator: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ($1, $2, $3, 'admin', true)
		ON CONFLICT (lower(email)) DO UPDATE SET aktiv = true
		RETURNING id`, vorname, nachname, email).Scan(&adminID); err != nil {
		t.Fatalf("Konto %s anlegen: %v", email, err)
	}
	sitzung, err = authenticator.GenerateToken(adminID, "PG-TEST-1", auth.RoleAdmin, "")
	if err != nil {
		t.Fatalf("Sitzung: %v", err)
	}
	return adminID, sitzung, NewServer(&db.Database{Pool: serverPool}, authenticator, sse.NewBroker(), false).Routes()
}

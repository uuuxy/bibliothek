// Command seed füllt eine Wegwerf-Datenbank mit Test-Admin, Schülern, Titeln und Exemplaren,
// die Vorstufe zum Lasttest mit k6 (docs/SCRIPTS.md). Es fragt nicht, bevor es schreibt.
//
// Der Lauf selbst steht in fuelle(): Umfang und Ausgabe kommen von außen, Fehler gehen zurück.
// main() liefert die Umgebung (DATABASE_URL, JWT_SECRET) und den vollen Umfang;
// main_pg_test.go fährt denselben Lauf klein gegen die Datenbank der Tests.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"bibliothek/auth"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Umfang ist die Größe eines Laufs. Exemplare werden in Portionen der Größe Portion eingefügt.
type Umfang struct {
	Schueler, Titel, Exemplare, Portion int
}

// vollerUmfang ist die Größe für den Lasttest.
var vollerUmfang = Umfang{Schueler: 2000, Titel: 5000, Exemplare: 80000, Portion: 10000}

const (
	adminEmail   = "scanner@test.local"
	adminBarcode = "ADMIN-SCANNER-TEST"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	geheimnis := os.Getenv("JWT_SECRET")
	if len(geheimnis) < 32 {
		log.Fatalf("FATAL: JWT_SECRET environment variable must be at least 32 characters long for security")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Fehler beim Verbinden mit der Datenbank: %v\n", err)
	}
	defer pool.Close()

	fmt.Println("Starte massiven Daten-Import für den Stresstest...")
	start := time.Now()

	adminID, err := fuelle(ctx, pool, vollerUmfang, os.Stdout)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	anmeldung, err := auth.NewAnmeldung(geheimnis, pool, 8760*time.Hour) // 1 Jahr gültig
	if err == nil {
		token, _ := anmeldung.ErzeugeToken(adminID, adminBarcode, auth.RolleAdmin, "") //nolint:errcheck
		fmt.Printf("\n========================================================\n")
		fmt.Printf("🛡️ DAST/SAST Scanner JWT (1 Jahr gültig):\n%s\n", token)
		fmt.Printf("========================================================\n\n")
	}

	fmt.Printf("🎉 Fertig in %v. Die Datenbank ist jetzt voll und bereit für k6.\n", time.Since(start))
}

// fuelle führt den Lauf aus und liefert die Kennung des Test-Admins, die gespeicherte, auch wenn
// das Konto schon vom vorigen Lauf da war. Jeder Schritt ist wiederholbar: Was es schon gibt,
// wird übersprungen (ON CONFLICT DO NOTHING), nichts entsteht doppelt.
func fuelle(ctx context.Context, pool *pgxpool.Pool, umfang Umfang, ausgabe io.Writer) (string, error) {
	adminID, err := legeTestAdminAn(ctx, pool)
	if err != nil {
		return "", err
	}
	if err := legeSchuelerAn(ctx, pool, umfang.Schueler, ausgabe); err != nil {
		return "", err
	}
	titelIDs, err := legeTitelAn(ctx, pool, umfang.Titel, ausgabe)
	if err != nil {
		return "", err
	}
	if err := legeExemplareAn(ctx, pool, titelIDs, umfang.Exemplare, umfang.Portion, ausgabe); err != nil {
		return "", err
	}
	return adminID, nil
}

// legeTestAdminAn legt das Konto des Scanners an oder findet es. Eine frische UUID mit
// ON CONFLICT DO NOTHING ließe beim zweiten Lauf die Zeile unberührt; der Ausweis fände keine
// Leserzeile, und das JWT trüge eine Kennung, die es nicht gibt
// (TestFuelle_ZweiterLaufIstDerselbeAdmin).
func legeTestAdminAn(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	// ON CONFLICT … DO UPDATE statt DO NOTHING: Nur so liefert RETURNING auch die
	// vorhandene Zeile. Geändert wird nichts (aktiv bleibt, was es ist).
	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO benutzer (vorname, nachname, email, rolle, aktiv)
		VALUES ('Scanner', 'TestAdmin', $1, 'admin', true)
		ON CONFLICT (email) DO UPDATE SET aktiv = benutzer.aktiv
		RETURNING id
	`, adminEmail).Scan(&adminID); err != nil {
		return "", fmt.Errorf("test-Admin anlegen: %w", err)
	}
	// Die Ausweisnummer steht an der Leserzeile, die der Trigger beim Anlegen des Kontos
	// erzeugt. Zwei Anweisungen und keine schreibende CTE: Eine Zeile, die dieselbe Anweisung
	// gerade eingefügt hat, liegt außerhalb des Schnappschusses des äußeren UPDATE; es fände sie
	// nicht und änderte stumm 0 Zeilen.
	tag, err := pool.Exec(ctx, `
		UPDATE leser SET barcode_id = $2
		WHERE id = (SELECT leser_id FROM benutzer WHERE id = $1)
	`, adminID, adminBarcode)
	if err != nil {
		return "", fmt.Errorf("ausweis des Test-Admins eintragen: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return "", fmt.Errorf("ausweis des Test-Admins eintragen: %d Leserzeilen statt 1", tag.RowsAffected())
	}
	return adminID, nil
}

func legeSchuelerAn(ctx context.Context, pool *pgxpool.Pool, n int, ausgabe io.Writer) error {
	stapel := &pgx.Batch{}
	for i := 1; i <= n; i++ {
		barcodeID := fmt.Sprintf("S%06d", i)
		vorname := fmt.Sprintf("Vorname%d", i)
		nachname := fmt.Sprintf("Nachname%d", i)
		klasse := fmt.Sprintf("%d%s", rand.Intn(8)+5, string(rune('A'+rand.Intn(4)))) // z.B. 7B
		abgaengerJahr := time.Now().Year() + rand.Intn(5) + 1

		stapel.Queue(`
			INSERT INTO schueler (barcode_id, vorname, nachname, klasse, abgaenger_jahr, ist_gesperrt)
			VALUES ($1, $2, $3, $4, $5, false)
			ON CONFLICT (barcode_id) WHERE deleted_at IS NULL DO NOTHING
		`, barcodeID, vorname, nachname, klasse, abgaengerJahr)
	}

	ergebnis := pool.SendBatch(ctx, stapel)
	if err := ergebnis.Close(); err != nil {
		return fmt.Errorf("schüler einfügen: %w", err)
	}
	fmt.Fprintf(ausgabe, "✅ %d Schüler erfolgreich generiert.\n", n) //nolint:errcheck // Fortschrittszeile
	return nil
}

// legeTitelAn liefert die Kennungen der Titel, an die die Exemplare gehängt werden. Titel, die
// es schon gab (zweiter Lauf), behalten ihre gespeicherte Kennung; die frische aus dem INSERT
// wäre eine Kennung ohne Zeile, und jedes Exemplar daran ein Fehler am Fremdschlüssel.
func legeTitelAn(ctx context.Context, pool *pgxpool.Pool, n int, ausgabe io.Writer) ([]uuid.UUID, error) {
	stapel := &pgx.Batch{}
	for i := 0; i < n; i++ {
		stapel.Queue(`
			INSERT INTO buecher_titel (id, titel, isbn)
			VALUES ($1, $2, $3)
			ON CONFLICT (isbn) DO NOTHING
		`, uuid.New(), fmt.Sprintf("Titel %d", i+1), fmt.Sprintf("ISBN-%010d", i+1))
	}
	ergebnis := pool.SendBatch(ctx, stapel)
	if err := ergebnis.Close(); err != nil {
		return nil, fmt.Errorf("titel einfügen: %w", err)
	}
	zeilen, err := pool.Query(ctx, `SELECT id FROM buecher_titel WHERE isbn LIKE 'ISBN-%' ORDER BY isbn LIMIT $1`, n)
	if err != nil {
		return nil, fmt.Errorf("titel-Ids lesen: %w", err)
	}
	titelIDs, err := pgx.CollectRows(zeilen, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("titel-Ids lesen: %w", err)
	}
	if len(titelIDs) != n {
		return nil, fmt.Errorf("%d Titel gespeichert, %d erwartet", len(titelIDs), n)
	}
	fmt.Fprintf(ausgabe, "✅ %d Titel erfolgreich generiert.\n", n) //nolint:errcheck // Fortschrittszeile
	return titelIDs, nil
}

func legeExemplareAn(ctx context.Context, pool *pgxpool.Pool, titelIDs []uuid.UUID, gesamt, portion int, ausgabe io.Writer) error {
	for i := 0; i < gesamt; i += portion {
		// Die letzte Portion ist kürzer, wenn gesamt kein Vielfaches von portion ist; sonst
		// entstünden mehr Exemplare als bestellt (TestFuelle_LegtGenauDenUmfangAn).
		bis := min(portion, gesamt-i)
		stapel := &pgx.Batch{}
		for j := 1; j <= bis; j++ {
			barcode := fmt.Sprintf("B%07d", i+j) // Barcodes wie B0000001
			titelID := titelIDs[rand.Intn(len(titelIDs))]

			stapel.Queue(`
				INSERT INTO buecher_exemplare (barcode_id, titel_id, ist_ausleihbar)
				VALUES ($1, $2, true)
				ON CONFLICT (barcode_id) DO NOTHING
			`, barcode, titelID)
		}

		ergebnis := pool.SendBatch(ctx, stapel)
		if err := ergebnis.Close(); err != nil {
			return fmt.Errorf("exemplare einfügen (Portion ab %d): %w", i, err)
		}
		fmt.Fprintf(ausgabe, "⏳ %d / %d Exemplare eingefügt...\n", i+bis, gesamt) //nolint:errcheck // Fortschrittszeile
	}

	fmt.Fprintf(ausgabe, "✅ Alle %d Exemplare erfolgreich generiert.\n", gesamt) //nolint:errcheck // Fortschrittszeile
	return nil
}

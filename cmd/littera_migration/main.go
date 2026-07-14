//go:build odbc

// Dieses Einmal-Migrationstool liest aus Litteras ODBC-/Access-Quelle und benötigt
// daher CGO + unixODBC-Header (sql.h). Es ist hinter dem Build-Tag "odbc" versteckt,
// damit der Standard-Build (go build ./..., CI, Linter) ohne unixODBC funktioniert.
// Bauen bei Bedarf mit:  go build -tags odbc ./cmd/littera_migration
package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"

	_ "github.com/alexbrainman/odbc"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	log.Println("Starte MS Access -> PostgreSQL Migrationsskript...")

	accessDSN := os.Getenv("ACCESS_DSN")
	if accessDSN == "" {
		accessDSN = "Driver={Microsoft Access Driver (*.mdb, *.accdb)};Dbq=littera.mdb;"
		log.Printf("Kein ACCESS_DSN gefunden, verwende Fallback: %s\n", accessDSN)
	}

	pgURL := os.Getenv("DATABASE_URL")
	if pgURL == "" {
		pgURL = "postgres://postgres:postgres@localhost:5432/bibliothek?sslmode=disable"
		log.Printf("Kein DATABASE_URL gefunden, verwende Fallback: %s\n", pgURL)
	}

	// 1. Mit Access verbinden
	log.Println("Verbinde mit MS Access Datenbank...")
	accessDB, err := sql.Open("odbc", accessDSN)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der Access-Datenbank: %v", err)
	}
	defer accessDB.Close()

	if err := accessDB.Ping(); err != nil {
		log.Fatalf("Ping zur Access-Datenbank fehlgeschlagen: %v", err)
	}

	// 2. Mit PostgreSQL verbinden
	log.Println("Verbinde mit PostgreSQL Datenbank...")
	pgDB, err := sql.Open("pgx", pgURL)
	if err != nil {
		log.Fatalf("Fehler beim Öffnen der PostgreSQL-Datenbank: %v", err)
	}
	defer pgDB.Close()

	if err := pgDB.Ping(); err != nil {
		log.Fatalf("Ping zur PostgreSQL-Datenbank fehlgeschlagen: %v", err)
	}

	// 3. PostgreSQL Transaktion starten
	ctx := context.Background()
	tx, err := pgDB.BeginTx(ctx, nil)
	if err != nil {
		log.Fatalf("Fehler beim Starten der Postgres-Transaktion: %v", err)
	}
	defer tx.Rollback() // Sicherer Rollback, falls Commit nicht erreicht wird

	log.Println("PostgreSQL Transaktion gestartet.")

	// 4. Metadaten (TITEL) migrieren
	titleMap := migriereTitel(ctx, tx, accessDB)

	// 5. Exemplare migrieren
	migriereExemplare(ctx, tx, accessDB, titleMap)

	// 6. Transaktion abschließen
	log.Println("Schließe PostgreSQL Transaktion ab (Commit)...")
	if err := tx.Commit(); err != nil {
		log.Fatalf("Fehler beim Commit der Transaktion: %v", err)
	}

	log.Println("Migration erfolgreich abgeschlossen!")
}

// migriereTitel liest die Access-TITEL-Tabelle und upsertet sie nach buecher_titel.
// Rückgabe: Zuordnung Access-TitelID → Postgres-TitelID für die Exemplar-Migration.
func migriereTitel(ctx context.Context, tx *sql.Tx, accessDB *sql.DB) map[string]string {
	log.Println("Lese Metadaten aus Access (TITEL)...")
	// WICHTIG: Ersetze 'TITEL' und die Spaltennamen durch die exakten Namen deiner Access-Tabelle.
	// Wir nehmen hier Standardnamen an, passend zu den üblichen Strukturierungen.
	titelQuery := `SELECT TitelID, Titel, Autor, ISBN, Verlag, Jahr, Signatur FROM TITEL`
	rowsTitel, err := accessDB.Query(titelQuery)
	if err != nil {
		log.Fatalf("Fehler beim Abrufen der TITEL: %v", err)
	}
	defer rowsTitel.Close()

	// Map Access TitelID -> Postgres UUID (bzw. ID), falls in Postgres eine serial ID verwendet wird.
	// Wir speichern hier die Zuordnung Access ID -> ISBN oder wir machen einen CTE beim Insert.
	// Da buecher_titel einen Serial 'id' nutzt, können wir die neue ID per RETURNING abfragen.
	titleMap := make(map[string]string) // Access_TitelID -> Postgres_TitelID (string representation)

	insertTitelStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO buecher_titel (titel, autor, isbn, verlag, erscheinungsjahr, signatur)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, 0), $6)
		ON CONFLICT (isbn) DO UPDATE SET
			titel = EXCLUDED.titel,
			autor = EXCLUDED.autor,
			verlag = EXCLUDED.verlag,
			erscheinungsjahr = EXCLUDED.erscheinungsjahr,
			signatur = EXCLUDED.signatur
		RETURNING id
	`)
	if err != nil {
		log.Fatalf("Fehler beim Prepare des Titel-Inserts: %v", err)
	}
	defer insertTitelStmt.Close()

	var countTitel int
	for rowsTitel.Next() {
		var accessID string
		var titel, autor, isbn, verlag, signatur sql.NullString
		var jahr sql.NullInt32

		if err := rowsTitel.Scan(&accessID, &titel, &autor, &isbn, &verlag, &jahr, &signatur); err != nil {
			log.Fatalf("Fehler beim Scannen der Titelzeile: %v", err)
		}

		var newPgID string
		cleanIsbn := strings.TrimSpace(isbn.String)
		err := insertTitelStmt.QueryRowContext(ctx,
			titel.String,
			autor.String,
			cleanIsbn,
			verlag.String,
			jahr.Int32,
			signatur.String,
		).Scan(&newPgID)

		if err != nil {
			log.Printf("Warnung: Konnte Titel '%s' (AccessID: %s) nicht einfügen: %v", titel.String, accessID, err)
			continue
		}

		titleMap[accessID] = newPgID
		countTitel++
		if countTitel%500 == 0 {
			log.Printf("%d Titel verarbeitet...", countTitel)
		}
	}
	if err := rowsTitel.Err(); err != nil {
		log.Fatalf("Fehler beim Lesen der TITEL: %v", err)
	}
	log.Printf("Erfolgreich %d Metadaten-Datensätze verarbeitet.\n", countTitel)
	return titleMap
}

// migriereExemplare liest die Access-EXEMPLARE-Tabelle und schreibt die Exemplare den
// (über titleMap aufgelösten) Postgres-Titeln zu.
func migriereExemplare(ctx context.Context, tx *sql.Tx, accessDB *sql.DB, titleMap map[string]string) {
	log.Println("Lese Exemplare aus Access (EXEMPLARE)...")
	// WICHTIG: Passe den Tabellennamen an deine Access-DB an!
	exemplareQuery := `SELECT ExemplarID, TitelID, Barcode, ErworbenAm FROM EXEMPLARE`
	rowsExemplare, err := accessDB.Query(exemplareQuery)
	if err != nil {
		log.Fatalf("Fehler beim Abrufen der EXEMPLARE: %v", err)
	}
	defer rowsExemplare.Close()

	insertExemplarStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO buecher_exemplare (titel_id, barcode_id, erworben_am)
		VALUES ($1, $2, $3)
		ON CONFLICT (barcode_id) DO NOTHING
	`)
	if err != nil {
		log.Fatalf("Fehler beim Prepare des Exemplar-Inserts: %v", err)
	}
	defer insertExemplarStmt.Close()

	var countExemplare int
	for rowsExemplare.Next() {
		if verarbeiteExemplarZeile(ctx, insertExemplarStmt, titleMap, rowsExemplare) {
			countExemplare++
			if countExemplare%500 == 0 {
				log.Printf("%d Exemplare verarbeitet...", countExemplare)
			}
		}
	}
	if err := rowsExemplare.Err(); err != nil {
		log.Fatalf("Fehler beim Lesen der EXEMPLARE: %v", err)
	}
	log.Printf("Erfolgreich %d Exemplar-Datensätze verarbeitet.\n", countExemplare)
}

// verarbeiteExemplarZeile scannt eine Exemplarzeile, löst den zugehörigen Titel auf und
// fügt das Exemplar ein. Liefert true, wenn ein Exemplar tatsächlich eingefügt wurde
// (fehlender Titel, leerer Barcode oder Insert-Fehler ⇒ false, übersprungen).
func verarbeiteExemplarZeile(ctx context.Context, stmt *sql.Stmt, titleMap map[string]string, rows *sql.Rows) bool {
	var exID, tID string
	var barcode sql.NullString
	var erworbenAm sql.NullTime

	// Wir lesen Barcode bewusst als String via sql.NullString, um führende Nullen zu behalten!
	if err := rows.Scan(&exID, &tID, &barcode, &erworbenAm); err != nil {
		log.Fatalf("Fehler beim Scannen der Exemplarzeile: %v", err)
	}

	pgTitelID, ok := titleMap[tID]
	if !ok {
		// Titel wurde scheinbar nicht migriert (z.B. weil Metadaten defekt oder fehlten)
		log.Printf("Überspringe Exemplar %s, da zugehöriger Titel (AccessID: %s) nicht gefunden wurde.", barcode.String, tID)
		return false
	}

	cleanBarcode := strings.TrimSpace(barcode.String)
	if cleanBarcode == "" {
		return false // Kein Barcode -> kein valides Exemplar im System
	}

	var pTime interface{}
	if erworbenAm.Valid {
		pTime = erworbenAm.Time
	}

	if _, err := stmt.ExecContext(ctx, pgTitelID, cleanBarcode, pTime); err != nil {
		log.Printf("Warnung: Konnte Exemplar (Barcode: %s) nicht einfügen: %v", cleanBarcode, err)
		return false
	}
	return true
}

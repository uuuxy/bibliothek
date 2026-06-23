package api

import (
	"log"

	"github.com/jackc/pgx/v5/pgconn"
)

// logExec protokolliert einen fehlgeschlagenen "fire-and-forget"-Schreibvorgang
// (z. B. Audit- oder Idempotenz-Inserts), ohne den bereits erfolgreich abgeschlossenen
// Hauptvorgang zu beeinflussen. Verwendung: logExec(s.DB.Pool.Exec(ctx, ...)).
func logExec(_ pgconn.CommandTag, err error) {
	if err != nil {
		log.Printf("audit/idempotenz: schreibvorgang fehlgeschlagen: %v", err)
	}
}

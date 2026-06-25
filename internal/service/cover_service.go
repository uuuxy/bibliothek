package service

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"bibliothek/db"
	"bibliothek/inventur"
)

// CoverService handles fetching book covers asynchronously.
type CoverService struct {
	db db.PgxPoolIface
}

// NewCoverService creates a new CoverService.
func NewCoverService(dbPool db.PgxPoolIface) *CoverService {
	return &CoverService{
		db: dbPool,
	}
}

// coverSyncConcurrency bestimmt, wie viele Cover gleichzeitig von den externen Quellen
// (DNB/Google/OpenLibrary) geholt werden. Moderat gewählt: beseitigt den sequenziellen
// Flaschenhals (bei ~20k Titeln von Stunden auf Minuten), ohne die APIs zu überlasten.
const coverSyncConcurrency = 8

// coverSyncRunning verhindert überlappende Sync-Läufe (Start, Cron und manueller
// Trigger erzeugen jeweils eine neue CoverService-Instanz) über einen prozessweiten Guard.
var coverSyncRunning atomic.Bool

type missingCover struct {
	ID   string
	ISBN string
}

// SyncMissingCoversAsync lädt für alle Titel ohne lokales Cover die Cover parallel nach.
// Es werden PENDING- (noch nie versucht) UND FAILED-Titel (erneuter Versuch) verarbeitet,
// sodass transiente Fehler bei einem späteren Lauf automatisch nachgeholt werden.
func (s *CoverService) SyncMissingCoversAsync() {
	if !coverSyncRunning.CompareAndSwap(false, true) {
		log.Println("Cover Sync: Lauf bereits aktiv – überspringe.")
		return
	}
	defer coverSyncRunning.Store(false)

	ctx := context.Background()

	// Verarbeitet: noch nie versuchte (PENDING), fehlgeschlagene (FAILED, erneuter Versuch)
	// sowie Alt-Titel mit externer Cover-URL (FOUND, aber nicht lokal) — diese werden auf
	// ein lokales WebP migriert, damit nichts mehr extern (mit Hotlink-/Bot-Risiko) lädt.
	query := `
		SELECT id, isbn FROM buecher_titel
		WHERE isbn IS NOT NULL AND isbn != ''
		  AND (
		        cover_status IN ('PENDING', 'FAILED')
		     OR (cover_url IS NOT NULL AND cover_url <> '' AND cover_url NOT LIKE '/uploads/%')
		      )`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		log.Printf("Cover Sync: Fehler beim Abrufen der fehlenden Cover: %v", err)
		return
	}

	var missing []missingCover
	for rows.Next() {
		var mc missingCover
		if err := rows.Scan(&mc.ID, &mc.ISBN); err == nil {
			missing = append(missing, mc)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		log.Printf("Cover Sync: Fehler beim Lesen der fehlenden Cover: %v", err)
		return
	}
	rows.Close()

	if len(missing) == 0 {
		log.Println("Cover Sync: Keine fehlenden Cover gefunden.")
		return
	}

	log.Printf("Cover Sync: Starte parallelen Download für %d Cover (%d Worker)...", len(missing), coverSyncConcurrency)

	client := inventur.NeuerMetadatenClient()

	var found, notFound, failed atomic.Int64
	jobs := make(chan missingCover)
	var wg sync.WaitGroup

	for i := 0; i < coverSyncConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for mc := range jobs {
				s.processCover(ctx, client, mc, &found, &notFound, &failed)
			}
		}()
	}

	for _, mc := range missing {
		jobs <- mc
	}
	close(jobs)
	wg.Wait()

	log.Printf("Cover Sync: Abgeschlossen. Gefunden: %d, Nicht gefunden: %d, Fehlgeschlagen: %d",
		found.Load(), notFound.Load(), failed.Load())
}

// processCover verarbeitet einen einzelnen Titel: Cover suchen (lädt es bei Erfolg als
// lokales WebP), und den Status in der Datenbank entsprechend setzen.
func (s *CoverService) processCover(ctx context.Context, client *inventur.MetadatenClient, mc missingCover, found, notFound, failed *atomic.Int64) {
	reqCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	res, err := client.SucheNachISBN(reqCtx, mc.ISBN)
	cancel()

	switch {
	case err != nil:
		failed.Add(1)
		s.setCoverStatus(ctx, mc.ID, "FAILED")
	case res.CoverURL != "":
		found.Add(1)
		if _, derr := s.db.Exec(ctx, `UPDATE buecher_titel SET cover_url = $1, cover_status = 'FOUND' WHERE id = $2`, res.CoverURL, mc.ID); derr != nil {
			log.Printf("Cover Sync: DB-Update für Titel %s fehlgeschlagen: %v", mc.ID, derr)
		}
	default:
		notFound.Add(1)
		s.setCoverStatus(ctx, mc.ID, "NOT_FOUND")
	}
}

// setCoverStatus aktualisiert den cover_status eines Titels (Best-Effort, geloggt).
func (s *CoverService) setCoverStatus(ctx context.Context, id, status string) {
	if _, err := s.db.Exec(ctx, `UPDATE buecher_titel SET cover_status = $1 WHERE id = $2`, status, id); err != nil {
		log.Printf("Cover Sync: Status %q für Titel %s konnte nicht gesetzt werden: %v", status, id, err)
	}
}

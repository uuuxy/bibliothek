package service

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/pkg/safego"
	"bibliothek/repository"
)

// CoverDienst handles fetching book covers asynchronously.
type CoverDienst struct {
	pool db.PgxPoolIface
	// katalog baut den Client für die Katalogdienste. Ein Test stellt die Dienste darüber
	// nach, statt DNB, Google Books und OpenLibrary im Netz zu fragen.
	katalog func() *inventur.MetadatenClient
}

// NewCoverDienst creates a new CoverDienst.
func NewCoverDienst(pool db.PgxPoolIface) *CoverDienst {
	return &CoverDienst{
		pool:    pool,
		katalog: inventur.NeuerMetadatenClient,
	}
}

// coverAbgleichArbeiter bestimmt, wie viele Cover gleichzeitig von den externen Quellen
// (DNB/Google/OpenLibrary) geholt werden. Moderat gewählt: beseitigt den sequenziellen
// Flaschenhals (bei ~20k Titeln von Stunden auf Minuten), ohne die APIs zu überlasten.
const coverAbgleichArbeiter = 8

// coverAbgleichTakt drosselt die Titel-Rate GLOBAL über alle Worker. Pro Titel fallen
// bis zu ~5 externe HTTP-Calls an (DNB-SRU-Lookup + Cover-Kandidaten). Ungedrosselt
// hämmern 8 Worker beim Altbestands-Backfill (~14.000 Littera-Titel) im zweistelligen
// req/s-Bereich auf die DNB — ein IP-Block dort würde auch den ISBN-Lookup am
// Ausleih-Pult lahmlegen, der über dieselbe Quell-IP läuft. 2 Titel/s ⇒ der komplette
// Altbestand ist in ~2 h durch; die Parallelität glättet dabei nur langsame Downloads.
const coverAbgleichTakt = 500 * time.Millisecond

// coverAbgleichLaeuft verhindert überlappende Sync-Läufe (Start, Cron und manueller
// Trigger erzeugen jeweils eine neue CoverDienst-Instanz) über einen prozessweiten Guard.
var coverAbgleichLaeuft atomic.Bool

// fehlendesCover ist ein Titel, dessen Cover der Abgleich holen soll.
type fehlendesCover = repository.TitelMitISBN

// GleicheFehlendeCoverAb lädt für alle Titel ohne lokales Cover die Cover parallel nach.
// Es werden PENDING- (noch nie versucht) UND FAILED-Titel (erneuter Versuch) verarbeitet,
// sodass transiente Fehler bei einem späteren Lauf automatisch nachgeholt werden.
func (d *CoverDienst) GleicheFehlendeCoverAb() {
	if !coverAbgleichLaeuft.CompareAndSwap(false, true) {
		log.Println("Cover Sync: Lauf bereits aktiv – überspringe.")
		return
	}
	defer coverAbgleichLaeuft.Store(false)
	// Panik-Netz für die ÄUSSERE Goroutine (nicht nur die Worker unten): Diese Funktion
	// läuft als `go GleicheFehlendeCoverAb()` ohne HTTP-Recovery. Ein Panik hier — etwa
	// beim Lesen der DB-Zeilen oder im Dispatch — risse sonst den gesamten Prozess mit.
	// Steht NACH dem Store(false)-Defer, damit auf einem Panik BEIDES läuft: recovern UND
	// das Running-Flag zurücksetzen (sonst überspringt jeder künftige Lauf für immer).
	defer safego.FangePanic("cover-sync")

	ctx := context.Background()

	// Verarbeitet: noch nie versuchte (PENDING), fehlgeschlagene (FAILED, erneuter Versuch)
	// sowie Alt-Titel mit externer Cover-URL (FOUND, aber nicht lokal) — diese werden auf
	// ein lokales WebP migriert, damit nichts mehr extern (mit Hotlink-/Bot-Risiko) lädt.
	fehlende, err := repository.LadeTitelFuerCoverAbgleich(ctx, d.pool)
	if err != nil {
		log.Printf("Cover Sync: Fehler beim Abrufen der fehlenden Cover: %v", err)
		return
	}

	if len(fehlende) == 0 {
		log.Println("Cover Sync: Keine fehlenden Cover gefunden.")
		return
	}

	log.Printf("Cover Sync: Starte parallelen Download für %d Cover (%d Worker)...", len(fehlende), coverAbgleichArbeiter)

	metadaten := d.katalog()

	var gefunden, nichtGefunden, gescheitert atomic.Int64
	// Gepuffert auf die Zahl der Aufgaben: Der Dispatcher blockiert damit NIE beim Senden.
	// Stirbt ein Worker (Panik, von seinem eigenen Guard gefangen) und liest nicht mehr,
	// hinge ein ungepufferter Kanal beim `jobs <- mc` für immer — und coverAbgleichLaeuft
	// bliebe true, sodass jeder weitere Lauf still übersprungen würde.
	aufgaben := make(chan fehlendesCover, len(fehlende))
	var wg sync.WaitGroup

	// Globale Drossel: jeder Worker wartet vor JEDEM Titel auf den Ticker —
	// damit ist die Gesamtrate unabhängig von der Worker-Zahl hart begrenzt.
	takt := time.NewTicker(coverAbgleichTakt)
	defer takt.Stop()

	for i := 0; i < coverAbgleichArbeiter; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer safego.FangePanic("cover-sync-worker")
			for titel := range aufgaben {
				<-takt.C
				d.verarbeiteCover(ctx, metadaten, titel, &gefunden, &nichtGefunden, &gescheitert)
			}
		}()
	}

	for _, titel := range fehlende {
		aufgaben <- titel
	}
	close(aufgaben)
	wg.Wait()

	log.Printf("Cover Sync: Abgeschlossen. Gefunden: %d, Nicht gefunden: %d, Fehlgeschlagen: %d",
		gefunden.Load(), nichtGefunden.Load(), gescheitert.Load())
}

// verarbeiteCover verarbeitet einen einzelnen Titel: Cover suchen (lädt es bei Erfolg als
// lokales WebP), und den Status in der Datenbank entsprechend setzen.
func (d *CoverDienst) verarbeiteCover(ctx context.Context, metadaten *inventur.MetadatenClient, titel fehlendesCover, gefunden, nichtGefunden, gescheitert *atomic.Int64) {
	anfrageCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	treffer, err := metadaten.SucheNachISBN(anfrageCtx, titel.ISBN)
	cancel()

	switch {
	case err != nil:
		gescheitert.Add(1)
		d.vermerkeCoverStatus(ctx, titel.ID, "FAILED")
	case treffer.CoverURL != "":
		gefunden.Add(1)
		if schreibFehler := repository.SetzeGefundenesCover(ctx, d.pool, titel.ID, treffer.CoverURL); schreibFehler != nil {
			log.Printf("Cover Sync: DB-Update für Titel %s fehlgeschlagen: %v", titel.ID, schreibFehler)
		}
	default:
		nichtGefunden.Add(1)
		d.vermerkeCoverStatus(ctx, titel.ID, "NOT_FOUND")
	}
}

// vermerkeCoverStatus aktualisiert den cover_status eines Titels (Best-Effort, geloggt).
func (d *CoverDienst) vermerkeCoverStatus(ctx context.Context, id, status string) {
	if err := repository.SetzeCoverStatus(ctx, d.pool, id, status); err != nil {
		log.Printf("Cover Sync: Status %q für Titel %s konnte nicht gesetzt werden: %v", status, id, err)
	}
}

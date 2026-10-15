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

// CoverDienst holt die Cover der Titel im Hintergrund von den Katalogdiensten.
type CoverDienst struct {
	pool db.Pool
	// katalog baut den Client für die Katalogdienste. Ein Test stellt die Dienste darüber
	// nach, statt DNB, Google Books und OpenLibrary im Netz zu fragen.
	katalog func() *inventur.MetadatenClient
	// laeuft verhindert, dass zwei Läufe sich überlappen. Start, Zeitplan und der Knopf in der
	// Verwaltung teilen sich deshalb einen Dienst (main.go).
	laeuft atomic.Bool
}

// NewCoverDienst baut den Dienst für den Abgleich der Cover.
func NewCoverDienst(pool db.Pool) *CoverDienst {
	return &CoverDienst{
		pool:    pool,
		katalog: inventur.NewMetadatenClient,
	}
}

// coverAbgleichArbeiter ist die Zahl der Cover, die gleichzeitig von DNB, Google und
// OpenLibrary geholt werden: genug, dass rund 20.000 Titel in Minuten statt Stunden
// durchlaufen, und wenig genug für die Dienste.
const coverAbgleichArbeiter = 8

// coverAbgleichTakt drosselt alle Arbeiter zusammen auf zwei Titel je Sekunde. Je Titel fallen
// bis zu fünf Anfragen an; ungedrosselt fragten acht Arbeiter die DNB beim Altbestand (rund
// 14.000 Titel) zweistellig oft je Sekunde, und eine Sperre dort träfe auch die Suche nach der
// ISBN an der Theke, die über dieselbe Adresse läuft. Der ganze Altbestand ist so in etwa zwei
// Stunden durch.
const coverAbgleichTakt = 500 * time.Millisecond

// fehlendesCover ist ein Titel, dessen Cover der Abgleich holen soll.
type fehlendesCover = repository.TitelMitISBN

// GleicheFehlendeCoverAb holt für alle Titel ohne eigenes Cover die Cover nach, mehrere
// zugleich. Auch Titel, deren letzter Versuch scheiterte, kommen wieder dran, damit ein
// vorübergehender Fehler beim nächsten Lauf verschwindet.
func (d *CoverDienst) GleicheFehlendeCoverAb() {
	if !d.laeuft.CompareAndSwap(false, true) {
		log.Println("Cover Sync: Lauf bereits aktiv – überspringe.")
		return
	}
	defer d.laeuft.Store(false)
	// Das Netz für die äußere Goroutine, nicht nur für die Arbeiter: Die Funktion läuft als `go
	// d.GleicheFehlendeCoverAb()` ohne die Erholung des HTTP-Servers, und eine Panik risse
	// sonst den Prozess mit. Es steht nach dem Defer, der den Schalter zurücksetzt, damit bei
	// einer Panik beides läuft; sonst übersprängen alle künftigen Läufe.
	defer safego.FangePanic("cover-sync")

	ctx := context.Background()

	// Die Auswahl: noch nie versucht (PENDING), gescheitert (FAILED) und ältere Titel mit
	// fremder Cover-Adresse (FOUND, aber nicht lokal). Diese bekommen ein eigenes WebP, damit
	// kein Cover mehr von fremden Servern lädt.
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
	// Gepuffert auf die Zahl der Aufgaben, damit das Verteilen nie beim Senden wartet: Stirbt
	// ein Arbeiter (eine Panik, die sein eigenes Netz fängt) und liest nicht mehr, hinge ein
	// ungepufferter Kanal beim `aufgaben <- titel` für immer, und laeuft bliebe
	// gesetzt, sodass jeder weitere Lauf still übersprungen würde.
	aufgaben := make(chan fehlendesCover, len(fehlende))
	var wg sync.WaitGroup

	// Jeder Arbeiter wartet vor jedem Titel auf den Takt; so hängt die Rate nicht an der Zahl
	// der Arbeiter.
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

// verarbeiteCover sucht das Cover eines Titels, legt es bei Erfolg als WebP ab und vermerkt den
// Stand in der Datenbank.
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

// vermerkeCoverStatus setzt cover_status eines Titels. Ein Fehler steht nur im Log: Der nächste
// Lauf nimmt den Titel ohnehin wieder.
func (d *CoverDienst) vermerkeCoverStatus(ctx context.Context, id, status string) {
	if err := repository.SetzeCoverStatus(ctx, d.pool, id, status); err != nil {
		log.Printf("Cover Sync: Status %q für Titel %s konnte nicht gesetzt werden: %v", status, id, err)
	}
}

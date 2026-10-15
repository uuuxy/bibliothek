package katalog

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"bibliothek/pkg/coverquelle"
	"bibliothek/pkg/isbnutil"
	"bibliothek/repository"
)

// Die Tür Cover nimmt ohne Anmeldung zwei voneinander unabhängige Angaben entgegen: die ISBN
// als Name im Cache und die Adresse des Bilds. Bestimmte die erste Anfrage, welches Bild unter
// einer ISBN liegt, vergiftete ein Aufruf mit der ISBN eines Schulbuchs und der Adresse eines
// beliebigen anderen Covers den Cache, und der Monitor im Flur zeigte dieses Bild; jede neue
// Adresse legte zudem eine Datei an, die nichts wieder räumt.
//
// Beides schließt dieselbe Regel: Der Server liefert nur Bilder aus, deren Adresse er für diese
// ISBN selbst kennt oder herleiten kann, und der Dateiname ist ISBN plus Hash der geprüften
// Adresse, sodass sich zwei Adressen nie eine Datei teilen. Die Prüfung dazu steht in
// cover_quelle_bindung_test.go.

// coverKandidatenFuerISBN sind die Adressen, die coverKandidaten (frontend/src/lib/utils/
// coverSrc.js) für eine ISBN selbst baut. Muster hier und dort müssen gleich bleiben. Die
// Adresse von Google Books trägt kein leeres `id=`: Mit ihm antwortet Google auf jede ISBN
// mit dem Ersatzbild.
func coverKandidatenFuerISBN(isbnSauber string) []string {
	return []string{
		fmt.Sprintf("https://books.google.com/books/content?vid=ISBN:%s&printsec=frontcover&img=1&zoom=1", isbnSauber),
		fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg", isbnSauber),
	}
}

// coverDateiname bindet die Datei an ISBN und Adresse.
func coverDateiname(isbn, sichereURL string) string {
	sum := sha256.Sum256([]byte(sichereURL))
	return isbnutil.OhneTrenner(isbn) + "-" + hex.EncodeToString(sum[:6]) + ".webp"
}

// istCoverQuelleErlaubt entscheidet, ob sichereURL für diese ISBN heruntergeladen werden
// darf. Erlaubt ist eine im Katalog gespeicherte Adresse dieser ISBN, oder eine der beiden
// herleitbaren Kandidaten-Adressen, diese ohne Anmeldung nur für ISBNs, die es im
// Katalog gibt (die ISBN-Suche beim Anlegen eines Titels läuft angemeldet). Fehler zählen
// als „nein": im Zweifel kein Download.
func (t *Tueren) istCoverQuelleErlaubt(r *http.Request, isbn, sichereURL string) bool {
	if t.DB == nil || t.DB.Pool == nil {
		return false
	}
	isbnSauber := isbnutil.OhneTrenner(isbn)
	quellen, imKatalog, err := repository.LadeCoverQuellenFuerISBN(r.Context(), t.DB.Pool, isbnSauber)
	if err != nil {
		return false
	}
	for _, q := range quellen {
		if gepr, ok := coverquelle.SichereURL(q, coverquelle.BildHosts); ok && gepr == sichereURL {
			return true
		}
	}
	for _, k := range coverKandidatenFuerISBN(isbnSauber) {
		if gepr, ok := coverquelle.SichereURL(k, coverquelle.BildHosts); ok && gepr == sichereURL {
			return imKatalog || t.istAngemeldet(r)
		}
	}
	return false
}

// istAngemeldet prüft die Sitzung, ohne den Zugriff zu erzwingen — der Endpunkt bleibt
// öffentlich, die Anmeldung erweitert nur, was heruntergeladen werden darf.
func (t *Tueren) istAngemeldet(r *http.Request) bool {
	if t.Auth == nil {
		return false
	}
	_, _, err := t.PruefeSitzung(r)
	return err == nil
}

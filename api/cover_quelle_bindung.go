package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"bibliothek/pkg/coverquelle"
	"bibliothek/pkg/isbnutil"
	"bibliothek/repository"
)

// Der Cover-Proxy nahm bis zum 05.09.2026 zwei voneinander unabhängige Angaben entgegen:
// den Cache-Schlüssel (isbn) und den Inhalt (url). Wer zuerst kam, bestimmte dauerhaft,
// welches Bild unter dieser ISBN ausgeliefert wird — ohne Anmeldung, denn der Endpunkt
// ist öffentlich. Ein Aufruf mit der ISBN eines Schulbuchs und der Adresse eines
// beliebigen anderen Covers bei Google Books vergiftete damit den Cache, und der
// Flur-Monitor zeigte fortan dieses Bild. Zweitens legte jeder ISBN-artige Schlüssel eine
// neue Datei an, ohne Räumung: ein Plattenfüller auf dem Host der Datenbank.
//
// Beides schließt dieselbe Regel: Der Server liefert nur Bilder aus, deren Adresse er
// für diese ISBN SELBST kennt oder selbst herleiten könnte. Dateiname ist ISBN plus Hash
// der geprüften URL — zwei Adressen teilen sich nie eine Datei. Gate: cover_quelle_bindung_test.go.

// coverKandidatenFuerISBN sind die Adressen, die coverKandidaten (frontend/src/lib/utils/
// coverSrc.js) für eine ISBN selbst baut. Muster hier und dort MÜSSEN gleich bleiben.
func coverKandidatenFuerISBN(isbnSauber string) []string {
	return []string{
		fmt.Sprintf("https://books.google.com/books/content?id=&vid=ISBN:%s&printsec=frontcover&img=1&zoom=1", isbnSauber),
		fmt.Sprintf("https://covers.openlibrary.org/b/isbn/%s-L.jpg", isbnSauber),
	}
}

// coverDateiname bindet die Datei an ISBN UND Adresse.
func coverDateiname(isbn, sichereURL string) string {
	sum := sha256.Sum256([]byte(sichereURL))
	return isbnutil.CleanISBN(isbn) + "-" + hex.EncodeToString(sum[:6]) + ".webp"
}

// coverQuelleErlaubt entscheidet, ob sichereURL für diese ISBN heruntergeladen werden
// darf. Erlaubt ist eine im Katalog gespeicherte Adresse dieser ISBN, oder eine der beiden
// herleitbaren Kandidaten-Adressen — letztere ohne Anmeldung nur für ISBNs, die es im
// Katalog gibt (die ISBN-Suche beim Anlegen eines Titels läuft angemeldet). Fehler zählen
// als „nein": im Zweifel kein Download.
func (s *Server) coverQuelleErlaubt(r *http.Request, isbn, sichereURL string) bool {
	if s.DB == nil || s.DB.Pool == nil {
		return false
	}
	isbnSauber := isbnutil.CleanISBN(isbn)
	quellen, imKatalog, err := repository.CoverQuellenFuerISBN(r.Context(), s.DB.Pool, isbnSauber)
	if err != nil {
		return false
	}
	for _, q := range quellen {
		if gepr, ok := coverquelle.SichereURL(q, coverquelle.CoverHosts); ok && gepr == sichereURL {
			return true
		}
	}
	for _, k := range coverKandidatenFuerISBN(isbnSauber) {
		if gepr, ok := coverquelle.SichereURL(k, coverquelle.CoverHosts); ok && gepr == sichereURL {
			return imKatalog || s.istAngemeldet(r)
		}
	}
	return false
}

// istAngemeldet prüft die Sitzung, ohne den Zugriff zu erzwingen — der Endpunkt bleibt
// öffentlich, die Anmeldung erweitert nur, was heruntergeladen werden darf.
func (s *Server) istAngemeldet(r *http.Request) bool {
	if s.Auth == nil {
		return false
	}
	_, _, err := s.claimsAusRequest(r)
	return err == nil
}

package service

import (
	"context"
	"fmt"
	"log"

	"bibliothek/db"
	"bibliothek/inventur"
	"bibliothek/pkg/isbnutil"
	"bibliothek/repository"
)

// BestellTreffer ist ein Treffer der Bestellsuche. Welche Felder gefüllt sind, hängt von der
// Quelle ab; die Kommentare der Felder sagen, welche nur bei source="local" gesetzt sind.
type BestellTreffer struct {
	ID       string `json:"id,omitempty"`
	Titel    string `json:"titel"`
	Autor    string `json:"autor"`
	ISBN     string `json:"isbn"`
	Verlag   string `json:"verlag,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// Signatur ist nur bei source="local" gesetzt: die Signatur eines schon katalogisierten
	// Titels. Ein Treffer der DNB hat noch keinen Titel und damit keine Signatur; sie entsteht
	// erst als Vorschlag beim Anlegen über /aus-isbn.
	Signatur    string `json:"signatur,omitempty"`
	Quelle      string `json:"source"`
	Bestand     int    `json:"current_stock,omitempty"`
	IstDublette bool   `json:"is_duplicate,omitempty"`
	// PreisVorschlag ist der Ladenpreis laut DNB (MARC21 020 $c), 0 heißt keiner. Er gilt
	// zum Erscheinen und ist nicht der Preis der Schule: Die Oberfläche füllt damit das
	// Preisfeld vor und übernimmt ihn nie ungefragt als Ausgabe.
	PreisVorschlag float64 `json:"preis_vorschlag,omitempty"`
	// IstLernmittel ist nur bei source="local" aussagekräftig: der Vorschlag für den Topf der
	// Bestellung (Lernmittel heißt Land, sonst Schulträger; mittel.js im Warenkorb). Ein
	// Treffer der DNB ist noch kein Titel und trägt false; beim Anlegen fragt die Oberfläche
	// nach.
	IstLernmittel bool `json:"ist_lernmittel"`
}

// FindeZumBestellen sucht einen Titel zum Bestellen im eigenen Katalog und bei der DNB. dnbAusfall
// sagt, dass die DNB nicht geantwortet hat: Die Liste trägt dann nur den eigenen Katalog, und
// ein Buch, das die DNB kennt, sähe ohne das Merkmal aus wie eines, das sie nicht kennt.
func FindeZumBestellen(ctx context.Context, pool db.PgxPoolIface, metadaten *inventur.MetadatenClient, text string) (treffer []BestellTreffer, dnbAusfall bool, err error) {
	treffer = findeImKatalog(ctx, pool, text)
	vonDNB, dnbAusfall := findeBeiDerDNB(ctx, pool, metadaten, text)
	return append(treffer, vonDNB...), dnbAusfall, nil
}

// findeImKatalog durchsucht den eigenen Katalog (repository.FindeTitelZumBestellen). Scheitert
// die Abfrage, bleibt die Liste leer: Die Treffer der DNB kommen ohnehin getrennt dazu.
func findeImKatalog(ctx context.Context, pool db.PgxPoolIface, text string) []BestellTreffer {
	treffer, err := repository.FindeTitelZumBestellen(ctx, pool, text)
	if err != nil {
		return nil
	}
	var liste []BestellTreffer
	for _, t := range treffer {
		liste = append(liste, BestellTreffer{
			ID:            t.ID,
			Titel:         t.Titel,
			Autor:         t.Autor,
			ISBN:          t.ISBN,
			Verlag:        t.Verlag,
			CoverURL:      t.CoverURL,
			Signatur:      t.Signatur,
			Quelle:        "local",
			Bestand:       t.Bestand,
			IstLernmittel: t.IstLernmittel,
		})
	}
	return liste
}

// findeBeiDerDNB fragt die DNB nach Titeln und markiert bereits lokal vorhandene ISBNs.
// Jeder Fehler der Abfrage ist ein Ausfall: Eine Suche ohne Treffer beantwortet die DNB mit
// einer leeren Liste, nicht mit einem Fehler.
func findeBeiDerDNB(ctx context.Context, pool db.PgxPoolIface, metadaten *inventur.MetadatenClient, text string) (treffer []BestellTreffer, ausfall bool) {
	dnbTreffer, dnbErr := metadaten.SucheTextDNB(ctx, text)
	if dnbErr != nil {
		log.Printf("bestellsuche: die DNB hat nicht geantwortet: %v", dnbErr)
		return nil, true
	}

	var isbns []string
	for _, dnb := range dnbTreffer {
		if n := isbnutil.Normalform(dnb.ISBN); n != "" {
			isbns = append(isbns, n)
		}
	}

	vorhanden := ladeVorhandeneISBNs(ctx, pool, isbns)

	for _, dnb := range dnbTreffer {
		treffer = append(treffer, baueDNBTreffer(dnb, vorhanden))
	}
	return treffer, false
}

// ladeVorhandeneISBNs fragt, welche der ISBNs schon ein Titel trägt, und liefert sie als
// Menge (repository.FindeISBNsImKatalog). Beide Seiten stehen in der Normalform: Die DNB nennt eine
// ISBN mit Bindestrichen und bei älteren Sätzen zehnstellig, der Katalog führt sie
// dreizehnstellig; der Treffer hieße sonst „Neu". Ein Fehler wird protokolliert und ergibt
// eine leere Menge; die Suche in der DNB scheitert daran nicht.
func ladeVorhandeneISBNs(ctx context.Context, pool db.PgxPoolIface, isbns []string) map[string]struct{} {
	vorhanden, err := repository.FindeISBNsImKatalog(ctx, pool, isbns)
	if err != nil {
		log.Printf("order-service: Bulk ISBN-Existenzprüfung fehlgeschlagen: %v", err)
		return map[string]struct{}{}
	}
	return vorhanden
}

// baueDNBTreffer macht aus einem Treffer der DNB einen Eintrag der Bestellsuche: ohne
// eigenes Cover die Cover-Adresse der DNB, und „Vorhanden", wenn die ISBN in der Menge der
// schon getragenen steht (ladeVorhandeneISBNs, Normalform).
func baueDNBTreffer(dnb inventur.MetadatenErgebnis, vorhanden map[string]struct{}) BestellTreffer {
	coverURL := dnb.CoverURL
	if coverURL == "" && dnb.ISBN != "" {
		coverURL = fmt.Sprintf("https://portal.dnb.de/opac/mvb/cover?isbn=%s", dnb.ISBN)
	}

	_, istVorhanden := vorhanden[isbnutil.Normalform(dnb.ISBN)]

	return BestellTreffer{
		Titel:          dnb.Titel,
		Autor:          dnb.Autor,
		ISBN:           dnb.ISBN,
		Verlag:         dnb.Verlag,
		CoverURL:       coverURL,
		Quelle:         "dnb",
		IstDublette:    istVorhanden,
		PreisVorschlag: dnb.Preis,
	}
}

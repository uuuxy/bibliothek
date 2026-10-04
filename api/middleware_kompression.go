package api

import (
	"net/http"

	"github.com/klauspost/compress/gzhttp"
)

// kompressionMindestgroesse: Kleinere Antworten passen ohnehin in ein Netzwerkpaket, das
// Packen spart dort nichts. Auch die Antwort mit dem CSRF-Token liegt darunter.
const kompressionMindestgroesse = 1024

// kompressionInhaltstypen sind die Textformen, die der Server ausliefert: die Antworten
// der Schnittstelle, der CSV-Export des Bestands und die Dateien der Oberfläche. Bilder,
// PDF und Excel sind in sich schon gepackt und gehen unverändert hinaus.
var kompressionInhaltstypen = []string{
	"application/json",
	"text/html",
	"text/css",
	"text/javascript",
	"image/svg+xml",
	"text/csv",
}

// KompressionMiddleware packt Antworten mit gzip, wenn der Client es anbietet. Der Proxy
// vor der Anwendung packt nicht; ohne dieses Glied geht die Titelliste des Katalogs mit
// mehreren Megabyte durchs Schulnetz.
//
// Der Live-Strom läuft daran vorbei: Die Hülle hält die ersten Bytes einer Antwort zurück,
// bis ihre Größe feststeht, und ein Strom soll sofort beim Client ankommen.
func KompressionMiddleware(next http.Handler) http.Handler {
	// Nur gzip: An Titelliste und Oberfläche gemessen liegen gzip und zstd dicht
	// beieinander, und jede weitere Kodierung ist ein weiterer Weg, den Proxy und
	// Werkzeuge verstehen müssen.
	huelle, err := gzhttp.NewWrapper(
		gzhttp.EnableZstd(false),
		gzhttp.MinSize(kompressionMindestgroesse),
		gzhttp.ContentTypes(kompressionInhaltstypen),
	)
	if err != nil {
		panic("Kompression: " + err.Error())
	}
	gepackt := huelle(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if istLiveStrom(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		gepackt.ServeHTTP(w, r)
	})
}

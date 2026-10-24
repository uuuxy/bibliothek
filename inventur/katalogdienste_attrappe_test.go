package inventur

import "net/http"

// metadatenUeber baut den Client der Katalogdienste mit dem HTTP-Client eines Tests, über die
// Naht, die auch die Tests anderer Pakete nehmen.
func metadatenUeber(httpClient *http.Client) *MetadatenClient {
	metadaten := NewMetadatenClient()
	metadaten.SetzeHTTPClientFuerTest(httpClient)
	return metadaten
}

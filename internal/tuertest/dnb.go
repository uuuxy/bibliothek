package tuertest

import (
	"io"
	"net/http"
	"strings"
)

// DNBAttrappe beantwortet die Anfrage an die DNB mit einem festen Satz und dem Status (ohne
// Angabe 200). Jede andere Adresse, etwa eine Cover-Quelle, gibt es bei ihr nicht.
type DNBAttrappe struct {
	Satz   string
	Status int
}

// RoundTrip macht die Attrappe zum Transport eines http.Client.
func (d DNBAttrappe) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Host == "services.dnb.de" {
		status := d.Status
		if status == 0 {
			status = http.StatusOK
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(d.Satz)), Header: http.Header{}, Request: r}, nil
	}
	return &http.Response{StatusCode: http.StatusNotFound, Body: http.NoBody, Header: http.Header{}, Request: r}, nil
}

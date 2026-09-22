package inventur

import "bibliothek/pkg/coverquelle"

// Die Host-Allowlist stand bis zum 04.08.2026 hier als eigene Kopie, mit einem
// Kommentar, der zur Pflege der Zwillingsliste in api/image_caching.go aufrief. Genau
// diese Konstruktion — Gleichheit, die nur ein Kommentar behauptet — war bereits
// auseinandergelaufen: Die dritte Kopie in metadaten_client.go kannte books.google.*
// nicht. Alle drei fragen jetzt pkg/coverquelle.

// IstErlaubteCoverHerkunft (nur speichern, nicht abrufen) ist am 22.09.2026 mit dem
// manuellen Cover-Update PUT /api/books/{id}/cover gefallen — der einzigen Stelle, die
// eine URL nur speicherte. Wer eine Cover-URL anfasst, ruft sie auch ab und nimmt
// SichereCoverURL.

// SichereCoverURL liefert die aus geprüften Teilen neu gebaute Cover-URL.
// ok=false heißt: Host nicht erlaubt oder URL unlesbar.
func SichereCoverURL(rohURL string) (string, bool) {
	return coverquelle.SichereURL(rohURL, coverquelle.CoverHosts)
}

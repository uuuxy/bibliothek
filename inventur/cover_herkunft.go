package inventur

import "bibliothek/pkg/coverquelle"

// Die Host-Allowlist stand bis zum 04.08.2026 hier als eigene Kopie, mit einem
// Kommentar, der zur Pflege der Zwillingsliste in api/image_caching.go aufrief. Genau
// diese Konstruktion — Gleichheit, die nur ein Kommentar behauptet — war bereits
// auseinandergelaufen: Die dritte Kopie in metadaten_client.go kannte books.google.*
// nicht. Alle drei fragen jetzt pkg/coverquelle.

// IstErlaubteCoverHerkunft prüft, ob eine Cover-URL von einem zugelassenen Host
// stammt. Für Stellen, die eine URL nur SPEICHERN (manuelles Cover-Update) — wer sie
// auch abruft, nimmt SichereCoverURL und schickt die neu gebaute Fassung los.
func IstErlaubteCoverHerkunft(rohURL string) bool {
	return coverquelle.IstErlaubt(rohURL, coverquelle.CoverHosts)
}

// SichereCoverURL liefert die aus geprüften Teilen neu gebaute Cover-URL.
// ok=false heißt: Host nicht erlaubt oder URL unlesbar.
func SichereCoverURL(rohURL string) (string, bool) {
	return coverquelle.SichereURL(rohURL, coverquelle.CoverHosts)
}

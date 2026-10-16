package inventur

import "bibliothek/pkg/coverquelle"

// Die Liste der erlaubten Hosts steht nur in pkg/coverquelle. Eigene Kopien hier und an anderen
// Stellen liefen auseinander, weil ihre Gleichheit nur ein Kommentar behauptete.

// Wer eine URL eines Covers anfasst, ruft sie auch ab und nimmt sichereCoverURL; eine Prüfung
// nur zum Speichern gibt es nicht.

// sichereCoverURL liefert die aus geprüften Teilen neu gebaute Cover-URL.
// ok=false heißt: Host nicht erlaubt oder URL unlesbar.
func sichereCoverURL(rohURL string) (string, bool) {
	return coverquelle.SichereURL(rohURL, coverquelle.BildHosts)
}

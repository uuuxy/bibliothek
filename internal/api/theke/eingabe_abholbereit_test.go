package theke

import (
	"testing"
	"time"

	"bibliothek/internal/service"
)

// Der Hinweis „liegt im Abholfach" erreicht die Theke nur über diese Übertragung; ohne Vormerkung
// bleibt das Feld ganz weg.
func TestZuAbholbereitenVormerkungen(t *testing.T) {
	if got := zuAbholbereitenVormerkungen(nil); got != nil {
		t.Errorf("ohne Vormerkung: %v, erwartet nil", got)
	}
	bis := time.Date(2026, 5, 1, 23, 59, 59, 0, time.UTC)
	got := zuAbholbereitenVormerkungen([]service.AbholbereiteVormerkung{{Titel: "Faust", BereitgestelltBis: &bis}})
	if len(got) != 1 || got[0].Titel != "Faust" || got[0].BereitgestelltBis != &bis {
		t.Errorf("zuAbholbereitenVormerkungen(…) = %+v, erwartet Faust mit der Frist", got)
	}
}

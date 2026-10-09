package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
)

// ── GDPR: Lesehistorie befristen ─────────────────────────────────────────────
//
// Bis zum 22.08.2026 behielt jede zurückgegebene Ausleihe ihre schueler_id bis zur
// Löschung des Schülers — die Titel-Historie zeigte bis zu 200 Entleiher mit Namen,
// das Profil die komplette Lesebiografie. Das HBDI-Muster-VVT „Schulbibliothek" verlangt
// Löschung, „sobald nicht mehr notwendig"; die Lesehistorie eines Kindes über Jahre zu
// führen ist kein Zweck der Ausleihe.
//
// Dieser Job trennt die Ausleihe vom Schüler (schueler_id = NULL), der Vorgang selbst
// bleibt für Statistik und Bestandskartei erhalten. Zwei Fristen, weil zwei
// Verarbeitungstätigkeiten: Schülerbücherei kurz, Lernmittel lang (Nachweis von
// Ausleihe UND Rücklauf, Schadensersatz über die Schulaufsicht). Lernmittel heißt hier
// wie im Leitfaden: aus Landesmitteln beschafft, also jedes Buch im Eigentum des Landes
// (repository.ExemplarTopfSQL, seit dem 29.09.2026; vorher ist_lernmittel am Titel). Beide
// Fristen stehen in den Einstellungen; 0 schaltet die jeweilige Befristung ab.
//
// Nicht getrennt werden Ausleihen, an denen ein OFFENER Schadensfall hängt — dort ist
// der Zweck (Forderung) noch nicht erreicht.
//
// Seit dem 16.09.2026 gilt die Befristung JEDEM LESER, auch dem Kollegium: Was ein
// Erwachsener gelesen hat, muss die Bücherei nach der Rückgabe so wenig wissen wie bei
// einem Kind. Eine laufende Dauerleihe ist davon nicht betroffen — die Frist beginnt mit
// der Rückgabe.

// RunLesehistorieBefristung trennt abgeschlossene Ausleihen nach Frist vom Schüler.
func (s *Scheduler) RunLesehistorieBefristung() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	einst, err := repository.NewSystemSettingsRepository(s.db).GetSettings(ctx)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	freihandTage := repository.TageOderStandard(einst.LesehistorieTage, repository.StandardLesehistorieTage)
	lernmittelTage := repository.TageOderStandard(einst.LesehistorieLernmittelTage, repository.StandardLesehistorieLernmittelTage)

	getrenntFreihand := s.trenneAusleihen(ctx, freihandTage, false)
	getrenntLernmittel := s.trenneAusleihen(ctx, lernmittelTage, true)
	protokollFreihand := s.tilgeAusleihProtokoll(ctx, freihandTage, false)
	protokollLernmittel := s.tilgeAusleihProtokoll(ctx, lernmittelTage, true)
	protokollVormerkung := s.tilgeVormerkSpur(ctx, freihandTage)

	protokoll := protokollFreihand + protokollLernmittel + protokollVormerkung
	gesamt := getrenntFreihand + getrenntLernmittel + protokoll
	log.Printf("Scheduler Lesehistorie: %d Ausleihen vom Schüler getrennt (Schülerbücherei %d nach %d Tagen, Lernmittel %d nach %d Tagen), %d Protokolleinträge bereinigt",
		getrenntFreihand+getrenntLernmittel, getrenntFreihand, freihandTage, getrenntLernmittel, lernmittelTage, protokoll)
	if gesamt == 0 {
		return
	}
	if err := s.auditRepo.LogSystemAktion(ctx, "ausleihen", "ANONYMIZE",
		"DSGVO Lesehistorie befristet: abgeschlossene Ausleihen vom Schüler getrennt",
		map[string]any{
			"schuelerbuecherei_getrennt": getrenntFreihand,
			"schuelerbuecherei_tage":     freihandTage,
			"lernmittel_getrennt":        getrenntLernmittel,
			"lernmittel_tage":            lernmittelTage,
			"protokoll_bereinigt":        protokoll,
			"ausgefuehrt_am":             time.Now().UTC().Format(time.RFC3339),
		},
	); err != nil {
		log.Printf("audit: Lesehistorie-ANONYMIZE konnte nicht protokolliert werden: %v", err)
	}
}

// trenneAusleihen nimmt abgeschlossenen Ausleihen nach der Frist den Leser. Scheitert eine
// Klasse, läuft die andere trotzdem: Der Fehler steht im Protokoll, die Zahl ist dann 0.
func (s *Scheduler) trenneAusleihen(ctx context.Context, tage int, lernmittel bool) int64 {
	getrennt, err := repository.TrenneAusleihenVomLeser(ctx, s.db, lernmittel, tage)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Trennung (lernmittel=%v) fehlgeschlagen: %v", lernmittel, err)
		return 0
	}
	return getrennt
}

// tilgeAusleihProtokoll nimmt dem Protokoll von Ausleihe und Rückgabe nach derselben Frist
// den Leser. Ohne das trüge das Protokoll die Lesehistorie bis zum Ende seiner Aufbewahrung
// weiter, und die Trennung der Ausleihe bliebe ohne Wirkung.
func (s *Scheduler) tilgeAusleihProtokoll(ctx context.Context, tage int, lernmittel bool) int64 {
	bereinigt, err := repository.TilgeLeserImAusleihProtokoll(ctx, s.db, lernmittel, tage)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Protokoll-Bereinigung (lernmittel=%v) fehlgeschlagen: %v", lernmittel, err)
		return 0
	}
	return bereinigt
}

// tilgeVormerkSpur nimmt der Spur einer Vormerkung, die mit ihrem Titel gelöscht wurde, den
// Leser. Es gilt die Frist der Schülerbücherei.
func (s *Scheduler) tilgeVormerkSpur(ctx context.Context, tage int) int64 {
	bereinigt, err := repository.TilgeLeserInVormerkSpuren(ctx, s.db, tage)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Bereinigung der Vormerk-Spuren fehlgeschlagen: %v", err)
		return 0
	}
	return bereinigt
}

package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/repository"
)

// ── DSGVO: Lesehistorie befristen ──────────────────────────────────────────────
//
// Ohne Befristung behielte jede zurückgegebene Ausleihe ihren Leser bis zu seiner Löschung: Die
// Historie eines Titels zeigte bis zu 200 Entleiher mit Namen, das Profil die ganze
// Lesebiografie. Das Muster-VVT „Schulbibliothek" des HBDI verlangt die Löschung, „sobald nicht
// mehr notwendig"; die Lesehistorie eines Kindes über Jahre zu führen, ist kein Zweck der
// Ausleihe.
//
// Der Lauf trennt die Ausleihe vom Leser (schueler_id = NULL), der Vorgang selbst bleibt für
// Statistik und Bestandskartei. Zwei Fristen, weil zwei Verarbeitungstätigkeiten:
// Schülerbücherei kurz, Lernmittel lang (Nachweis von Ausleihe und Rücklauf, Schadensersatz über
// die Schulaufsicht). Lernmittel heißt hier wie im Leitfaden: aus Landesmitteln beschafft, also
// jedes Buch im Eigentum des Landes (repository.ExemplarTopfSQL). Beide Fristen stehen in den
// Einstellungen; 0 schaltet die jeweilige Befristung ab.
//
// Nicht getrennt werden Ausleihen, an denen ein offener Schadensfall hängt: Dort ist der Zweck,
// die Forderung, noch nicht erreicht.
//
// Die Befristung gilt jedem Leser, auch dem Kollegium: Was ein Erwachsener gelesen hat, muss die
// Bücherei nach der Rückgabe so wenig wissen wie bei einem Kind. Eine laufende Dauerleihe ist
// nicht betroffen; die Frist beginnt mit der Rückgabe.

// fuehreLesehistorieBefristungAus trennt abgeschlossene Ausleihen nach der Frist vom Leser.
func (z *Zeitplan) fuehreLesehistorieBefristungAus() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	einst, err := repository.NewEinstellungsAbfragen(z.pool).Lade(ctx)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Einstellungen nicht lesbar, Lauf übersprungen: %v", err)
		return
	}
	freihandTage := repository.TageOderVorgabe(einst.LesehistorieTage, repository.LesehistorieTageVorgabe)
	lernmittelTage := repository.TageOderVorgabe(einst.LesehistorieLernmittelTage, repository.LesehistorieLernmittelTageVorgabe)

	getrenntFreihand := z.trenneAusleihen(ctx, freihandTage, false)
	getrenntLernmittel := z.trenneAusleihen(ctx, lernmittelTage, true)
	protokollFreihand := z.tilgeAusleihProtokoll(ctx, freihandTage, false)
	protokollLernmittel := z.tilgeAusleihProtokoll(ctx, lernmittelTage, true)
	protokollVormerkung := z.tilgeVormerkSpur(ctx, freihandTage)

	protokoll := protokollFreihand + protokollLernmittel + protokollVormerkung
	gesamt := getrenntFreihand + getrenntLernmittel + protokoll
	log.Printf("Scheduler Lesehistorie: %d Ausleihen vom Schüler getrennt (Schülerbücherei %d nach %d Tagen, Lernmittel %d nach %d Tagen), %d Protokolleinträge bereinigt",
		getrenntFreihand+getrenntLernmittel, getrenntFreihand, freihandTage, getrenntLernmittel, lernmittelTage, protokoll)
	if gesamt == 0 {
		return
	}
	if err := z.protokollAbfragen.ProtokolliereSystemAktion(ctx, "ausleihen", "ANONYMIZE",
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
// Klasse, läuft die andere trotzdem: Der Fehler steht im Log, die Zahl ist dann 0.
func (z *Zeitplan) trenneAusleihen(ctx context.Context, tage int, lernmittel bool) int64 {
	getrennt, err := repository.TrenneAusleihenVomLeser(ctx, z.pool, lernmittel, tage)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Trennung (lernmittel=%v) fehlgeschlagen: %v", lernmittel, err)
		return 0
	}
	return getrennt
}

// tilgeAusleihProtokoll nimmt dem Protokoll von Ausleihe und Rückgabe nach derselben Frist
// den Leser. Ohne das trüge das Protokoll die Lesehistorie bis zum Ende seiner Aufbewahrung
// weiter, und die Trennung der Ausleihe bliebe ohne Wirkung.
func (z *Zeitplan) tilgeAusleihProtokoll(ctx context.Context, tage int, lernmittel bool) int64 {
	bereinigt, err := repository.TilgeLeserImAusleihProtokoll(ctx, z.pool, lernmittel, tage)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Protokoll-Bereinigung (lernmittel=%v) fehlgeschlagen: %v", lernmittel, err)
		return 0
	}
	return bereinigt
}

// tilgeVormerkSpur nimmt der Spur einer Vormerkung, die mit ihrem Titel gelöscht wurde, den
// Leser. Es gilt die Frist der Schülerbücherei.
func (z *Zeitplan) tilgeVormerkSpur(ctx context.Context, tage int) int64 {
	bereinigt, err := repository.TilgeLeserInVormerkSpuren(ctx, z.pool, tage)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Bereinigung der Vormerk-Spuren fehlgeschlagen: %v", err)
		return 0
	}
	return bereinigt
}

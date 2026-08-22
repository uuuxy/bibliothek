package jobs

import (
	"context"
	"log"
	"time"

	"bibliothek/pkg/lmf"
	"bibliothek/repository"
)

// ── GDPR: Lesehistorie befristen ─────────────────────────────────────────────
//
// Bis zum 22.08.2026 behielt jede zurückgegebene Ausleihe ihre schueler_id bis zur
// Löschung des Schülers — die Titel-Historie zeigte bis zu 200 Entleiher mit Namen,
// das Profil die komplette Lesebiografie. Das HBDI-Muster-VVT „Schulbibliothek" verlangt
// Löschung, „sobald nicht mehr notwendig"; die Lesehistorie eines Kindes über Jahre zu
// führen ist kein Zweck der Ausleihe (docs/datenschutz_offene_punkte.md, A1).
//
// Dieser Job trennt die Ausleihe vom Schüler (schueler_id = NULL), der Vorgang selbst
// bleibt für Statistik und Bestandskartei erhalten. Zwei Fristen, weil zwei
// Verarbeitungstätigkeiten: Schülerbücherei kurz, Lernmittel lang (Nachweis von
// Ausleihe UND Rücklauf, Schadensersatz über die Schulaufsicht). Beide stehen in den
// Einstellungen; 0 schaltet die jeweilige Befristung ab.
//
// Nicht getrennt werden Ausleihen, an denen ein OFFENER Schadensfall hängt — dort ist
// der Zweck (Forderung) noch nicht erreicht. Lehrer-Ausleihen (Handapparat) sind
// dienstlich und bleiben unberührt.

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

	gesamt := getrenntFreihand + getrenntLernmittel
	log.Printf("Scheduler Lesehistorie: %d Ausleihen vom Schüler getrennt (Schülerbücherei %d nach %d Tagen, Lernmittel %d nach %d Tagen)",
		gesamt, getrenntFreihand, freihandTage, getrenntLernmittel, lernmittelTage)
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
			"ausgefuehrt_am":             time.Now().UTC().Format(time.RFC3339),
		},
	); err != nil {
		log.Printf("audit: Lesehistorie-ANONYMIZE konnte nicht protokolliert werden: %v", err)
	}
}

// trenneAusleihen setzt schueler_id = NULL für abgeschlossene Ausleihen, deren Rückgabe
// länger als `tage` zurückliegt. lernmittel wählt die Klasse: true = nur Lernmittel
// (LMF-Titel), false = alles andere (Freihand, Medien und Geräte — Geräte haben kein
// Exemplar und fallen damit automatisch in die kurze Frist). tage <= 0 = aus.
func (s *Scheduler) trenneAusleihen(ctx context.Context, tage int, lernmittel bool) int64 {
	if tage <= 0 {
		return 0
	}
	istLernmittel := `EXISTS (
		SELECT 1 FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.id = a.exemplar_id AND ` + lmf.SQLBedingung("t.titel", "t.signatur") + `)`
	klasse := "NOT " + istLernmittel
	if lernmittel {
		klasse = istLernmittel
	}
	query := `
		UPDATE ausleihen a
		SET schueler_id = NULL
		WHERE a.schueler_id IS NOT NULL
		  AND a.rueckgabe_am IS NOT NULL
		  AND a.rueckgabe_am < NOW() - make_interval(days => $1)
		  AND NOT EXISTS (
		        SELECT 1 FROM schadensfaelle sf
		        WHERE sf.ausleihe_id = a.id
		          AND sf.ist_bezahlt = false
		          AND sf.storniert_am IS NULL)
		  AND ` + klasse
	tag, err := s.db.Exec(ctx, query, tage)
	if err != nil {
		log.Printf("Scheduler Lesehistorie: Trennung (lernmittel=%v) fehlgeschlagen: %v", lernmittel, err)
		return 0
	}
	return tag.RowsAffected()
}

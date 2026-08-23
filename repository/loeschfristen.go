package repository

// loeschfristen.go — die Fragen, die die nächtlichen Löschroutinen stellen. Genau
// einmal aufgeschrieben.
//
// Warum diese Datei existiert: Bis zum 23.08.2026 stand jedes Lösch-Prädikat zweimal
// im Baum — einmal im Job (der löscht) und, für die Anonymisierung, ein zweites Mal in
// der Selbstprüfung (die zählt, was noch dastehen müsste). Zwei Wahrheitsquellen für
// dieselbe Frage sind hier kein Schönheitsfehler, sondern der schlimmste denkbare
// Fehler: Der Wächter beruhigt, während der Job schläft. Genau diese Klasse hat das
// Projekt zweimal getroffen (foto_url, goose-022) — eine DSGVO-Routine lief monatelang
// ins Leere, und nur eine Logzeile wusste es.
//
// Deshalb: Job und Wächter setzen DENSELBEN String ein. Der einzige Unterschied ist ein
// Wert, nicht ein Satz — die Kulanz.
//
// ── Kulanz ────────────────────────────────────────────────────────────────────
// Der Job läuft nachts. Eine Zeile, die heute um 09:00 fällig wird, ist bis zur
// nächsten Nacht zu Recht noch da; der Wächter dürfte sie nicht anmahnen. Die Kulanz
// verlängert deshalb die FRIST des Wächters um einen Tag: Er zählt nur, was der Job
// bereits mindestens einmal in der Hand hatte. Job = KulanzJob (0), Wächter =
// KulanzWaechter (1). Dieselbe Abfrage, ein anderer Parameter.

import (
	"time"

	"bibliothek/pkg/lmf"
)

const (
	// KulanzJob ist die Kulanz des Löschers selbst: keine. Er ist die Instanz, die die
	// Frist definiert.
	KulanzJob = 0
	// KulanzWaechter ist der eine Tag Luft, den die Selbstprüfung dem letzten
	// nächtlichen Lauf zugesteht.
	KulanzWaechter = 1

	// StandardAnonymisierungSoftDeleteTage sind die Tage nach dem Soft-Delete
	// (Papierkorb), nach denen ein Schüler-Datensatz anonymisiert wird.
	StandardAnonymisierungSoftDeleteTage = 180
	// StandardAnonymisierungAbgaengerTage sind die Tage nach der letzten Änderung, nach
	// denen ein als Abgänger markierter Schüler anonymisiert wird.
	StandardAnonymisierungAbgaengerTage = 360

	// StandardAuditAufbewahrungMonate ist die Vorgabe-Aufbewahrung der beiden
	// Protokolltabellen.
	StandardAuditAufbewahrungMonate = 24
	// MindestAuditAufbewahrungMonate ist die Untergrenze: Eine versehentliche 0 in den
	// Einstellungen darf nicht das komplette Protokoll wegräumen — Revisionsfähigkeit
	// (wer hat die Gebühr storniert?) ist der Zweck der Tabellen.
	MindestAuditAufbewahrungMonate = 6
	// AuditAufbewahrungSchluessel ist die Zeile in system_einstellungen, aus der Job
	// UND Wächter die Frist lesen.
	AuditAufbewahrungSchluessel = "audit_aufbewahrung_monate"
)

// ── Schüler-Anonymisierung ($1 Soft-Delete-Tage, $2 Abgänger-Tage, $3 Kulanz) ──

// PredikatAnonymisierung liefert die WHERE-Bedingung von RunGDPRAnonymizeOldData.
// Ein Schüler mit offener Ausleihe oder unbezahltem Schaden bleibt stehen — dort ist
// der Zweck der Speicherung noch nicht erreicht.
func PredikatAnonymisierung() string {
	return `anonymized_at IS NULL
		  AND (
		      (deleted_at IS NOT NULL AND deleted_at < NOW() - make_interval(days => $1::int + $3::int))
		      OR
		      (ist_abgaenger = true AND aktualisiert_am < NOW() - make_interval(days => $2::int + $3::int))
		  )
		  AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = schueler.id AND rueckgabe_am IS NULL)
		  AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = schueler.id AND ist_bezahlt = false)`
}

// ── Abgänger endgültig löschen ($1 Stichjahr) ─────────────────────────────────

// PredikatAbgaengerLoeschung liefert die WHERE-Bedingung, mit der
// RunGDPRDeleteAbgaenger die endgültig löschbaren Abgänger auswählt. Die Frist steckt
// hier nicht in einem Intervall, sondern im Stichjahr — siehe AbgaengerStichjahr.
func PredikatAbgaengerLoeschung() string {
	return `ist_abgaenger = true
		  AND deleted_at IS NULL
		  AND abgaenger_jahr < $1
		  AND NOT EXISTS (
		      SELECT 1 FROM ausleihen
		      WHERE schueler_id = schueler.id AND rueckgabe_am IS NULL
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM schadensfaelle
		      WHERE schueler_id = schueler.id AND ist_bezahlt = false
		  )`
}

// AbgaengerStichjahr liefert das Jahr, unter dem ein Abgangsjahr liegen muss, damit der
// Datensatz endgültig gelöscht werden darf — die 30-tägige Karenzzeit nach Schuljahres-
// ende als Näherung. Vor dem 30. Januar gilt das Vorjahr als Stichjahr, die Abgänger des
// letzten Jahres sind dann noch in der Karenz. Job UND Wächter rechnen hiermit; eine
// zweite Jahresrechnung wäre eine zweite Frist.
func AbgaengerStichjahr(jetzt time.Time) int {
	jahr := jetzt.Year()
	if jetzt.Before(time.Date(jahr, time.January, 30, 0, 0, 0, 0, time.UTC)) {
		jahr--
	}
	return jahr
}

// ── Lesehistorie befristen ($1 Tage, $2 Kulanz) ───────────────────────────────

// istLernmittelExemplar formuliert die Klassenfrage über das Exemplar in `spalte`.
func istLernmittelExemplar(spalte string) string {
	return `EXISTS (
		SELECT 1 FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		WHERE e.id = ` + spalte + ` AND ` + lmf.SQLBedingung("t.titel", "t.signatur") + `)`
}

// klasse wählt die Frist-Klasse: true = nur Lernmittel (LMF-Titel), false = alles andere
// (Freihand, Medien und Geräte — Geräte haben kein Exemplar und fallen damit automatisch
// in die kurze Frist).
func klasse(spalte string, lernmittel bool) string {
	if lernmittel {
		return istLernmittelExemplar(spalte)
	}
	return "NOT " + istLernmittelExemplar(spalte)
}

// PredikatLesehistorieAusleihen liefert die WHERE-Bedingung, mit der die Ausleihe vom
// Schüler getrennt wird (Alias `a`). Ausleihen mit OFFENEM Schadensfall bleiben
// zugeordnet — dort ist der Zweck (Forderung) noch nicht erreicht. Lehrer-Ausleihen
// haben keine schueler_id und sind ohnehin nicht betroffen.
func PredikatLesehistorieAusleihen(lernmittel bool) string {
	return `a.schueler_id IS NOT NULL
		  AND a.rueckgabe_am IS NOT NULL
		  AND a.rueckgabe_am < NOW() - make_interval(days => $1::int + $2::int)
		  AND NOT EXISTS (
		        SELECT 1 FROM schadensfaelle sf
		        WHERE sf.ausleihe_id = a.id
		          AND sf.ist_bezahlt = false
		          AND sf.storniert_am IS NULL)
		  AND ` + klasse("a.exemplar_id", lernmittel)
}

// PredikatLesehistorieProtokoll liefert die WHERE-Bedingung, mit der dem Ausleih-
// Protokoll (audit_log, Alias `al`) die Schüler-Zuordnung genommen wird. datensatz_id
// ist dort das EXEMPLAR (so schreibt logLoanEvent), die Klasse kommt über den Titel.
// Ein Eintrag bleibt, solange dieser Schüler dieses Exemplar noch offen hat oder ein
// offener Schadensfall daran hängt.
func PredikatLesehistorieProtokoll(lernmittel bool) string {
	return `al.tabelle = 'ausleihen'
		  AND al.details ? 'schueler_id'
		  AND al.timestamp < NOW() - make_interval(days => $1::int + $2::int)
		  AND NOT EXISTS (
		        SELECT 1 FROM ausleihen a
		        WHERE a.exemplar_id = al.datensatz_id
		          AND a.schueler_id::text = al.details->>'schueler_id'
		          AND a.rueckgabe_am IS NULL)
		  AND NOT EXISTS (
		        SELECT 1 FROM schadensfaelle sf
		        WHERE sf.exemplar_id = al.datensatz_id
		          AND sf.schueler_id::text = al.details->>'schueler_id'
		          AND sf.ist_bezahlt = false
		          AND sf.storniert_am IS NULL)
		  AND ` + klasse("al.datensatz_id", lernmittel)
}

// ── Erledigte Anliegen ($1 Tage, $2 Kulanz) ───────────────────────────────────

// PredikatAnliegen liefert die WHERE-Bedingung der Anliegen-Befristung. Die Rechnung
// läuft über den Erledigungszeitpunkt, nicht über das Anlegen: Ein offenes Anliegen ist
// eine laufende Sache und hat keine Frist.
func PredikatAnliegen() string {
	return `erledigt_am IS NOT NULL
		  AND erledigt_am < NOW() - make_interval(days => $1::int + $2::int)`
}

// ── Audit-Aufbewahrung ($1 Monate, $2 Kulanz-Tage) ────────────────────────────
//
// Die Kulanz ist hier ein zusätzlicher TAG im selben Intervall, nicht ein zusätzlicher
// Monat — sonst gäbe der Wächter dem Job einen ganzen Monat Blindheit.

// PredikatAuditLog ist die Bedingung für audit_log (fachliche Datensatz-Historie).
func PredikatAuditLog() string {
	return `timestamp < NOW() - make_interval(months => $1::int, days => $2::int)`
}

// PredikatAuditLogs ist die Bedingung für audit_logs (Admin-Aktionen inkl. IP-Adressen).
// Andere Tabelle, andere Zeitspalte, dieselbe Frist.
func PredikatAuditLogs() string {
	return `zeitstempel < NOW() - make_interval(months => $1::int, days => $2::int)`
}

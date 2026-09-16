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

	"bibliothek/pkg/schulzeit"
)

// AbgangSeit ist der Ausdruck für „seit wann ist der weg": der Abgangsstempel, und für
// Altzeilen ohne Stempel ersatzweise die letzte Änderung. EINE Formulierung für die
// Löschuhr (PredikatAnonymisierung) und den Wächter „Ehemalige mit offenen Vorgängen"
// (BetriebszustandRepository) — bis zum 12.09.2026 verlangte der Wächter
// `abgaenger_seit IS NOT NULL` und sah die Altzeilen deshalb nie, obwohl gerade ihr Name
// auf Dauer stehen bleibt (#593).
//
// alias ist der Tabellen-Alias der schueler-Zeile in der jeweiligen Abfrage.
func AbgangSeit(alias string) string {
	return "COALESCE(" + alias + ".abgaenger_seit, " + alias + ".aktualisiert_am)"
}

// ── Schüler-Anonymisierung ──────────────────────────────────────────────────

// PredikatAnonymisierung liefert die Bedingung von RunGDPRAnonymizeOldData.
// Ein Schüler mit offener Ausleihe oder unbezahltem Schaden bleibt stehen — dort ist
// der Zweck der Speicherung noch nicht erreicht. Die Soft-Delete-Frist ist eine
// Konstante; die Abgänger-Frist ist die einstellbare Karenzzeit (abgaengerKarenzTage,
// Migration 094).
//
// Die Uhr der Karenz ist der SPÄTESTE von drei Zeitpunkten: der Abgang (abgaenger_seit;
// aktualisiert_am nur als Rückfall für Altzeilen ohne Stempel), die letzte Rückgabe
// einer Ausleihe und der letzte Abschluss eines Schadensfalls (Bezahlung oder Storno —
// beides setzt ist_bezahlt, repository/audit_system.go). Bis 05.09.2026 zählte nur der
// Abgang. Weil offene Vorgänge die Zeile schützen, kippte der Schutz damit genau mit der
// Rückgabe: Wer am Tag 10 zurückgab, hatte 80 Tage Reparaturfenster; wer am Tag 120
// zurückgab, wurde in der Folgenacht anonymisiert. Die Karenz ist für die Korrektur an
// der Theke da, und die Rückgabe IST der Thekenkontakt. Die endgültige Löschung
// (PredikatAbgaengerLoeschung) trifft seit 05.09.2026 nur anonymisierte Zeilen — sie
// wartet die Karenz also ab, statt sie abzuschneiden.
//
// Der Soft-Delete-Zweig behält seine eigene Uhr (deleted_at): Eine Löschung von Hand ist
// eine Entscheidung, keine Zuordnung, die sich noch als falsch herausstellen könnte.
// GREATEST übergeht NULL — ein Schüler ohne je einen Vorgang rechnet allein ab dem Abgang.
func PredikatAnonymisierung(abgaengerKarenzTage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{StandardAnonymisierungSoftDeleteTage, abgaengerKarenzTage, kulanz}, Where: `art = 'schueler'
		  AND anonymized_at IS NULL
		  AND (
		      (deleted_at IS NOT NULL AND deleted_at < NOW() - make_interval(days => $1::int + $3::int))
		      OR
		      (ist_abgaenger = true AND GREATEST(
		          ` + AbgangSeit("schueler") + `,
		          (SELECT max(a.rueckgabe_am) FROM ausleihen a WHERE a.schueler_id = schueler.id),
		          (SELECT max(GREATEST(sf.aktualisiert_am, sf.storniert_am)) FROM schadensfaelle sf
		            WHERE sf.schueler_id = schueler.id AND sf.ist_bezahlt)
		      ) < NOW() - make_interval(days => $2::int + $3::int))
		  )
		  AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = schueler.id AND rueckgabe_am IS NULL)
		  AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = schueler.id AND ist_bezahlt = false)`}
}

// Beide Prädikate beginnen seit Migration 123 mit art = 'schueler'.
//
// Der Grund liegt im SOFT-DELETE-Zweig der Anonymisierung: Er hängt allein am Papierkorb
// (deleted_at) und fragt nicht nach der Art. Eine gelöschte Kollegenzeile wäre damit
// löschreif geworden — und weil chk_leser_nur_schueler_werden_abgaenger das Setzen von
// anonymized_at bei einem Kollegen verbietet, hätte das UPDATE abgebrochen und den GANZEN
// Nachtlauf mitgenommen. Eine einzige Zeile hätte die Anonymisierung aller echten
// Abgänger stillgelegt, Nacht für Nacht, mit einer Zeile im Protokoll (nachgestellt in
// api/leser_lusd_schutz_pg_test.go).
//
// Hier und nicht in den Jobs, weil diese Bedingungen die EINE Quelle sind: Der Wächter der
// Selbstprüfung stellt dieselbe Frage als count(*). Stünde die Einschränkung nur im Job,
// zählte der Wächter weiter Kollegen mit und meldete Arbeit, die niemand tun kann.

// ── Abgänger endgültig löschen ($1 Stichjahr) ─────────────────────────────────

// PredikatAbgaengerLoeschung liefert die WHERE-Bedingung, mit der
// RunGDPRDeleteAbgaenger die endgültig löschbaren Abgänger auswählt. Die Frist steckt
// hier nicht in einem Intervall, sondern im Stichjahr — siehe AbgaengerStichjahr.
//
// Gelöscht wird nur, was schon anonymisiert ist (anonymized_at). Bis 05.09.2026 löschte
// der Job ab dem 30. Januar jeden Abgänger ohne offene Vorgänge in der nächsten Nacht —
// und lief im Cron VOR der Anonymisierung. Die Karenz (PredikatAnonymisierung, ab dem
// letzten Vorgang) galt damit nur bis zum Stichtag: Wer im November zurückgab, hatte
// 77 statt 90 Tage Reparaturfenster, wer nach dem Stichtag zurückgab, keines. Mit dieser
// einen Bedingung gilt die EINE Einstellung abgaenger_karenz_tage für beides; 0 heißt
// weiter „sofort", weil dann schon der Import anonymisiert (anonymisiereAbgaenger).
func PredikatAbgaengerLoeschung(jetzt time.Time) Loeschbedingung {
	return Loeschbedingung{Args: []any{AbgaengerStichjahr(jetzt)}, Where: `art = 'schueler'
		  AND ist_abgaenger = true
		  AND deleted_at IS NULL
		  AND anonymized_at IS NOT NULL
		  AND abgaenger_jahr < $1
		  AND NOT EXISTS (
		      SELECT 1 FROM ausleihen
		      WHERE schueler_id = schueler.id AND rueckgabe_am IS NULL
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM schadensfaelle
		      WHERE schueler_id = schueler.id AND ist_bezahlt = false
		  )`}
}

// AbgaengerStichjahr liefert das Jahr, unter dem ein Abgangsjahr liegen muss, damit der
// Datensatz endgültig gelöscht werden darf — die 30-tägige Karenzzeit nach Schuljahres-
// ende als Näherung. Vor dem 30. Januar gilt das Vorjahr als Stichjahr, die Abgänger des
// letzten Jahres sind dann noch in der Karenz. Job UND Wächter rechnen hiermit; eine
// zweite Jahresrechnung wäre eine zweite Frist.
//
// Gerechnet wird in der ZEITZONE DER SCHULE, nicht in der des Containers. Vorher nahm
// die Funktion das Jahr aus der lokalen Zeit und verglich es mit einem fest in UTC
// gebauten 30. Januar — zwei Zeitzonen in einer Rechnung, die nur zufällig übereinstimmen,
// solange der Container auf UTC läuft. Am 30.01. um 00:30 Berliner Zeit lieferte das
// noch das Vorjahr, obwohl der Stichtag lokal längst erreicht war. Die Richtung war
// harmlos (es wurde später gelöscht, nicht früher), aber das Ergebnis hing an einer
// Umgebungsvariablen statt am Schulkalender. Dieselbe Regel wie bei den PDF-Datumsangaben
// (pkg/schulzeit).
func AbgaengerStichjahr(jetzt time.Time) int {
	inSchulzeit := jetzt.In(schulzeit.Zone())
	jahr := inSchulzeit.Year()
	if inSchulzeit.Before(time.Date(jahr, time.January, 30, 0, 0, 0, 0, schulzeit.Zone())) {
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
		WHERE e.id = ` + spalte + ` AND t.ist_lernmittel)`
}

// klasse wählt die Frist-Klasse: true = nur Lernmittel, false = alles andere
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
// zugeordnet — dort ist der Zweck (Forderung) noch nicht erreicht.
//
// Die Befristung gilt SCHÜLERN. Bis Migration 125 galt das von selbst: Eine
// Lehrerausleihe stand in einer anderen Spalte und hatte keine schueler_id. Seit alle
// Leser in einer Tabelle stehen, muss die Einschränkung dastehen — sonst nähme der
// Nachtlauf still auch dem Kollegium seine Ausleihhistorie. Ob er das SOLL, ist eine
// Frage an den Betrieb und keine, die dieser Umbau nebenbei beantwortet
// (docs/OFFEN.md 5.16); bis dahin bleibt das Verhalten, wie es war.
func PredikatLesehistorieAusleihen(lernmittel bool, tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `a.schueler_id IS NOT NULL
		  AND EXISTS (SELECT 1 FROM leser l WHERE l.id = a.schueler_id AND l.art = 'schueler')
		  AND a.rueckgabe_am IS NOT NULL
		  AND a.rueckgabe_am < NOW() - make_interval(days => $1::int + $2::int)
		  AND NOT EXISTS (
		        SELECT 1 FROM schadensfaelle sf
		        WHERE sf.ausleihe_id = a.id
		          AND sf.ist_bezahlt = false
		          AND sf.storniert_am IS NULL)
		  AND ` + klasse("a.exemplar_id", lernmittel)}
}

// PredikatLesehistorieProtokoll liefert die WHERE-Bedingung, mit der dem Ausleih-
// Protokoll (audit_log, Alias `al`) die Schüler-Zuordnung genommen wird. datensatz_id
// ist dort das EXEMPLAR (so schreibt logLoanEvent), die Klasse kommt über den Titel.
// Ein Eintrag bleibt, solange dieser Schüler dieses Exemplar noch offen hat oder ein
// offener Schadensfall daran hängt.
func PredikatLesehistorieProtokoll(lernmittel bool, tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `al.tabelle = 'ausleihen'
		  AND al.details ? 'schueler_id'
		  -- Wie beim Prädikat der Ausleihen: nur Schüler (Migration 125).
		  AND EXISTS (SELECT 1 FROM leser l WHERE l.id::text = al.details->>'schueler_id' AND l.art = 'schueler')
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
		  AND ` + klasse("al.datensatz_id", lernmittel)}
}

// ── Erledigte Anliegen ($1 Tage, $2 Kulanz) ───────────────────────────────────

// PredikatAnliegen liefert die WHERE-Bedingung der Anliegen-Befristung. Die Rechnung
// läuft über den Erledigungszeitpunkt, nicht über das Anlegen: Ein offenes Anliegen ist
// eine laufende Sache und hat keine Frist.
func PredikatAnliegen(tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `erledigt_am IS NOT NULL
		  AND erledigt_am < NOW() - make_interval(days => $1::int + $2::int)`}
}

// ── Nachbuch-Meldungen ($1 Tage, $2 Kulanz) ─────────────────────────────────────
//
// Quittierte Meldungen fallen nach der Lesehistorie-Frist der Schülerbücherei, höchstens
// nach 30 Tagen (entschieden am 13.09.2026): Eine quittierte Meldung ist erledigt,
// ihre Beteiligten stehen mit Namen darin, und länger als die Lesehistorie darf nichts
// den Schüler an ein Buch binden. Offene Meldungen haben keine Frist — sie sind Arbeit,
// die noch aussteht, und der Wächter der Betriebsbereitschaft nennt sie nach 14 Tagen.

// HoechstNachbuchMeldungenTage ist die Obergrenze der Frist, unabhängig von der Einstellung.
const HoechstNachbuchMeldungenTage = 30

// NachbuchMeldungenTage liefert die Frist: Lesehistorie-Frist, höchstens 30 Tage.
func NachbuchMeldungenTage(einst *SystemEinstellungen) int {
	tage := TageOderStandard(einst.LesehistorieTage, StandardLesehistorieTage)
	if tage <= 0 || tage > HoechstNachbuchMeldungenTage {
		return HoechstNachbuchMeldungenTage
	}
	return tage
}

// PredikatNachbuchMeldungen ist die Bedingung für quittierte Nachbuch-Meldungen.
func PredikatNachbuchMeldungen(tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `quittiert_am IS NOT NULL
		  AND quittiert_am < NOW() - make_interval(days => $1::int + $2::int)`}
}

// ── Audit-Aufbewahrung ($1 Monate, $2 Kulanz-Tage) ────────────────────────────
//
// Die Kulanz ist hier ein zusätzlicher TAG im selben Intervall, nicht ein zusätzlicher
// Monat — sonst gäbe der Wächter dem Job einen ganzen Monat Blindheit.

// PredikatAuditLog ist die Bedingung für audit_log (fachliche Datensatz-Historie).
func PredikatAuditLog(monate, kulanzTage int) Loeschbedingung {
	return Loeschbedingung{Args: []any{monate, kulanzTage},
		Where: `timestamp < NOW() - make_interval(months => $1::int, days => $2::int)`}
}

// PredikatAuditLogs ist die Bedingung für audit_logs (Admin-Aktionen inkl. IP-Adressen).
// Andere Tabelle, andere Zeitspalte, dieselbe Frist.
func PredikatAuditLogs(monate, kulanzTage int) Loeschbedingung {
	return Loeschbedingung{Args: []any{monate, kulanzTage},
		Where: `zeitstempel < NOW() - make_interval(months => $1::int, days => $2::int)`}
}

package repository

// Die Fragen, die die nächtlichen Löschroutinen stellen, genau einmal aufgeschrieben.
//
// Job und Wächter setzen denselben String ein: Der Job löscht, die Selbstprüfung zählt, was
// noch dastehen müsste. Stünde die Frage zweimal im Baum, beruhigte der Wächter, während der
// Job schläft; eine Löschroutine liefe dann ins Leere, und nur eine Logzeile wüsste es. Der
// einzige Unterschied zwischen beiden ist ein Wert, nicht ein Satz: die Kulanz.
//
// Der Job läuft nachts. Eine Zeile, die heute um 09:00 fällig wird, ist bis zur nächsten Nacht
// zu Recht noch da; der Wächter dürfte sie nicht anmahnen. Die Kulanz verlängert deshalb die
// Frist des Wächters um einen Tag: Er zählt nur, was der Job schon mindestens einmal in der
// Hand hatte. Job = KulanzJob (0), Wächter = KulanzWaechter (1). Dieselbe Abfrage, ein anderer
// Parameter.

import (
	"time"

	"bibliothek/pkg/mitteltopf"
	"bibliothek/pkg/schulzeit"
)

// sqlAbgangSeit ist der Ausdruck für „seit wann ist der weg": der Abgangsstempel, und für
// Altzeilen ohne Stempel ersatzweise die letzte Änderung. Eine Formulierung für die Löschuhr
// (PraedikatAnonymisierung) und den Wächter „Ehemalige mit offenen Vorgängen"
// (BetriebszustandAbfragen): Verlangte der Wächter `abgaenger_seit IS NOT NULL`, sähe er die
// Altzeilen nie, obwohl gerade ihr Name auf Dauer stehen bleibt.
//
// alias ist der Tabellen-Alias der schueler-Zeile in der jeweiligen Abfrage.
func sqlAbgangSeit(alias string) string {
	return "COALESCE(" + alias + ".abgaenger_seit, " + alias + ".aktualisiert_am)"
}

// sqlKarenzUhr ist der Zeitpunkt, ab dem die Karenz eines Abgängers läuft: der spätere von
// Abgang (sqlAbgangSeit) und letztem abgeschlossenem Vorgang (letzter_vorgang_am: letzte
// Rückgabe oder letzter Abschluss eines Schadensfalls). Eine Formulierung für die Löschuhr
// (PraedikatAnonymisierung) und den Wächter „Ehemalige mit offenen Vorgängen"
// (ZaehleEhemaligeMitOffenenVorgaengen): Rechnete der Wächter allein mit dem Abgang, meldete er
// „Karenz abgelaufen", wo die Löschuhr noch läuft. GREATEST übergeht NULL; ein Leser ohne je
// einen Vorgang rechnet allein ab dem Abgang.
//
// Der Leser trägt den Zeitpunkt selbst, von Triggern gestempelt: Er überlebt das Trennen der
// Ausleihe durch den Lesehistorie-Lauf (jobs/cron_dsgvo_lesehistorie.go) und sagt nichts
// darüber, was gelesen wurde. Zwei Unterabfragen über ausleihen.schueler_id und
// schadensfaelle.schueler_id verlören die Rückgabe, sobald die Karenz länger eingestellt ist
// als die Lesehistorie-Frist; die Uhr fiele auf den Abgang zurück, und die Zeile würde früher
// anonymisiert als eingestellt.
func sqlKarenzUhr(alias string) string {
	return "GREATEST(" + sqlAbgangSeit(alias) + ", " + alias + ".letzter_vorgang_am)"
}

// ── Schüler-Anonymisierung ──────────────────────────────────────────────────

// PraedikatAnonymisierung liefert die Bedingung von RunGDPRAnonymizeOldData. Ein Schüler mit
// offener Ausleihe oder unbezahltem Schaden bleibt stehen: Dort ist der Zweck der Speicherung
// noch nicht erreicht. Die Frist im Papierkorb ist eine Konstante; die Frist der Abgänger ist
// die einstellbare Karenzzeit (abgaengerKarenzTage).
//
// Die Uhr der Karenz (sqlKarenzUhr) ist der spätere von zwei Zeitpunkten: dem Abgang
// (abgaenger_seit; aktualisiert_am nur als Rückfall für Altzeilen ohne Stempel) und dem letzten
// abgeschlossenen Vorgang (letzter_vorgang_am: die letzte Rückgabe einer Ausleihe oder der
// letzte Abschluss eines Schadensfalls, Bezahlung wie Storno). Zählte nur der Abgang, kippte
// der Schutz mit der Rückgabe, weil offene Vorgänge die Zeile schützen: Wer am Tag 10
// zurückgibt, hätte 80 Tage Reparaturfenster; wer am Tag 120 zurückgibt, würde in der
// Folgenacht anonymisiert. Die Karenz ist für die Korrektur an der Theke da, und die Rückgabe
// ist der Thekenkontakt. Die endgültige Löschung (praedikatAbgaengerLoeschung) trifft nur
// anonymisierte Zeilen; sie wartet die Karenz also ab, statt sie abzuschneiden.
//
// Der Zweig des Papierkorbs behält seine eigene Uhr (deleted_at): Eine Löschung von Hand ist
// eine Entscheidung, keine Zuordnung, die sich noch als falsch herausstellen könnte.
func PraedikatAnonymisierung(abgaengerKarenzTage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{AnonymisierungPapierkorbTage, abgaengerKarenzTage, kulanz}, Where: `art = 'schueler'
		  AND anonymized_at IS NULL
		  AND (
		      (deleted_at IS NOT NULL AND deleted_at < NOW() - make_interval(days => $1::int + $3::int))
		      OR
		      (ist_abgaenger = true AND ` + sqlKarenzUhr("schueler") + ` < NOW() - make_interval(days => $2::int + $3::int))
		  )
		  AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = schueler.id AND rueckgabe_am IS NULL)
		  AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = schueler.id AND ist_bezahlt = false)`}
}

// Beide Prädikate beginnen mit art = 'schueler'.
//
// Der Grund liegt im Zweig des Papierkorbs: Er hängt allein an deleted_at und fragt nicht nach
// der Art. Eine gelöschte Kollegenzeile würde damit löschreif, und weil
// chk_leser_nur_schueler_werden_abgaenger das Setzen von anonymized_at bei einem Kollegen
// verbietet, bräche das UPDATE ab und nähme den ganzen Nachtlauf mit. Eine einzige Zeile legte
// die Anonymisierung aller echten Abgänger still, Nacht für Nacht, mit einer Zeile im Protokoll
// (nachgestellt in repository/leser_lusd_schutz_pg_test.go).
//
// Hier und nicht in den Jobs, weil diese Bedingungen die eine Quelle sind: Der Wächter der
// Selbstprüfung stellt dieselbe Frage als count(*). Stünde die Einschränkung nur im Job, zählte
// der Wächter weiter Kollegen mit und meldete Arbeit, die niemand tun kann.

// ── Gelöschte Kollegen endgültig löschen ($1 Tage, $2 Kulanz) ─────────────────

// praedikatKollegenPapierkorb liefert die Bedingung, mit der RunPapierkorbKollegenLoeschung
// Kollegen auswählt, die seit AnonymisierungPapierkorbTage im Papierkorb liegen.
// Dieselbe Frist wie beim Schüler, der dort
// anonymisiert wird; einen Kollegen verbietet chk_leser_nur_schueler_werden_abgaenger zu
// anonymisieren, deshalb fällt er ganz — über denselben Weg wie „Endgültig löschen" von Hand
// (LoescheLeserEndgueltig). Offene Ausleihen und unbezahlte Forderungen halten ihn, wie dort.
//
// Gefragt wird die Tabelle leser, nicht die Sicht schueler: Die zeigt keinen Kollegen.
func praedikatKollegenPapierkorb(kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{AnonymisierungPapierkorbTage, kulanz}, Where: `art <> 'schueler'
		  AND deleted_at IS NOT NULL
		  AND deleted_at < NOW() - make_interval(days => $1::int + $2::int)
		  AND NOT EXISTS (SELECT 1 FROM ausleihen WHERE schueler_id = leser.id AND rueckgabe_am IS NULL)
		  AND NOT EXISTS (SELECT 1 FROM schadensfaelle WHERE schueler_id = leser.id AND ist_bezahlt = false)`}
}

// ── Abgänger endgültig löschen ($1 Stichjahr) ─────────────────────────────────

// praedikatAbgaengerLoeschung liefert die WHERE-Bedingung, mit der RunGDPRDeleteAbgaenger die
// endgültig löschbaren Abgänger auswählt. Die Frist steckt hier nicht in einem Intervall,
// sondern im Stichjahr (AbgaengerStichjahr).
//
// Gelöscht wird nur, was schon anonymisiert ist (anonymized_at). Löschte der Job ab dem 30.
// Januar jeden Abgänger ohne offene Vorgänge, gälte die Karenz (PraedikatAnonymisierung, ab dem
// letzten Vorgang) nur bis zum Stichtag: Wer im November zurückgibt, hätte 77 statt 90 Tage
// Reparaturfenster, wer nach dem Stichtag zurückgibt, keines. Mit dieser einen Bedingung gilt
// die eine Einstellung abgaenger_karenz_tage für beides; 0 heißt „sofort", weil dann schon der
// Import anonymisiert (AnonymisiereAbgaenger).
func praedikatAbgaengerLoeschung(jetzt time.Time) Loeschbedingung {
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
// Datensatz endgültig gelöscht werden darf: die 30-tägige Karenzzeit nach dem Ende des
// Schuljahres als Näherung. Vor dem 30. Januar gilt das Vorjahr als Stichjahr, die Abgänger des
// letzten Jahres sind dann noch in der Karenz. Job und Wächter rechnen hiermit; eine zweite
// Jahresrechnung wäre eine zweite Frist.
//
// Gerechnet wird in der Zeitzone der Schule, nicht in der des Containers: Das Jahr aus der
// lokalen Zeit und ein in UTC gebauter 30. Januar wären zwei Zeitzonen in einer Rechnung, die
// nur übereinstimmen, solange der Container auf UTC läuft. Das Ergebnis hinge an einer
// Umgebungsvariablen statt am Schulkalender. Dieselbe Regel wie bei den Datumsangaben der PDFs
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

// istLandExemplar formuliert die Klassenfrage über das Exemplar in `spalte`: Gehört es dem
// Land? Dieselbe Regel wie Etikett, Bestandsbücher und Bescheid (ExemplarTopfSQL: Eigentum am
// Exemplar, sonst Topf der Bestellung, sonst ist_lernmittel am Titel).
//
// Die lange Frist hat ihren Grund im Verfahren des Landes, Bestandskartei (Leitfaden
// Lernmittelfreiheit 11.3) und Schadensersatz über die Schulaufsicht, und der Leitfaden knüpft
// die Bestandsverzeichnisse an „alle aus Landesmitteln beschafften Gegenstände" (11.1), also an
// das Eigentum. Entschiede ist_lernmittel allein, ginge eine Lektüre des Landes an die
// Schulaufsicht und verlöre ihren Ausleiher nach der Frist der Schülerbücherei.
func istLandExemplar(spalte string) string {
	return `EXISTS (
		SELECT 1 FROM buecher_exemplare e
		JOIN buecher_titel t ON t.id = e.titel_id
		` + ExemplarTopfJoin + `
		WHERE e.id = ` + spalte + ` AND ` + ExemplarTopfSQL + ` = '` + mitteltopf.Land + `')`
}

// klasse wählt die Frist-Klasse: true = Bücher des Landes, false = alles andere
// (Schülerbücherei, Medien und Geräte — Geräte haben kein Exemplar und fallen damit
// automatisch in die kurze Frist).
func klasse(spalte string, land bool) string {
	if land {
		return istLandExemplar(spalte)
	}
	return "NOT " + istLandExemplar(spalte)
}

// praedikatLesehistorieAusleihen liefert die WHERE-Bedingung, mit der die Ausleihe vom Leser
// getrennt wird (Alias `a`). Ausleihen mit offenem Schadensfall bleiben zugeordnet: Dort ist
// der Zweck (die Forderung) noch nicht erreicht.
//
// Die Befristung gilt jedem Leser, Schülern wie Kollegium: eine Regel für alle. Das kostet
// nichts, weil die Frist ab der Rückgabe läuft: Eine Dauerleihe des Kollegiums ist nie
// zurückgegeben und bleibt unberührt, solange sie läuft. Die Paarung von Frist und Eigentum
// bleibt: Ein Buch des Landes behält seine 730 Tage für die Bestandskartei, ein Buch der
// Schülerbücherei einen Tag.
func praedikatLesehistorieAusleihen(land bool, tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `a.schueler_id IS NOT NULL
		  AND a.rueckgabe_am IS NOT NULL
		  AND a.rueckgabe_am < NOW() - make_interval(days => $1::int + $2::int)
		  AND NOT EXISTS (
		        SELECT 1 FROM schadensfaelle sf
		        WHERE sf.ausleihe_id = a.id
		          AND sf.ist_bezahlt = false
		          AND sf.storniert_am IS NULL)
		  AND ` + klasse("a.exemplar_id", land)}
}

// praedikatLesehistorieProtokoll liefert die WHERE-Bedingung, mit der dem Ausleih-
// Protokoll (audit_log, Alias `al`) die Schüler-Zuordnung genommen wird. datensatz_id
// ist dort das Exemplar (so schreibt protokolliereLeihvorgang), die Klasse kommt über sein Eigentum.
// Ein Eintrag bleibt, solange dieser Schüler dieses Exemplar noch offen hat oder ein
// offener Schadensfall daran hängt.
func praedikatLesehistorieProtokoll(land bool, tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `al.tabelle = 'ausleihen'
		  AND al.details ? 'schueler_id'
		  -- Wie beim Prädikat der Ausleihen: jeder Leser.
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
		  AND ` + klasse("al.datensatz_id", land)}
}

// praedikatLesehistorieVormerkspur liefert die WHERE-Bedingung, mit der der Spur einer
// Vormerkung, die mit ihrem Titel gelöscht wurde (audit_log, Alias `al`), der Leser genommen
// wird: Kennung und Name. Die Zeile nennt den Titel des Buchs, und länger als die Lesehistorie
// bindet nichts einen Leser an ein Buch. Den Titel gibt es nicht mehr, ein Eigentum also auch
// nicht: Es gilt die Frist der Schülerbücherei, wie für die Ausleihspur eines gelöschten Titels.
//
// Die Spur einer gelöschten Forderung (tabelle 'schadensfaelle') fällt nicht darunter: Sie
// belegt, wessen Forderung mit dem Titel gelöscht wurde, und behält den Leser bis zu seiner
// Tilgung.
func praedikatLesehistorieVormerkspur(tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `al.tabelle = 'vormerkungen'
		  AND al.details ?| ARRAY['schueler_id', 'betrifft']
		  AND al.timestamp < NOW() - make_interval(days => $1::int + $2::int)`}
}

// ── Bearbeiter an Ausleihen ($1 Tage, $2 Kulanz) ──────────────────────────────

// praedikatBearbeiterKennung liefert die Bedingung, mit der einer zurückgegebenen Ausleihe die
// Kennungen der Bearbeiter von Ausgabe und Rückgabe genommen werden. Die Frist läuft ab der
// Rückgabe: Eine laufende Ausleihe behält ihren Bearbeiter.
func praedikatBearbeiterKennung(kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{BearbeiterKennungTage, kulanz}, Where: `rueckgabe_am < NOW() - make_interval(days => $1::int + $2::int)
		  AND (bearbeiter_id IS NOT NULL OR rueckgabe_bearbeiter_id IS NOT NULL)`}
}

// ── Erledigte Anliegen ($1 Tage, $2 Kulanz) ───────────────────────────────────

// praedikatAnliegen liefert die WHERE-Bedingung der Anliegen-Befristung. Die Rechnung
// läuft über den Erledigungszeitpunkt, nicht über das Anlegen: Ein offenes Anliegen ist
// eine laufende Sache und hat keine Frist.
func praedikatAnliegen(tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `erledigt_am IS NOT NULL
		  AND erledigt_am < NOW() - make_interval(days => $1::int + $2::int)`}
}

// praedikatKlassensatzReservierungen liefert die Bedingung für erledigte
// Klassensatz-Reservierungen. Dieselbe Frist wie die erledigten Anliegen (anliegen_tage): Eine
// Reservierung ist eine Anfrage aus dem Kollegiums-Portal wie ein Wunsch; erledigt ist ihr
// Zweck erreicht, offen bleibt sie.
//
// Anders als beim Anliegen steht „erledigt" hier in einer eigenen Spalte mit dem Zeitpunkt
// erledigt_am. Eine alte Reservierung ohne Zeitpunkt bleibt stehen: Ein erfundener wäre
// schlechter als keiner.
func praedikatKlassensatzReservierungen(tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `erledigt = true
		  AND erledigt_am IS NOT NULL
		  AND erledigt_am < NOW() - make_interval(days => $1::int + $2::int)`}
}

// ── Nachbuch-Meldungen ($1 Tage, $2 Kulanz) ─────────────────────────────────────
//
// Quittierte Meldungen fallen nach der Lesehistorie-Frist der Schülerbücherei, höchstens
// nach 30 Tagen: Eine quittierte Meldung ist erledigt,
// ihre Beteiligten stehen mit Namen darin, und länger als die Lesehistorie darf nichts
// den Schüler an ein Buch binden. Offene Meldungen haben keine Frist — sie sind Arbeit,
// die noch aussteht, und der Wächter der Betriebsbereitschaft nennt sie nach 14 Tagen.

// hoechstNachbuchMeldungenTage ist die Obergrenze der Frist, unabhängig von der Einstellung.
const hoechstNachbuchMeldungenTage = 30

// NachbuchMeldungenTage liefert die Frist: Lesehistorie-Frist, höchstens 30 Tage.
func NachbuchMeldungenTage(einst *SystemEinstellungen) int {
	tage := TageOderVorgabe(einst.LesehistorieTage, LesehistorieTageVorgabe)
	if tage <= 0 || tage > hoechstNachbuchMeldungenTage {
		return hoechstNachbuchMeldungenTage
	}
	return tage
}

// praedikatNachbuchMeldungen ist die Bedingung für quittierte Nachbuch-Meldungen.
func praedikatNachbuchMeldungen(tage, kulanz int) Loeschbedingung {
	return Loeschbedingung{Args: []any{tage, kulanz}, Where: `quittiert_am IS NOT NULL
		  AND quittiert_am < NOW() - make_interval(days => $1::int + $2::int)`}
}

// ── Audit-Aufbewahrung ($1 Monate, $2 Kulanz-Tage) ────────────────────────────
//
// Die Kulanz ist hier ein zusätzlicher Tag im selben Intervall, nicht ein zusätzlicher
// Monat — sonst gäbe der Wächter dem Job einen ganzen Monat Blindheit.

// praedikatProtokoll ist die Bedingung für audit_log (fachliche Datensatz-Historie).
func praedikatProtokoll(monate, kulanzTage int) Loeschbedingung {
	return Loeschbedingung{Args: []any{monate, kulanzTage},
		Where: `timestamp < NOW() - make_interval(months => $1::int, days => $2::int)`}
}

// praedikatVerwaltungsProtokoll ist die Bedingung für audit_logs (Admin-Aktionen inkl. IP-Adressen).
// Andere Tabelle, andere Zeitspalte, dieselbe Frist.
func praedikatVerwaltungsProtokoll(monate, kulanzTage int) Loeschbedingung {
	return Loeschbedingung{Args: []any{monate, kulanzTage},
		Where: `zeitstempel < NOW() - make_interval(months => $1::int, days => $2::int)`}
}

package bereitschaft

import (
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/jobs"
	"bibliothek/repository"
)

// testJetzt ist der feste Bezugspunkt statt time.Now(): Die Altersrechnung des Backups soll
// im Test nicht an der Wanduhr hängen.
var testJetzt = time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

// vorStunden liefert einen Zeitpunkt n Stunden vor testJetzt.
func vorStunden(n int) *time.Time {
	ts := testJetzt.Add(-time.Duration(n) * time.Hour)
	return &ts
}

// lageEingerichtet liefert eine Lage, in der alles eingerichtet ist. Jeder Fall unten nimmt
// genau eine Sache weg und prüft, dass genau diese gemeldet wird.
func lageEingerichtet() Lage {
	return Lage{
		Umgebung:              "production",
		S3Endpunkt:            "https://s3.example.net",
		S3Zugangsschluessel:   "AK",
		S3Geheimschluessel:    "SK",
		S3Bucket:              "bibliothek-backups",
		GeheimnisSchutzScharf: true,
		JWTGeheimnis:          "ein-eigenes-langes-geheimnis-mit-genug-zeichen",
		Schluessel:            "eigener-schluessel-32-zeichen-ab",
		IMAPHost:              "srv1.philipp-reis-schule.de",
		SelbstanmeldeDomain:   "philipp-reis-schule.de",
		OeffentlicheAdresse:   "https://flasch3.herzog-dupont.de",
		SMTPHost:              "srv1.philipp-reis-schule.de",
		DemoSchueler:          0,
		DemoExemplare:         0,
		// Lieferanten: erhoben, keiner trägt eine Kennung der früheren Startdaten.
		BeispielLieferanten: []string{},
		RechteLive:          rechteWieVorgabe(),
		AdminKonten:         []string{"Nina Berger (nberger@philipp-reis-schule.de)"},
		// Klassen: erhoben und leer, also alles verbunden.
		KlassenOhneLehrkraft:  []string{},
		VerwaisteZuordnungen:  []string{},
		VerwaisteBuecherliste: []string{},
		// Nächtliches Backup: Schlüssel stark, letzter Lauf heute Nacht.
		BackupSchluesselGesetzt: true,
		BackupSchluesselSchwach: false,
		LetztesBackup:           vorStunden(10),
		Jetzt:                   testJetzt,
		// Restore-Probe: letzter Wochenlauf erfolgreich.
		RestoreProbe: &jobs.RestoreProbeErgebnis{
			Zeitpunkt: *vorStunden(3 * 24), Erfolg: true,
			BackupDatei: "backup_2026-08-17T023000Z.sql.gz.enc", Tabellen: 31,
		},
		// Löschroutinen: alle erhoben, nichts überfällig, keine Frist auf 0.
		LoeschRueckstand: rueckstandSauber(),
		// Ehemalige mit offenen Vorgängen: erhoben, keiner.
		EhemaligeMitOffenenVorgaengen: zahl(0),
		// Nachbuch-Meldungen: erhoben, keine liegt länger als zwei Wochen.
		NachbuchMeldungenOffen: zahl(0),
		// Ferientabelle: reicht weit genug (testJetzt 2026 + 2 ≤ 2030).
		FerientabelleBis: 2030,
		// Ferien für Leihfristen: ebenso (Programmtabelle bis 2030).
		UebrigeFerienBis: 2030,
		// Schadensersatz-Bescheid: erhoben, keine Pflichtangabe fehlt.
		BescheidFehlend: []string{},
		// Schlüssel und Bestand: geprüft, beide Spalten lesbar.
		SchluesselProbe: &repository.SchluesselProbe{Geprueft: 2},
	}
}

// zahl liefert einen Zeiger auf n — für die „erhoben"-Felder der Lage.
func zahl(n int) *int { return &n }

// rechteWieVorgabe baut die Rechte der Datenbank genau wie die Vorgabe im Code. Jeder Fall
// zu den Rechten verstellt davon ausgehend genau eine Zeile.
func rechteWieVorgabe() map[string]map[string]bool {
	live := map[string]map[string]bool{}
	for _, v := range db.RechteVorgabe {
		if live[v.Role] == nil {
			live[v.Role] = map[string]bool{}
		}
		live[v.Role][v.Permission] = v.Allowed
	}
	return live
}

func befundZu(t *testing.T, befunde []Befund, bereich string) Befund {
	t.Helper()
	for _, b := range befunde {
		if b.Bereich == bereich {
			return b
		}
	}
	t.Fatalf("kein Befund für %q — geprüft wurden: %v", bereich, bereiche(befunde))
	return Befund{}
}

func bereiche(befunde []Befund) []string {
	out := make([]string, 0, len(befunde))
	for _, b := range befunde {
		out = append(out, b.Bereich)
	}
	return out
}

// Die Gegenprobe zuerst: Ist alles eingerichtet, wird nichts gemeldet. Ohne sie wäre ein
// Wächter, der immer meldet, von einem, der richtig meldet, nicht zu unterscheiden.
func TestBetriebsbereitschaft_EingerichteteAnlageIstStill(t *testing.T) {
	for _, b := range Pruefe(lageEingerichtet()) {
		if b.Stufe != StufeOK {
			t.Errorf("%s meldet %q, obwohl alles eingerichtet ist: %s", b.Bereich, b.Stufe, b.Befund)
		}
	}
}

func TestBetriebsbereitschaft_MeldetJedeLuecke(t *testing.T) {
	faelle := []struct {
		name    string
		aendere func(*Lage)
		bereich string
		stufe   string
		// Ein Wort, das im Befund oder in der Abhilfe vorkommen muss, damit eine Meldung
		// mit der richtigen Stufe nicht von etwas anderem spricht.
		enthaelt string
	}{
		{
			name:     "kein Ziel ausser Haus",
			aendere:  func(l *Lage) { l.S3Bucket = "" },
			bereich:  "Auslagerung der Backups",
			stufe:    StufeKritisch,
			enthaelt: "S3_BUCKET",
		},
		{
			name: "Schlüssel passt nicht zum Bestand",
			aendere: func(l *Lage) {
				l.SchluesselProbe = &repository.SchluesselProbe{Geprueft: 2, NichtLesbar: []string{"SMTP-Passwort"}}
			},
			bereich:  "Schlüssel und Bestand",
			stufe:    StufeKritisch,
			enthaelt: "SMTP-Passwort",
		},
		{
			name:     "Schlüssel-Probe nicht lesbar",
			aendere:  func(l *Lage) { l.SchluesselProbe = nil },
			bereich:  "Schlüssel und Bestand",
			stufe:    StufeWarnung,
			enthaelt: "nicht prüfen",
		},
		{
			name:     "Mailversand darf Klartext",
			aendere:  func(l *Lage) { l.SMTPKlartextErlaubt = true },
			bereich:  "Mailversand (Mahnwesen)",
			stufe:    StufeWarnung,
			enthaelt: "SMTP_ALLOW_PLAINTEXT",
		},
		{
			name:     "Mailversand prüft das Zertifikat nicht",
			aendere:  func(l *Lage) { l.SMTPUnsicheresTLSErlaubt = true },
			bereich:  "Mailversand (Mahnwesen)",
			stufe:    StufeWarnung,
			enthaelt: "SMTP_ALLOW_INSECURE_TLS",
		},
		{
			name:     "Bescheid ohne Schulnummer",
			aendere:  func(l *Lage) { l.BescheidFehlend = []string{"Schulnummer"} },
			bereich:  "Schadensersatz-Bescheid",
			stufe:    StufeWarnung,
			enthaelt: "Schulnummer",
		},
		{
			name:     "Bescheid-Angaben nicht erhoben",
			aendere:  func(l *Lage) { l.BescheidFehlend = nil },
			bereich:  "Schadensersatz-Bescheid",
			stufe:    StufeWarnung,
			enthaelt: "nicht erhoben",
		},
		{
			name:     "Ehemalige mit offenem Vorgang seit über einem Jahr",
			aendere:  func(l *Lage) { l.EhemaligeMitOffenenVorgaengen = zahl(3) },
			bereich:  "Ehemalige mit offenen Vorgängen",
			stufe:    StufeWarnung,
			enthaelt: "Verlust",
		},
		{
			// 2026 + 2 > 2027: zwei Jahre vor dem Ende der Tabelle kommt die Warnung.
			name:     "Ferientabelle läuft in unter zwei Jahren aus",
			aendere:  func(l *Lage) { l.FerientabelleBis = 2027 },
			bereich:  "Ferientabelle",
			stufe:    StufeWarnung,
			enthaelt: "kmk.org",
		},
		{
			// Dieselbe Schwelle für Herbst-, Weihnachts- und Osterferien; die Abhilfe ist ein
			// Programm-Update, keine Einstellung.
			name:     "Ferien für Leihfristen laufen in unter zwei Jahren aus",
			aendere:  func(l *Lage) { l.UebrigeFerienBis = 2027 },
			bereich:  "Ferien für Leihfristen",
			stufe:    StufeWarnung,
			enthaelt: "kultus.hessen.de",
		},
		{
			name:     "Ehemalige nicht erhoben",
			aendere:  func(l *Lage) { l.EhemaligeMitOffenenVorgaengen = nil },
			bereich:  "Ehemalige mit offenen Vorgängen",
			stufe:    StufeWarnung,
			enthaelt: "Nicht erhoben",
		},
		{
			name:     "Beispiel-Geheimnis im Echtbetrieb",
			aendere:  func(l *Lage) { l.JWTGeheimnis = "super-secret-default-key-at-least-32-bytes" },
			bereich:  "Geheimnisse",
			stufe:    StufeKritisch,
			enthaelt: "Sitzungen fälschen",
		},
		{
			name:     "eigene Geheimnisse, Absicherung aber nicht scharf",
			aendere:  func(l *Lage) { l.GeheimnisSchutzScharf = false },
			bereich:  "Geheimnisse",
			stufe:    StufeWarnung,
			enthaelt: "ENFORCE_PROD_SECRETS",
		},
		{
			name:     "mock-Anmeldung im Echtbetrieb",
			aendere:  func(l *Lage) { l.IMAPHost = "mock" },
			bereich:  "Anmeldung",
			stufe:    StufeKritisch,
			enthaelt: "JEDES Passwort",
		},
		{
			name:     "kein IMAP-Host",
			aendere:  func(l *Lage) { l.IMAPHost = "" },
			bereich:  "Anmeldung",
			stufe:    StufeKritisch,
			enthaelt: "anmelden",
		},
		{
			name:     "Selbstanmeldung nicht eingerichtet",
			aendere:  func(l *Lage) { l.SelbstanmeldeDomain = "" },
			bereich:  "Selbstanmeldung Kollegium",
			stufe:    StufeWarnung,
			enthaelt: "SELBSTANMELDUNG_DOMAIN",
		},
		{
			name:     "keine oeffentliche Adresse",
			aendere:  func(l *Lage) { l.OeffentlicheAdresse = "" },
			bereich:  "Bestell-Bestätigungslink",
			stufe:    StufeWarnung,
			enthaelt: "bestätigen",
		},
		{
			name:     "kein SMTP-Server",
			aendere:  func(l *Lage) { l.SMTPHost = "" },
			bereich:  "Mailversand (Mahnwesen)",
			stufe:    StufeWarnung,
			enthaelt: "Mahnungen",
		},
		{
			name:     "Demo-Daten im Bestand",
			aendere:  func(l *Lage) { l.DemoSchueler = 2000 },
			bereich:  "Demo-Daten",
			stufe:    StufeWarnung,
			enthaelt: "2000",
		},
		{
			// Halb aufgeräumt: Schüler weg, Exemplare noch da.
			name:     "Demo-Exemplare ohne Demo-Schüler",
			aendere:  func(l *Lage) { l.DemoExemplare = 300 },
			bereich:  "Demo-Daten",
			stufe:    StufeWarnung,
			enthaelt: "300 Demo-Exemplare",
		},
		{
			// Die früheren Startdaten: Die Bestellung geht hinaus, Bücher kommen nie.
			name:     "Beispiel-Lieferant im Bestand",
			aendere:  func(l *Lage) { l.BeispielLieferanten = []string{"Klett Verlag"} },
			bereich:  "Lieferanten",
			stufe:    StufeKritisch,
			enthaelt: "Klett Verlag",
		},
		{
			name:     "Lieferantenliste nicht lesbar",
			aendere:  func(l *Lage) { l.BeispielLieferanten = nil },
			bereich:  "Lieferanten",
			stufe:    StufeWarnung,
			enthaelt: "nicht geprüft",
		},
		{
			// Der Seed fasst bestehende Zeilen nie an: Ein im Code geänderter Wert erreicht
			// die Anlage nicht, die Zeile in der Datenbank bleibt.
			name:     "Rechte-Wert weicht von der Vorgabe ab",
			aendere:  func(l *Lage) { l.RechteLive["HELFER"]["view_students"] = true },
			bereich:  "Rechte-Vorgabe",
			stufe:    StufeWarnung,
			enthaelt: "HELFER/view_students live an, Vorgabe aus",
		},
		{
			name:     "Rechte-Zeile fehlt live",
			aendere:  func(l *Lage) { delete(l.RechteLive["ADMIN"], "manage_users") },
			bereich:  "Rechte-Vorgabe",
			stufe:    StufeWarnung,
			enthaelt: "ADMIN/manage_users fehlt live",
		},
		{
			// Eine umbenannte Rolle, deren Zeilen die Migration nicht mitnahm.
			name:     "verwaiste Live-Zeile",
			aendere:  func(l *Lage) { l.RechteLive["LEHRER"] = map[string]bool{"view_books": true} },
			bereich:  "Rechte-Vorgabe",
			stufe:    StufeWarnung,
			enthaelt: "LEHRER/view_books live an, in der Vorgabe unbekannt",
		},
		{
			name:     "Admin-Konten nicht lesbar",
			aendere:  func(l *Lage) { l.AdminKonten = nil },
			bereich:  "Admin-Konten",
			stufe:    StufeWarnung,
			enthaelt: "nicht gelesen",
		},
		{
			// Ohne aktiven Admin erreichen die Kritisch-Alarme niemanden, und niemand kann
			// das System verwalten.
			name:     "kein aktives Admin-Konto",
			aendere:  func(l *Lage) { l.AdminKonten = []string{} },
			bereich:  "Admin-Konten",
			stufe:    StufeKritisch,
			enthaelt: "Kein aktives Admin-Konto",
		},
		{
			name:     "Live-Rechte nicht lesbar",
			aendere:  func(l *Lage) { l.RechteLive = nil },
			bereich:  "Rechte-Vorgabe",
			stufe:    StufeWarnung,
			enthaelt: "nicht gelesen",
		},
		// Nächtliches Backup: Mit diesen Befunden erreicht ein stehender Backup-Job die
		// tägliche Alarm-Mail.
		{
			name:     "Backup-Key fehlt",
			aendere:  func(l *Lage) { l.BackupSchluesselGesetzt = false },
			bereich:  "Nächtliches Backup",
			stufe:    StufeKritisch,
			enthaelt: "BACKUP_ENCRYPTION_KEY",
		},
		{
			name:     "noch nie ein Backup",
			aendere:  func(l *Lage) { l.LetztesBackup = nil },
			bereich:  "Nächtliches Backup",
			stufe:    StufeKritisch,
			enthaelt: "Noch nie",
		},
		{
			name:     "Backup älter als 48h",
			aendere:  func(l *Lage) { l.LetztesBackup = vorStunden(50) },
			bereich:  "Nächtliches Backup",
			stufe:    StufeKritisch,
			enthaelt: "vor 50 Stunden",
		},
		{
			name:     "ein Lauf verpasst (26h)",
			aendere:  func(l *Lage) { l.LetztesBackup = vorStunden(30) },
			bereich:  "Nächtliches Backup",
			stufe:    StufeWarnung,
			enthaelt: "verpasst",
		},
		{
			name:     "schwacher Backup-Key",
			aendere:  func(l *Lage) { l.BackupSchluesselSchwach = true },
			bereich:  "Nächtliches Backup",
			stufe:    StufeWarnung,
			enthaelt: "zu kurz",
		},
		// Restore-Probe: Ein Backup, das sich nicht einspielen lässt, ist so viel wert wie
		// keins, und ohne diese Befunde erführe das niemand vor dem Ernstfall.
		{
			name:     "Restore-Probe nie gelaufen",
			aendere:  func(l *Lage) { l.RestoreProbe = nil },
			bereich:  "Restore-Probe",
			stufe:    StufeWarnung,
			enthaelt: "kein Probelauf",
		},
		{
			name: "Restore-Probe fehlgeschlagen",
			aendere: func(l *Lage) {
				l.RestoreProbe = &jobs.RestoreProbeErgebnis{
					Zeitpunkt: *vorStunden(24), Erfolg: false,
					Fehler: "entschlüsselung fehlgeschlagen (falscher Schlüssel oder beschädigte Datei)",
				}
			},
			bereich:  "Restore-Probe",
			stufe:    StufeKritisch,
			enthaelt: "FEHLGESCHLAGEN",
		},
		{
			name: "Restore-Probe steht still",
			aendere: func(l *Lage) {
				l.RestoreProbe = &jobs.RestoreProbeErgebnis{
					Zeitpunkt: *vorStunden(12 * 24), Erfolg: true, Tabellen: 31,
				}
			},
			bereich:  "Restore-Probe",
			stufe:    StufeKritisch,
			enthaelt: "steht still",
		},
	}

	for _, f := range faelle {
		t.Run(f.name, func(t *testing.T) {
			l := lageEingerichtet()
			f.aendere(&l)
			befunde := Pruefe(l)

			b := befundZu(t, befunde, f.bereich)
			if b.Stufe != f.stufe {
				t.Errorf("Stufe %q statt %q: %s", b.Stufe, f.stufe, b.Befund)
			}
			zusammen := b.Befund + " " + b.Folge + " " + b.Abhilfe
			if !strings.Contains(zusammen, f.enthaelt) {
				t.Errorf("die Meldung nennt %q nicht: %s", f.enthaelt, zusammen)
			}

			// Ein Mangel ohne Abhilfe landet auf einem Zettel statt in der .env.
			if b.Abhilfe == "" {
				t.Errorf("%s meldet einen Mangel, sagt aber nicht, was zu tun ist", b.Bereich)
			}
			if b.Folge == "" {
				t.Errorf("%s meldet einen Mangel, sagt aber nicht, warum er zählt", b.Bereich)
			}

			// Die anderen Bereiche bleiben still: Löst eine fehlende Einstellung fünf
			// Meldungen aus, ist der Wächter nicht zu gebrauchen.
			for _, andere := range befunde {
				if andere.Bereich != f.bereich && andere.Stufe != StufeOK {
					t.Errorf("Nebenwirkung: %s meldet %q, obwohl nur %q verstellt wurde",
						andere.Bereich, andere.Stufe, f.bereich)
				}
			}
		})
	}
}

// Auf dem Entwicklungsrechner sind mock-Anmeldung und Beispiel-Geheimnisse richtig. Meldete
// die Prüfung sie dort als Mangel, wäre sie dauerhaft rot und damit wertlos.
func TestBetriebsbereitschaft_SpielwieseIstKeinMangel(t *testing.T) {
	for _, env := range []string{"local", "development", "test"} {
		l := lageEingerichtet()
		l.Umgebung = env
		l.IMAPHost = "mock"
		l.JWTGeheimnis = "super-secret-default-key-at-least-32-bytes"
		l.Schluessel = "super-secure-aes-key-32-chars-ok"
		l.GeheimnisSchutzScharf = false
		// Auf dem Entwicklungsrechner gibt es weder Schlüssel noch Backup-Dateien.
		l.BackupSchluesselGesetzt = false
		l.LetztesBackup = nil

		for _, bereich := range []string{"Anmeldung", "Geheimnisse", "Nächtliches Backup"} {
			if b := befundZu(t, Pruefe(l), bereich); b.Stufe != StufeOK {
				t.Errorf("APP_ENV=%s: %s meldet %q — auf der Spielwiese ist das richtig so (%s)",
					env, bereich, b.Stufe, b.Befund)
			}
		}
	}
}

// rueckstandSauber liefert den Rückstand einer Nacht, die getan hat, was sie soll: alle
// Routinen erhoben, keine überfällige Zeile, keine Frist auf 0.
func rueckstandSauber() []repository.LoeschRueckstand {
	return []repository.LoeschRueckstand{
		{Routine: "Schüler-Anonymisierung", Frist: "180 Tage (gelöscht) / 360 Tage (Abgänger)"},
		{Routine: "Abgänger endgültig löschen", Frist: "30 Tage nach Schuljahresende"},
		{Routine: "Lesehistorie Schülerbücherei", Frist: "90 Tage"},
		{Routine: "Lesehistorie Lernmittel", Frist: "730 Tage"},
		{Routine: "Erledigte Anliegen", Frist: "365 Tage"},
		{Routine: "Audit-Aufbewahrung", Frist: "24 Monate"},
	}
}

// Die schärfste Stufe gewinnt, gleich an welcher Stelle sie steht; ohne Befund ist alles in
// Ordnung.
func TestGesamt(t *testing.T) {
	befunde := func(stufen ...string) []Befund {
		var b []Befund
		for _, s := range stufen {
			b = append(b, Befund{Stufe: s})
		}
		return b
	}
	for _, f := range []struct {
		stufen []string
		want   string
	}{
		{nil, StufeOK},
		{[]string{StufeOK, StufeOK}, StufeOK},
		{[]string{StufeOK, StufeWarnung, StufeOK}, StufeWarnung},
		{[]string{StufeWarnung, StufeKritisch, StufeOK}, StufeKritisch},
		{[]string{StufeKritisch, StufeWarnung}, StufeKritisch},
		{[]string{StufeOK, StufeKritisch}, StufeKritisch},
	} {
		if ist := Gesamt(befunde(f.stufen...)); ist != f.want {
			t.Errorf("Gesamt über %v: %s, erwartet %s", f.stufen, ist, f.want)
		}
	}
}

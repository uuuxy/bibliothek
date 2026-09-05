package api

import (
	"strings"
	"testing"
	"time"

	"bibliothek/db"
	"bibliothek/jobs"
	"bibliothek/repository"
)

// Eine Lage, in der ALLES eingerichtet ist. Ausgangspunkt aller Fälle unten: Jeder Test
// nimmt genau eine Sache weg und prüft, dass genau diese eine gemeldet wird.
// Fester Bezugspunkt statt time.Now(): Die Altersrechnung des Backup-Befundes soll im
// Test nicht an der Wanduhr hängen.
var testJetzt = time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)

// vorStunden liefert einen Zeitpunkt n Stunden vor testJetzt.
func vorStunden(n int) *time.Time {
	ts := testJetzt.Add(-time.Duration(n) * time.Hour)
	return &ts
}

func lageEingerichtet() Lage {
	return Lage{
		AppEnv:              "production",
		S3Endpoint:          "https://s3.example.net",
		S3AccessKey:         "AK",
		S3SecretKey:         "SK",
		S3Bucket:            "bibliothek-backups",
		EnforceProdSecrets:  true,
		JWTSecret:           "ein-eigenes-langes-geheimnis-mit-genug-zeichen",
		AppEncryptionKey:    "eigener-schluessel-32-zeichen-ab",
		ImapHost:            "srv1.philipp-reis-schule.de",
		SelbstanmeldeDomain: "philipp-reis-schule.de",
		OeffentlicheAdresse: "https://flasch3.herzog-dupont.de",
		SmtpHost:            "srv1.philipp-reis-schule.de",
		DemoSchueler:        0,
		RechteLive:          rechteWieVorgabe(),
		AdminKonten:         []string{"Peter Flasch (pflasch@philipp-reis-schule.de)"},
		// Klassen-Drift (F3): erhoben und leer = alles verbunden.
		KlassenOhneLehrkraft:  []string{},
		VerwaisteZuordnungen:  []string{},
		VerwaisteBuecherliste: []string{},
		// Nächtliches Backup (B3): Key stark, letzter Lauf heute Nacht.
		BackupKeySet:  true,
		BackupKeyWeak: false,
		LetztesBackup: vorStunden(10),
		Jetzt:         testJetzt,
		// Restore-Probe: letzter Wochenlauf erfolgreich.
		RestoreProbe: &jobs.RestoreProbeErgebnis{
			Zeitpunkt: *vorStunden(3 * 24), Erfolg: true,
			BackupDatei: "backup_2026-08-17T023000Z.sql.gz.enc", Tabellen: 31,
		},
		// DSGVO-Löschroutinen: alle erhoben, nichts überfällig, keine Frist auf 0.
		LoeschRueckstand: rueckstandSauber(),
		// Ehemalige mit offenen Vorgängen: erhoben, keiner.
		EhemaligeMitOffenenVorgaengen: zahl(0),
	}
}

// zahl liefert einen Zeiger auf n — für die „erhoben"-Felder der Lage.
func zahl(n int) *int { return &n }

// rechteWieVorgabe baut die Live-Rechte deckungsgleich zur Code-Vorgabe — der
// Ausgangspunkt „keine Drift". Jeder Rechte-Testfall verstellt davon ausgehend
// genau eine Zeile.
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

// Die Gegenprobe zuerst: Ist alles eingerichtet, darf NICHTS gemeldet werden.
//
// Ohne sie wäre ein Wächter, der immer meckert, von einem, der richtig meckert, nicht zu
// unterscheiden — und ein Wächter, der immer rot ist, wird abgeschaltet statt gelesen.
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
		// Ein Wort, das im Befund oder in der Abhilfe vorkommen MUSS. Verhindert, dass
		// eine Meldung zwar die richtige Stufe trägt, aber von etwas anderem spricht.
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
			name:     "Ehemalige mit offenem Vorgang seit über einem Jahr",
			aendere:  func(l *Lage) { l.EhemaligeMitOffenenVorgaengen = zahl(3) },
			bereich:  "Ehemalige mit offenen Vorgängen",
			stufe:    StufeWarnung,
			enthaelt: "Verlust",
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
			aendere:  func(l *Lage) { l.JWTSecret = "super-secret-default-key-at-least-32-bytes" },
			bereich:  "Geheimnisse",
			stufe:    StufeKritisch,
			enthaelt: "Sitzungen fälschen",
		},
		{
			name:     "eigene Geheimnisse, Absicherung aber nicht scharf",
			aendere:  func(l *Lage) { l.EnforceProdSecrets = false },
			bereich:  "Geheimnisse",
			stufe:    StufeWarnung,
			enthaelt: "ENFORCE_PROD_SECRETS",
		},
		{
			name:     "mock-Anmeldung im Echtbetrieb",
			aendere:  func(l *Lage) { l.ImapHost = "mock" },
			bereich:  "Anmeldung",
			stufe:    StufeKritisch,
			enthaelt: "JEDES Passwort",
		},
		{
			name:     "kein IMAP-Host",
			aendere:  func(l *Lage) { l.ImapHost = "" },
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
			aendere:  func(l *Lage) { l.SmtpHost = "" },
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
			// Die Kollegium-Wunde: Der Seed fasst bestehende Zeilen nie an — ein im
			// Code geänderter Wert erreicht die Anlage nicht, die Live-Zeile bleibt.
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
			// Reste alter Vokabulare — etwa eine umbenannte Rolle, deren Zeilen die
			// Migration nicht mitnahm.
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
			// Ohne aktiven Admin erreichen die Kritisch-Alarme niemanden — und
			// niemand kann das System verwalten.
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
		// Nächtliches Backup (Ausfallmatrix 20.08.2026, B3): Erst mit diesen Befunden
		// erreicht ein stehender Backup-Job die tägliche Kritisch-Alarm-Mail — vorher
		// sah ihn nur das Dashboard-Badge, und nur, wer die Seite öffnete.
		{
			name:     "Backup-Key fehlt",
			aendere:  func(l *Lage) { l.BackupKeySet = false },
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
			aendere:  func(l *Lage) { l.BackupKeyWeak = true },
			bereich:  "Nächtliches Backup",
			stufe:    StufeWarnung,
			enthaelt: "zu kurz",
		},
		// Restore-Probe (Schema-Erweiterung 21.08.2026): Ein Backup, das sich nicht
		// einspielen lässt, ist so viel wert wie keins — und ohne diese Befunde
		// erführe das niemand vor dem Ernstfall.
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

			// Und die anderen Bereiche bleiben still: Ein Wächter, bei dem eine fehlende
			// Einstellung fünf Meldungen auslöst, ist nicht zu gebrauchen.
			for _, andere := range befunde {
				if andere.Bereich != f.bereich && andere.Stufe != StufeOK {
					t.Errorf("Nebenwirkung: %s meldet %q, obwohl nur %q verstellt wurde",
						andere.Bereich, andere.Stufe, f.bereich)
				}
			}
		})
	}
}

// Optionale Rechte (db.RechteOptional) sind zum Umschalten GEBAUT: Schaltet ein
// Admin dem Helfer die Vormerkungen zu, ist das der vorgesehene Gebrauch — keine
// Drift. Ohne diese Ausnahme stünde jede nutzende Anlage dauerhaft gelb, und
// Dauerwarnungen erziehen zum Wegsehen. Die Existenz-Prüfung bleibt scharf:
// Verschwindet die Zeile ganz, fehlt die Tür in der Rechte-Matrix — Warnung.
func TestBetriebsbereitschaft_OptionalesRechtIstKeineDrift(t *testing.T) {
	t.Run("eingeschaltet ist kein Befund", func(t *testing.T) {
		l := lageEingerichtet()
		l.RechteLive["HELFER"]["manage_vormerkungen"] = true
		b := befundZu(t, Pruefe(l), "Rechte-Vorgabe")
		if b.Stufe != StufeOK {
			t.Errorf("optionales Recht eingeschaltet → %q statt OK: %s", b.Stufe, b.Befund)
		}
	})
	t.Run("fehlende Zeile warnt weiterhin", func(t *testing.T) {
		l := lageEingerichtet()
		delete(l.RechteLive["HELFER"], "manage_vormerkungen")
		b := befundZu(t, Pruefe(l), "Rechte-Vorgabe")
		if b.Stufe != StufeWarnung || !strings.Contains(b.Befund+b.Folge, "HELFER/manage_vormerkungen fehlt live") {
			t.Errorf("fehlende optionale Zeile muss warnen, got %q: %s", b.Stufe, b.Befund)
		}
	})
}

// Auf dem Entwicklungsrechner sind mock-Anmeldung und Beispiel-Geheimnisse RICHTIG.
// Meldete die Prüfung sie dort als Mangel, wäre sie dauerhaft rot — und damit wertlos.
func TestBetriebsbereitschaft_SpielwieseIstKeinMangel(t *testing.T) {
	for _, env := range []string{"local", "development", "test"} {
		l := lageEingerichtet()
		l.AppEnv = env
		l.ImapHost = "mock"
		l.JWTSecret = "super-secret-default-key-at-least-32-bytes"
		l.AppEncryptionKey = "super-secure-aes-key-32-chars-ok"
		l.EnforceProdSecrets = false
		// Auf dem Entwicklungsrechner gibt es weder Key noch Backup-Dateien —
		// dauerhaft rot wäre der Wächter wertlos.
		l.BackupKeySet = false
		l.LetztesBackup = nil

		for _, bereich := range []string{"Anmeldung", "Geheimnisse", "Nächtliches Backup"} {
			if b := befundZu(t, Pruefe(l), bereich); b.Stufe != StufeOK {
				t.Errorf("APP_ENV=%s: %s meldet %q — auf der Spielwiese ist das richtig so (%s)",
					env, bereich, b.Stufe, b.Befund)
			}
		}
	}
}

// Die Liste der Beispiel-Geheimnisse ist mit main.go geteilt. Bricht sie, startet der
// Server mit einem öffentlich bekannten Schlüssel — deshalb hier festgehalten.
func TestIstBekanntesDefaultGeheimnis(t *testing.T) {
	for _, wert := range []string{
		"super-secret-default-key-at-least-32-bytes",
		"super-secure-aes-key-32-chars-ok",
		"supergeheim_lokal",
	} {
		if !IstBekanntesDefaultGeheimnis(wert) {
			t.Errorf("%q wird nicht als Beispiel-Geheimnis erkannt", wert)
		}
	}
	for _, wert := range []string{"", "ein-eigenes-langes-geheimnis-mit-genug-zeichen"} {
		if IstBekanntesDefaultGeheimnis(wert) {
			t.Errorf("%q gilt fälschlich als Beispiel-Geheimnis", wert)
		}
	}
}

// Viele Abweichungen dürfen die Auskunft nicht fluten: gezeigt werden sechs,
// der Rest wird gezählt. (Der Extremfall — leere, aber lesbare Tabelle — meldet
// JEDE Vorgabe-Zeile als fehlend.)
func TestBetriebsbereitschaft_RechteAbweichungenWerdenGekappt(t *testing.T) {
	l := lageEingerichtet()
	l.RechteLive = map[string]map[string]bool{}

	b := befundZu(t, Pruefe(l), "Rechte-Vorgabe")
	if b.Stufe != StufeWarnung {
		t.Fatalf("Stufe %q statt Warnung: %s", b.Stufe, b.Befund)
	}
	if !strings.Contains(b.Befund, "weitere") {
		t.Errorf("die Kappung fehlt — der Befund listet alles: %s", b.Befund)
	}
	if strings.Count(b.Befund, ";") > 6 {
		t.Errorf("mehr als sechs Einträge gezeigt: %s", b.Befund)
	}
}

// Die Klassen-Drift-Prüfung (F3): nicht erhoben ≠ alles verbunden, und jede
// gerissene Text-Verbindung wird benannt statt still geschluckt.
func TestKlassenDrift(t *testing.T) {
	t.Run("nicht erhoben ist eine Warnung, kein OK", func(t *testing.T) {
		l := lageEingerichtet()
		l.KlassenOhneLehrkraft = nil
		b := pruefeKlassenDrift(l)
		if b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "nicht geprüft") {
			t.Errorf("nil muss als 'nicht geprüft' warnen, got %+v", b)
		}
	})
	t.Run("Klasse ohne Lehrkraft wird benannt", func(t *testing.T) {
		l := lageEingerichtet()
		l.KlassenOhneLehrkraft = []string{"06a"}
		b := pruefeKlassenDrift(l)
		if b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "06a") {
			t.Errorf("06a muss in der Warnung stehen, got %+v", b)
		}
		if b.Abhilfe == "" || b.Folge == "" {
			t.Error("Warnung ohne Folge/Abhilfe landet auf einem Zettel statt in der Zuordnung")
		}
	})
	t.Run("verwaiste Buecherliste wird benannt", func(t *testing.T) {
		l := lageEingerichtet()
		l.VerwaisteBuecherliste = []string{"7x"}
		if b := pruefeKlassenDrift(l); b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "7x") {
			t.Errorf("7x muss in der Warnung stehen, got %+v", b)
		}
	})
}

// rueckstandSauber: alle Routinen erhoben, keine überfällige Zeile, keine Frist auf 0 —
// der Ausgangspunkt „die Nacht hat getan, was sie soll".
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

// Prüfung 22.08.2026: Die DSGVO-Anonymisierung lief zweimal monatelang ins Leere; nur eine
// Logzeile wusste es. Jetzt urteilt die Selbstprüfung über den ZUSTAND — seit dem
// 23.08.2026 über den ALLER Löschroutinen, nicht nur der Anonymisierung.
func TestPruefeDsgvoRoutinen(t *testing.T) {
	t.Run("nicht erhoben ist eine Warnung, kein OK", func(t *testing.T) {
		if b := pruefeDsgvoRoutinen(Lage{LoeschRueckstand: nil}); b.Stufe != StufeWarnung {
			t.Fatalf("nicht erhoben = Warnung: %+v", b)
		}
	})

	t.Run("sauber ist ok", func(t *testing.T) {
		b := pruefeDsgvoRoutinen(Lage{LoeschRueckstand: rueckstandSauber()})
		if b.Stufe != StufeOK || !strings.Contains(b.Befund, "6 Löschroutinen") {
			t.Fatalf("sauber = ok über alle Routinen: %+v", b)
		}
	})

	// Der eigentliche Zugewinn: JEDE Routine muss den Alarm allein auslösen können.
	// Vorher konnte das nur die Anonymisierung — die anderen liefen unbeobachtet.
	for i, r := range rueckstandSauber() {
		t.Run("überfällig: "+r.Routine, func(t *testing.T) {
			stand := rueckstandSauber()
			stand[i].Zeilen = 3
			b := pruefeDsgvoRoutinen(Lage{LoeschRueckstand: stand})
			if b.Stufe != StufeKritisch {
				t.Fatalf("%s überfällig muss kritisch sein: %+v", r.Routine, b)
			}
			if !strings.Contains(b.Befund, r.Routine) || !strings.Contains(b.Befund, "3") {
				t.Errorf("Befund nennt Routine und Anzahl nicht: %s", b.Befund)
			}
			if !strings.Contains(b.Befund, r.Frist) {
				t.Errorf("Befund nennt die geltende Frist nicht: %s", b.Befund)
			}
		})
	}

	t.Run("abgeschaltete Frist ist eine Warnung, kein Fehler", func(t *testing.T) {
		stand := rueckstandSauber()
		stand[2].Aus = true
		b := pruefeDsgvoRoutinen(Lage{LoeschRueckstand: stand})
		if b.Stufe != StufeWarnung || !strings.Contains(b.Befund, "Lesehistorie Schülerbücherei") {
			t.Fatalf("Frist auf 0 = Warnung mit Namen der Routine: %+v", b)
		}
	})

	// Rangfolge: Ein echter Rückstand darf nicht von einer abgeschalteten Frist
	// verdeckt werden — sonst meldete die Seite „Warnung", während Daten stehen bleiben.
	t.Run("Rückstand schlägt abgeschaltete Frist", func(t *testing.T) {
		stand := rueckstandSauber()
		stand[2].Aus = true
		stand[5].Zeilen = 1
		if b := pruefeDsgvoRoutinen(Lage{LoeschRueckstand: stand}); b.Stufe != StufeKritisch {
			t.Fatalf("Rückstand muss die Warnung überstimmen: %+v", b)
		}
	})
}

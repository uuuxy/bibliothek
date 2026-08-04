// littera-altbestand überträgt einen Littera-Export nach PostgreSQL.
//
// Die Quelle sind die CSV-Dateien, die `mdb-export` aus littera_sav.mdb erzeugt (siehe
// internal/littera.Dateien) — nicht die Access-Datei selbst: Deren ODBC-Treiber gibt es
// nur unter Windows, weder auf dem Arbeitsrechner noch im Container.
//
// Aufruf:
//
//	go run ./cmd/littera-altbestand -csv ./littera-export -db postgres://…
//	go run ./cmd/littera-altbestand -csv ./littera-export -trocken
//
// Was übernommen wird, steuern die drei Schalter -bestand, -personen und -ausleihen.
// Die Vorgabe ist NUR der Bestand, und das ist eine bewusste Entscheidung: Der geprüfte
// Export ist ein Stand von 2010 (Schüler-Geburtsjahre 1989–2001, letzte Ausleihe 2010,
// Fristen bis 2011). Wer Personen und Ausleihen daraus übernimmt, legt Schüler an, die
// heute Mitte dreißig sind, und meldet ein Viertel des Bestands als verliehen. Für einen
// aktuellen Export ist beides richtig — deshalb die Schalter statt einer Sperre.
//
// Rückgabewerte:
//
//	0 – vollständig übernommen. Warnungen können im Protokoll stehen, aber jeder
//	    Quelldatensatz ist angekommen.
//	1 – abgebrochen. Ab dem gemeldeten Punkt wurde nichts mehr geschrieben.
//	2 – abgeschlossen, aber unvollständig. Welche Datensätze fehlen, steht als
//	    FEHLER-Zeile im Protokoll.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"bibliothek/internal/littera"
	"bibliothek/internal/uebernahme"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	exitOK             = 0
	exitAbgebrochen    = 1
	exitUnvollstaendig = 2
)

const protokollPfad = "littera_import.log"

type schalter struct {
	csvVerzeichnis string
	dbURL          string
	trocken        bool
	bestand        bool
	personen       bool
	ausleihen      bool
	barcodes       string
	batch          int
	schuljahrEnde  int
	lehrerAktiv    bool
	erzwingen      bool
}

func main() { os.Exit(run()) }

// run hält die gesamte Arbeit, damit die defer-Kette wirklich läuft: Das Protokoll wird
// über einen 64-KB-Puffer geschrieben, ein os.Exit mitten im Ablauf ließe ihn ungeleert —
// und zwar genau in dem Lauf, für den man es braucht.
func run() int {
	s := lies()

	if s.csvVerzeichnis == "" {
		log.Print("FEHLER: -csv fehlt (Verzeichnis mit den mdb-export-CSVs)")
		return exitAbgebrochen
	}

	log.Printf("Lese Littera-Export aus %s …", s.csvVerzeichnis)
	ab, err := littera.LeseAltbestand(s.csvVerzeichnis)
	if err != nil {
		log.Printf("FEHLER: Export nicht lesbar: %v", err)
		return exitAbgebrochen
	}
	log.Printf("Gelesen: %d Titel, %d Exemplare, %d Leser, %d Ausleihen",
		len(ab.Titel), len(ab.Exemplare), len(ab.Leser), len(ab.Ausleihen))

	if s.trocken {
		trockenlauf(ab)
		return exitOK
	}
	return uebertrage(s, ab)
}

func lies() schalter {
	var s schalter
	flag.StringVar(&s.csvVerzeichnis, "csv", "", "Verzeichnis mit den mdb-export-CSVs")
	flag.StringVar(&s.dbURL, "db", os.Getenv("DATABASE_URL"), "PostgreSQL-Verbindung")
	flag.BoolVar(&s.trocken, "trocken", false, "nur lesen und berichten, nicht schreiben")
	flag.BoolVar(&s.bestand, "bestand", true, "Titel und Exemplare übernehmen")
	flag.BoolVar(&s.personen, "personen", false, "Schüler und Lehrkräfte übernehmen")
	flag.BoolVar(&s.ausleihen, "ausleihen", false, "Ausleihen übernehmen (setzt -personen voraus)")
	flag.StringVar(&s.barcodes, "barcodes", string(littera.BarcodeLittera),
		"littera = Exemplarnummer vom vorhandenen Etikett, neu = frische B-XXXXX aus barcode_seq")
	flag.IntVar(&s.batch, "batch", 200, "Datensätze je Transaktion")
	flag.IntVar(&s.schuljahrEnde, "schuljahr-ende", 0,
		"Jahr, in dem das laufende Schuljahr endet (0 = aus dem heutigen Datum)")
	flag.BoolVar(&s.lehrerAktiv, "lehrer-aktiv", false,
		"Lehrkräfte als aktive Benutzer anlegen (Vorgabe: inaktiv, weil die Anmeldung über den Barcode läuft)")
	flag.BoolVar(&s.erzwingen, "erzwingen", false,
		"trotz vorhandener Littera-Daten in der Zieldatenbank laufen (legt sie ein zweites Mal an)")
	flag.Parse()
	return s
}

// trockenlauf berichtet, was der Export hergibt, ohne die Datenbank anzufassen.
func trockenlauf(ab *littera.Altbestand) {
	log.Print("TROCKENLAUF: es wird nichts geschrieben")
	log.Printf("  Titel mit Signatur:      %d", len(ab.Signaturen))
	log.Printf("  davon uneinheitlich:     %d (häufigster Wert gewinnt)", len(ab.SignaturAbweichend))
	log.Printf("  Verlage / Medienarten:   %d / %d", len(ab.Verlage), len(ab.Medienarten))

	nach := map[littera.LeserArt]int{}
	for _, l := range ab.Leser {
		nach[l.Art]++
	}
	log.Printf("  Leser: %d Schüler, %d Lehrkräfte, %d abgegangen, %d sonstige, %d unklar",
		nach[littera.ArtSchueler], nach[littera.ArtLehrkraft], nach[littera.ArtAbgegangen],
		nach[littera.ArtSonstige], nach[littera.ArtUnbekannt])

	bekannt := make(map[string]bool, len(ab.Exemplare))
	for _, e := range ab.Exemplare {
		bekannt[e.ID] = true
	}
	log.Printf("  Ausleihen: %d gesamt, %d offen, %d ohne Exemplar, %d ohne Frist",
		len(ab.Ausleihen), len(littera.NurOffene(ab.Ausleihen)),
		len(littera.OhneExemplar(ab.Ausleihen, bekannt)), len(littera.OhneFrist(ab.Ausleihen)))
}

func uebertrage(s schalter, ab *littera.Altbestand) int {
	if s.dbURL == "" {
		log.Print("FEHLER: -db fehlt und DATABASE_URL ist nicht gesetzt")
		return exitAbgebrochen
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	prot, err := uebernahme.NeuesProtokoll(protokollPfad, "littera_id")
	if err != nil {
		log.Printf("FEHLER: Protokoll konnte nicht geöffnet werden: %v", err)
		return exitAbgebrochen
	}
	defer prot.Schliessen()

	pool, err := pgxpool.New(ctx, s.dbURL)
	if err != nil {
		log.Printf("FEHLER: Datenbankverbindung: %v", err)
		return exitAbgebrochen
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Printf("FEHLER: Datenbank nicht erreichbar: %v", err)
		return exitAbgebrochen
	}

	schreiber := littera.NeuerSchreiber(pool, prot, optionen(s))
	if !s.erzwingen {
		if err := schreiber.PruefeZielbestand(ctx); err != nil {
			log.Printf("FEHLER: %v", err)
			log.Print("       (-erzwingen läuft trotzdem, legt die Daten aber doppelt an)")
			return exitAbgebrochen
		}
	}

	bericht := fuehreAus(ctx, schreiber, s, ab)
	bericht.Warnungen, bericht.Fehler = prot.Warnungen(), prot.FehlerAnzahl()
	drucke(bericht, s)
	return rueckgabewert(bericht)
}

func optionen(s schalter) littera.Optionen {
	opt := littera.StandardOptionen(time.Now())
	opt.Barcodes = littera.Barcodequelle(s.barcodes)
	opt.BatchGroesse = s.batch
	opt.LehrerAktiv = s.lehrerAktiv
	if s.schuljahrEnde > 0 {
		opt.SchuljahrEnde = s.schuljahrEnde
	}
	return opt
}

// fuehreAus arbeitet die eingeschalteten Teile der Reihe nach ab. Die Reihenfolge ist
// zwingend: Ausleihen brauchen die UUIDs aus Bestand UND Personen.
func fuehreAus(
	ctx context.Context, schreiber *littera.Schreiber, s schalter, ab *littera.Altbestand,
) littera.Bericht {
	var b littera.Bericht
	b.Bestand.AbgleichOK, b.Personen.AbgleichOK, b.Ausleihen.AbgleichOK = true, true, true

	var err error
	if s.bestand {
		log.Printf("Übernehme Bestand (%d Titel) …", len(ab.Titel))
		if b.Bestand, err = schreiber.SchreibeBestand(ctx, ab); err != nil {
			b.Abbruch = fmt.Errorf("beim Bestand: %w", err)
			return b
		}
		log.Printf("  → %d Titel, %d Exemplare, %d übersprungen",
			b.Bestand.Titel, b.Bestand.Exemplare, b.Bestand.Uebersprungen)
	}

	if s.personen {
		log.Printf("Übernehme Personen (%d Leser) …", len(ab.Leser))
		if b.Personen, err = schreiber.SchreibePersonen(ctx, ab); err != nil {
			b.Abbruch = fmt.Errorf("bei den Personen: %w", err)
			return b
		}
		log.Printf("  → %d Schüler, %d Lehrkräfte, %d nicht übernommen",
			b.Personen.Schueler, b.Personen.Lehrkraefte, b.Personen.Uebersprungen)
	}

	if s.ausleihen {
		log.Printf("Übernehme Ausleihen (%d) …", len(ab.Ausleihen))
		if b.Ausleihen, err = schreiber.SchreibeAusleihen(ctx, ab, b.Bestand, b.Personen); err != nil {
			b.Abbruch = fmt.Errorf("bei den Ausleihen: %w", err)
			return b
		}
		log.Printf("  → %d übernommen", b.Ausleihen.Geschrieben)
	}
	return b
}

func rueckgabewert(b littera.Bericht) int {
	switch {
	case b.Abbruch != nil:
		return exitAbgebrochen
	case b.Vollstaendig():
		return exitOK
	default:
		return exitUnvollstaendig
	}
}

func drucke(b littera.Bericht, s schalter) {
	log.Print("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if s.bestand {
		log.Printf("Bestand    Quelle %6d Titel / %6d Exemplare", b.Bestand.QuellTitel, b.Bestand.QuellExemplare)
		log.Printf("           geschrieben %6d / %6d, übersprungen %d",
			b.Bestand.Titel, b.Bestand.Exemplare, b.Bestand.Uebersprungen)
		abgleich(b.Bestand.AbgleichOK, fmt.Sprintf("%d Titel / %d Exemplare tatsächlich neu",
			b.Bestand.IstTitel, b.Bestand.IstExemplare))
	}
	if s.personen {
		log.Printf("Personen   Quelle %6d Leser", b.Personen.QuellLeser)
		log.Printf("           geschrieben %6d Schüler / %4d Lehrkräfte, nicht übernommen %d",
			b.Personen.Schueler, b.Personen.Lehrkraefte, b.Personen.Uebersprungen)
		abgleich(b.Personen.AbgleichOK, fmt.Sprintf("%d Schüler / %d Lehrkräfte tatsächlich neu",
			b.Personen.IstSchueler, b.Personen.IstLehrkraefte))
	}
	if s.ausleihen {
		a := b.Ausleihen
		log.Printf("Ausleihen  Quelle %6d", a.QuellAusleihen)
		log.Printf("           geschrieben %6d; ohne Exemplar %d, ohne Entleiher %d, "+
			"Doppelbelegung %d, Fehler %d",
			a.Geschrieben, a.OhneExemplar, a.OhneEntleiher, a.Doppelbelegung, a.Uebersprungen)
		if a.FristVorAusgabe > 0 {
			log.Printf("           %d Ausleihen tragen eine Frist VOR dem Verleihdatum", a.FristVorAusgabe)
		}
		abgleich(a.AbgleichOK, fmt.Sprintf("%d Ausleihen tatsächlich neu", a.IstAusleihen))
	}
	log.Print("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("  Warnungen (abgewertet übernommen): %d → %s", b.Warnungen, protokollPfad)
	log.Printf("  Fehler (NICHT übernommen):         %d → %s", b.Fehler, protokollPfad)

	if b.Abbruch != nil {
		log.Printf("⚠  Die Übernahme endete vorzeitig: %v", b.Abbruch)
		log.Print("⚠  Alles ab diesem Punkt wurde NICHT geschrieben. Ursache beheben und erneut starten.")
		return
	}
	if b.Fehler > 0 {
		log.Printf("⚠  %d Datensätze fehlen in der neuen Datenbank: grep FEHLER %s", b.Fehler, protokollPfad)
	}
	if b.Warnungen > 0 {
		log.Printf("ℹ  %d Datensätze wurden abgewertet übernommen: grep WARNUNG %s", b.Warnungen, protokollPfad)
	}
}

// abgleich meldet die Gegenprobe an der Datenbank. Eine Abweichung heißt: die Zahlen
// darüber sind falsch — und zwar zugunsten des Werkzeugs. Das muss lauter stehen als sie.
func abgleich(ok bool, ist string) {
	if ok {
		log.Printf("           ✓ Abgleich: %s", ist)
		return
	}
	log.Printf("           ⚠ ABGLEICH FEHLGESCHLAGEN: %s", ist)
	log.Print("           ⚠ Den Zahlen ist nicht zu trauen. Bestand prüfen, bevor Littera abgeschaltet wird.")
}

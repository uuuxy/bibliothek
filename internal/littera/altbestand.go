package littera

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Altbestand ist ein vollständig gelesener Littera-Export.
//
// Die Nachschlagetabellen sind hier bereits aufgelöst gehalten (Schlüssel → Name), damit
// der Schreibpfad keine Littera-Interna mehr kennen muss.
type Altbestand struct {
	Titel       []Titel
	Exemplare   []Exemplar
	Leser       []Leser
	Ausleihen   []Ausleihe
	Verlage     map[string]string // Verlag.Buchungsnummer → Name
	Medienarten map[string]string // Medienart.Buchungsnummer → Bezeichnung
	Signaturen  map[string]string // Titel.Buchungsnummer → Signatur (aus den Exemplaren)
	// SignaturAbweichend nennt die Titel, deren Exemplare untereinander verschiedene
	// Signaturen tragen. Sie werden übernommen (häufigster Wert gewinnt), stehen aber
	// hier, damit der Bericht sie nennen kann.
	SignaturAbweichend []string
	// Standortvermerke bildet Titel.Buchungsnummer → Vermerke am Titel ab (standort.go).
	// Sie gelten für jedes Exemplar des Titels ohne eigenen Sonderstandort.
	Standortvermerke map[string][]string

	// Ausweisnummern bildet Leser.Buchungsnummer → Barcode des Schülerausweises ab,
	// Fremdbarcodes Exemplar.Buchungsnummer → Barcode am Buch. Beide stammen aus
	// Littera-Tabellen für FREMDE Nummern (siehe fremdnummern.go) und sind leer, wenn die
	// Schule keine herstellerbedruckten Ausweise oder Ersatzetiketten benutzt.
	Ausweisnummern map[string]string
	Fremdbarcodes  map[string]string
	// AusweisMehrfach nennt Leser mit mehr als einer hinterlegten Karte.
	AusweisMehrfach []string

	// Schlagworte der Titel (schlagworte.go, docs/OFFEN.md 4.20).
	Schlagworte SchlagwortQuelle
	// Interessenkreise der Titel, geschrieben als Schlagworte (schlagworte.go); die
	// Verweis-Zähler bleiben hier leer.
	Interessenkreise SchlagwortQuelle
}

// Dateien sind die mdb-export-Ausgaben, die LeseAltbestandMit erwartet.
//
//	mdb-export littera_sav.mdb Titel              > titel.csv
//	mdb-export littera_sav.mdb Exemplar           > exemplar.csv
//	mdb-export littera_sav.mdb Verlag             > verlag.csv
//	mdb-export littera_sav.mdb Medienart          > medienart.csv
//	mdb-export littera_sav.mdb Personen           > personen.csv
//	mdb-export littera_sav.mdb Personen_Zuordnung > personen_zuordnung.csv
//	mdb-export littera_sav.mdb Leser              > leser.csv
//	mdb-export littera_sav.mdb Leser_UG           > leser_ug.csv
//	mdb-export littera_sav.mdb Verleih            > verleih.csv
//	mdb-export littera_sav.mdb Schlagworte            > schlagworte.csv
//	mdb-export littera_sav.mdb Schlag_zuord           > schlag_zuord.csv
//	mdb-export littera_sav.mdb Verweise_Schlagworte   > verweise_schlagworte.csv
//	mdb-export littera_sav.mdb Verweis_Zu_Schlagworte > verweis_zu_schlagworte.csv
//	mdb-export littera_sav.mdb Interessenskreis       > interessenskreis.csv
//	mdb-export littera_sav.mdb IntZuMed               > intzumed.csv
const (
	dateiTitel             = "titel.csv"
	dateiExemplar          = "exemplar.csv"
	dateiVerlag            = "verlag.csv"
	dateiMedienart         = "medienart.csv"
	dateiPersonen          = "personen.csv"
	dateiPersonenZuordnung = "personen_zuordnung.csv"
	dateiLeser             = "leser.csv"
	dateiLeserUG           = "leser_ug.csv"
	dateiVerleih           = "verleih.csv"
	// Die Schlagworte sind Pflicht wie die übrigen (seit dem 30.09.2026, docs/OFFEN.md 4.20):
	// Fehlte eine Datei, kämen die Titel still ohne ihre Schlagworte an. Littera führt alle vier
	// Tabellen schon im Stand von 2010, die beiden Verweis-Tabellen dort leer.
	dateiSchlagworte          = "schlagworte.csv"
	dateiSchlagZuord          = "schlag_zuord.csv"
	dateiVerweiseSchlagworte  = "verweise_schlagworte.csv"
	dateiVerweisZuSchlagworte = "verweis_zu_schlagworte.csv"
	// Die Interessenkreise ebenso (seit dem 30.09.2026, docs/OFFEN.md 4.20); die Sicherung von
	// 2010 führt beide Tabellen.
	dateiInteressenkreise = "interessenskreis.csv"
	dateiIntZuMed         = "intzumed.csv"
	// Die beiden Fremdnummern-Tabellen sind OPTIONAL: In einer Installation ohne
	// herstellerbedruckte Ausweise sind sie leer, und mdb-export erzeugt die Dateien
	// dann gar nicht erst. Fehlen sie, läuft der Import mit den Littera-eigenen Nummern.
	dateiFremdLeserNummer = "fremdlesernummer.csv"
	dateiFremdBarcode     = "fremdbarcode.csv"
)

// LeseAltbestandMit liest alle Tabellen aus einem Verzeichnis mit mdb-export-CSVs und nimmt
// aus, was nach der Regel kein Standort ist.
//
// Alle Dateien sind Pflicht. Eine fehlende Nachschlagetabelle würde nicht auffallen,
// sondern still „11" statt „Klett" in den Katalog schreiben — der Lauf soll dann
// abbrechen, nicht mit halbem Ergebnis weitermachen.
func LeseAltbestandMit(verzeichnis string, regel Standortregel) (*Altbestand, error) {
	ab := &Altbestand{}
	var err error

	if ab.Titel, err = mitDatei(verzeichnis, dateiTitel, leseTitel); err != nil {
		return nil, err
	}
	if ab.Exemplare, err = mitDatei(verzeichnis, dateiExemplar, leseExemplare); err != nil {
		return nil, err
	}
	regel.wendeStandortregelAn(ab.Exemplare)
	if ab.Verlage, err = mitDatei(verzeichnis, dateiVerlag, leseVerlage); err != nil {
		return nil, err
	}
	if ab.Medienarten, err = mitDatei(verzeichnis, dateiMedienart, leseMedienarten); err != nil {
		return nil, err
	}
	if err = leseAutoren(verzeichnis, ab, regel); err != nil {
		return nil, err
	}
	if err = leseSchlagworteUndInteressenkreise(verzeichnis, ab); err != nil {
		return nil, err
	}
	if err = leseLeserUndAusleihen(verzeichnis, ab); err != nil {
		return nil, err
	}

	if err = leseFremdnummern(verzeichnis, ab); err != nil {
		return nil, err
	}

	ab.Signaturen, ab.SignaturAbweichend = signaturJeTitel(ab.Exemplare)
	return ab, nil
}

// leseFremdnummern liest die beiden optionalen Tabellen mit herstellereigenen Barcodes.
//
// Eine fehlende Datei ist hier ausdrücklich kein Fehler — anders als bei den
// Nachschlagetabellen, deren Fehlen still „11" statt „Klett" in den Katalog schriebe.
// Ist eine Datei aber DA und unlesbar, bricht der Lauf ab: Dann stimmt etwas mit dem
// Export nicht, und stillschweigend auf die Littera-Nummern zurückzufallen hieße, die
// Etiketten zu ignorieren, die tatsächlich an Karte und Buch kleben.
func leseFremdnummern(verzeichnis string, ab *Altbestand) error {
	ab.Ausweisnummern = map[string]string{}
	ab.Fremdbarcodes = map[string]string{}

	if istVorhanden(verzeichnis, dateiFremdLeserNummer) {
		erg, err := mitDatei(verzeichnis, dateiFremdLeserNummer, leseAusweisPaare)
		if err != nil {
			return err
		}
		ab.Ausweisnummern, ab.AusweisMehrfach = erg.nummern, erg.mehrfach
	}
	if istVorhanden(verzeichnis, dateiFremdBarcode) {
		barcodes, err := mitDatei(verzeichnis, dateiFremdBarcode, leseFremdbarcodes)
		if err != nil {
			return err
		}
		ab.Fremdbarcodes = barcodes
	}
	return nil
}

// ausweispaare bündelt die zwei Rückgabewerte von leseAusweisnummern, damit mitDatei
// (das genau einen Wert liefert) sie durchreichen kann.
type ausweispaare struct {
	nummern  map[string]string
	mehrfach []string
}

func leseAusweisPaare(r io.Reader) (ausweispaare, error) {
	nummern, mehrfach, err := leseAusweisnummern(r)
	return ausweispaare{nummern: nummern, mehrfach: mehrfach}, err
}

func istVorhanden(verzeichnis, name string) bool {
	_, err := os.Stat(filepath.Join(verzeichnis, name))
	return err == nil
}

func leseAutoren(verzeichnis string, ab *Altbestand, regel Standortregel) error {
	personen, err := mitDatei(verzeichnis, dateiPersonen, lesePersonen)
	if err != nil {
		return err
	}
	z, err := mitDatei(verzeichnis, dateiPersonenZuordnung,
		func(r io.Reader) (personenZuordnungen, error) { return lesePersonenZuordnungen(personen, r, regel) })
	if err != nil {
		return err
	}
	for i := range ab.Titel {
		ab.Titel[i].Autor = ohneVermerke(ab.Titel[i].Autor, z.VermerkNamen)
	}
	ab.Titel = mitAutoren(ab.Titel, z.Autoren)
	ab.Standortvermerke = z.Standortvermerke
	return nil
}

func leseSchlagworteUndInteressenkreise(verzeichnis string, ab *Altbestand) error {
	woerter, err := mitDatei(verzeichnis, dateiSchlagworte, leseSchlagworte)
	if err != nil {
		return err
	}
	if ab.Schlagworte, err = mitDatei(verzeichnis, dateiSchlagZuord,
		func(r io.Reader) (SchlagwortQuelle, error) { return schlagworteJeTitel(woerter, r) }); err != nil {
		return err
	}
	if ab.Schlagworte.VerweisWoerter, err = mitDatei(verzeichnis, dateiVerweiseSchlagworte, zaehleZeilen); err != nil {
		return err
	}
	if ab.Schlagworte.VerweisZuordnungen, err = mitDatei(verzeichnis, dateiVerweisZuSchlagworte, zaehleZeilen); err != nil {
		return err
	}
	kreise, err := mitDatei(verzeichnis, dateiInteressenkreise, leseInteressenkreise)
	if err != nil {
		return err
	}
	ab.Interessenkreise, err = mitDatei(verzeichnis, dateiIntZuMed,
		func(r io.Reader) (SchlagwortQuelle, error) { return interessenkreiseJeTitel(kreise, r) })
	return err
}

func leseLeserUndAusleihen(verzeichnis string, ab *Altbestand) error {
	gruppen, err := mitDatei(verzeichnis, dateiLeserUG, leseLesergruppen)
	if err != nil {
		return err
	}
	if ab.Leser, err = mitDatei(verzeichnis, dateiLeser,
		func(r io.Reader) ([]Leser, error) { return leseLeser(r, gruppen) }); err != nil {
		return err
	}
	ab.Ausleihen, err = mitDatei(verzeichnis, dateiVerleih, leseAusleihen)
	return err
}

// mitDatei öffnet eine Exportdatei, reicht sie an den Leser weiter und schließt sie.
func mitDatei[T any](verzeichnis, name string, lies func(io.Reader) (T, error)) (T, error) {
	var leer T
	pfad := filepath.Join(verzeichnis, name)
	f, err := os.Open(pfad) // #nosec G304 - Pfad kommt aus dem Aufruf des Werkzeugs
	if err != nil {
		return leer, fmt.Errorf("%s: %w", name, err)
	}
	defer func() { _ = f.Close() }() //nolint:errcheck

	wert, err := lies(f)
	if err != nil {
		return leer, fmt.Errorf("%s: %w", pfad, err)
	}
	return wert, nil
}

// exemplareJeTitel gruppiert die Exemplare nach ihrem Titel.
//
// Der Schreibpfad braucht diese Sicht, weil der TITEL die atomare Einheit ist: Er trägt
// Exemplar-Zeilen, und ein Titel mit 5 Stück, dem nur drei Exemplare folgen, wäre ein
// stiller Bestandsfehler.
func exemplareJeTitel(exemplare []Exemplar) map[string][]Exemplar {
	jeTitel := make(map[string][]Exemplar)
	for _, e := range exemplare {
		jeTitel[e.TitelID] = append(jeTitel[e.TitelID], e)
	}
	return jeTitel
}

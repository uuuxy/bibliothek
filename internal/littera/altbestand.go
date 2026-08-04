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
}

// Dateien sind die mdb-export-Ausgaben, die LeseAltbestand erwartet.
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
const (
	DateiTitel             = "titel.csv"
	DateiExemplar          = "exemplar.csv"
	DateiVerlag            = "verlag.csv"
	DateiMedienart         = "medienart.csv"
	DateiPersonen          = "personen.csv"
	DateiPersonenZuordnung = "personen_zuordnung.csv"
	DateiLeser             = "leser.csv"
	DateiLeserUG           = "leser_ug.csv"
	DateiVerleih           = "verleih.csv"
)

// LeseAltbestand liest alle Tabellen aus einem Verzeichnis mit mdb-export-CSVs.
//
// Alle Dateien sind Pflicht. Eine fehlende Nachschlagetabelle würde nicht auffallen,
// sondern still „11" statt „Klett" in den Katalog schreiben — der Lauf soll dann
// abbrechen, nicht mit halbem Ergebnis weitermachen.
func LeseAltbestand(verzeichnis string) (*Altbestand, error) {
	ab := &Altbestand{}
	var err error

	if ab.Titel, err = mitDatei(verzeichnis, DateiTitel, LeseTitel); err != nil {
		return nil, err
	}
	if ab.Exemplare, err = mitDatei(verzeichnis, DateiExemplar, LeseExemplare); err != nil {
		return nil, err
	}
	if ab.Verlage, err = mitDatei(verzeichnis, DateiVerlag, VerlagNamen); err != nil {
		return nil, err
	}
	if ab.Medienarten, err = mitDatei(verzeichnis, DateiMedienart, MedienartNamen); err != nil {
		return nil, err
	}
	if err = leseAutoren(verzeichnis, ab); err != nil {
		return nil, err
	}
	if err = leseLeserUndAusleihen(verzeichnis, ab); err != nil {
		return nil, err
	}

	ab.Signaturen, ab.SignaturAbweichend = SignaturJeTitel(ab.Exemplare)
	return ab, nil
}

func leseAutoren(verzeichnis string, ab *Altbestand) error {
	personen, err := mitDatei(verzeichnis, DateiPersonen, LesePersonen)
	if err != nil {
		return err
	}
	autoren, err := mitDatei(verzeichnis, DateiPersonenZuordnung,
		func(r io.Reader) (map[string]string, error) { return AutorenJeTitel(personen, r) })
	if err != nil {
		return err
	}
	ab.Titel = MitAutoren(ab.Titel, autoren)
	return nil
}

func leseLeserUndAusleihen(verzeichnis string, ab *Altbestand) error {
	gruppen, err := mitDatei(verzeichnis, DateiLeserUG, LeseLesergruppen)
	if err != nil {
		return err
	}
	if ab.Leser, err = mitDatei(verzeichnis, DateiLeser,
		func(r io.Reader) ([]Leser, error) { return LeseLeser(r, gruppen) }); err != nil {
		return err
	}
	ab.Ausleihen, err = mitDatei(verzeichnis, DateiVerleih, LeseAusleihen)
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

// ExemplareJeTitel gruppiert die Exemplare nach ihrem Titel.
//
// Der Schreibpfad braucht diese Sicht, weil der TITEL die atomare Einheit ist: Er trägt
// buecher_titel.stock, und ein Titel mit stock=5, dem nur drei Exemplare folgen, wäre ein
// stiller Bestandsfehler.
func ExemplareJeTitel(exemplare []Exemplar) map[string][]Exemplar {
	jeTitel := make(map[string][]Exemplar)
	for _, e := range exemplare {
		jeTitel[e.TitelID] = append(jeTitel[e.TitelID], e)
	}
	return jeTitel
}

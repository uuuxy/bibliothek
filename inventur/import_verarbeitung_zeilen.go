package inventur

import (
	"context"
	"math"
	"strconv"
	"strings"

	"bibliothek/internal/katalogdienste"
	"bibliothek/pkg/lmf"
	"bibliothek/repository"
)

// importZeile bündelt die Parameter für die Verarbeitung einer Importzeile.
type importZeile struct {
	Zeile     []string
	Spalten   map[string]int
	Abfragen  *repository.TitelAbfragen
	Metadaten *katalogdienste.MetadatenClient
}

// verarbeiteImportZeile verarbeitet eine einzelne Excel-Zeile und erstellt das
// entsprechende Book-Objekt für den Batch-Import.
func verarbeiteImportZeile(ctx context.Context, konfig importZeile) (*repository.KatalogTitel, error) {
	spalteLesen := func(name string) string {
		stelle := konfig.Spalten[name]
		if stelle >= 0 && stelle < len(konfig.Zeile) {
			return strings.TrimSpace(konfig.Zeile[stelle])
		}
		return ""
	}

	isbn := spalteLesen("isbn")
	if isbn == "" {
		return nil, nil // Leere Zeile überspringen
	}

	titelText := spalteLesen("titel")
	autor := spalteLesen("autor")
	// Die Spalte des Fachs ist Freitext, buecher_titel.subject aber ein Fremdschlüssel auf die
	// Systematik: Was hier durchgeht, wird eine Kategorie und steht danach in der Auswahl der Fächer
	// in der Maske und als Fach im Portal. Deshalb dieselbe Prüfung wie im Import des Bestands aus
	// CSV (fachDerZeile): nur ein echter Fachname oder eine bekannte Schreibvariante, sonst leer.
	//
	// Ein Rückfall auf das wörtliche Fach „Unbekannt" legte eine Kategorie dieses Namens an, die kein
	// Fach ist. Leer heißt überall sonst „ohne Fach", und genau das ist hier gemeint.
	fach := lmf.FachExakt(spalteLesen("fach"))

	// Die Spalte „klasse" nennt einen Jahrgang. Er steht als Spanne N bis N am Titel: Mit ihr
	// rechnen Inventur und Portal, und die Titelliste zeigt sie. 0 heißt unbekannt.
	klasse := leseKlassenstufe(spalteLesen("klasse"))

	eintrag := repository.KatalogTitel{
		ISBN:            isbn,
		Titel:           titelText,
		Autor:           autor,
		Fach:            fach,
		JahrgangVon:     klasse,
		JahrgangBis:     klasse,
		Bestand:         leseBestand(spalteLesen("bestand")),
		ZuletztGezaehlt: nil,
	}

	ergaenzeMetadaten(ctx, konfig.Metadaten, &eintrag)

	if eintrag.Fach == "" || strings.EqualFold(eintrag.Fach, "unbekannt") {
		if erratenesFach := errateFachAusTitel(eintrag.Titel); erratenesFach != "" {
			eintrag.Fach = erratenesFach
		}
	}

	return &eintrag, nil
}

// leseKlassenstufe liest den Jahrgang aus der Spalte „klasse" (lmf.IstJahrgang). Fehlt die
// Spalte, ist sie leer oder liegt der Wert daneben, ist der Jahrgang unbekannt (0), und beide
// Upserts schreiben NULL in die Spanne. Geraten wird nicht: Ein geratener Jahrgang ist in der
// Datenbank von einem gepflegten nicht zu unterscheiden, und das Upsert behält eine vorhandene
// Spanne.
func leseKlassenstufe(stufeText string) int {
	stufe, err := strconv.Atoi(stufeText)
	if err != nil || !lmf.IstJahrgang(stufe) {
		return 0
	}
	return stufe
}

// leseBestand liest den Bestand aus der Import-Spalte. Nur Werte aus [0, MaxInt32]
// werden übernommen: negative Bestände sind Datenfehler, und die DB-Spalte ist int4 —
// der Bulk-Upsert konvertiert nach int32. Ungültiges wird wie ein Parse-Fehler
// behandelt (Bestand 0).
func leseBestand(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > math.MaxInt32 {
		return 0
	}
	return n
}

// ergaenzeMetadaten ergänzt fehlende Buch-Metadaten über externe APIs.
func ergaenzeMetadaten(ctx context.Context, metadaten *katalogdienste.MetadatenClient, eintrag *repository.KatalogTitel) {
	if eintrag.Titel != "" && eintrag.Autor != "" && eintrag.CoverURL != "" {
		return
	}

	nachschlagen, _ := metadaten.SucheNachISBN(ctx, eintrag.ISBN) //nolint:errcheck
	if nachschlagen == nil {
		return
	}

	if eintrag.Titel == "" || eintrag.Titel == "Unbekannter Titel" {
		eintrag.Titel = nachschlagen.Titel
	}
	if eintrag.Autor == "" || eintrag.Autor == "Unbekannter Autor" {
		eintrag.Autor = nachschlagen.Autor
	}
	if eintrag.CoverURL == "" {
		eintrag.CoverURL = nachschlagen.CoverURL
	}

	if eintrag.Titel == "" {
		eintrag.Titel = "Unbekannter Titel"
	}
	if eintrag.Autor == "" {
		eintrag.Autor = "Unbekannter Autor"
	}
}

// Package lusd gleicht den Bestand der Schüler mit einem Export der LUSD ab: Es liest die
// Datei, ordnet jede Zeile dem Bestand zu, bildet die Vorschau und wendet die Änderungen in
// einer Transaktion an. Die Tür steht in api/lusd.go, die Anweisungen an die Datenbank stehen
// in repository/lusd_import.go.
package lusd

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"
)

// StudentDiff ist ein Schüler-Eintrag der Vorschau. ID ist der Listenschlüssel fürs
// Frontend — die LUSD-ID (ID-Modus) oder die schueler-UUID, bei Neuzugängen im
// Namensmodus die Zeilennummer. Nie eine interne Kennung, die es nicht ohnehin gibt.
type StudentDiff struct {
	ID         string `json:"id"`
	Vorname    string `json:"vorname"`
	Nachname   string `json:"nachname"`
	AlteKlasse string `json:"alte_klasse,omitempty"`
	NeueKlasse string `json:"neue_klasse,omitempty"`
}

// AdoptionDiff beschreibt eine geplante Adoption: Eine Zeile der Datei, deren LUSD-ID im
// Bestand fehlt, trifft über Name und Geburtsdatum auf einen vorhandenen Schüler ohne
// LUSD-ID (von Hand angelegt oder aus Littera übernommen). Statt ihn ein zweites Mal
// anzulegen, wird die LUSD-ID nachgetragen. SchuelerID ist der vorhandene Datensatz, LusdID
// die Kennung, die er bekommt.
type AdoptionDiff struct {
	SchuelerID   string `json:"schueler_id"`
	LusdID       string `json:"lusd_id"`
	Vorname      string `json:"vorname"`
	Nachname     string `json:"nachname"`
	Geburtsdatum string `json:"geburtsdatum"`
	AlteKlasse   string `json:"alte_klasse,omitempty"`
	NeueKlasse   string `json:"neue_klasse,omitempty"`
}

// PreviewResult ist die Vorschau, anhand derer das Sekretariat Namen und
// Klassenwechsel prüft, bevor es bestätigt. ActiveDbStudents ist die Bezugsgröße der
// Abgänger-Quote (abgleichbare aktive Schüler), nicht die Zahl der Zeilen in der Datei.
type PreviewResult struct {
	Modus            string         `json:"modus"` // "lusd_id" oder "name_geburtsdatum" (Modus.String)
	NewStudents      []StudentDiff  `json:"new_students"`
	ClassChanges     []StudentDiff  `json:"class_changes"`
	Adoptions        []AdoptionDiff `json:"adoptions"`         // ID-Modus: ID-lose Bestandsschüler bekommen ihre LUSD-ID
	Rueckkehrer      []StudentDiff  `json:"rueckkehrer"`       // Abgänger, die wieder im Export stehen — werden reaktiviert
	Graduates        []StudentDiff  `json:"graduates"`         // Abgänger (im Bestand, fehlen in der Datei)
	NichtImExport    []StudentDiff  `json:"nicht_im_export"`   // Namensmodus: nie bestätigte Handanlagen — bleiben unverändert
	NichtAbgleichbar []StudentDiff  `json:"nicht_abgleichbar"` // Namensmodus: ohne Geburtsdatum — bleiben unverändert
	Mehrdeutig       []StudentDiff  `json:"mehrdeutig"`        // gleicher Schlüssel mehrfach — wird nicht angefasst
	// Umbenennungen: Abgänger + Neuzugang, die nach Geburtsdatum/Schuleintritt/Klasse/
	// Anschrift dieselbe Person sind (paarung.go). Der Admin bestätigt je Paar.
	Umbenennungen []UmbenennungDiff `json:"umbenennungen"`
	// KarenzTage: so lange bleiben Abgänger ohne offene Vorgänge nur gesperrt, bevor
	// sie anonymisiert werden (Einstellung abgaenger_karenz_tage; 0 = sofort).
	KarenzTage       int `json:"karenz_tage"`
	TotalCsvRecords  int `json:"total_csv_records"`
	ActiveDbStudents int `json:"active_db_students"`
	SkippedNoID      int `json:"skipped_no_id"`      // ID-Modus: CSV-Zeilen ohne LUSD-ID — werden nie importiert
	DublettenInDatei int `json:"dubletten_in_datei"` // Zeilen mit demselben Schlüssel, letzte gewann
	// DublettenAbweichend: die davon, deren Zeilen verschiedene Klassen trugen — die
	// Vorschau nennt sie beim Namen, damit ein stiller Zusammenfall zweier Schüler
	// (gleicher Name, Tippfehler im Geburtsdatum) vor dem Import auffällt.
	DublettenAbweichend []StudentDiff `json:"dubletten_abweichend"`
}

// massGraduationThresholdPct: Ab diesem Anteil an Abgängern, bezogen auf die abgleichbaren
// aktiven Schüler des Bestands, verlangt der Import eine ausdrückliche Bestätigung. Die
// Behandlung der Abgänger ist nicht umkehrbar; die Schwelle fängt einen Teilexport ab, etwa
// eine Datei mit nur einem Jahrgang.
const massGraduationThresholdPct = 30

// minStudentsForThreshold hält die Schwelle von sehr kleinen Beständen fern
// (Ersteinrichtung, Testsysteme).
const minStudentsForThreshold = 10

// MassenabgangFehler meldet, dass der Lauf die Schwelle überschritte und die Bestätigung
// fehlt. Er trägt die Zahlen für die Antwort der Tür.
type MassenabgangFehler struct {
	Graduates int
	Active    int
}

func (e *MassenabgangFehler) Error() string {
	return fmt.Sprintf(
		"%d von %d aktiven Schülern würden zu Abgängern (Schwelle: %d%%). Datei prüfen — falls der Massenabgang beabsichtigt ist (Schuljahreswechsel), Import mit Bestätigung wiederholen.",
		e.Graduates, e.Active, massGraduationThresholdPct)
}

// Lauf sind die Vorgaben eines Laufs: Vorschau oder Anwenden, ob der Massenabgang bestätigt
// ist, welche Umbenennungs-Paare gewählt sind und wie viele Tage ein Abgänger ohne offene
// Vorgänge nur gesperrt bleibt, bevor er anonymisiert wird (0 = sofort).
type Lauf struct {
	Anwenden, MassenabgangBestaetigt bool
	Umbenennungen                    []UmbenennungWahl
	KarenzTage                       int
}

// Fuehre vergleicht die Datei mit dem Bestand in einer Transaktion und liefert die Vorschau
// oder wendet die Änderungen an, samt der gewählten Umbenennungen. Die Karenzzeit liest der
// Aufrufer vorher aus den Einstellungen: Sie gehört nicht in den Stand, den die Transaktion
// sieht.
func Fuehre(ctx context.Context, pool db.PgxPoolIface, datei Datei, lauf Lauf) (*PreviewResult, error) {
	apply := lauf.Anwenden
	// Eine Transaktion für den ganzen Lauf; bei einem frühen Rückweg wird zurückgerollt.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	// Beim Anwenden reiht die Sperre gleichzeitige Läufe ein (repository.SperreLusdImport).
	// Die Vorschau liest nur und nimmt sie nicht.
	if apply {
		if err := repository.SperreLusdImport(ctx, tx); err != nil {
			return nil, fmt.Errorf("lusd-import sperren fehlgeschlagen: %w", err)
		}
	}

	bestand, err := ladeLusdBestand(ctx, tx)
	if err != nil {
		return nil, err
	}
	idx := baueLusdIndex(bestand, datei.Modus)

	res := &PreviewResult{
		Modus:            datei.Modus.String(),
		NewStudents:      []StudentDiff{},
		ClassChanges:     []StudentDiff{},
		Adoptions:        []AdoptionDiff{},
		Rueckkehrer:      []StudentDiff{},
		Graduates:        []StudentDiff{},
		NichtImExport:    []StudentDiff{},
		NichtAbgleichbar: []StudentDiff{},
		Mehrdeutig:       []StudentDiff{},
		Umbenennungen:    []UmbenennungDiff{},
		TotalCsvRecords:  len(datei.Zeilen),
		ActiveDbStudents: len(idx.aktiv),
		DublettenInDatei: datei.DublettenInDatei,
		KarenzTage:       lauf.KarenzTage,
	}
	res.DublettenAbweichend = append([]StudentDiff{}, datei.Zusammengelegt...)

	// Erst nur zuordnen: Die Schwelle des Massenabgangs muss entscheiden, bevor die erste
	// Anweisung etwas ändert.
	zuordnung := klassifiziereLusd(datei, idx, res)
	res.Adoptions = append(res.Adoptions, zuordnung.adoptionen...)
	// Umbenennungen aus Abgängern und Neuzugängen paaren; beim Anwenden die Wahl des
	// Admins einarbeiten (bestätigte Paare verlassen beide Listen).
	res.Umbenennungen = append(res.Umbenennungen, findeUmbenennungen(datei, bestand, idx, zuordnung)...)
	if apply {
		if err := uebernimmUmbenennungen(datei, lauf.Umbenennungen, res.Umbenennungen, &zuordnung, res); err != nil {
			return nil, err
		}
	}

	if !apply {
		return res, nil
	}

	// Die Schwelle des Massenabgangs gilt hier, am Server: Die Behandlung der Abgänger sperrt
	// und anonymisiert bei Karenz 0 sofort; das ist nicht umkehrbar.
	if !lauf.MassenabgangBestaetigt &&
		res.ActiveDbStudents >= minStudentsForThreshold &&
		len(res.Graduates)*100 >= res.ActiveDbStudents*massGraduationThresholdPct {
		return nil, &MassenabgangFehler{Graduates: len(res.Graduates), Active: res.ActiveDbStudents}
	}

	// Adoptionen zuerst: Sie heften die LUSD-ID an den vorhandenen Schüler ohne ID. Danach
	// findet WendeAenderungenAn ihn über repository.FindeAktivenSchuelerNachLusdID und
	// übernimmt Klasse und Kontaktdaten wie bei jedem Schüler des Bestands, statt ihn ein
	// zweites Mal anzulegen.
	if err := adoptiereWaisen(ctx, tx, zuordnung.adoptionen); err != nil {
		return nil, err
	}
	if err := WendeAenderungenAn(ctx, tx, datei, zuordnung); err != nil {
		return nil, err
	}
	if err := behandleAbgaenger(ctx, tx, zuordnung.abgaengerIDs, res.KarenzTage); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return res, nil
}

// Package lusd gleicht den Bestand der Schüler mit einem Export der LUSD ab: Es liest die
// Datei, ordnet jede Zeile dem Bestand zu, bildet die Vorschau und wendet die Änderungen in
// einer Transaktion an. Die Tür steht in api/leser/lusd.go, die Anweisungen an die Datenbank stehen
// in repository/lusd_import.go.
package lusd

import (
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/repository"
)

// VorschauSchueler ist ein Schüler-Eintrag der Vorschau. ID ist der Listenschlüssel der
// Oberfläche — die LUSD-ID (ID-Modus) oder die schueler-UUID, bei Neuzugängen im
// Namensmodus die Zeilennummer. Nie eine interne Kennung, die es nicht ohnehin gibt.
type VorschauSchueler struct {
	ID         string `json:"id"`
	Vorname    string `json:"vorname"`
	Nachname   string `json:"nachname"`
	AlteKlasse string `json:"alte_klasse,omitempty"`
	NeueKlasse string `json:"neue_klasse,omitempty"`
}

// VorschauAdoption beschreibt eine geplante Adoption: Eine Zeile der Datei, deren LUSD-ID im
// Bestand fehlt, trifft über Name und Geburtsdatum auf einen vorhandenen Schüler ohne
// LUSD-ID (von Hand angelegt oder aus Littera übernommen). Statt ihn ein zweites Mal
// anzulegen, wird die LUSD-ID nachgetragen. SchuelerID ist der vorhandene Datensatz, LusdID
// die Kennung, die er bekommt.
type VorschauAdoption struct {
	SchuelerID   string `json:"schueler_id"`
	LUSDID       string `json:"lusd_id"`
	Vorname      string `json:"vorname"`
	Nachname     string `json:"nachname"`
	Geburtsdatum string `json:"geburtsdatum"`
	AlteKlasse   string `json:"alte_klasse,omitempty"`
	NeueKlasse   string `json:"neue_klasse,omitempty"`
}

// Vorschau ist die Vorschau, anhand derer das Sekretariat Namen und
// Klassenwechsel prüft, bevor es bestätigt. AktiveImBestand ist die Bezugsgröße der
// Abgänger-Quote (abgleichbare aktive Schüler), nicht die Zahl der Zeilen in der Datei.
type Vorschau struct {
	Modus            string             `json:"modus"` // "lusd_id" oder "name_geburtsdatum" (Modus.String)
	Neuzugaenge      []VorschauSchueler `json:"new_students"`
	Klassenwechsel   []VorschauSchueler `json:"class_changes"`
	Adoptionen       []VorschauAdoption `json:"adoptions"`         // ID-Modus: ID-lose Bestandsschüler bekommen ihre LUSD-ID
	Rueckkehrer      []VorschauSchueler `json:"rueckkehrer"`       // Abgänger, die wieder im Export stehen — werden reaktiviert
	Abgaenger        []VorschauSchueler `json:"graduates"`         // Abgänger (im Bestand, fehlen in der Datei)
	NichtImExport    []VorschauSchueler `json:"nicht_im_export"`   // Namensmodus: nie bestätigte Handanlagen — bleiben unverändert
	NichtAbgleichbar []VorschauSchueler `json:"nicht_abgleichbar"` // Namensmodus: ohne Geburtsdatum — bleiben unverändert
	Mehrdeutig       []VorschauSchueler `json:"mehrdeutig"`        // gleicher Schlüssel mehrfach — wird nicht angefasst
	// Umbenennungen: Abgänger + Neuzugang, die nach Geburtsdatum/Schuleintritt/Klasse/
	// Anschrift dieselbe Person sind (paarung.go). Der Admin bestätigt je Paar.
	Umbenennungen []VorschauUmbenennung `json:"umbenennungen"`
	// KarenzTage: so lange bleiben Abgänger ohne offene Vorgänge nur gesperrt, bevor
	// sie anonymisiert werden (Einstellung abgaenger_karenz_tage; 0 = sofort).
	KarenzTage       int `json:"karenz_tage"`
	ZeilenInDatei    int `json:"total_csv_records"`
	AktiveImBestand  int `json:"active_db_students"`
	OhneID           int `json:"skipped_no_id"`      // ID-Modus: CSV-Zeilen ohne LUSD-ID — werden nie importiert
	DublettenInDatei int `json:"dubletten_in_datei"` // Zeilen mit demselben Schlüssel, letzte gewann
	// DublettenAbweichend: die davon, deren Zeilen verschiedene Klassen trugen — die
	// Vorschau nennt sie beim Namen, damit ein stiller Zusammenfall zweier Schüler
	// (gleicher Name, Tippfehler im Geburtsdatum) vor dem Import auffällt.
	DublettenAbweichend []VorschauSchueler `json:"dubletten_abweichend"`
}

// massenabgangSchwelleProzent: Ab diesem Anteil an Abgängern, bezogen auf die abgleichbaren
// aktiven Schüler des Bestands, verlangt der Import eine ausdrückliche Bestätigung. Die
// Behandlung der Abgänger ist nicht umkehrbar; die Schwelle fängt einen Teilexport ab, etwa
// eine Datei mit nur einem Jahrgang.
const massenabgangSchwelleProzent = 30

// massenabgangMindestbestand hält die Schwelle von sehr kleinen Beständen fern
// (Ersteinrichtung, Testsysteme).
const massenabgangMindestbestand = 10

// MassenabgangFehler meldet, dass der Lauf die Schwelle überschritte und die Bestätigung
// fehlt. Er trägt die Zahlen für die Antwort der Tür.
type MassenabgangFehler struct {
	Abgaenger int
	Aktive    int
}

func (e *MassenabgangFehler) Error() string {
	return fmt.Sprintf(
		"%d von %d aktiven Schülern würden zu Abgängern (Schwelle: %d%%). Datei prüfen — falls der Massenabgang beabsichtigt ist (Schuljahreswechsel), Import mit Bestätigung wiederholen.",
		e.Abgaenger, e.Aktive, massenabgangSchwelleProzent)
}

// Lauf sind die Vorgaben eines Laufs: Vorschau oder Anwenden, ob der Massenabgang bestätigt
// ist, welche Umbenennungs-Paare gewählt sind und wie viele Tage ein Abgänger ohne offene
// Vorgänge nur gesperrt bleibt, bevor er anonymisiert wird (0 = sofort).
type Lauf struct {
	Anwenden, MassenabgangBestaetigt bool
	Umbenennungen                    []UmbenennungWahl
	KarenzTage                       int
}

// FuehreAus vergleicht die Datei mit dem Bestand in einer Transaktion und liefert die Vorschau
// oder wendet die Änderungen an, samt der gewählten Umbenennungen. Die Karenzzeit liest der
// Rufer vorher aus den Einstellungen, weil sie nicht zu dem Stand gehört, den die Transaktion
// sieht.
func FuehreAus(ctx context.Context, pool db.PgxPoolIface, datei Datei, lauf Lauf) (*Vorschau, error) {
	anwenden := lauf.Anwenden
	// Eine Transaktion für den ganzen Lauf; bei einem frühen Rückweg wird zurückgerollt.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer db.SafeRollback(ctx, tx)

	// Beim Anwenden reiht die Sperre gleichzeitige Läufe ein (repository.SperreLUSDImport).
	// Die Vorschau liest nur und nimmt sie nicht.
	if anwenden {
		if err := repository.SperreLUSDImport(ctx, tx); err != nil {
			return nil, fmt.Errorf("lusd-import sperren fehlgeschlagen: %w", err)
		}
	}

	bestand, err := ladeBestand(ctx, tx)
	if err != nil {
		return nil, err
	}
	index := baueIndex(bestand, datei.Modus)

	vorschau := &Vorschau{
		Modus:            datei.Modus.String(),
		Neuzugaenge:      []VorschauSchueler{},
		Klassenwechsel:   []VorschauSchueler{},
		Adoptionen:       []VorschauAdoption{},
		Rueckkehrer:      []VorschauSchueler{},
		Abgaenger:        []VorschauSchueler{},
		NichtImExport:    []VorschauSchueler{},
		NichtAbgleichbar: []VorschauSchueler{},
		Mehrdeutig:       []VorschauSchueler{},
		Umbenennungen:    []VorschauUmbenennung{},
		ZeilenInDatei:    len(datei.Zeilen),
		AktiveImBestand:  len(index.aktiv),
		DublettenInDatei: datei.DublettenInDatei,
		KarenzTage:       lauf.KarenzTage,
	}
	vorschau.DublettenAbweichend = append([]VorschauSchueler{}, datei.Zusammengelegt...)

	// Erst nur zuordnen: Die Schwelle des Massenabgangs muss entscheiden, bevor die erste
	// Anweisung etwas ändert.
	zuordnung := klassifiziere(datei, index, vorschau)
	vorschau.Adoptionen = append(vorschau.Adoptionen, zuordnung.adoptionen...)
	// Umbenennungen aus Abgängern und Neuzugängen paaren; beim Anwenden die Wahl des
	// Admins einarbeiten (bestätigte Paare verlassen beide Listen).
	vorschau.Umbenennungen = append(vorschau.Umbenennungen, findeUmbenennungen(datei, bestand, index, zuordnung)...)
	if anwenden {
		if err := uebernimmUmbenennungen(datei, lauf.Umbenennungen, vorschau.Umbenennungen, &zuordnung, vorschau); err != nil {
			return nil, err
		}
	}

	if !anwenden {
		return vorschau, nil
	}

	// Die Schwelle des Massenabgangs gilt hier, am Server: Die Behandlung der Abgänger sperrt
	// und anonymisiert bei Karenz 0 sofort; das ist nicht umkehrbar.
	if !lauf.MassenabgangBestaetigt &&
		vorschau.AktiveImBestand >= massenabgangMindestbestand &&
		len(vorschau.Abgaenger)*100 >= vorschau.AktiveImBestand*massenabgangSchwelleProzent {
		return nil, &MassenabgangFehler{Abgaenger: len(vorschau.Abgaenger), Aktive: vorschau.AktiveImBestand}
	}

	// Adoptionen zuerst: Sie heften die LUSD-ID an den vorhandenen Schüler ohne ID. Danach
	// findet WendeAenderungenAn ihn über repository.FindeAktivenSchuelerZuLUSDID und
	// übernimmt Klasse und Kontaktdaten wie bei jedem Schüler des Bestands, statt ihn ein
	// zweites Mal anzulegen.
	if err := adoptiereWaisen(ctx, tx, zuordnung.adoptionen); err != nil {
		return nil, err
	}
	if err := WendeAenderungenAn(ctx, tx, datei, zuordnung); err != nil {
		return nil, err
	}
	if err := behandleAbgaenger(ctx, tx, zuordnung.abgaengerIDs, vorschau.KarenzTage); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return vorschau, nil
}

package repository

import (
	"time"

	"bibliothek/db"
	"bibliothek/internal/ausweis"
)

// Leser ist jeder, der ausleiht: Schüler, Lehrkraft oder LiV (Tabelle leser), unterscheidbar an
// Art. Listen, Mahnungen und Klassenansichten meinen nur Schüler und lesen die Sicht schueler;
// die Theke fragt nach einem Leser (FindeZuBarcode, Finde) und bekommt auch Kollegen.
type Leser struct {
	// ID ist die UUID des Lesers.
	ID string `json:"id"`
	// Art ist Schüler, Lehrkraft oder LiV (chk_leser_art). Sie entscheidet keine Rechte, nur,
	// wen der LUSD-Abgleich und der Löschjob erfassen und was die Theke am Treffer anzeigt.
	// Leer, wo die Abfrage die Sicht schueler liest: Dort ist jede Zeile ein Schüler.
	Art string `json:"art,omitempty"`
	// BarcodeID ist die Ausweisnummer.
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	// Klasse ist die Klasse oder Kursstufe (etwa "07B"); leer bei Lehrkraft und LiV.
	Klasse string `json:"klasse"`
	// AbgaengerJahr ist das Jahr, in dem der Schüler die Schule voraussichtlich verlässt. Es
	// steuert die Löschung nach der DSGVO und die Archivierung im Stapel, nicht die Gültigkeit
	// des Ausweises: Ein Gymnasialschüler bleibt bis Jahrgang 13, sein Ausweis gilt aber nur
	// bis zum Ende der Mittelstufe.
	AbgaengerJahr int `json:"abgaenger_jahr"`
	// AusweisGueltigBis ist das Jahr, bis zu dem der Schülerausweis gilt (31.07.), aus dem
	// Bildungsgang der Klasse gerechnet (internal/ausweis). Kein Feld der Datenbank, sondern
	// beim Lesen gesetzt. nil heißt: aus dieser Klassenbezeichnung nicht ableitbar; dann fragt
	// der Druckdialog nach, statt ein Datum zu erfinden.
	AusweisGueltigBis *int `json:"ausweis_gueltig_bis,omitempty"`
	// IstGesperrt ist die Sperre, die das Programm setzt: Ehemalige (Versetzung, LUSD-Import),
	// Papierkorb, Anonymisierung. Die Theke lässt dann nur die Rückgabe zu, außer beim
	// Lernmittel (service.pruefeAusleihSperren). Offene Forderungen und überfällige Medien
	// setzen sie nicht; die zählt die Theke bei jeder Ausleihe selbst.
	IstGesperrt bool `json:"ist_gesperrt"`
	// LUSDID ist die Schülernummer aus der LUSD; an ihr erkennt der Abgleich den Schüler
	// wieder.
	LUSDID *string `json:"lusd_id,omitempty"`
	// IstAbgaenger kennzeichnet Schüler, die die Schule verlassen haben.
	IstAbgaenger bool `json:"ist_abgaenger"`
	// Geburtsdatum ist der Tag als Text (DATE in Postgres); nil ohne Eintrag.
	Geburtsdatum   *string   `json:"geburtsdatum,omitempty"`
	ErstelltAm     time.Time `json:"erstellt_am"`
	AktualisiertAm time.Time `json:"aktualisiert_am"`
	// IstVonHandGesperrt ist die Sperre von Hand (PATCH /api/admin/students/{id}/lock). Sie gilt
	// auch beim Lernmittel; aufgehoben wird sie in der Akte.
	IstVonHandGesperrt bool `json:"is_manually_blocked"`
	// Sperrgrund ist der Grund der Sperre, von Hand oder vom Programm: eine Spalte für beide.
	Sperrgrund *string `json:"block_reason,omitempty"`
	// Strasse, Hausnummer, Plz und Ort sind die Postanschrift; alle vier können leer sein.
	Strasse    string `json:"strasse"`
	Hausnummer string `json:"hausnummer"`
	Plz        string `json:"plz"`
	Ort        string `json:"ort"`
	// ElternEmail ist die Adresse der Erziehungsberechtigten.
	ElternEmail string `json:"eltern_email"`
	// IstAnonymisiert sagt, dass anonymized_at gesetzt ist: Die Zeile ist keine Person mehr.
	// Die Theke gibt ihr nichts aus, auch kein Lernmittel (service.pruefeAusleihSperren); die
	// Namenssuche findet sie noch („Abgänger Anonymisiert-…"). Gefüllt nur über die
	// Spaltenliste der Leser-Abfragen (spaltenLeser), nicht in der API.
	IstAnonymisiert bool `json:"-"`
}

// scanner ist eine Zeile, die sich lesen lässt: pgx.Row und pgx.Rows erfüllen die Schnittstelle
// beide, und die Helfer, die eine Zeile lesen, nehmen damit beides.
type scanner interface {
	Scan(dest ...any) error
}

// LeserAbfragen liest die Leser: einzeln für Theke und Akte, als Suche und als Liste der
// Leserdatei.
type LeserAbfragen struct {
	db db.Pool
}

// NewLeserAbfragen liefert die Abfragen der Leser an Postgres.
func NewLeserAbfragen(db db.Pool) *LeserAbfragen {
	return &LeserAbfragen{db: db}
}

// leseLeserMitZusatz liest die feste Spaltenliste eines Lesers (spaltenLeser) und danach
// beliebige Zusatzspalten, etwa count(*) OVER () für die Gesamtzahl der Treffer. So steht die
// Reihenfolge der Felder an einer Stelle; eine zweite Kopie liefe beim nächsten neuen Feld
// still auseinander.
func leseLeserMitZusatz(row scanner, zusatz ...any) (*Leser, error) {
	var s Leser
	ziele := []any{
		&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.LUSDID, &s.IstAbgaenger, &s.Geburtsdatum, &s.ErstelltAm, &s.AktualisiertAm, &s.IstVonHandGesperrt, &s.Sperrgrund,
		&s.Strasse, &s.Hausnummer, &s.Plz, &s.Ort, &s.ElternEmail, &s.IstAnonymisiert,
	}
	if err := row.Scan(append(ziele, zusatz...)...); err != nil {
		return nil, err
	}
	s.AusweisGueltigBis = ausweisGueltigBis(s.Klasse)
	return &s, nil
}

// ausweisGueltigBis rechnet die Ausweisgültigkeit aus der Klasse. Kein Datenbankfeld:
// Die Regel darf sich ändern (Schulform, Zweigwechsel), ohne dass Altbestände
// nachgezogen werden müssen — und ein gespeicherter Wert wäre nach jedem
// Schuljahreswechsel still veraltet.
//
// Rückgabe nil bei einer Klassenbezeichnung, aus der sich nichts ableiten lässt. Der
// Druckdialog fragt dann nach, statt ein Datum zu erfinden.
func ausweisGueltigBis(klasse string) *int {
	jahr, ok := ausweis.GueltigBisJahr(klasse, ausweis.SchuljahrEnde(time.Now()))
	if !ok {
		return nil
	}
	return &jahr
}

// spaltenLeser ist die Spaltenliste von leseLeserMitZusatz — einmal, damit die Abfragen über
// die Sicht `schueler` und die über die Tabelle `leser` nicht auseinanderlaufen können.
const spaltenLeser = `id, coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, ''), coalesce(abgaenger_jahr, 0), coalesce(ist_gesperrt, false), lusd_id, coalesce(ist_abgaenger, false), TO_CHAR(geburtsdatum, 'YYYY-MM-DD'), erstellt_am, aktualisiert_am, coalesce(is_manually_blocked, false), block_reason, coalesce(strasse, ''), coalesce(hausnummer, ''), coalesce(plz, ''), coalesce(ort, ''), coalesce(eltern_email, ''), anonymized_at IS NOT NULL`

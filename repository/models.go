package repository

import (
	"time"
)

// User represents a system administrator, teacher, or library staff member.
type User struct {
	ID        string `json:"id"`
	BarcodeID string `json:"barcode_id"`
	Vorname   string `json:"vorname"`
	Nachname  string `json:"nachname"`
	Rolle     string `json:"rolle"`
}

// Student represents the schueler table model in PostgreSQL.
type Student struct {
	ID             string    `json:"id"`
	BarcodeID      string    `json:"barcode_id"`
	Vorname        string    `json:"vorname"`
	Nachname       string    `json:"nachname"`
	Klasse         string    `json:"klasse"`
	AbgaengerJahr  int       `json:"abgaenger_jahr"`
	IstGesperrt    bool      `json:"ist_gesperrt"`
	ErstelltAm     time.Time `json:"erstellt_am"`
	AktualisiertAm time.Time `json:"aktualisiert_am"`
}

// BookTitle represents the buecher_titel table metadata.
type BookTitle struct {
	ID                      string         `json:"id"`
	Titel                   string         `json:"titel"`
	Untertitel              string         `json:"untertitel,omitempty"`
	Autor                   string         `json:"autor,omitempty"`
	ISBN                    string         `json:"isbn,omitempty"`
	Verlag                  string         `json:"verlag,omitempty"`
	Erscheinungsjahr        int            `json:"erscheinungsjahr,omitempty"`
	Beschreibung            string         `json:"beschreibung,omitempty"`
	CoverURL                string         `json:"cover_url,omitempty"`
	Medientyp               string         `json:"medientyp,omitempty"`
	ErstelltAm              time.Time      `json:"erstellt_am"`
	AktualisiertAm          time.Time      `json:"aktualisiert_am"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften,omitempty"`
}

// BookCopy represents the buecher_exemplare physical item model, combined with title details.
type BookCopy struct {
	ID              string    `json:"id"`
	TitelID         string    `json:"titel_id"`
	BarcodeID       string    `json:"barcode_id"`
	ZustandNotiz    string    `json:"zustand_notiz"`
	ErworbenAm      time.Time `json:"erworben_am"`
	IstAusleihbar   bool      `json:"ist_ausleihbar"`
	IstAusgesondert bool      `json:"ist_ausgesondert"`
	ErstelltAm      time.Time `json:"erstellt_am"`
	AktualisiertAm  time.Time `json:"aktualisiert_am"`

	// Joined fields from associated BookTitle
	Titel                   string         `json:"titel"`
	Autor                   string         `json:"autor"`
	Verlag                  string         `json:"verlag"`
	ISBN                    string         `json:"isbn"`
	CoverURL                string         `json:"cover_url,omitempty"`
	Medientyp               string         `json:"medientyp,omitempty"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften,omitempty"`
}

// Loan represents the ausleihen table model for historical and active transactions.
type Loan struct {
	ID                    string     `json:"id"`
	ExemplarID            string     `json:"exemplar_id"`
	SchuelerID            *string    `json:"schueler_id,omitempty"`
	AusleiherBenutzerID   *string    `json:"ausleiher_benutzer_id,omitempty"`
	AusgeliehenAm         time.Time  `json:"ausgeliehen_am"`
	RueckgabeFrist        time.Time  `json:"rueckgabe_frist"`
	RueckgabeAm           *time.Time `json:"rueckgabe_am,omitempty"`
	BearbeiterID          string     `json:"bearbeiter_id"`
	RueckgabeBearbeiterID *string    `json:"rueckgabe_bearbeiter_id,omitempty"`
	IstFremdrueckgabe     bool       `json:"ist_fremdrueckgabe"`
	IstHandapparat        bool       `json:"ist_handapparat"`
}

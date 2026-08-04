package inventur

// Book bildet die Tabelle buecher_titel im Code ab.
type Book struct {
	ID                      string         `json:"id" db:"id"`
	ISBN                    string         `json:"isbn" db:"isbn"`
	Title                   string         `json:"title" db:"title"`
	Author                  string         `json:"author" db:"author"`
	// Signatur steht physisch auf dem Buchrücken-Etikett (Littera-Systematik,
	// z. B. "Bio 5" oder "Row") — Importe dürfen befüllte Werte NIE leeren.
	Signatur                string         `json:"signatur" db:"signatur"`
	CoverURL                string         `json:"coverUrl" db:"cover_url"`
	Subject                 string         `json:"subject" db:"subject"`
	GradeLevel              int16          `json:"gradeLevel" db:"grade_level"`
	Track                   string         `json:"track" db:"track"`
	Stock                   int            `json:"stock" db:"stock"`
	Verfuegbar              int            `json:"verfuegbar"`
	Gesamt                  int            `json:"gesamt"`
	LastCounted             *string        `json:"lastCounted" db:"last_counted"`
	SortOrder               int            `json:"sortOrder" db:"sort_order"`
	Medientyp               string         `json:"medientyp" db:"medientyp"`
	JahrgangVon             int            `json:"jahrgangVon" db:"jahrgang_von"`
	JahrgangBis             int            `json:"jahrgangBis" db:"jahrgang_bis"`
	Untertitel              string         `json:"untertitel" db:"untertitel"`
	Verlag                  string         `json:"verlag" db:"verlag"`
	Erscheinungsjahr        int            `json:"erscheinungsjahr" db:"erscheinungsjahr"`
	Beschreibung            string         `json:"beschreibung" db:"beschreibung"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften" db:"erweiterte_eigenschaften"`
}

// ClassBookAssignment represents a book assigned to a class.

type ClassBookAssignment struct {
	ClassName string `json:"className" db:"class_name"`
	BookID    string `json:"bookId" db:"book_id"`
	Title     string `json:"title" db:"title"`
	Subject   string `json:"subject" db:"subject"`
	Track     string `json:"track" db:"track"`
	CoverURL  string `json:"coverUrl" db:"cover_url"`
}

// BuchEingabe repräsentiert die erwartete JSON-Struktur für das Erstellen oder Aktualisieren eines Buches.
type BuchEingabe struct {
	ISBN                    string         `json:"isbn"`
	Fach                    string         `json:"subject"`
	KlassenStufe            int16          `json:"gradeLevel"`
	Schulzweig              string         `json:"track"`
	Bestand                 int            `json:"stock"`
	Titel                   string         `json:"title"`
	Autor                   string         `json:"author"`
	CoverURL                string         `json:"coverUrl"`
	ZaehlDatum              *string        `json:"lastCounted"`
	Medientyp               string         `json:"medientyp"`
	JahrgangVon             int            `json:"jahrgangVon"`
	JahrgangBis             int            `json:"jahrgangBis"`
	Untertitel              string         `json:"untertitel"`
	Verlag                  string         `json:"verlag"`
	Erscheinungsjahr        int            `json:"erscheinungsjahr"`
	Beschreibung            string         `json:"beschreibung"`
	Signatur                string         `json:"signatur"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften"`
}

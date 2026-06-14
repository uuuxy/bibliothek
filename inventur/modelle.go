package inventur

// Book bildet die Tabelle buecher_titel im Code ab.
type Book struct {
	ID                      string         `json:"id" db:"id"`
	ISBN                    string         `json:"isbn" db:"isbn"`
	Title                   string         `json:"title" db:"title"`
	Author                  string         `json:"author" db:"author"`
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

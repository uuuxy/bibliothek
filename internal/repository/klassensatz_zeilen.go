package repository

// KlassensatzTitel ist ein Klassensatz-Titel, wie ihn das Lehrerportal anzeigt.
type KlassensatzTitel struct {
	ID         string `json:"id"`
	Titel      string `json:"title"`
	Fach       string `json:"subject"`
	Zweig      string `json:"track"`
	CoverURL   string `json:"coverUrl"`
	ISBN       string `json:"isbn"`
	Verfuegbar int    `json:"verfuegbar"`
	Gesamt     int    `json:"gesamt"`
	ImZulauf   int    `json:"imZulauf"`
	// Quelle: "hand" = in class_books gepflegt; "ausleihe" = live aus den laufenden
	// Ausleihen abgeleitet (mehr als die Hälfte der Klasse hält den Titel). Leser = wie
	// viele Kinder der Klasse ihn gerade haben (nur bei "ausleihe" gefüllt).
	Quelle string `json:"quelle"`
	Leser  int    `json:"leser"`
	// Auflagen: Haben Kinder der Klasse eine andere Auflage desselben Buchs als die der
	// Kachel, stehen hier alle Auflagen, die die Klasse hat, mit der Zahl der Kinder — die
	// meisten zuerst. Sonst fehlt das Feld.
	Auflagen []AuflageInKlasse `json:"auflagen,omitempty"`
}

// KlassensatzGruppe bündelt die Klassensatz-Titel einer Klasse — die Gruppierung passiert
// serverseitig, damit die Oberfläche nicht über alle Zuordnungen iterieren muss.
type KlassensatzGruppe struct {
	Klasse string             `json:"className"`
	Titel  []KlassensatzTitel `json:"books"`
}

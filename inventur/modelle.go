package inventur

// Book bildet die Tabelle buecher_titel im Code ab.
type Book struct {
	ID     string `json:"id" db:"id"`
	ISBN   string `json:"isbn" db:"isbn"`
	Title  string `json:"title" db:"title"`
	Author string `json:"author" db:"author"`
	// Signatur steht physisch auf dem Buchrücken-Etikett (Littera-Systematik,
	// z. B. "Bio 5" oder "Row") — Importe dürfen befüllte Werte NIE leeren.
	Signatur string `json:"signatur" db:"signatur"`
	CoverURL string `json:"coverUrl" db:"cover_url"`
	Subject  string `json:"subject" db:"subject"`
	Track    string `json:"track" db:"track"`
	// IstLernmittel: Schulbuch der Lernmittelfreiheit (Migration 093). Vorher stand das
	// im Text („LMF" vor Titel oder Signatur); heute schaltet die Maske es, Importe
	// lesen es aus Litteras Kennung.
	IstLernmittel bool    `json:"istLernmittel" db:"ist_lernmittel"`
	Stock         int     `json:"stock" db:"stock"`
	Verfuegbar    int     `json:"verfuegbar"`
	Gesamt        int     `json:"gesamt"`
	LastCounted   *string `json:"lastCounted" db:"last_counted"`
	SortOrder     int     `json:"sortOrder" db:"sort_order"`
	Medientyp     string  `json:"medientyp" db:"medientyp"`
	JahrgangVon   int     `json:"jahrgangVon" db:"jahrgang_von"`
	JahrgangBis   int     `json:"jahrgangBis" db:"jahrgang_bis"`
	// Mehrjahresband (Migration 134, Antwort der Schule vom 22.09.2026, docs/OFFEN.md 9.6):
	// Das Buch bleibt über die Spanne JahrgangVon..JahrgangBis beim Kind; die Frist rechnet
	// bis zum Stichtag des Schuljahres, in dem das Kind JahrgangBis beendet. Nur an einem
	// Lernmittel und nur mit einer Spanne über mehr als einen Jahrgang (pruefeMehrjahresband,
	// CHECK chk_mehrjahresband_spanne). Die Jahreszahl ist JahrgangBis, eine zweite gibt es nicht.
	Mehrjahresband bool   `json:"mehrjahresband" db:"mehrjahresband"`
	Untertitel     string `json:"untertitel" db:"untertitel"`
	// Auflage: die Auflagenbezeichnung („4. Aufl. 2023", Migration 126). Eine neue
	// Auflage ist ein eigener Titel mit eigener ISBN — dieses Feld unterscheidet die
	// beiden Zeilen in Liste, Akte und Ausgabe.
	Auflage string `json:"auflage" db:"auflage"`
	// Listenpreis: was ein Ersatz HEUTE kostet (Migration 127) — der „Neupreis zum
	// Zeitpunkt des Verlusts" der Arbeitshilfe, in der Sprache des Medienzentrums der
	// Listenpreis. Ab dem zweiten Verleihjahr rechnet die Staffel darauf.
	//
	// ZEIGER, nicht float64: „nicht erfasst" (nil) und „kostet nichts" (0) sind zwei
	// verschiedene Aussagen, und der Unterschied steht am Ende in einem Bescheid an
	// Erziehungsberechtigte. Bei nil weicht die Staffel auf den Kaufpreis aus und sagt
	// das; bei 0 nennte sie 0,00 €.
	Listenpreis             *float64       `json:"listenpreis" db:"listenpreis"`
	Verlag                  string         `json:"verlag" db:"verlag"`
	Erscheinungsjahr        int            `json:"erscheinungsjahr" db:"erscheinungsjahr"`
	Beschreibung            string         `json:"beschreibung" db:"beschreibung"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften" db:"erweiterte_eigenschaften"`
}

// BuchEingabe repräsentiert die erwartete JSON-Struktur für das Erstellen oder Aktualisieren eines Buches.
type BuchEingabe struct {
	ISBN          string `json:"isbn"`
	Fach          string `json:"subject"`
	Schulzweig    string `json:"track"`
	IstLernmittel bool   `json:"istLernmittel"`
	// Zeiger, nicht int: "nicht mitgeschickt" muss sich von "null" unterscheiden lassen.
	// Beim Aktualisieren gleicht syncBookStock die physischen Exemplare an diese Zahl an
	// — eine fehlende 0 sonderte bis zum 23.08.2026 den GESAMTEN Bestand aus, im
	// Rückfallzweig auch ausgeliehene Exemplare. `Number(undefined)` im Formular wird zu
	// NaN und in JSON zu null; genau das ist der Weg dorthin.
	Bestand        *int    `json:"stock"`
	Titel          string  `json:"title"`
	Autor          string  `json:"author"`
	CoverURL       string  `json:"coverUrl"`
	ZaehlDatum     *string `json:"lastCounted"`
	Medientyp      string  `json:"medientyp"`
	JahrgangVon    int     `json:"jahrgangVon"`
	JahrgangBis    int     `json:"jahrgangBis"`
	Mehrjahresband bool    `json:"mehrjahresband"`
	Untertitel     string  `json:"untertitel"`
	// Auflage: Auflagenbezeichnung des Titels (Migration 126). Ohne dieses Feld käme der
	// Wert aus der Maske nie am Repository an — die Tür wäre gebaut und nicht verdrahtet.
	Auflage string `json:"auflage"`
	// Listenpreis aus der Maske (Migration 127). Zeiger aus demselben Grund wie oben:
	// Ein leeres Feld ist „nicht erfasst", eine getippte 0 ist eine Aussage.
	Listenpreis             *float64       `json:"listenpreis"`
	Verlag                  string         `json:"verlag"`
	Erscheinungsjahr        int            `json:"erscheinungsjahr"`
	Beschreibung            string         `json:"beschreibung"`
	Signatur                string         `json:"signatur"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften"`
}

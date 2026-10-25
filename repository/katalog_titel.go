package repository

import "bibliothek/pkg/betrag"

// KatalogTitel ist ein Titel, wie ihn der Medienkatalog liest und schreibt: mit den Zahlen seiner
// Exemplare, den Feldern der Buchmaske und den Namen, unter denen die Titelliste ihn ausliefert.
// Titel ist dieselbe Zeile für Theke, Bestellwesen und Übernahme, mit anderen Namen in JSON.
type KatalogTitel struct {
	ID    string `json:"id" db:"id"`
	ISBN  string `json:"isbn" db:"isbn"`
	Titel string `json:"title" db:"title"`
	Autor string `json:"author" db:"author"`
	// Signatur steht physisch auf dem Buchrücken-Etikett (Littera-Systematik,
	// z. B. "Bio 5" oder "Row") — Importe dürfen befüllte Werte nie leeren.
	Signatur string `json:"signatur" db:"signatur"`
	CoverURL string `json:"coverUrl" db:"cover_url"`
	Fach     string `json:"subject" db:"subject"`
	Zweig    string `json:"track" db:"track"`
	// IstLernmittel: Schulbuch der Lernmittelfreiheit. Die Maske schaltet es; Importe lesen es aus
	// der Kennung von Littera.
	IstLernmittel   bool    `json:"istLernmittel" db:"ist_lernmittel"`
	Bestand         int     `json:"stock" db:"stock"`
	Verfuegbar      int     `json:"verfuegbar"`
	Gesamt          int     `json:"gesamt"`
	ImZulauf        int     `json:"imZulauf"` // bestellt, noch nicht eingetroffen (SQLBestandImZulauf)
	ZuletztGezaehlt *string `json:"lastCounted" db:"last_counted"`
	Reihenfolge     int     `json:"sortOrder" db:"sort_order"`
	Medientyp       string  `json:"medientyp" db:"medientyp"`
	// JahrgangVon, JahrgangBis: „im Unterricht von Jahrgang … bis". 0 und 0 heißt „unbekannt"
	// und ist eine eigene Angabe; in der Datenbank steht dann NULL.
	JahrgangVon int `json:"jahrgangVon" db:"jahrgang_von"`
	JahrgangBis int `json:"jahrgangBis" db:"jahrgang_bis"`
	// Mehrjahresband: Das Buch bleibt über die Spanne JahrgangVon..JahrgangBis beim Kind; die Frist
	// rechnet bis zum Stichtag des Schuljahres, in dem das Kind JahrgangBis beendet. Nur an einem
	// Lernmittel und nur mit einer Spanne über mehr als einen Jahrgang (pruefeMehrjahresband, CHECK
	// chk_mehrjahresband_spanne). Die Jahreszahl ist JahrgangBis, eine zweite gibt es nicht.
	Mehrjahresband bool   `json:"mehrjahresband" db:"mehrjahresband"`
	Untertitel     string `json:"untertitel" db:"untertitel"`
	// Auflage: die Bezeichnung der Auflage („4. Aufl. 2023"). Eine neue Auflage ist ein eigener Titel
	// mit eigener ISBN; dieses Feld unterscheidet die beiden Zeilen in Liste, Akte und Ausgabe.
	Auflage string `json:"auflage" db:"auflage"`
	// Listenpreis: was ein Ersatz heute kostet — der „Neupreis zum
	// Zeitpunkt des Verlusts" der Arbeitshilfe, in der Sprache des Medienzentrums der
	// Listenpreis. Ab dem zweiten Verleihjahr rechnet die Staffel darauf.
	//
	// Zeiger, nicht float64: „nicht erfasst" (nil) und „kostet nichts" (0) sind zwei
	// verschiedene Aussagen, und der Unterschied steht am Ende in einem Bescheid an
	// Erziehungsberechtigte. Bei nil weicht die Staffel auf den Kaufpreis aus und sagt
	// das; bei 0 nennte sie 0,00 €.
	Listenpreis             *betrag.Cent   `json:"listenpreis" db:"listenpreis"`
	Verlag                  string         `json:"verlag" db:"verlag"`
	Erscheinungsjahr        int            `json:"erscheinungsjahr" db:"erscheinungsjahr"`
	ErweiterteEigenschaften map[string]any `json:"erweiterteEigenschaften" db:"erweiterte_eigenschaften"`
	// Schlagworte stehen in einer eigenen Tabelle. nil heißt „keine
	// Aussage": Die Katalogliste lädt sie nicht, und beim Speichern lässt nil die
	// vorhandenen unangetastet. Eine leere Liste ist eine Aussage — keine Schlagworte.
	Schlagworte []string `json:"schlagworte"`
	// Suchwoerter: die Wörter, über die die Katalogsuche im Browser den Titel findet —
	// seine Schlagworte und die Verweise darauf (ladeSuchwoerterDerTitel). Nur in
	// der Katalogliste. Ein Suchindex, keine Anzeige und kein Schreibweg: Ein Verweis ist
	// kein Schlagwort des Titels, und das Speichern liest das Feld nicht.
	Suchwoerter []string `json:"suchwoerter,omitempty"`
	// WerkID und WerkRang: das Buch, zu dem die
	// Auflage gehört, und ihr Rang darin — 1 ist die neueste nach
	// sqlNeuesteAuflageZuerst, gezählt über alle Auflagen des Buchs. Der
	// Medienkatalog fasst damit die Auflagen eines Buchs zu einer Kachel zusammen. Ohne Werk
	// "" und 0. Nur Ausgabe: Geschrieben wird werk_id allein in repository/auflagen.go.
	WerkID   string `json:"werkId,omitempty"`
	WerkRang int    `json:"werkRang,omitempty"`
	// Standorte: je Standort die Zahl der Exemplare im Bestand, die ihn tragen
	// (ladeStandorteDerTitel). Nur in der Katalogliste, für die Spalte „Standort" der
	// Titel-Verwaltung; geändert wird der Standort am Exemplar.
	Standorte []StandortZahl `json:"standorte,omitempty"`
}

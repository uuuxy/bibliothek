package repository

import (
	"context"

	"bibliothek/pkg/isbnutil"

	"github.com/jackc/pgx/v5"
)

// TitelBestand ist der vorhandene Titelbestand, wie ein Import ihn zum Zuordnen braucht. Katalog-
// Import und Bestands-Import stellen ihm dieselbe Frage (TitelZu). Mit einer Fassung der Frage
// je Import ordnete dieselbe Zeile je nach Weg einem anderen Titel zu, oder sie legte ihn ein
// zweites Mal an.
type TitelBestand struct {
	nachISBN  map[string]string // Normalform der ISBN → Kennung des Titels
	nachTitel map[string]string // NormalisiereTitelKey des Titels → Kennung des Titels
}

// NeuerTitelBestand liefert einen leeren Bestand.
func NeuerTitelBestand() TitelBestand {
	return TitelBestand{nachISBN: map[string]string{}, nachTitel: map[string]string{}}
}

// TitelZu ordnet eine Zeile eines Imports einem Titel des Bestands zu: zuerst über die ISBN in
// ihrer Normalform, sonst über den Titeltext. Der Titeltext zählt auch für eine Zeile mit ISBN,
// weil der Bestand denselben Titel ohne ISBN führen kann. bekannt ist false, wenn der Bestand
// keinen solchen Titel führt.
func (b TitelBestand) TitelZu(isbn, titel string) (id string, bekannt bool) {
	if schluessel := isbnutil.Normalform(isbn); schluessel != "" {
		if id, bekannt = b.nachISBN[schluessel]; bekannt {
			return id, true
		}
	}
	id, bekannt = b.nachTitel[NormalisiereTitelKey(titel)]
	return id, bekannt
}

// Merke trägt einen Titel nach, den der laufende Import angelegt hat: Die folgenden Zeilen der
// Datei finden ihn dann. Ohne ISBN steht er nur unter seinem Titeltext.
func (b TitelBestand) Merke(isbn, titel, id string) {
	if schluessel := isbnutil.Normalform(isbn); schluessel != "" {
		b.nachISBN[schluessel] = id
	}
	b.nachTitel[NormalisiereTitelKey(titel)] = id
}

// LadeTitelBestand lädt den vorhandenen Titelbestand; beide Importe ordnen damit zu. Tragen
// zwei Titel dieselbe ISBN in verschiedener Schreibweise (eine Dublette, die Migration 140
// oder 157 stehen ließ), gilt der, der sie in der Normalform trägt.
func LadeTitelBestand(ctx context.Context, tx pgx.Tx) (TitelBestand, error) {
	rows, err := tx.Query(ctx, "SELECT id, COALESCE(isbn, ''), titel FROM buecher_titel")
	if err != nil {
		return TitelBestand{}, err
	}
	defer rows.Close()

	bestand := NeuerTitelBestand()
	for rows.Next() {
		var id, isbn, titel string
		if err := rows.Scan(&id, &isbn, &titel); err != nil {
			return TitelBestand{}, err
		}
		if schluessel := isbnutil.Normalform(isbn); schluessel != "" {
			if _, belegt := bestand.nachISBN[schluessel]; !belegt || isbn == schluessel {
				bestand.nachISBN[schluessel] = id
			}
		}
		bestand.nachTitel[NormalisiereTitelKey(titel)] = id
	}
	if err := rows.Err(); err != nil {
		return TitelBestand{}, err
	}
	return bestand, nil
}

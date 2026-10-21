package repository

import "context"

// LeserEtikettZeile ist das Minimum, das auf einem Klebeetikett steht.
//
// Bewusst ein eigener, schmaler Typ statt Leser: Der Etikettenbogen braucht vier
// Felder, und ein Aufruf, der das ganze Schülerprofil holt, zieht Adresse, Sperrgrund
// und Elternmail durch eine Schicht, die davon nichts wissen muss (Datenminimierung,
// dieselbe Haltung wie bei den Statistik-Abfragen).
type LeserEtikettZeile struct {
	BarcodeID string
	Vorname   string
	Nachname  string
	Klasse    string
}

// LadeEtikettZeilen liest die Angaben für die Etiketten der genannten Leser: Ausweisnummer, Name
// und Klasse.
//
// Das Kollegium steht vorn, danach ordnen Klasse, Nachname und Vorname, nicht die Reihenfolge der
// übergebenen Kennungen:
// Die Reihenfolge des Markierens in der Leserdatei ist zufällig (mal von oben durchgeklickt,
// mal per Häkchen im Kopf), und ein Bogen Klebeetiketten wird in der Reihenfolge abgezogen, in
// der man die Namen sucht.
//
// Die Klasse steht zuerst: Ohne sie laufen zwei gemeinsam markierte Klassen auf dem Bogen
// ineinander (gemessen: 7A, 7A, 7A, 7B, 7B, 7B, 7B, 7B, 7A, …), und wer die Etiketten
// klassenweise austeilt, klaubt sie einzeln heraus. Es ist außerdem dieselbe Reihenfolge, die
// die Leserdatei selbst führt (Kollegium zuerst, dann die gewohnte Reihenfolge der Kartei,
// ladeListe) und der der Stapeldruck der Ausweise folgt: Zwei Ausgaben derselben Auswahl sollen
// nicht verschieden sortiert aus dem Drucker kommen.
//
// Gelesen wird die Tabelle `leser`, nicht die Sicht `schueler`: Schüler und Kollegium stehen in
// derselben Liste, mit demselben Häkchen davor. Über die Sicht fiele ein markierter Kollege
// lautlos vom Bogen; der Druck gelänge, der Bogen wäre nur kürzer, und gemerkt würde es beim
// Verteilen.
//
// Gelöschte Leser bleiben außen vor, und ihre Kennungen fallen dabei weg. Das ist hier kein
// Fehler: Ein gleichzeitig gelöschter Mensch soll den ganzen Bogen nicht verhindern. Es ist der
// einzige Grund, aus dem eine Zeile fehlen kann.
func (a *LeserAbfragen) LadeEtikettZeilen(ctx context.Context, ids []string) ([]LeserEtikettZeile, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := a.db.Query(ctx, `
		SELECT coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, '')
		FROM leser
		WHERE id = ANY($1) AND deleted_at IS NULL
		ORDER BY (art = 'schueler'), klasse COLLATE deutsch, nachname COLLATE deutsch, vorname COLLATE deutsch
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := make([]LeserEtikettZeile, 0, len(ids))
	for rows.Next() {
		var z LeserEtikettZeile
		if err := rows.Scan(&z.BarcodeID, &z.Vorname, &z.Nachname, &z.Klasse); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
}

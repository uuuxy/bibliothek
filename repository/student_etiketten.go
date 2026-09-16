package repository

import "context"

// SchuelerEtikettZeile ist das Minimum, das auf einem Klebeetikett steht.
//
// Bewusst ein eigener, schmaler Typ statt Student: Der Etikettenbogen braucht vier
// Felder, und ein Aufruf, der das ganze Schülerprofil holt, zieht Adresse, Sperrgrund
// und Elternmail durch eine Schicht, die davon nichts wissen muss (Datenminimierung,
// dieselbe Haltung wie bei den Statistik-Abfragen).
type SchuelerEtikettZeile struct {
	BarcodeID string
	Vorname   string
	Nachname  string
	Klasse    string
}

// EtikettenZeilen liest die Etikettenangaben zu den angegebenen LESER-IDs.
//
// Sortiert nach Klasse, Nachname, Vorname — nicht in der Reihenfolge der übergebenen
// IDs: Die Markierungsreihenfolge in der Schülerdatei ist zufällig (mal von oben
// durchgeklickt, mal per Kopf-Häkchen), und ein Bogen Klebeetiketten wird in der
// Reihenfolge abgezogen, in der man die Namen sucht.
//
// Die Klasse steht ZUERST, und das ist der Punkt: Ohne sie laufen zwei gemeinsam
// markierte Klassen auf dem Bogen ineinander (gemessen: 7A, 7A, 7A, 7B, 7B, 7B, 7B,
// 7B, 7A, …), und wer die Etiketten klassenweise austeilt, klaubt sie einzeln
// heraus. Es ist außerdem dieselbe Reihenfolge, die die Leserdatei selbst führt
// (Kollegium zuerst, dann die gewohnte Kartei-Reihenfolge — listSchuelerMitStats) und
// der der Ausweis-Stapeldruck folgt: Zwei Ausgaben derselben Auswahl sollen nicht
// verschieden sortiert aus dem Drucker kommen.
//
// Gelesen wird die TABELLE `leser`, nicht die Sicht `schueler`. Seit dem 16.09.2026
// stehen Schüler und Kollegium in derselben Liste, mit demselben Häkchen davor; über die
// Sicht fiel ein markierter Kollege LAUTLOS vom Bogen — der Druck gelang, der Bogen war
// nur kürzer, und gemerkt hätte man es beim Verteilen.
//
// Gelöschte Leser bleiben außen vor, und ihre IDs fallen dabei weg. Das ist hier kein
// Fehler: Ein gleichzeitig gelöschter Mensch soll den ganzen Bogen nicht verhindern.
// Seit die Abfrage die Tabelle liest, ist das auch der einzige Grund, aus dem eine Zeile
// fehlen kann.
func (r *pgStudentRepository) EtikettenZeilen(ctx context.Context, ids []string) ([]SchuelerEtikettZeile, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, '')
		FROM leser
		WHERE id = ANY($1) AND deleted_at IS NULL
		ORDER BY (art = 'schueler'), klasse, nachname, vorname
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	zeilen := make([]SchuelerEtikettZeile, 0, len(ids))
	for rows.Next() {
		var z SchuelerEtikettZeile
		if err := rows.Scan(&z.BarcodeID, &z.Vorname, &z.Nachname, &z.Klasse); err != nil {
			return nil, err
		}
		zeilen = append(zeilen, z)
	}
	return zeilen, rows.Err()
}

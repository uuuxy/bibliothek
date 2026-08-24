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

// EtikettenZeilen liest die Etikettenangaben zu den angegebenen Schüler-IDs.
//
// Sortiert nach Nachname, Vorname — nicht in der Reihenfolge der übergebenen IDs: Ein
// Bogen Klebeetiketten wird in der Reihenfolge abgezogen, in der man die Namen sucht.
// Die Markierungsreihenfolge in der Schülerdatei ist dagegen zufällig (mal von oben
// durchgeklickt, mal per Kopf-Häkchen).
//
// Gelöschte Schüler bleiben außen vor. IDs ohne Treffer fallen still weg — der Aufrufer
// vergleicht die Anzahl und sagt es der Theke; hier ist es kein Fehler, weil ein
// gleichzeitig gelöschter Schüler den ganzen Bogen nicht verhindern soll.
func (r *pgStudentRepository) EtikettenZeilen(ctx context.Context, ids []string) ([]SchuelerEtikettZeile, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := r.db.Query(ctx, `
		SELECT coalesce(barcode_id, ''), coalesce(vorname, ''), coalesce(nachname, ''), coalesce(klasse, '')
		FROM schueler
		WHERE id = ANY($1) AND deleted_at IS NULL
		ORDER BY nachname, vorname
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

package repository

import (
	"context"
	"time"
)

// etikettenStatusBedingung übersetzt den Status der Nachdruck-Liste in sein Prädikat. Vorgabe
// ist „offen" (EtikettOffenBedingung): Die Liste heißt „Fehlende Etiketten". Liste und Zähler
// lesen es hier, damit die Fußzeile der Liste keine Zahl aus einer anderen Menge nennt als die
// Zeilen darüber.
func etikettenStatusBedingung(status string) string {
	switch status {
	case "erledigt":
		return `e.etikett_gedruckt = true AND e.ist_ausgesondert = false`
	case "alle":
		return `e.ist_ausgesondert = false`
	default:
		return EtikettOffenBedingung
	}
}

// EtikettExemplar ist ein Exemplar der Nachdruck-Liste mit dem Tag seines Zugangs (JJJJ-MM-TT;
// für ein Exemplar ohne Zugangsdatum der Tag des Erwerbs) und dem Vermerk „Etikett gedruckt".
type EtikettExemplar struct {
	BarcodeID       string
	Titel           string
	Autor           string
	ZugangAm        string
	EtikettGedruckt bool
}

// ListeEtikettenExemplare liefert die Exemplare eines Status (offen, erledigt, alle), die
// neuesten zuerst und höchstens limit Zeilen; suche filtert über Titel und Barcode. Scheitert
// das Lesen der Zeilen, trägt der Fehler ErrZeileUnlesbar.
func ListeEtikettenExemplare(ctx context.Context, db DBQueryer, status, suche string, limit int) ([]EtikettExemplar, error) {
	statusBedingung := etikettenStatusBedingung(status)
	rows, err := db.Query(ctx, `
			SELECT e.barcode_id, t.titel, coalesce(t.autor, ''), to_char(COALESCE(e.zugang_am, e.erworben_am), 'YYYY-MM-DD'),
			       e.etikett_gedruckt
			FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE `+statusBedingung+`
			  AND ($1 = '' OR t.titel ILIKE '%' || $1 || '%' OR e.barcode_id ILIKE '%' || $1 || '%')
			ORDER BY COALESCE(e.zugang_am, e.erworben_am) DESC, e.erstellt_am DESC, e.barcode_id
			LIMIT $2
		`, suche, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	liste := make([]EtikettExemplar, 0)
	for rows.Next() {
		var e EtikettExemplar
		if err := rows.Scan(&e.BarcodeID, &e.Titel, &e.Autor, &e.ZugangAm, &e.EtikettGedruckt); err != nil {
			return nil, zeileUnlesbar(err)
		}
		liste = append(liste, e)
	}
	if err := rows.Err(); err != nil {
		return nil, zeileUnlesbar(err)
	}
	return liste, nil
}

// ZaehleEtikettenExemplare zählt die Exemplare eines Status mit denselben Filtern wie die
// Liste; mit bis nur die, die bis zu diesem Tag in den Bestand kamen. Derselbe JOIN wie in der
// Liste, er ist verlustfrei: titel_id ist NOT NULL.
func ZaehleEtikettenExemplare(ctx context.Context, db DBQueryer, status string, bis *time.Time, suche string) (int, error) {
	var anzahl int
	err := db.QueryRow(ctx, `
			SELECT count(*)
			FROM buecher_exemplare e
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE `+etikettenStatusBedingung(status)+`
			  AND ($1::date IS NULL OR COALESCE(e.zugang_am, e.erworben_am) <= $1)
			  AND ($2 = '' OR t.titel ILIKE '%' || $2 || '%' OR e.barcode_id ILIKE '%' || $2 || '%')`,
		bis, suche,
	).Scan(&anzahl)
	return anzahl, err
}

// VermerkeAltbestandEtiketten vermerkt jedes Exemplar ohne Vermerk, das bis zum Stichtag in
// den Bestand kam, als beklebt und liefert die Zahl der Zeilen. Ausgesonderte bleiben außen
// vor.
func VermerkeAltbestandEtiketten(ctx context.Context, db DBQueryer, bis time.Time) (int64, error) {
	tag, err := db.Exec(ctx, `
			UPDATE buecher_exemplare e SET etikett_gedruckt = true, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE `+EtikettOffenBedingung+` AND COALESCE(e.zugang_am, e.erworben_am) <= $1
		`, bis)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// VermerkeEtikettenGedruckt vermerkt die genannten Exemplare als beklebt und liefert die Zahl
// der Zeilen, die vorher keinen Vermerk trugen.
func VermerkeEtikettenGedruckt(ctx context.Context, db DBQueryer, barcodes []string) (int64, error) {
	tag, err := db.Exec(ctx, `
			UPDATE buecher_exemplare SET etikett_gedruckt = true, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE barcode_id = ANY($1) AND etikett_gedruckt = false
		`, barcodes)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// NimmEtikettenVermerk nimmt den Vermerk der genannten Exemplare zurück und liefert die Zahl
// der Zeilen. Ausgesonderte bleiben außen vor: Für ein Buch, das nicht mehr im Regal steht,
// wäre ein Etikett immer falsch.
func NimmEtikettenVermerk(ctx context.Context, db DBQueryer, barcodes []string) (int64, error) {
	tag, err := db.Exec(ctx, `
			UPDATE buecher_exemplare SET etikett_gedruckt = false, aktualisiert_am = CURRENT_TIMESTAMP
			WHERE barcode_id = ANY($1) AND etikett_gedruckt = true AND ist_ausgesondert = false
		`, barcodes)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

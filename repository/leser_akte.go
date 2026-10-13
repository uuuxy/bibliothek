package repository

import (
	"context"
	"time"
)

// LaufendeAusleihe represents a currently checked out book copy detail for the student.
type LaufendeAusleihe struct {
	ID         string `json:"id"`
	AusleiheID string `json:"ausleihe_id"`
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	// ISBN wird nicht angezeigt, sondern gebraucht: Ohne sie kann die Ausleihliste kein
	// Cover nachladen, wenn keins am Titel gespeichert ist (CoverPeek fragt darüber den
	// Cover-Proxy) — und das ist bei importierten Beständen der Normalfall.
	ISBN     string `json:"isbn,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// IstLernmittel steuert die Marke „Lernmittel" an der Ausleihzeile — aus der
	// Spalte (Migration 093), nicht mehr aus einem LMF-Präfix in Titel oder Signatur.
	IstLernmittel  bool      `json:"ist_lernmittel"`
	AusgeliehenAm  time.Time `json:"ausgeliehen_am"`
	RueckgabeFrist time.Time `json:"rueckgabe_frist"`
	// IstDauerleihe ist die Ausleihe an jemanden, der kein Schüler ist
	// (`ausleihen.ist_handapparat`, gesetzt in erzeugeAusleihe und im Geräte-Pfad).
	//
	// Sie steht hier, weil die Oberfläche sonst eine Frist anzeigt und überschreitet, die
	// es nicht gibt: Entschieden am 16.09.2026 — „kollegen haben keine frist bzw werden einfach
	// nie gesperrt!" Die Sperr-Automatik hält das seit jeher (sie zählt nur Ausleihen mit
	// ist_handapparat = false); die Akte rechnete daneben ihr eigenes „überfällig" aus
	// dem Datum und färbte die Zeile nach einem Jahr rot.
	IstDauerleihe bool `json:"ist_dauerleihe"`
}

// HatFoto checks if an encrypted photo exists for the student.
func (a *pgLeserAbfragen) HatFoto(ctx context.Context, leserID string) (bool, error) {
	var hatFoto bool
	err := a.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler_fotos WHERE schueler_id = $1)", leserID).Scan(&hatFoto)
	return hatFoto, err
}

// ZaehleOffeneSchaeden zählt die unbezahlten Forderungen eines Lesers aus jedem Topf und nennt ihre
// Summe. Ein Storno setzt ist_bezahlt mit, die eine Bedingung deckt beides. Die Akte und der
// Hinweis an der Theke (service.pruefeHinweise) fragen beide hier, damit sie dieselbe Zahl nennen.
func ZaehleOffeneSchaeden(ctx context.Context, q Datenbank, leserID string) (anzahl int, summe float64, err error) {
	err = q.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(betrag), 0)::float8 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`,
		leserID).Scan(&anzahl, &summe)
	return anzahl, summe, err
}

// ZaehleOffeneSchaeden: Anzahl und Summe der unbezahlten Forderungen des Lesers.
func (a *pgLeserAbfragen) ZaehleOffeneSchaeden(ctx context.Context, leserID string) (int, float64, error) {
	return ZaehleOffeneSchaeden(ctx, a.db, leserID)
}

// LadeLaufendeAusleihen retrieves all books currently borrowed by the student.
func (a *pgLeserAbfragen) LadeLaufendeAusleihen(ctx context.Context, leserID string) ([]LaufendeAusleihe, error) {
	query := `
		SELECT 
			e.id, 
			a.id AS ausleihe_id,
			e.barcode_id, 
			t.titel, 
			coalesce(t.autor, ''), 
			coalesce(t.isbn, ''),
			coalesce(t.cover_url, ''),
			t.ist_lernmittel,
			a.ausgeliehen_am,
			a.rueckgabe_frist,
			a.ist_handapparat
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE a.schueler_id = $1 AND a.rueckgabe_am IS NULL
		ORDER BY a.ausgeliehen_am DESC
	`
	rows, err := a.db.Query(ctx, query, leserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ausleihen []LaufendeAusleihe
	for rows.Next() {
		var b LaufendeAusleihe
		if err := rows.Scan(
			&b.ID,
			&b.AusleiheID,
			&b.BarcodeID,
			&b.Titel,
			&b.Autor,
			&b.ISBN,
			&b.CoverURL,
			&b.IstLernmittel,
			&b.AusgeliehenAm,
			&b.RueckgabeFrist,
			&b.IstDauerleihe,
		); err != nil {
			return nil, err
		}
		ausleihen = append(ausleihen, b)
	}
	return ausleihen, rows.Err()
}

package repository

import (
	"context"
	"time"

	"bibliothek/pkg/betrag"
)

// LaufendeAusleihe ist eine Zeile der Ausleihliste in der Akte: das Exemplar, das der Leser
// gerade hat, mit Titel und Frist. ID ist die Kennung des Exemplars.
type LaufendeAusleihe struct {
	ID         string `json:"id"`
	AusleiheID string `json:"ausleihe_id"`
	BarcodeID  string `json:"barcode_id"`
	Titel      string `json:"titel"`
	Autor      string `json:"autor"`
	// ISBN wird nicht angezeigt, sondern gebraucht: Ohne sie kann die Ausleihliste kein
	// Cover nachladen, wenn keins am Titel gespeichert ist (CoverVorschau fragt darüber den
	// Cover-Proxy) — und das ist bei importierten Beständen der Normalfall.
	ISBN     string `json:"isbn,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// IstLernmittel steuert die Marke „Lernmittel" an der Zeile; der Wert kommt aus der Spalte
	// ist_lernmittel des Titels.
	IstLernmittel  bool      `json:"ist_lernmittel"`
	AusgeliehenAm  time.Time `json:"ausgeliehen_am"`
	RueckgabeFrist time.Time `json:"rueckgabe_frist"`
	// IstDauerleihe ist die Ausleihe an jemanden, der kein Schüler ist
	// (`ausleihen.ist_handapparat`, gesetzt in legeAusleiheAn und im Geräte-Pfad).
	//
	// Sie steht hier, weil die Oberfläche sonst eine Frist anzeigt und überschreitet, die es
	// nicht gibt: Kollegen haben keine Frist und werden nie gesperrt. Die Sperr-Automatik zählt
	// nur Ausleihen mit ist_handapparat = false; ohne das Feld rechnete die Akte ihr eigenes
	// „überfällig" aus dem Datum und färbte die Zeile nach einem Jahr rot.
	IstDauerleihe bool `json:"ist_dauerleihe"`
}

// HatFoto sagt, ob zu dem Leser ein verschlüsseltes Foto gespeichert ist.
func (a *LeserAbfragen) HatFoto(ctx context.Context, leserID string) (bool, error) {
	var hatFoto bool
	err := a.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler_fotos WHERE schueler_id = $1)", leserID).Scan(&hatFoto)
	return hatFoto, err
}

// ZaehleOffeneSchaeden zählt die unbezahlten Forderungen eines Lesers aus jedem Topf und nennt ihre
// Summe. Ein Storno setzt ist_bezahlt mit, die eine Bedingung deckt beides. Die Akte und der
// Hinweis an der Theke (service.pruefeHinweise) fragen beide hier, damit sie dieselbe Zahl nennen.
func ZaehleOffeneSchaeden(ctx context.Context, q Datenbank, leserID string) (anzahl int, summe betrag.Cent, err error) {
	err = q.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(betrag), 0) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`,
		leserID).Scan(&anzahl, &summe)
	return anzahl, summe, err
}

// ZaehleOffeneSchaeden zählt die unbezahlten Forderungen des Lesers und nennt ihre Summe.
func (a *LeserAbfragen) ZaehleOffeneSchaeden(ctx context.Context, leserID string) (int, betrag.Cent, error) {
	return ZaehleOffeneSchaeden(ctx, a.db, leserID)
}

// LadeLaufendeAusleihen liest die offenen Ausleihen des Lesers mit Exemplar und Titel, die
// jüngste zuerst.
func (a *LeserAbfragen) LadeLaufendeAusleihen(ctx context.Context, leserID string) ([]LaufendeAusleihe, error) {
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

package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// SperreOffeneAusleiheMitFrist liest die Frist einer offenen Ausleihe und hält ihre Zeile bis
// zum Ende der Transaktion gesperrt, damit zwei Verlängerungen zugleich nacheinander rechnen.
// pgx.ErrNoRows, wenn es die Ausleihe nicht gibt oder sie zurückgegeben ist.
func SperreOffeneAusleiheMitFrist(ctx context.Context, tx pgx.Tx, ausleiheID string) (time.Time, error) {
	var frist time.Time
	err := tx.QueryRow(ctx, `SELECT rueckgabe_frist FROM ausleihen WHERE id = $1 AND rueckgabe_am IS NULL FOR UPDATE`,
		ausleiheID).Scan(&frist)
	return frist, err
}

// VerlaengereAusleihe setzt die neue Frist einer offenen Ausleihe und beginnt die Mahnfolge
// neu: Sonst überspränge dasselbe Buch beim nächsten Überziehen die erste Mahnstufe. Liefert
// Kennung und gespeicherte Frist; pgx.ErrNoRows, wenn die Ausleihe nicht mehr offen ist.
func VerlaengereAusleihe(ctx context.Context, db DBQueryer, ausleiheID string, neueFrist time.Time) (id string, frist time.Time, err error) {
	q := `
			UPDATE ausleihen
			SET rueckgabe_frist = $2,
			    mahnstufe = 0,
			    letztes_mahndatum = NULL
			WHERE id = $1 AND rueckgabe_am IS NULL
			RETURNING id, rueckgabe_frist
		`
	err = db.QueryRow(ctx, q, ausleiheID, neueFrist).Scan(&id, &frist)
	return id, frist, err
}

// SetzeAusleihFrist setzt die Frist einer offenen Ausleihe von Hand und liefert Kennung und
// gespeicherte Frist; pgx.ErrNoRows, wenn die Ausleihe nicht mehr offen ist.
//
// Wie bei der regulären Verlängerung: Eine neue Frist in der Zukunft macht die
// Ausleihe wieder "nicht überfällig" und setzt die Mahn-Eskalation zurück. Ein
// vorgezogenes Datum (Rückruf) lässt die Mahnstufe unberührt.
func SetzeAusleihFrist(ctx context.Context, db DBQueryer, ausleiheID string, neueFrist time.Time) (id string, frist time.Time, err error) {
	q := `
			UPDATE ausleihen
			SET rueckgabe_frist = $1,
			    mahnstufe = CASE WHEN $1 > CURRENT_TIMESTAMP THEN 0 ELSE mahnstufe END,
			    letztes_mahndatum = CASE WHEN $1 > CURRENT_TIMESTAMP THEN NULL ELSE letztes_mahndatum END
			WHERE id = $2 AND rueckgabe_am IS NULL
			RETURNING id, rueckgabe_frist
		`
	err = db.QueryRow(ctx, q, neueFrist, ausleiheID).Scan(&id, &frist)
	return id, frist, err
}

// VerlaengereLernmittelDerKlasse setzt die Frist aller offenen Lernmittel-Ausleihen einer
// Klasse und liefert die Zahl der angepassten Ausleihen.
//
// Mass-Verlängerung setzt zugleich die Mahn-Eskalation der betroffenen Ausleihen
// zurück (sofern die neue Frist in der Zukunft liegt) — sonst würde ein ganzer
// Klassensatz nach der Verlängerung fälschlich auf der alten Mahnstufe weiterlaufen.
func VerlaengereLernmittelDerKlasse(ctx context.Context, db DBQueryer, klasse string, neueFrist time.Time) (int64, error) {
	q := `
			UPDATE ausleihen a
			SET rueckgabe_frist = $1,
			    mahnstufe = CASE WHEN $1 > CURRENT_TIMESTAMP THEN 0 ELSE a.mahnstufe END,
			    letztes_mahndatum = CASE WHEN $1 > CURRENT_TIMESTAMP THEN NULL ELSE a.letztes_mahndatum END
			FROM schueler s, buecher_exemplare e, buecher_titel t
			WHERE a.schueler_id = s.id
			  AND a.exemplar_id = e.id
			  AND e.titel_id = t.id
			  AND a.rueckgabe_am IS NULL
			  -- Die Sperre von Hand nimmt ein Kind aus: Sie soll zur Rückgabe zwingen, nicht
			  -- durch eine Fristverlängerung ausgehebelt werden. Die der Ehemaligen zählt beim
			  -- Schulbuch nicht, wie an der Theke.
			  AND ` + SchulbuchFristGehtMitSQL + `
			  -- Über den Normal-Schlüssel (Migration 079/087): „5a" aus einem Formular und
			  -- „05A" als registrierte Anzeigeform meinen dieselbe Klasse.
			  AND klassen_normkey(s.klasse) = klassen_normkey($2)
			  AND t.ist_lernmittel
		`
	tag, err := db.Exec(ctx, q, neueFrist, klasse)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LeserUndLernmittelDerAusleihe liest zu einer offenen Ausleihe ihren Leser (nil, wenn sie von
// ihm getrennt ist) und ob ihr Titel ein Lernmittel ist. pgx.ErrNoRows, wenn es die Ausleihe
// nicht gibt oder sie zurückgegeben ist.
func LeserUndLernmittelDerAusleihe(ctx context.Context, db DBQueryer, ausleiheID string) (leserID *string, lernmittel bool, err error) {
	err = db.QueryRow(ctx, `
		SELECT a.schueler_id, COALESCE(t.ist_lernmittel, false)
		FROM ausleihen a
		LEFT JOIN buecher_exemplare e ON e.id = a.exemplar_id
		LEFT JOIN buecher_titel t ON t.id = e.titel_id
		WHERE a.id = $1 AND a.rueckgabe_am IS NULL
	`, ausleiheID).Scan(&leserID, &lernmittel)
	return leserID, lernmittel, err
}

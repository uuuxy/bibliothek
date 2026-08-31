package repository

import (
	"context"
	"fmt"
	"strings"
)

// HasPhoto checks if an encrypted photo exists for the student.
func (repo *pgStudentRepository) HasPhoto(ctx context.Context, studentID string) (bool, error) {
	var hasPhoto bool
	err := repo.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schueler_fotos WHERE schueler_id = $1)", studentID).Scan(&hasPhoto)
	return hasPhoto, err
}

// HasOpenDamages checks if the student has any unpaid damage fees.
func (repo *pgStudentRepository) HasOpenDamages(ctx context.Context, studentID string) (bool, error) {
	var hasOpenDamages bool
	err := repo.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false)", studentID).Scan(&hasOpenDamages)
	return hasOpenDamages, err
}

// GetActiveBorrowedBooks retrieves all books currently borrowed by the student.
func (repo *pgStudentRepository) GetActiveBorrowedBooks(ctx context.Context, studentID string) ([]BorrowedBook, error) {
	query := `
		SELECT 
			e.id, 
			a.id AS ausleihe_id,
			e.barcode_id, 
			t.titel, 
			coalesce(t.autor, ''), 
			coalesce(t.isbn, ''),
			coalesce(t.cover_url, ''),
			coalesce(t.signatur, ''),
			a.ausgeliehen_am,
			a.rueckgabe_frist
		FROM ausleihen a
		JOIN buecher_exemplare e ON a.exemplar_id = e.id
		JOIN buecher_titel t ON e.titel_id = t.id
		WHERE a.schueler_id = $1 AND a.rueckgabe_am IS NULL
		ORDER BY a.ausgeliehen_am DESC
	`
	rows, err := repo.db.Query(ctx, query, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []BorrowedBook
	for rows.Next() {
		var b BorrowedBook
		if err := rows.Scan(
			&b.ID,
			&b.AusleiheID,
			&b.BarcodeID,
			&b.Titel,
			&b.Autor,
			&b.ISBN,
			&b.CoverURL,
			&b.Signatur,
			&b.AusgeliehenAm,
			&b.RueckgabeFrist,
		); err != nil {
			return nil, err
		}
		books = append(books, b)
	}
	return books, rows.Err()
}

// GetDistinctClasses returns a list of all active classes.
func (repo *pgStudentRepository) GetDistinctClasses(ctx context.Context) ([]string, error) {
	rows, err := repo.db.Query(ctx, "SELECT DISTINCT klasse FROM schueler WHERE klasse != '' AND deleted_at IS NULL ORDER BY klasse")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classes []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err == nil {
			classes = append(classes, k)
		}
	}
	return classes, rows.Err()
}

// ListStudentsWithStatsLimit ist die Obergrenze der ungefilterten Schülerdatei. Sie
// existiert, damit eine Schule mit 20.000 Schülern nicht bei jedem Seitenaufruf die
// komplette Kartei über die Leitung schickt. Wer jemanden dahinter sucht, findet ihn
// über suche — die Suche läuft auf dem Server und kennt keine Grenze bei 500.
const ListStudentsWithStatsLimit = 500

// ListStudentsWithStats liefert Schüler samt Ausleihzahlen, optional auf eine Klasse
// eingegrenzt und/oder auf einen Suchbegriff (Name oder Barcode).
//
// Gesucht wird über dieselben SQL-Bausteine wie in der Omnibox an der Theke
// (SchuelerSuchCTE/-Bedingung/-Rang, siehe student_queries.go): Tokens in beliebiger
// Reihenfolge, diakritikfrei, deutsche Ersatzschreibung in beide Richtungen. Vorher
// filterte die Schülerdatei im Browser über die ersten 500 gelieferten Zeilen — wer
// alphabetisch dahinter lag, war über die Suche nicht erreichbar, und zwar abhängig vom
// Klassennamen, also für den Benutzer ohne erkennbares Muster.
func (repo *pgStudentRepository) ListStudentsWithStats(ctx context.Context, klasse, suche string) ([]StudentListStat, error) {
	// Die Bedingungen werden zusammengesetzt, weil jede für sich optional ist. Die
	// Platzhalternummern stehen fest ($1/$2 Suche, $3 Klasse) und die zugehörigen
	// Argumente werden nur dann angehängt, wenn ihre Bedingung auch im SQL landet —
	// Postgres lehnt sonst überzählige Bind-Parameter ab.
	bedingungen := []string{"s.deleted_at IS NULL"}
	args := []any{}
	praefix := ""
	limit := ""

	if tokens := suchTokens(suche); len(tokens) > 0 {
		praefix = SchuelerSuchCTE
		// mitKlasse: Die Schülerdatei durchsuchte schon immer auch die Klasse ("10a"
		// listet die Klasse). Das darf die Serversuche nicht stillschweigend verlieren.
		bedingungen = append(bedingungen, SchuelerSuchBedingung(true))
		args = append(args, tokens, tokens[0])
	} else {
		// Nur die reine Liste wird gekappt. Eine Suche darf das nicht — genau daran
		// ist sie vorher gescheitert.
		limit = fmt.Sprintf(" LIMIT %d", ListStudentsWithStatsLimit)
	}

	if klasse != "" {
		args = append(args, klasse)
		bedingungen = append(bedingungen, fmt.Sprintf("s.klasse = $%d", len(args)))
	}

	// Bei einer Suche zuerst die besten Treffer, sonst die gewohnte Kartei-Reihenfolge.
	sortierung := "s.klasse, s.nachname, s.vorname"
	if praefix != "" {
		sortierung = SchuelerSuchRang + ", s.nachname ASC, s.vorname ASC"
	}

	rows, err := repo.db.Query(ctx, praefix+`
		SELECT s.id, s.barcode_id, s.vorname, s.nachname, s.klasse, s.abgaenger_jahr, s.ist_gesperrt,
			COALESCE(s.is_manually_blocked, false) as is_manually_blocked,
			COALESCE(l.ausgeliehen_anzahl, 0) as ausgeliehen_anzahl,
			COALESCE(l.ueberfaellig_anzahl, 0) as ueberfaellig_anzahl,
			EXISTS(SELECT 1 FROM schueler_fotos sf WHERE sf.schueler_id = s.id) as has_foto
		FROM schueler s
		LEFT JOIN LATERAL (
			SELECT
				COUNT(*) as ausgeliehen_anzahl,
				COUNT(*) FILTER (WHERE a.rueckgabe_frist < CURRENT_TIMESTAMP) as ueberfaellig_anzahl
			FROM ausleihen a
			WHERE a.schueler_id = s.id AND a.rueckgabe_am IS NULL
		) l ON true
		WHERE `+strings.Join(bedingungen, " AND ")+`
		ORDER BY `+sortierung+limit, args...)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []StudentListStat
	for rows.Next() {
		var s StudentListStat
		if err := rows.Scan(&s.ID, &s.BarcodeID, &s.Vorname, &s.Nachname, &s.Klasse, &s.AbgaengerJahr, &s.IstGesperrt, &s.IsManuallyBlocked, &s.AusgeliehenCount, &s.UeberfaelligCount, &s.HasFoto); err != nil {
			return nil, err
		}
		if s.BarcodeID != "" && s.HasFoto {
			s.FotoURL = fmt.Sprintf("/api/schueler/%s/photo", s.BarcodeID)
		}
		// Wie in scanStudentMitZusatz: Die Ausweisgültigkeit wird beim Lesen gerechnet,
		// nicht gespeichert. Der Klassensatz-Druck im Ausweis-Designer liest über diese
		// Liste — ohne die Zeile trüge dort jede Karte "Gültig bis: –".
		s.AusweisGueltigBis = ausweisGueltigBis(s.Klasse)
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

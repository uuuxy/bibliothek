package inventur

import (
	"bibliothek/repository"
	"context"
	"fmt"

	"bibliothek/db"
	"bibliothek/pkg/isbnutil"
)

// UpdateBook updates metadata fields of a book.
//
// aktualisiert_am wurde bis zum 10.08.2026 als EINZIGE Spalte von buecher_titel beim
// Aktualisieren nicht gesetzt. Sie steht in der API-Antwort (repository/book_search.go
// liefert sie mit) und behauptete dort den Zeitpunkt des ANLEGENS. In der Oberfläche liest
// sie heute niemand — aber ein Feld, das etwas anderes sagt als sein Name, ist eine Falle
// für den Nächsten, etwa für einen Abgleich, der „was hat sich seit gestern geändert"
// darüber beantworten will. Gefunden hat es schema_paritaet_test.go.
//
// Die Erklärung steht hier und nicht als SQL-Kommentar in der Anweisung: Ein Kommentar im
// Query-String reist bei jedem Aufruf zum Server mit und lässt jeden Test scheitern, der
// die Anweisung als Ganzes festhält.
// bestand ist bewusst ein Zeiger: nil heißt "der Aufrufer hat zum Bestand nichts
// gesagt" und lässt die physischen Exemplare unangetastet. Bis zum 23.08.2026 war es
// ein int, und eine fehlende Angabe kam als 0 an — syncBookStock sonderte daraufhin
// JEDES Exemplar des Titels aus, im Rückfallzweig auch die gerade ausgeliehenen. Der
// Weg dorthin war kurz: `Number(undefined)` im Formular ist NaN, in JSON null, in Go 0,
// und die Warnung im Formular ("du verringerst den Bestand") greift bei NaN nicht, weil
// `NaN < 5` falsch ist.
func (repo *BookRepository) UpdateBook(ctx context.Context, id string, book Book, bestand *Bestandsangabe) error {
	// subject ist FK auf die Systematik (Migration 078): unbekannte Fächer erst
	// registrieren, die kanonische Schreibweise schreiben, Leerwert wird NULL.
	kanonisch, err := StelleFaecherSicher(ctx, repo.db, []string{book.Subject})
	if err != nil {
		return err
	}

	query := `
		UPDATE buecher_titel
		SET isbn = NULLIF($1, ''),
			titel = $2,
			autor = $3,
			cover_url = $4,
			subject = NULLIF($5, ''),
			grade_level = $6,
			track = $7,
			last_counted = NULLIF($8::text, '')::date,
			medientyp = $9,
			erweiterte_eigenschaften = $10,
			jahrgang_von = $11,
			jahrgang_bis = $12,
			untertitel = $13,
			verlag = $14,
			erscheinungsjahr = $15,
			signatur = COALESCE(NULLIF($17, ''), signatur),
			ist_lernmittel = $18,
			auflage = NULLIF($19, ''),
			listenpreis = $20,
			mehrjahresband = $21,
			aktualisiert_am = NOW()
		WHERE id = $16`

	medientyp := book.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}

	properties := book.ErweiterteEigenschaften
	if properties == nil {
		properties = make(map[string]any)
	}

	// Titel-Update und Bestands-Synchronisierung atomar: Schlägt der Sync fehl (z. B. die
	// zweistufige Aussonderung), bleibt sonst ein Titel mit falschem Bestand zurück, den
	// der Handler als erfolgreich meldet (Transaktionsgrenzen-Sweep A3). Der Sync-Fehler
	// wird deshalb ZURÜCKGEGEBEN, nicht nur geloggt.
	tx, err := repo.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", err)
	}
	defer db.SafeRollback(ctx, tx)

	if err := pruefeAenderung(ctx, tx, id, book); err != nil {
		return err
	}

	result, err := tx.Exec(
		ctx,
		query,
		book.ISBN,
		book.Title,
		book.Author,
		book.CoverURL,
		kanonisch[book.Subject],
		book.GradeLevel,
		book.Track,
		book.LastCounted,
		medientyp,
		properties,
		book.JahrgangVon,
		book.JahrgangBis,
		book.Untertitel,
		book.Verlag,
		book.Erscheinungsjahr,
		id,
		book.Signatur,       // $17 — leerer Wert lässt die verklebte Signatur unangetastet
		book.IstLernmittel,  // $18 — die Maske entscheidet ausdrücklich (Migration 093)
		book.Auflage,        // $19 — die Maske ist der Ort der Angabe, leer heißt „keine"
		book.Listenpreis,    // $20 — Zeiger: nil löscht den Wert, das ist hier gewollt
		book.Mehrjahresband, // $21 — Mehrjahresband: bleibt über die Spanne beim Kind (Migration 134)
	)
	if err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", handleDbError(err))
	}

	if result.RowsAffected() == 0 {
		return ErrBookNotFound
	}

	// Schlagworte in derselben Transaktion (Migration 138). nil heißt „nicht
	// mitgeschickt" und lässt die vorhandenen stehen — wie beim Bestand oben.
	if book.Schlagworte != nil {
		if _, err := repository.SetzeSchlagworte(ctx, tx, id, book.Schlagworte); err != nil {
			return fmt.Errorf("schlagworte konnten nicht gespeichert werden: %w", err)
		}
	}

	// Nach dem UPDATE des Titels: Dessen Zeilensperre lässt einen zweiten Speichervorgang
	// warten, bis dieser festgeschrieben ist, und erst dann zählen.
	if bestand != nil {
		if err := repo.setzeBestand(ctx, tx, id, *bestand); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", err)
	}
	return nil
}

// pruefeAenderung hält die Änderung gegen den gespeicherten Titel und prüft, was sich ändert.
// Ein leerer Autor ist nur ein Fehler, wenn der Titel einen trägt: Dann hat ein Formular das
// Feld nie befüllt, oder jemand hat es geleert. Die ISBN wird nur geprüft, wenn sie eine
// andere ist als die gespeicherte: Ein Titel ohne ISBN oder mit einer Nummer, die keine ISBN
// ist, bleibt speicherbar, wenn ein anderes Feld geändert wird. Die Sperre hält die Zeile bis
// zum UPDATE derselben Transaktion.
func pruefeAenderung(ctx context.Context, tx repository.DBQueryer, id string, book Book) error {
	var isbn, autor string
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(isbn, ''), COALESCE(autor, '') FROM buecher_titel WHERE id = $1 FOR UPDATE`, id).Scan(&isbn, &autor)
	switch {
	case istKeineZeile(err):
		return ErrBookNotFound
	case err != nil:
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", err)
	case book.Author == "" && autor != "":
		return ErrAutorGeleert
	case book.ISBN == "" || isbnutil.Normalform(book.ISBN) == isbnutil.Normalform(isbn):
		return nil
	case !validiereISBN(book.ISBN):
		return ErrISBNFormat
	}
	return isbnVergeben(ctx, tx, book.ISBN, id)
}

// Bestandsangabe ist, was eine Maske zum Bestand eines vorhandenen Titels sagt: die Zahl,
// die gelten soll, und die Zahl, die sie beim Öffnen geladen hat. Gesehen ist nil bei einem
// Aufrufer, der die zweite Zahl nicht kennt (eine vor dem Update geladene Seite).
type Bestandsangabe struct {
	Soll    int
	Gesehen *int
}

// BestandVeraltet lehnt eine Bestandsänderung ab, deren Maske einen anderen Stand gesehen
// hat, als jetzt gilt: Zwischen Öffnen und Speichern hat jemand Exemplare angelegt,
// ausgesondert oder eingebucht. Die Tür antwortet mit 409 und nennt den Stand.
type BestandVeraltet struct {
	Gesehen *int
	Aktuell int
}

func (e *BestandVeraltet) Error() string {
	if e.Gesehen == nil {
		return fmt.Sprintf("bestand ohne gesehenen stand geändert, aktuell %d", e.Aktuell)
	}
	return fmt.Sprintf("bestand veraltet: gesehen %d, aktuell %d", *e.Gesehen, e.Aktuell)
}

// Meldung ist der Satz für die Maske. Ohne gesehene Zahl kommt die Anfrage von einer Seite,
// die vor dem Update geladen wurde: Sie kann die Zahl nicht mitschicken, bis sie neu lädt.
func (e *BestandVeraltet) Meldung() string {
	if e.Gesehen == nil {
		return fmt.Sprintf("Der Bestand lässt sich mit dieser Fassung der Seite nicht ändern (er steht bei %d). "+
			"Nichts gespeichert: bitte die Seite neu laden und den Titel neu öffnen.", e.Aktuell)
	}
	return fmt.Sprintf("Der Bestand wurde inzwischen an anderer Stelle geändert: jetzt %d statt %d. "+
		"Nichts gespeichert: bitte die gewünschte Zahl neu eintragen.", e.Aktuell, *e.Gesehen)
}

// setzeBestand gleicht die Exemplare nur an, wenn die Maske den Stand gesehen hat, der jetzt
// gilt. Ohne den Vergleich setzte eine länger offene Maske den Bestand auf ihre alte Zahl
// zurück und sonderte aus, was ein anderer Platz inzwischen angelegt hatte.
//
// Ein Aufrufer ohne gesehene Zahl darf den Stand bestätigen (Soll gleich Stand, nichts
// geschieht), aber nicht ändern.
func (repo *BookRepository) setzeBestand(ctx context.Context, q repository.DBQueryer, titelID string, angabe Bestandsangabe) error {
	aktuell, err := zaehleBestand(ctx, q, titelID)
	if err != nil {
		return fmt.Errorf("exemplare konnten nicht synchronisiert werden: %w", err)
	}
	if angabe.Gesehen == nil && angabe.Soll == aktuell {
		return nil
	}
	if angabe.Gesehen == nil || *angabe.Gesehen != aktuell {
		return &BestandVeraltet{Gesehen: angabe.Gesehen, Aktuell: aktuell}
	}
	if err := repo.gleicheExemplareAn(ctx, q, titelID, aktuell, angabe.Soll); err != nil {
		return fmt.Errorf("exemplare konnten nicht synchronisiert werden: %w", err)
	}
	return nil
}

// zaehleBestand zählt die Exemplare im Bestand (repository.SQLExemplarImBestand), die Zahl,
// die die Maske zeigt. Bestellte Exemplare gehören dem Wareneingang und zählen nicht mit.
func zaehleBestand(ctx context.Context, q repository.DBQueryer, titelID string) (int, error) {
	var bestand int
	err := q.QueryRow(ctx, `SELECT COUNT(*) FROM buecher_exemplare e WHERE e.titel_id = $1 AND `+repository.SQLExemplarImBestand, titelID).Scan(&bestand)
	if err != nil {
		return 0, fmt.Errorf("fehler beim ermitteln des aktuellen bestands: %w", err)
	}
	return bestand, nil
}

// syncBookStock synchronizes the physical buecher_exemplare records to match the expected stock.
//
// q ist repository.DBQueryer statt dbSchreiber, weil die Nummernvergabe (barcode_seq gegen
// Bestandsabgleich) Query braucht — pgx.Tx und der Pool erfüllen beides.
func (repo *BookRepository) syncBookStock(ctx context.Context, q repository.DBQueryer, titelID string, expectedStock int) error {
	currentStock, err := zaehleBestand(ctx, q, titelID)
	if err != nil {
		return err
	}
	return repo.gleicheExemplareAn(ctx, q, titelID, currentStock, expectedStock)
}

// gleicheExemplareAn legt fehlende Exemplare an oder sondert überzählige aus, bis der Bestand
// expectedStock erreicht. Ausgesondert wird nur aus dem Bestand: Bestellte Exemplare lassen
// sich über die Zahl nicht aussondern.
func (repo *BookRepository) gleicheExemplareAn(ctx context.Context, q repository.DBQueryer, titelID string, currentStock, expectedStock int) error {
	if expectedStock > currentStock {
		// Nummern aus barcode_seq — dieselbe Quelle wie Bestellwesen, Handvergabe und
		// Littera-Import (Migration 068: eine Quelle für alle Wege).
		barcodes, err := repository.ZieheFreieExemplarBarcodes(ctx, q, expectedStock-currentStock)
		if err != nil {
			return fmt.Errorf("fehler beim generieren von exemplaren im batch: %w", err)
		}
		_, err = q.Exec(ctx, `
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz)
				SELECT $1, unnest($2::text[]), true, 'Automatisch generiert'
			`, titelID, barcodes)
		if err != nil {
			return fmt.Errorf("fehler beim generieren von exemplaren im batch: %w", err)
		}
		return nil
	}
	if expectedStock == currentStock {
		return nil
	}
	numToRetire := currentStock - expectedStock

	// 1. Versuchen, nicht-ausgeliehene Exemplare auszusondern
	//
	// ist_ausleihbar = false gehört dazu, wie in den anderen Aussonderungswegen
	// (repository/audit_books.go, damage.go, book_inventory.go): Ein ausgesondertes
	// Exemplar, das sich weiterhin „ausleihbar" nennt, verleiht der erste Leser, der nur
	// diese Spalte prüft.
	query := `
			UPDATE buecher_exemplare
			SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'BESTANDSKORREKTUR', bestellstatus = NULL, letzte_bewegung_am = CURRENT_TIMESTAMP,
			    zustand_notiz = COALESCE(zustand_notiz || ' | ', '') || 'Automatisch ausgesondert'
			WHERE id IN (
				SELECT e.id
				FROM buecher_exemplare e
				LEFT JOIN ausleihen a ON a.exemplar_id = e.id AND a.rueckgabe_am IS NULL
				WHERE e.titel_id = $1 AND ` + repository.SQLExemplarImBestand + ` AND a.id IS NULL
				LIMIT $2
			)
		`
	result, err := q.Exec(ctx, query, titelID, numToRetire)
	if err != nil {
		return fmt.Errorf("fehler beim aussondern von exemplaren: %w", err)
	}

	retired := result.RowsAffected()
	if retired >= int64(numToRetire) {
		return nil
	}
	// 2. Fallback: Auch ausgeliehene Exemplare aussondern, falls nötig
	remainingToRetire := int64(numToRetire) - retired
	fallbackQuery := `
				UPDATE buecher_exemplare
				SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'BESTANDSKORREKTUR', bestellstatus = NULL, letzte_bewegung_am = CURRENT_TIMESTAMP,
				    zustand_notiz = COALESCE(zustand_notiz || ' | ', '') || 'Automatisch ausgesondert (war ausgeliehen)'
				WHERE id IN (
					SELECT e.id
					FROM buecher_exemplare e
					WHERE e.titel_id = $1 AND ` + repository.SQLExemplarImBestand + `
					LIMIT $2
				)
			`
	_, err = q.Exec(ctx, fallbackQuery, titelID, remainingToRetire)
	if err != nil {
		return fmt.Errorf("fehler beim aussondern (fallback): %w", err)
	}
	return nil
}

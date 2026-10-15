package inventur

import (
	"bibliothek/repository"
	"context"
	"fmt"
	"slices"
	"strings"

	"bibliothek/db"
	"bibliothek/pkg/isbnutil"
)

// titelFeld ist ein Feld des Titels: der Name, unter dem die Eingabe es nennt, seine Spalte
// und der Ausdruck, der ihren Wert schreibt.
type titelFeld struct{ name, spalte, wert string }

// titelFelder sind die Felder, die UpdateBook schreibt, mit den Parametern seiner Anweisung.
// Geschrieben wird nur, was der Aufrufer nennt: Zwei Plätze, die denselben Titel offen haben,
// überschrieben sich sonst gegenseitig Felder, die keiner von beiden angefasst hat.
var titelFelder = []titelFeld{
	{"isbn", "isbn", "NULLIF($1, '')"},
	{"title", "titel", "$2"},
	{"author", "autor", "$3"},
	{"coverUrl", "cover_url", "$4"},
	{"subject", "subject", "NULLIF($5, '')"},
	{"track", "track", "$6"},
	{"lastCounted", "last_counted", "NULLIF($7::text, '')::date"},
	{"medientyp", "medientyp", "$8"},
	{"erweiterteEigenschaften", "erweiterte_eigenschaften", "$9"},
	// 0 heißt „unbekannt" und wird NULL (Migration 162).
	{"jahrgangVon", "jahrgang_von", "NULLIF($10, 0)"},
	{"jahrgangBis", "jahrgang_bis", "NULLIF($11, 0)"},
	{"untertitel", "untertitel", "$12"},
	{"verlag", "verlag", "$13"},
	{"erscheinungsjahr", "erscheinungsjahr", "$14"},
	// Ein leerer Wert lässt die verklebte Signatur unangetastet.
	{"signatur", "signatur", "COALESCE(NULLIF($16, ''), signatur)"},
	{"istLernmittel", "ist_lernmittel", "$17"},
	{"auflage", "auflage", "NULLIF($18, '')"},
	// Ein Listenpreis ohne Wert (nil) löscht ihn: „nicht erfasst" ist etwas anderes als 0.
	{"listenpreis", "listenpreis", "$19"},
	{"mehrjahresband", "mehrjahresband", "$20"},
}

// sqlTitelAendern setzt je Spalte den neuen Wert, wenn $21 ihr Feld nennt, und sonst den, der
// in der Zeile steht. Die Anweisung nennt jede Spalte (schema_paritaet_test.go) und ist für
// jede Änderung dieselbe.
//
// Die Erklärung steht hier und nicht als SQL-Kommentar: Der reiste bei jedem Aufruf zum
// Server mit und ließe jeden Test scheitern, der die Anweisung als Ganzes festhält.
var sqlTitelAendern = func() string {
	var b strings.Builder
	b.WriteString("UPDATE buecher_titel SET ")
	for _, f := range titelFelder {
		fmt.Fprintf(&b, "%[2]s = CASE WHEN '%[1]s' = ANY($21::text[]) THEN %[3]s ELSE %[2]s END, ", f.name, f.spalte, f.wert)
	}
	b.WriteString("aktualisiert_am = NOW() WHERE id = $15")
	return b.String()
}()

// pruefeFeldnamen lehnt einen Namen ab, den titelFelder nicht kennt: Die Anweisung überginge
// ihn, und der Aufrufer hielte sein Feld für gespeichert.
func pruefeFeldnamen(felder []string) error {
	for _, name := range felder {
		if !slices.ContainsFunc(titelFelder, func(f titelFeld) bool { return f.name == name }) {
			return fmt.Errorf("buch konnte nicht aktualisiert werden: unbekanntes feld %q", name)
		}
	}
	return nil
}

// AendereTitel schreibt die genannten Felder eines Titels (felder, Namen wie in titelFelder)
// und gleicht auf Wunsch den Bestand an, beides in einer Transaktion: Scheitert der Abgleich,
// bleibt auch der Titel, wie er war. Ein Feld, das felder nicht nennt, bleibt, wie es in der
// Datenbank steht. bestand ist ein Zeiger: nil heißt, der Aufrufer hat zum Bestand nichts
// gesagt, und lässt die Exemplare unangetastet — eine fehlende Angabe käme sonst als 0 an und
// sonderte jedes Exemplar aus. aktualisiert_am wird bei jedem Speichern gesetzt.
func (a *TitelAbfragen) AendereTitel(ctx context.Context, id string, eintrag Titel, felder []string, bestand *Bestandsangabe) error {
	werte, err := a.titelWerte(ctx, id, eintrag, felder)
	if err != nil {
		return err
	}

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", err)
	}
	defer db.Verwerfe(ctx, tx)

	if err := pruefeAenderung(ctx, tx, id, eintrag, felder); err != nil {
		return err
	}

	ergebnis, err := tx.Exec(ctx, sqlTitelAendern, werte...)
	if err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", uebersetzeDatenbankfehler(err))
	}

	if ergebnis.RowsAffected() == 0 {
		return ErrTitelNichtGefunden
	}

	// Schlagworte in derselben Transaktion (Migration 138). nil heißt „nicht
	// mitgeschickt" und lässt die vorhandenen stehen — wie beim Bestand oben.
	if eintrag.Schlagworte != nil {
		if _, err := repository.SetzeSchlagworte(ctx, tx, id, eintrag.Schlagworte); err != nil {
			return fmt.Errorf("schlagworte konnten nicht gespeichert werden: %w", err)
		}
	}

	// Nach dem UPDATE des Titels: Dessen Zeilensperre lässt einen zweiten Speichervorgang
	// warten, bis dieser festgeschrieben ist, und erst dann zählen.
	if bestand != nil {
		if err := a.setzeBestand(ctx, tx, id, *bestand); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", err)
	}
	return nil
}

// titelWerte sind die Parameter von sqlTitelAendern, der letzte nennt die Felder. Ein
// genanntes Fach wird vorher registriert, wenn die Systematik es nicht kennt (subject ist FK,
// Migration 078), und in ihrer Schreibweise geschrieben; ein Leerwert wird NULL.
func (a *TitelAbfragen) titelWerte(ctx context.Context, id string, eintrag Titel, felder []string) ([]any, error) {
	if err := pruefeFeldnamen(felder); err != nil {
		return nil, err
	}
	fach := ""
	if slices.Contains(felder, "subject") {
		kanonisch, err := StelleFaecherSicher(ctx, a.pool, []string{eintrag.Fach})
		if err != nil {
			return nil, err
		}
		fach = kanonisch[eintrag.Fach]
	}
	medientyp := eintrag.Medientyp
	if medientyp == "" {
		medientyp = "Buch"
	}
	eigenschaften := eintrag.ErweiterteEigenschaften
	if eigenschaften == nil {
		eigenschaften = make(map[string]any)
	}
	if felder == nil {
		felder = []string{}
	}
	return []any{
		eintrag.ISBN, eintrag.Titel, eintrag.Autor, eintrag.CoverURL, fach, eintrag.Zweig,
		eintrag.ZuletztGezaehlt, medientyp, eigenschaften, eintrag.JahrgangVon, eintrag.JahrgangBis,
		eintrag.Untertitel, eintrag.Verlag, eintrag.Erscheinungsjahr, id, eintrag.Signatur,
		eintrag.IstLernmittel, eintrag.Auflage, eintrag.Listenpreis, eintrag.Mehrjahresband, felder,
	}, nil
}

// pruefeAenderung hält die genannten Felder gegen den gespeicherten Titel und prüft, was sich
// ändert. Ein leerer Autor ist nur ein Fehler, wenn der Titel einen trägt: Dann hat jemand das
// Feld geleert. Die ISBN wird nur geprüft, wenn sie eine andere ist als die gespeicherte: Ein
// Titel ohne ISBN oder mit einer Nummer, die keine ISBN ist, bleibt speicherbar, wenn ein
// anderes Feld geändert wird. Die Sperre hält die Zeile bis zum UPDATE derselben Transaktion.
func pruefeAenderung(ctx context.Context, tx repository.Datenbank, id string, eintrag Titel, felder []string) error {
	nennt := func(name string) bool { return slices.Contains(felder, name) }
	var isbn, autor string
	var lernmittel, mehrjahresband bool
	var von, bis int
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(isbn, ''), COALESCE(autor, ''), ist_lernmittel, mehrjahresband, COALESCE(jahrgang_von, 0), COALESCE(jahrgang_bis, 0) FROM buecher_titel WHERE id = $1 FOR UPDATE`, id).
		Scan(&isbn, &autor, &lernmittel, &mehrjahresband, &von, &bis)
	switch {
	case istKeineZeile(err):
		return ErrTitelNichtGefunden
	case err != nil:
		return fmt.Errorf("buch konnte nicht aktualisiert werden: %w", err)
	case nennt("author") && eintrag.Autor == "" && autor != "":
		return ErrAutorGeleert
	}
	if err := pruefeSpanneNachAenderung(eintrag, nennt, titelSpanne{lernmittel, mehrjahresband, von, bis}); err != nil {
		return err
	}
	switch {
	case !nennt("isbn") || eintrag.ISBN == "" || isbnutil.Normalform(eintrag.ISBN) == isbnutil.Normalform(isbn):
		return nil
	case !validiereISBN(eintrag.ISBN):
		return ErrISBNFormat
	}
	return isbnVergeben(ctx, tx, eintrag.ISBN, id)
}

// titelSpanne ist, woran das Mehrjahresband hängt: Lernmittel, der Schalter und die Jahrgänge.
type titelSpanne struct {
	lernmittel, mehrjahresband bool
	von, bis                   int
}

// pruefeSpanneNachAenderung prüft Spanne und Mehrjahresband am Stand nach der Änderung: die
// genannten Felder aus der Eingabe, die übrigen vom Titel (stand). Nennt die Änderung keins der
// vier, prüft sie nichts: Was unverändert am Titel steht, hindert das Speichern eines anderen
// Felds nicht.
func pruefeSpanneNachAenderung(eintrag Titel, nennt func(string) bool, stand titelSpanne) error {
	if !nennt("istLernmittel") && !nennt("mehrjahresband") && !nennt("jahrgangVon") && !nennt("jahrgangBis") {
		return nil
	}
	if nennt("istLernmittel") {
		stand.lernmittel = eintrag.IstLernmittel
	}
	if nennt("mehrjahresband") {
		stand.mehrjahresband = eintrag.Mehrjahresband
	}
	if nennt("jahrgangVon") {
		stand.von = eintrag.JahrgangVon
	}
	if nennt("jahrgangBis") {
		stand.bis = eintrag.JahrgangBis
	}
	if err := pruefeJahrgangsSpanne(stand.von, stand.bis); err != nil {
		return err
	}
	return pruefeMehrjahresband(stand.lernmittel, stand.mehrjahresband, stand.von, stand.bis)
}

// Bestandsangabe ist, was eine Maske zum Bestand eines vorhandenen Titels sagt: die Zahl,
// die gelten soll, und die Zahl, die sie beim Öffnen geladen hat. Gesehen ist nil bei einem
// Aufrufer, der die zweite Zahl nicht kennt (eine vor dem Update geladene Seite).
type Bestandsangabe struct {
	Soll    int
	Gesehen *int
	// BearbeiterID nennt, wer speichert. Ein kleinerer Bestand verlangt sie: Die überzähligen
	// Exemplare stehen mit der Person im Protokoll.
	BearbeiterID string
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
func (a *TitelAbfragen) setzeBestand(ctx context.Context, q repository.Datenbank, titelID string, angabe Bestandsangabe) error {
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
	if err := a.gleicheExemplareAn(ctx, q, titelID, aktuell, angabe.Soll, angabe.BearbeiterID); err != nil {
		return fmt.Errorf("exemplare konnten nicht synchronisiert werden: %w", err)
	}
	return nil
}

// zaehleBestand zählt die Exemplare im Bestand (repository.SQLExemplarImBestand), die Zahl,
// die die Maske zeigt. Bestellte Exemplare gehören dem Wareneingang und zählen nicht mit.
func zaehleBestand(ctx context.Context, q repository.Datenbank, titelID string) (int, error) {
	var bestand int
	err := q.QueryRow(ctx, `SELECT COUNT(*) FROM buecher_exemplare e WHERE e.titel_id = $1 AND `+repository.SQLExemplarImBestand, titelID).Scan(&bestand)
	if err != nil {
		return 0, fmt.Errorf("fehler beim ermitteln des aktuellen bestands: %w", err)
	}
	return bestand, nil
}

// gleicheBestandAb synchronizes the physical buecher_exemplare records to match the expected stock.
//
// q ist repository.Datenbank statt dbSchreiber, weil die Nummernvergabe (barcode_seq gegen
// Bestandsabgleich) Query braucht — pgx.Tx und der Pool erfüllen beides. bearbeiterID darf
// leer sein, solange der Bestand nicht sinkt: Das Anlegen eines Titels sondert nichts aus.
func (a *TitelAbfragen) gleicheBestandAb(ctx context.Context, q repository.Datenbank, titelID string, erwarteterBestand int, bearbeiterID string) error {
	aktuellerBestand, err := zaehleBestand(ctx, q, titelID)
	if err != nil {
		return err
	}
	return a.gleicheExemplareAn(ctx, q, titelID, aktuellerBestand, erwarteterBestand, bearbeiterID)
}

// gleicheExemplareAn legt fehlende Exemplare an oder sondert überzählige aus, bis der Bestand
// expectedStock erreicht. Ausgesondert wird nur aus dem Bestand: Bestellte Exemplare lassen
// sich über die Zahl nicht aussondern. Ein neues Exemplar erbt den Standort der vorhandenen
// (repository.SQLGeerbterStandort). Jedes ausgesonderte Exemplar steht mit bearbeiterID im
// Protokoll (repository.ProtokolliereAussonderung); ohne sie sondert der Abgleich nichts aus.
func (a *TitelAbfragen) gleicheExemplareAn(ctx context.Context, q repository.Datenbank, titelID string, aktuellerBestand, erwarteterBestand int, bearbeiterID string) error {
	if erwarteterBestand > aktuellerBestand {
		// Nummern aus barcode_seq — dieselbe Quelle wie Bestellwesen, Handvergabe und
		// Littera-Import (Migration 068: eine Quelle für alle Wege).
		barcodes, err := repository.ZieheFreieExemplarBarcodes(ctx, q, erwarteterBestand-aktuellerBestand)
		if err != nil {
			return fmt.Errorf("fehler beim generieren von exemplaren im batch: %w", err)
		}
		_, err = q.Exec(ctx, `
				INSERT INTO buecher_exemplare (titel_id, barcode_id, ist_ausleihbar, zustand_notiz, standort)
				SELECT $1, unnest($2::text[]), true, 'Automatisch generiert', `+repository.SQLGeerbterStandort("$1::uuid")+`
			`, titelID, barcodes)
		if err != nil {
			return fmt.Errorf("fehler beim generieren von exemplaren im batch: %w", err)
		}
		return nil
	}
	if erwarteterBestand == aktuellerBestand {
		return nil
	}
	if bearbeiterID == "" {
		return repository.ErrAussonderungOhneBearbeiter
	}
	auszusondern := aktuellerBestand - erwarteterBestand

	// 1. Versuchen, nicht-ausgeliehene Exemplare auszusondern
	//
	// ist_ausleihbar = false gehört dazu, wie in den anderen Aussonderungswegen
	// (repository/exemplar_aussondern.go, schaden.go, exemplar_status.go): Ein ausgesondertes
	// Exemplar, das sich weiterhin „ausleihbar" nennt, verleiht der erste Leser, der nur
	// diese Spalte prüft.
	abfrage := `
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
			RETURNING id::text
		`
	ausgesondert, err := sondereAus(ctx, q, abfrage, titelID, auszusondern)
	if err != nil {
		return fmt.Errorf("fehler beim aussondern von exemplaren: %w", err)
	}

	// 2. Fallback: Auch ausgeliehene Exemplare aussondern, falls nötig
	if len(ausgesondert) < auszusondern {
		ersatzAbfrage := `
				UPDATE buecher_exemplare
				SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = 'BESTANDSKORREKTUR', bestellstatus = NULL, letzte_bewegung_am = CURRENT_TIMESTAMP,
				    zustand_notiz = COALESCE(zustand_notiz || ' | ', '') || 'Automatisch ausgesondert (war ausgeliehen)'
				WHERE id IN (
					SELECT e.id
					FROM buecher_exemplare e
					WHERE e.titel_id = $1 AND ` + repository.SQLExemplarImBestand + `
					LIMIT $2
				)
				RETURNING id::text
			`
		rest, err := sondereAus(ctx, q, ersatzAbfrage, titelID, auszusondern-len(ausgesondert))
		if err != nil {
			return fmt.Errorf("fehler beim aussondern (fallback): %w", err)
		}
		ausgesondert = append(ausgesondert, rest...)
	}
	return repository.ProtokolliereAussonderung(ctx, q, ausgesondert, bearbeiterID, repository.AussonderungsWegBestandskorrektur, nil)
}

// sondereAus führt eine der beiden Anweisungen von gleicheExemplareAn aus und nennt die
// Exemplare, die sie ausgesondert hat.
func sondereAus(ctx context.Context, q repository.Datenbank, anweisung, titelID string, anzahl int) ([]string, error) {
	zeilen, err := q.Query(ctx, anweisung, titelID, anzahl)
	if err != nil {
		return nil, err
	}
	defer zeilen.Close()
	var ids []string
	for zeilen.Next() {
		var id string
		if err := zeilen.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, zeilen.Err()
}

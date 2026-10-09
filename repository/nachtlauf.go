package repository

// nachtlauf.go — die Anweisungen der nächtlichen Löschroutinen (jobs/). Wo eine Frist aus
// loeschfristen.go gilt, setzt die Funktion deren Bedingung selbst ein, mit der Kulanz des
// Laufs: Der Wächter in loeschrueckstand.go zählt mit derselben Bedingung, und kein Aufrufer
// kann einer Tabelle eine fremde Bedingung reichen.

import (
	"context"
	"time"
)

// AnonymisiereBearbeiterAlterAusleihen nimmt Ausleihen, die PredikatBearbeiterKennung trifft,
// die Kennungen der Bearbeiter von Ausgabe und Rückgabe, und liefert die Zahl der Zeilen.
func AnonymisiereBearbeiterAlterAusleihen(ctx context.Context, db DBQueryer) (int64, error) {
	bedingung := PredikatBearbeiterKennung(KulanzJob)
	tag, err := db.Exec(ctx, `
		UPDATE ausleihen
		SET bearbeiter_id = NULL,
		    rueckgabe_bearbeiter_id = NULL
		WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// AnonymisiereSchueler leert bei jedem Schüler, den PredikatAnonymisierung trifft, die Spalten,
// die ihn erkennbar machen, und setzt anonymized_at. barcode_id und block_reason bekommen
// feste Ersatzwerte: Ein Schüler braucht eine Ausweisnummer, die unter den Lesern einmalig
// ist, ein gesperrter einen Grund, und ein alter Freitext könnte selbst eine Person nennen.
// Dieselben Spalten leert AnonymisiereAbgaenger im Abgleich mit der LUSD; kommt eine Spalte
// dazu, die eine Person erkennbar macht, gehört sie in beide Anweisungen.
func AnonymisiereSchueler(ctx context.Context, db DBQueryer, karenzTage int) (int64, error) {
	bedingung := PredikatAnonymisierung(karenzTage, KulanzJob)
	tag, err := db.Exec(ctx, `
		UPDATE schueler
		SET vorname = left(md5(random()::text), 8),
		    nachname = 'Anonym',
		    klasse = '',
		    barcode_id = 'ANON-' || id::text,
		    geburtsdatum = NULL,
		    schul_eintritt_am = NULL,
		    lusd_id = NULL,
		    strasse = NULL,
		    hausnummer = NULL,
		    plz = NULL,
		    ort = NULL,
		    eltern_email = NULL,
		    block_reason = 'Anonymisiert (DSGVO)',
		    anonymized_at = NOW(),
		    aktualisiert_am = NOW()
		WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LoescheFotosAnonymisierterSchueler entfernt die Passfotos aller anonymisierten Schüler. Die
// Anweisung fragt nach anonymized_at und nicht nach dem Lauf dieser Nacht: So fällt auch ein
// Foto, dessen Löschen in einer früheren Nacht gescheitert ist.
func LoescheFotosAnonymisierterSchueler(ctx context.Context, db DBQueryer) error {
	_, err := db.Exec(ctx,
		"DELETE FROM schueler_fotos WHERE schueler_id IN (SELECT id FROM schueler WHERE anonymized_at IS NOT NULL)",
	)
	return err
}

// AnonymisierteSchuelerIDs nennt die Kennungen aller anonymisierten Schüler; für sie tilgt
// der Nachtlauf die Spuren in den Nebentabellen (SpurTilgungen).
func AnonymisierteSchuelerIDs(ctx context.Context, db DBQueryer) ([]string, error) {
	return leseKennungen(ctx, db, `SELECT id::text FROM schueler WHERE anonymized_at IS NOT NULL`)
}

// leseKennungen liest eine Spalte von Kennungen ganz und schließt die Zeilen, bevor der
// Aufrufer weitere Anweisungen schickt.
func leseKennungen(ctx context.Context, db DBQueryer, query string, args ...any) ([]string, error) {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// LoeschreiferAbgaenger ist ein Abgänger, den PredikatAbgaengerLoeschung zur endgültigen
// Löschung freigibt.
type LoeschreiferAbgaenger struct {
	ID            string
	Vorname       string
	Nachname      string
	Klasse        string
	BarcodeID     string
	AbgaengerJahr int
}

// LoeschreifeAbgaenger liest die Abgänger, die zum Zeitpunkt jetzt endgültig gelöscht werden
// dürfen. Die Zeilen sind gelesen und geschlossen, bevor der Aufrufer je Abgänger eine eigene
// Transaktion öffnet.
func LoeschreifeAbgaenger(ctx context.Context, db DBQueryer, jetzt time.Time) ([]LoeschreiferAbgaenger, error) {
	bedingung := PredikatAbgaengerLoeschung(jetzt)
	query := `
		SELECT id, vorname, nachname, klasse, barcode_id, abgaenger_jahr
		FROM schueler
		WHERE ` + bedingung.Where
	rows, err := db.Query(ctx, query, bedingung.Args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var abgaenger []LoeschreiferAbgaenger
	for rows.Next() {
		var a LoeschreiferAbgaenger
		if err := rows.Scan(&a.ID, &a.Vorname, &a.Nachname, &a.Klasse, &a.BarcodeID, &a.AbgaengerJahr); err != nil {
			return nil, err
		}
		abgaenger = append(abgaenger, a)
	}
	return abgaenger, rows.Err()
}

// KollegenImPapierkorb nennt die Kennungen der Kollegen, die PredikatKollegenPapierkorb zur
// endgültigen Löschung freigibt.
func KollegenImPapierkorb(ctx context.Context, db DBQueryer) ([]string, error) {
	bedingung := PredikatKollegenPapierkorb(KulanzJob)
	return leseKennungen(ctx, db, `SELECT id::text FROM leser WHERE `+bedingung.Where, bedingung.Args...)
}

// TrenneAusleihenVomLeser nimmt abgeschlossenen Ausleihen den Leser, wenn die Rückgabe länger
// als tage zurückliegt; land wählt die Bücher des Landes oder alles andere. Eine Frist von 0
// schaltet die Befristung ab: Die Anweisung läuft dann nicht.
func TrenneAusleihenVomLeser(ctx context.Context, db DBQueryer, land bool, tage int) (int64, error) {
	if tage <= 0 {
		return 0, nil
	}
	bedingung := PredikatLesehistorieAusleihen(land, tage, KulanzJob)
	query := `
		UPDATE ausleihen a
		SET schueler_id = NULL
		WHERE ` + bedingung.Where
	tag, err := db.Exec(ctx, query, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// TilgeLeserImAusleihProtokoll nimmt den Protokolleinträgen zu Ausleihe und Rückgabe nach
// derselben Frist die Kennung und den Namen des Lesers. Bliebe eines von beiden stehen, trüge
// das Protokoll die Lesehistorie weiter, die der Ausleihe genommen ist. Eine Frist von 0
// schaltet die Befristung ab.
func TilgeLeserImAusleihProtokoll(ctx context.Context, db DBQueryer, land bool, tage int) (int64, error) {
	if tage <= 0 {
		return 0, nil
	}
	bedingung := PredikatLesehistorieProtokoll(land, tage, KulanzJob)
	query := `
		UPDATE audit_log al
		SET details = al.details - 'schueler_id' - 'entleiher'
		WHERE ` + bedingung.Where
	tag, err := db.Exec(ctx, query, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// TilgeLeserInVormerkSpuren nimmt der Spur einer Vormerkung, die mit ihrem Titel gelöscht
// wurde, Kennung und Name des Lesers; die Zeile bleibt mit Titel, Zeitpunkt und Anlass. Eine
// Frist von 0 schaltet die Befristung ab.
func TilgeLeserInVormerkSpuren(ctx context.Context, db DBQueryer, tage int) (int64, error) {
	if tage <= 0 {
		return 0, nil
	}
	bedingung := PredikatLesehistorieVormerkspur(tage, KulanzJob)
	query := `
		UPDATE audit_log al
		SET details = al.details - 'schueler_id' - 'betrifft'
		WHERE ` + bedingung.Where
	tag, err := db.Exec(ctx, query, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LoescheErledigteAnliegen löscht Anliegen aus dem Portal des Kollegiums, die länger als tage
// erledigt sind. Eine Frist von 0 schaltet die Befristung ab.
func LoescheErledigteAnliegen(ctx context.Context, db DBQueryer, tage int) (int64, error) {
	if tage <= 0 {
		return 0, nil
	}
	bedingung := PredikatAnliegen(tage, KulanzJob)
	tag, err := db.Exec(ctx, `DELETE FROM lehrer_anliegen WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LoescheErledigteKlassensatzReservierungen löscht Reservierungen eines Klassensatzes, die
// länger als tage erledigt sind. Eine Frist von 0 schaltet die Befristung ab.
func LoescheErledigteKlassensatzReservierungen(ctx context.Context, db DBQueryer, tage int) (int64, error) {
	if tage <= 0 {
		return 0, nil
	}
	bedingung := PredikatKlassensatzReservierungen(tage, KulanzJob)
	tag, err := db.Exec(ctx, `DELETE FROM klassensatz_reservierungen WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LoescheQuittierteNachbuchMeldungen löscht Meldungen des Nachbuchens, die länger als tage
// quittiert sind.
func LoescheQuittierteNachbuchMeldungen(ctx context.Context, db DBQueryer, tage int) (int64, error) {
	bedingung := PredikatNachbuchMeldungen(tage, KulanzJob)
	tag, err := db.Exec(ctx, `DELETE FROM nachbuch_meldungen WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LoescheAuditLog löscht aus dem Protokoll der Datensätze, was älter ist als monate.
func LoescheAuditLog(ctx context.Context, db DBQueryer, monate int) (int64, error) {
	bedingung := PredikatAuditLog(monate, KulanzJob)
	tag, err := db.Exec(ctx, `DELETE FROM audit_log WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// LoescheAuditLogs löscht aus dem Protokoll der Verwaltung, was älter ist als monate.
func LoescheAuditLogs(ctx context.Context, db DBQueryer, monate int) (int64, error) {
	bedingung := PredikatAuditLogs(monate, KulanzJob)
	tag, err := db.Exec(ctx, `DELETE FROM audit_logs WHERE `+bedingung.Where, bedingung.Args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

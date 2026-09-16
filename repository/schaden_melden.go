package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// meldeSchaden bucht einen Verlust oder Schaden an einer Ausleihe: Exemplar aussondern,
// Forderung anlegen, Ausleihe beenden, Abholfach lösen — alles innerhalb der
// Transaktion des Aufrufers.
//
// Bis zum 15.09.2026 war das der Rumpf von ReportDamage mit eigener Transaktion. Seit
// Stufe 2 des Mahnverfahrens bucht auch der Schadensersatz-Bescheid Verluste, und zwar
// in DERSELBEN Transaktion wie Nummer und Brief: Papier und Datenbank dürfen nicht
// auseinanderlaufen (ein Brief über ein Buch, dessen Ausleihe weiterläuft, oder ein
// beendetes Buch ohne Brief). Deshalb liegt die Transaktionsgrenze beim Aufrufer.
//
// Der Aussonderungsgrund folgt der Fallgruppe: „nicht zurückgegeben" ist VERLUST,
// „beschädigt" BESCHAEDIGUNG. Bis zum 15.09.2026 stand bei beiden BESCHAEDIGUNG; das
// endgültige Löschen und die Fund-Meldung des Fehlbestandsberichts kennen aber nur
// VERLUST (OFFEN.md 5.3), und die Verlustquote (api/stats.go) zählt beide.
func meldeSchaden(ctx context.Context, tx pgx.Tx, copyID, loanID, benutzerID, beschreibung string, art SchadensArt, betrag float64) (string, error) {
	// Idempotenz + Serialisierung gegen Doppelklick: Zwei parallel abgeschickte
	// "Schaden melden"-Klicks mit derselben ausleihe_id würden sonst beide den
	// fremdeAktive-Check passieren und JE einen Schadensfall anlegen — der Schüler würde
	// für dasselbe Buch doppelt belastet. Wir sperren zuerst die Ausleihe-Zeile
	// (FOR UPDATE): der zweite Aufruf blockiert, bis der erste committet hat, und liest
	// danach den bereits angelegten Schadensfall. Existiert für diese Ausleihe schon ein
	// (nicht stornierter) Schadensfall, geben wir dessen ID idempotent zurück, statt einen
	// zweiten anzulegen.
	// Der Schuldner steht an der AUSLEIHE, nicht im Request: Ein beschädigtes Buch
	// gehört dem, der es geliehen hat. Früher übernahm der INSERT die schueler_id
	// ungeprüft aus dem Client-Body — eine falsche (vertippte oder manipulierte) ID
	// hätte den Gebührenbescheid einem unbeteiligten Schüler zugeschrieben. Wir lesen
	// sie stattdessen aus der ohnehin gesperrten Ausleihe-Zeile; das Client-Feld ist
	// nur noch Anzeige. FOR UPDATE bleibt dieselbe Sperre wie zuvor.
	//
	// *string, weil schueler_id nullable ist: Die DSGVO-Anonymisierung löst die Ausleihe
	// von der Person. Ein Scan in einen nackten string stürbe an "cannot scan NULL". Ist
	// die Ausleihe personenlos, bleibt schueler_id im Schadensfall NULL.
	var loanSchuelerID *string
	if err := tx.QueryRow(ctx,
		`SELECT schueler_id FROM ausleihen WHERE id = $1 FOR UPDATE`, loanID,
	).Scan(&loanSchuelerID); err != nil {
		return "", err // pgx.ErrNoRows: Ausleihe existiert nicht
	}

	// Ein Kollege bekommt KEINE Forderung (entschieden am 16.09.2026).
	//
	// Der Weg, den eine Forderung nimmt, endet im Schadensersatz-Bescheid, und der ist ein
	// Schreiben an Erziehungsberechtigte: Er braucht Klasse, Anschrift und die Frage der
	// Volljährigkeit (EmpfaengerFuerBescheid). Von einer Lehrkraft steht davon nichts in der
	// Akte — bewusst, denn ihre Privatanschrift gehört nicht in die Bücherei. Dazu haftet
	// eine Lehrkraft ihrem Dienstherrn nur bei Vorsatz oder grober Fahrlässigkeit; das
	// festzustellen ist Sache der Schulleitung, nicht dieser Anwendung.
	//
	// Gebucht wird trotzdem alles, was den BESTAND angeht: Das Exemplar ist weg oder kaputt
	// und wird ausgesondert, die Ausleihe endet, eine Vormerkung darauf wird gelöst. Nur die
	// Forderung entsteht nicht. Vorher entstand sie und tauchte in keiner Übersicht auf: Der
	// Reiter „Schadensersatz" liest die Sicht `schueler`, und „Bescheid erstellen" antwortete
	// „Schüler nicht gefunden". Das Geld stand offen, und niemand konnte es einziehen
	// (Rasterdurchgang 16.09.2026, OFFEN.md 5.17).
	ohneForderung := false
	if loanSchuelerID != nil {
		if err := tx.QueryRow(ctx,
			`SELECT art <> 'schueler' FROM leser WHERE id = $1`, *loanSchuelerID,
		).Scan(&ohneForderung); err != nil {
			return "", err
		}
	}

	var bestehenderSchaden string
	err := tx.QueryRow(ctx,
		`SELECT id FROM schadensfaelle WHERE ausleihe_id = $1 AND storniert_am IS NULL LIMIT 1`,
		loanID,
	).Scan(&bestehenderSchaden)
	if err == nil {
		// Schadensfall existiert bereits — idempotent zurückgeben, nichts doppelt buchen.
		return bestehenderSchaden, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	// Race-Schutz: Bleibt das Schadensformular offen, während das Buch zurückgegeben
	// und neu ausgeliehen wird, würde der "Melden"-Klick ein aktiv verliehenes Exemplar
	// aussondern. Gibt es für dieses Exemplar eine aktive Ausleihe, die NICHT die hier
	// gemeldete ist, brechen wir ab, statt die neue Ausleihe blind zu überschreiben.
	var fremdeAktive int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL AND id <> $2
	`, copyID, loanID).Scan(&fremdeAktive); err != nil {
		return "", err
	}
	if fremdeAktive > 0 {
		return "", ErrExemplarNeuVerliehen
	}

	grund := "BESCHAEDIGUNG"
	if art == SchadensArtNichtZurueck {
		grund = "VERLUST"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = $1,
		    zustand_notiz = $2, aktualisiert_am = CURRENT_TIMESTAMP,
		    letzte_bewegung_am = `+sqlStempelJetzt+`
		WHERE id = $3
	`, grund, beschreibung, copyID); err != nil {
		return "", err
	}

	// Geister-Zuteilung verhindern: Wurde dieses Exemplar bei der Rückgabe gerade einem
	// wartenden Schüler als 'abholbereit' zugewiesen und wird nun als beschädigt ausgesondert,
	// zeigt dessen Profil ein abholbereites, aber physisch defektes Buch. Die Vormerkung
	// zurück auf 'wartend' setzen und die Exemplar-Bindung lösen — der Schüler rückt damit für
	// das nächste verfügbare Exemplar wieder in die Warteschlange.
	if _, err := tx.Exec(ctx, `
		UPDATE vormerkungen
		SET status = 'wartend', bereitgestellt_exemplar_id = NULL, bereitgestellt_bis = NULL
		WHERE bereitgestellt_exemplar_id = $1 AND status = 'abholbereit'
	`, copyID); err != nil {
		return "", err
	}

	var schadensID string
	if !ohneForderung {
		if err := tx.QueryRow(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, art)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, copyID, loanID, loanSchuelerID, beschreibung, betrag, string(art)).Scan(&schadensID); err != nil {
			return "", err
		}
	}

	// `AND rueckgabe_am IS NULL`: Ohne Forderung gibt es keine Zeile, an der die Idempotenz
	// oben hängt — ein zweiter Klick verschöbe sonst das Rückgabedatum auf JETZT. Für den
	// Weg mit Forderung ändert die Bedingung nichts; dort ist schon der Schadensfall die
	// Bremse.
	if _, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = NULLIF($1, '')::uuid
		WHERE id = $2 AND rueckgabe_am IS NULL
	`, benutzerID, loanID); err != nil {
		return "", err
	}
	// Ohne Forderung ist die Kennung leer — es gibt keinen Schadensfall, auf den sie zeigen
	// könnte. Der Bescheid-Weg (bescheid_verlust.go) kommt hier nie mit einem Kollegen an:
	// Seine Liste liest die Sicht `schueler`.
	return schadensID, nil
}

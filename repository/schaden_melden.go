package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// meldeSchadenParams trägt die Angaben einer Schadensmeldung. Als Struktur, weil vier
// Kennungen und Texte (Exemplar, Ausleihe, Bearbeiter, Beschreibung) nebeneinander
// string sind: Positionsweise vertauscht, liefe die Meldung ohne Fehler auf das falsche
// Exemplar oder die falsche Ausleihe.
type meldeSchadenParams struct {
	copyID       string
	loanID       string
	benutzerID   string
	beschreibung string
	art          SchadensArt
	betrag       float64
}

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
// VERLUST (OFFEN.md 5.3), und die Verlustquote (statistik.go) zählt beide.
func meldeSchaden(ctx context.Context, tx pgx.Tx, params meldeSchadenParams) (string, error) {
	ausleihe, err := ladeSchadensAusleihe(ctx, tx, params)
	if err != nil {
		return "", err
	}
	vorhanden, erledigt, err := pruefeSchadensmeldung(ctx, tx, params, ausleihe)
	if err != nil || erledigt {
		return vorhanden, err
	}
	loanSchuelerID, exemplarID, ohneForderung := ausleihe.schuelerID, ausleihe.exemplarID, ausleihe.ohneForderung

	grund := "BESCHAEDIGUNG"
	if params.art == SchadensArtNichtZurueck {
		grund = "VERLUST"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = $1,
		    zustand_notiz = $2, aktualisiert_am = CURRENT_TIMESTAMP,
		    letzte_bewegung_am = `+sqlStempelJetzt+`
		WHERE id = $3
	`, grund, params.beschreibung, exemplarID); err != nil {
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
	`, exemplarID); err != nil {
		return "", err
	}

	var schadensID string
	if !ohneForderung {
		if err := tx.QueryRow(ctx, `
			INSERT INTO schadensfaelle (exemplar_id, ausleihe_id, schueler_id, beschreibung, betrag, art)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`, exemplarID, params.loanID, loanSchuelerID, params.beschreibung, params.betrag, string(params.art)).Scan(&schadensID); err != nil {
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
	`, params.benutzerID, params.loanID); err != nil {
		return "", err
	}
	// Ohne Forderung ist die Kennung leer — es gibt keinen Schadensfall, auf den sie zeigen
	// könnte. Der Bescheid-Weg (bescheid_verlust.go) kommt hier nie mit einem Kollegen an:
	// Seine Liste liest die Sicht `schueler`.
	return schadensID, nil
}

// schadensAusleihe ist, was eine Meldung von der gesperrten Ausleihe liest.
type schadensAusleihe struct {
	// schuelerID ist nil bei einer anonymisierten Ausleihe; der Schadensfall bleibt dann
	// ohne Person.
	schuelerID           *string
	exemplarID           string
	zurueck              bool
	exemplarAusgesondert bool
	ohneForderung        bool
}

// ladeSchadensAusleihe sperrt die Ausleihe und liest Schuldner und Exemplar von ihr. Die
// Sperre reiht zwei Klicks auf dieselbe Ausleihe hintereinander: Der zweite findet den
// Schadensfall des ersten. Schuldner und Exemplar stehen an der Ausleihe, nicht in der
// Anfrage; mit einer fremden Kennung träfe die Forderung sonst ein unbeteiligtes Kind und die
// Aussonderung ein Exemplar, das nie verliehen war. Die Kennung der Anfrage gilt nur für eine
// Ausleihe ohne Buch.
//
// Ein Kollege bekommt keine Forderung: Der Bescheid ist ein Schreiben an
// Erziehungsberechtigte, und über den Ersatz einer Lehrkraft entscheidet die Schulleitung.
// Gebucht wird trotzdem, was den Bestand angeht.
func ladeSchadensAusleihe(ctx context.Context, tx pgx.Tx, params meldeSchadenParams) (schadensAusleihe, error) {
	a := schadensAusleihe{exemplarID: params.copyID}
	var exemplarID *string
	if err := tx.QueryRow(ctx, `
		SELECT a.schueler_id, a.exemplar_id::text, a.rueckgabe_am IS NOT NULL,
		       COALESCE(e.ist_ausgesondert, false)
		FROM ausleihen a
		LEFT JOIN buecher_exemplare e ON e.id = a.exemplar_id
		WHERE a.id = $1 FOR UPDATE OF a`, params.loanID,
	).Scan(&a.schuelerID, &exemplarID, &a.zurueck, &a.exemplarAusgesondert); err != nil {
		return a, err // pgx.ErrNoRows: Ausleihe existiert nicht
	}
	if exemplarID != nil {
		a.exemplarID = *exemplarID
	}
	if a.schuelerID != nil {
		if err := tx.QueryRow(ctx,
			`SELECT art <> 'schueler' FROM leser WHERE id = $1`, *a.schuelerID,
		).Scan(&a.ohneForderung); err != nil {
			return a, err
		}
	}
	return a, nil
}

// pruefeSchadensmeldung sagt, ob die Meldung noch etwas zu buchen hat. erledigt mit der
// Kennung des vorhandenen Schadensfalls: Die Ausleihe ist schon gemeldet, nichts wird doppelt
// gebucht.
func pruefeSchadensmeldung(ctx context.Context, tx pgx.Tx, params meldeSchadenParams, a schadensAusleihe) (vorhanden string, erledigt bool, err error) {
	err = tx.QueryRow(ctx,
		`SELECT id FROM schadensfaelle WHERE ausleihe_id = $1 AND storniert_am IS NULL LIMIT 1`,
		params.loanID,
	).Scan(&vorhanden)
	if err == nil {
		return vorhanden, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, err
	}

	// Blieb der Dialog offen, während das Buch zurückkam und neu verliehen wurde, träfe die
	// Aussonderung die Ausleihe eines anderen.
	var fremdeAktive int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM ausleihen
		WHERE exemplar_id = $1 AND rueckgabe_am IS NULL AND id <> $2
	`, a.exemplarID, params.loanID).Scan(&fremdeAktive); err != nil {
		return "", false, err
	}
	if fremdeAktive > 0 {
		return "", false, ErrExemplarNeuVerliehen
	}

	// „Nicht zurückgegeben" für eine beendete Ausleihe ohne Schadensfall: Das Buch kam an einem
	// anderen Platz zurück, während die Akte offen stand, und steht im Regal. „Beschädigt
	// zurückgegeben" bleibt nach der Rückgabe möglich. Beim Kollegium gibt es keine Forderung,
	// an der die eigene frühere Meldung zu erkennen wäre; dort zeigt sie das ausgesonderte
	// Exemplar, und der zweite Klick ändert nichts.
	if a.zurueck && params.art == SchadensArtNichtZurueck {
		if a.ohneForderung && a.exemplarAusgesondert {
			return "", true, nil
		}
		return "", false, ErrAusleiheInzwischenZurueck
	}
	return "", false, nil
}

package repository

import (
	"context"
	"errors"

	"bibliothek/pkg/betrag"

	"github.com/jackc/pgx/v5"
)

// schadensMeldung trägt die Angaben einer Schadensmeldung. Eine Struktur, weil Ausleihe,
// Bearbeiter und Beschreibung nebeneinander Texte sind: In der Reihenfolge vertauscht, liefe die
// Meldung ohne Fehler auf die falsche Ausleihe. Ein Exemplar trägt sie nicht, es steht an der
// Ausleihe (ladeSchadensAusleihe).
type schadensMeldung struct {
	ausleiheID   string
	bearbeiterID string
	beschreibung string
	art          SchadensArt
	betrag       betrag.Cent
}

// meldeSchaden bucht einen Verlust oder Schaden an einer Ausleihe: Exemplar aussondern,
// Forderung anlegen, Ausleihe beenden, Abholfach lösen. Die Transaktion gehört dem Rufer, weil
// der Schadensersatz-Bescheid Verluste zusammen mit Nummer und Brief bucht: Sonst gäbe es einen
// Brief über ein Buch, dessen Ausleihe weiterläuft, oder eine beendete Ausleihe ohne Brief.
//
// Der Grund der Aussonderung folgt der Fallgruppe: „nicht zurückgegeben" ist VERLUST,
// „beschädigt" BESCHAEDIGUNG. Der Fund im Fehlbestandsbericht und das endgültige Löschen dort
// kennen nur VERLUST (inventur_verluste.go); die Verlustquote (statistik.go) zählt
// beide.
func meldeSchaden(ctx context.Context, tx pgx.Tx, meldung schadensMeldung) (string, error) {
	ausleihe, err := ladeSchadensAusleihe(ctx, tx, meldung)
	if err != nil {
		return "", err
	}
	vorhanden, erledigt, err := pruefeSchadensmeldung(ctx, tx, meldung, ausleihe)
	if err != nil || erledigt {
		return vorhanden, err
	}
	schuldnerID, exemplarID, ohneForderung := ausleihe.schuelerID, ausleihe.exemplarID, ausleihe.ohneForderung

	grund := "BESCHAEDIGUNG"
	if meldung.art == SchadensArtNichtZurueck {
		grund = "VERLUST"
	}
	if _, err := tx.Exec(ctx, `
		UPDATE buecher_exemplare
		SET ist_ausgesondert = true, ist_ausleihbar = false, aussonderung_grund = $1,
		    zustand_notiz = $2, aktualisiert_am = CURRENT_TIMESTAMP,
		    letzte_bewegung_am = `+sqlStempelJetzt+`
		WHERE id = $3
	`, grund, meldung.beschreibung, exemplarID); err != nil {
		return "", err
	}

	// Wurde das Exemplar bei der Rückgabe gerade einem Wartenden als abholbereit zugewiesen,
	// zeigte dessen Profil nach der Aussonderung ein Buch zum Abholen, das es nicht mehr gibt.
	// Die Vormerkung geht zurück auf wartend und verliert das Exemplar; der Leser rückt für das
	// nächste freie Exemplar wieder in die Warteschlange.
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
		`, exemplarID, meldung.ausleiheID, schuldnerID, meldung.beschreibung, meldung.betrag, string(meldung.art)).Scan(&schadensID); err != nil {
			return "", err
		}
	}

	// Die Bedingung rueckgabe_am IS NULL: Ohne Forderung gibt es keine Zeile, an der ein
	// zweiter Klick zu erkennen wäre, und er verschöbe das Rückgabedatum auf jetzt. Mit
	// Forderung ändert die Bedingung nichts; dort bremst schon der Schadensfall.
	if _, err := tx.Exec(ctx, `
		UPDATE ausleihen
		SET rueckgabe_am = CURRENT_TIMESTAMP, rueckgabe_bearbeiter_id = NULLIF($1, '')::uuid
		WHERE id = $2 AND rueckgabe_am IS NULL
	`, meldung.bearbeiterID, meldung.ausleiheID); err != nil {
		return "", err
	}
	// Ohne Forderung ist die Kennung leer: Es gibt keinen Schadensfall, auf den sie zeigen
	// könnte. Der Bescheid (bescheid_verlust.go) kommt hier nie mit einem Kollegen an, seine
	// Liste liest die Sicht schueler.
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
// Aussonderung ein Exemplar, das nie verliehen war. Eine Ausleihe ohne Buch ist eine
// Geräteausleihe und bekommt ErrAusleiheOhneBuch.
//
// Ein Kollege bekommt keine Forderung: Der Bescheid ist ein Schreiben an
// Erziehungsberechtigte, und über den Ersatz einer Lehrkraft entscheidet die Schulleitung.
// Gebucht wird trotzdem, was den Bestand angeht.
func ladeSchadensAusleihe(ctx context.Context, tx pgx.Tx, meldung schadensMeldung) (schadensAusleihe, error) {
	var a schadensAusleihe
	var exemplarID *string
	if err := tx.QueryRow(ctx, `
		SELECT a.schueler_id, a.exemplar_id::text, a.rueckgabe_am IS NOT NULL,
		       COALESCE(e.ist_ausgesondert, false)
		FROM ausleihen a
		LEFT JOIN buecher_exemplare e ON e.id = a.exemplar_id
		WHERE a.id = $1 FOR UPDATE OF a`, meldung.ausleiheID,
	).Scan(&a.schuelerID, &exemplarID, &a.zurueck, &a.exemplarAusgesondert); err != nil {
		return a, err // pgx.ErrNoRows: Ausleihe existiert nicht
	}
	if exemplarID == nil {
		return a, ErrAusleiheOhneBuch
	}
	a.exemplarID = *exemplarID
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
func pruefeSchadensmeldung(ctx context.Context, tx pgx.Tx, meldung schadensMeldung, a schadensAusleihe) (vorhanden string, erledigt bool, err error) {
	err = tx.QueryRow(ctx,
		`SELECT id FROM schadensfaelle WHERE ausleihe_id = $1 AND storniert_am IS NULL LIMIT 1`,
		meldung.ausleiheID,
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
	`, a.exemplarID, meldung.ausleiheID).Scan(&fremdeAktive); err != nil {
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
	if a.zurueck && meldung.art == SchadensArtNichtZurueck {
		if a.ohneForderung && a.exemplarAusgesondert {
			return "", true, nil
		}
		return "", false, ErrAusleiheInzwischenZurueck
	}
	return "", false, nil
}

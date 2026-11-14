package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Zwei Hinweise beim Ausgeben eines Schulbuchs, neben den gemischten Auflagen
// (auflagen_klasse.go): ein Buch, das sonst niemand in der Klasse hat, und ein Buch, das das
// Kind schon hat. Beide fragen nur lesend nach einer Ausleihe, die schon gebucht ist; die Theke
// zeigt sie als Zeile über dem Konto, die Ausleihe bleibt.
//
// Verglichen wird mit den Schulbüchern, die die Klasse für dasselbe Schuljahr hat, gelesen an
// der Frist. Beim Büchertausch hat die Klasse noch die alten Bücher, während die neuen ausgehen,
// und ab dem ersten Tag des Tauschs gilt jedes neue Schulbuch dem nächsten Schuljahr. Gegen
// alles verglichen, was die Klasse hat, bekäme das erste Kind mit dem neuen Buch den Hinweis,
// und ein altes Buch im neuen Stapel bliebe still.

// BuchFremdInKlasse ist der Hinweis an der Theke: Mehr als die Hälfte der Klasse hat ihre
// Schulbücher für das Schuljahr schon, und kein anderes Kind davon hat dieses Buch, in keiner
// Auflage. Klasse und Zahlen, keine Namen.
type BuchFremdInKlasse struct {
	Klasse string `json:"klasse"`
	// MitBuechern zählt die Kinder der Klasse mit Schulbüchern für das Schuljahr, das Kind der
	// Ausleihe eingeschlossen.
	MitBuechern int `json:"mit_buechern"`
	// Kinder zählt die aktiven Kinder der Klasse.
	Kinder int `json:"kinder"`
}

// FindeBuchFremdInKlasse liefert den Hinweis zur eben gebuchten Ausleihe eines Exemplars von
// titelID an schuelerID mit der Frist frist, oder nil. Klasse und Schwelle sind die der
// Klassensatz-Übersicht (LadeKlassensatzGruppen): klassen_normkey über die Sicht schueler, nur
// aktive Kinder, mehr als die Hälfte und mindestens KlassensatzMindestLeser. Dasselbe Buch ist
// auch eine andere Auflage (COALESCE(werk_id, id)); die meldet LadeAuflagenMischungInKlasse. Die
// Oberstufe fragt es nicht, dort lernen Kurse statt Klassen.
func FindeBuchFremdInKlasse(ctx context.Context, q Datenbank, titelID, schuelerID string, frist time.Time) (*BuchFremdInKlasse, error) {
	beginn := SchuljahrBeginn(frist)
	var h BuchFremdInKlasse
	var andereHabenEs bool
	err := q.QueryRow(ctx, `
		WITH kind AS (
			SELECT s.id, s.klasse, klassen_normkey(s.klasse) AS k
			FROM schueler s
			WHERE s.id = $2 AND s.deleted_at IS NULL AND s.ist_abgaenger = false
			  AND `+JahrgangSQL("s.klasse")+` BETWEEN 1 AND 10
		),
		klasse AS (
			SELECT s.id
			FROM schueler s
			JOIN kind ON klassen_normkey(s.klasse) = kind.k
			WHERE s.deleted_at IS NULL AND s.ist_abgaenger = false
		),
		schulbuecher AS (
			SELECT a.schueler_id, COALESCE(t.werk_id, t.id) AS buch
			FROM ausleihen a
			JOIN klasse ON klasse.id = a.schueler_id
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE a.rueckgabe_am IS NULL AND t.ist_lernmittel
			  AND a.rueckgabe_frist >= $3 AND a.rueckgabe_frist < $4
		)
		SELECT kind.klasse,
		       (SELECT count(*) FROM klasse)::int,
		       (SELECT count(DISTINCT schueler_id) FROM schulbuecher)::int,
		       EXISTS (SELECT 1 FROM schulbuecher sb JOIN buecher_titel t ON t.id = $1
		               WHERE sb.schueler_id <> kind.id AND sb.buch = COALESCE(t.werk_id, t.id))
		FROM kind`, titelID, schuelerID, beginn, beginn.AddDate(1, 0, 0)).
		Scan(&h.Klasse, &h.Kinder, &h.MitBuechern, &andereHabenEs)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("buch fremd in der klasse: %w", err)
	}
	if andereHabenEs || h.MitBuechern < KlassensatzMindestLeser || h.MitBuechern*2 <= h.Kinder {
		return nil, nil
	}
	return &h, nil
}

// HatBuchSchon sagt, ob schuelerID neben exemplarID ein weiteres Exemplar desselben Buchs offen
// ausgeliehen hat, auch in einer anderen Auflage (COALESCE(werk_id, id) wie in der
// Klassensatz-Übersicht).
func HatBuchSchon(ctx context.Context, q Datenbank, exemplarID, schuelerID string) (bool, error) {
	var hat bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM buecher_exemplare neu
			JOIN buecher_titel nt ON nt.id = neu.titel_id
			JOIN ausleihen a ON a.schueler_id = $2 AND a.rueckgabe_am IS NULL AND a.exemplar_id <> neu.id
			JOIN buecher_exemplare e ON e.id = a.exemplar_id
			JOIN buecher_titel t ON t.id = e.titel_id
			WHERE neu.id = $1 AND COALESCE(t.werk_id, t.id) = COALESCE(nt.werk_id, nt.id))`,
		exemplarID, schuelerID).Scan(&hat)
	if err != nil {
		return false, fmt.Errorf("buch schon ausgeliehen: %w", err)
	}
	return hat, nil
}

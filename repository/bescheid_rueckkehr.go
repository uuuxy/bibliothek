package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"bibliothek/pkg/betrag"
	"bibliothek/pkg/schulzeit"

	"github.com/jackc/pgx/v5"
)

// rueckkehrForderung ist eine offene Forderung des Exemplars samt ihrem Brief.
type rueckkehrForderung struct {
	id             string
	betrag         betrag.Cent
	bescheidStatus string
	bescheidID     string
	referenznummer string
}

// RueckkehrBefund sagt, was das Wiederauftauchen eines Exemplars ausgelöst hat.
type RueckkehrBefund struct {
	// StornierteForderungen zählt die Forderungen, die die Schule selbst beendet hat.
	StornierteForderungen int
	// StornierterBetrag ist ihre Summe, für die Meldung an der Theke.
	StornierterBetrag betrag.Cent
	// AufsichtInformieren nennt die Referenznummern der Bescheide, die schon übergeben waren.
	AufsichtInformieren []string
}

// AufsichtHinweis ist der Satz für den Fall, den das Programm nicht erledigen kann: Der
// Bescheid liegt bei der Schulaufsicht, die Forderung bleibt offen, und jemand muss dort
// anrufen. Leer, wenn nichts zu tun ist. Theke und Fehlbestandsbericht sagen denselben Satz.
func (b RueckkehrBefund) AufsichtHinweis() string {
	if len(b.AufsichtInformieren) == 0 {
		return ""
	}
	return fmt.Sprintf("Die Forderung steht auf Bescheid %s, der bereits an die Schulaufsicht übergeben wurde — sie ist unverzüglich zu informieren. Die Forderung bleibt bis dahin offen.",
		strings.Join(b.AufsichtInformieren, ", "))
}

// verbucheRueckkehr behandelt die offenen Forderungen „nicht zurückgegeben" eines Exemplars,
// das wieder aufgetaucht ist. Was geschieht, hängt daran, wo der Bescheid liegt:
//
//   - Noch bei der Schule: Die Schule storniert die Forderung selbst. Damit fallen Sperre und
//     Löschsperre, die an der offenen Forderung hängen.
//   - Schon bei der Schulaufsicht (übergeben): Die Schule storniert nichts. Der Anspruch wird
//     dort durchgesetzt, und sie ist „unverzüglich zu informieren" (Leitfaden LMF Nr. 12.7).
//     Der Bescheid bekommt dafür den Merker rueckgabe_nach_uebergabe.
//
// Bezahlte Forderungen bleiben unberührt: Wer bezahlt hat und das Buch später bringt, ist ein
// Fall für eine Erstattung, und den entscheidet ein Mensch.
//
// Die Transaktion gehört dem Rufer, wie bei MarkiereVerlustAlsGefunden: Das Zurückholen des
// Exemplars und das Ende der Forderung gehören zusammen, sonst steht das Buch wieder im Regal
// und die Forderung bleibt.
func verbucheRueckkehr(ctx context.Context, q Datenbank, exemplarID, bearbeiterID string) (RueckkehrBefund, error) {
	var befund RueckkehrBefund

	// FOR UPDATE OF f: Scannen zwei Arbeitsplätze denselben Barcode gleichzeitig, sähe der
	// zweite die Forderung sonst noch offen, und beide stornierten.
	rows, err := q.Query(ctx, `
		SELECT f.id, f.betrag, coalesce(b.status, ''), coalesce(b.id::text, ''), coalesce(b.referenznummer, '')
		FROM schadensfaelle f
		LEFT JOIN schadensersatz_bescheide b ON b.id = f.bescheid_id
		WHERE f.exemplar_id = $1
		  AND f.art = 'nicht_zurueckgegeben'
		  AND f.ist_bezahlt = false
		  AND f.storniert_am IS NULL
		FOR UPDATE OF f
	`, exemplarID)
	if err != nil {
		return befund, fmt.Errorf("offene Forderungen des Exemplars lesen: %w", err)
	}

	var offene []rueckkehrForderung
	for rows.Next() {
		var f rueckkehrForderung
		if err := rows.Scan(&f.id, &f.betrag, &f.bescheidStatus, &f.bescheidID, &f.referenznummer); err != nil {
			rows.Close()
			return befund, fmt.Errorf("offene Forderung lesen: %w", err)
		}
		offene = append(offene, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return befund, fmt.Errorf("offene Forderungen des Exemplars lesen: %w", err)
	}

	// Das Datum steht im Stornierungsgrund und damit in der Akte des Kindes und im Protokoll.
	// Deshalb schulzeit.Jetzt(): time.Now() ist im Container UTC, und eine Rückgabe zwischen 0
	// und 2 Uhr Berliner Zeit trüge das Datum des Vortags.
	grund := "Rückgabe am " + schulzeit.Jetzt().Format(datumsformat)
	for _, f := range offene {
		if f.bescheidStatus == "uebergeben" {
			if err := merkeRueckgabeNachUebergabe(ctx, q, f, bearbeiterID); err != nil {
				return befund, err
			}
			befund.AufsichtInformieren = append(befund.AufsichtInformieren, f.referenznummer)
			continue
		}
		if err := storniereWeilZurueck(ctx, q, f.id, f.betrag, bearbeiterID, grund); err != nil {
			return befund, err
		}
		befund.StornierteForderungen++
		befund.StornierterBetrag += f.betrag
	}
	return befund, nil
}

// storniereWeilZurueck beendet die Forderung wie StorniereForderung: ist_bezahlt wird mit
// gesetzt, weil alle Filter auf „offen" daran hängen, und storniert_am unterscheidet sie von
// einer Zahlung. Der Eintrag trägt dieselbe Aktion STORNIERUNG, damit das Protokoll eine Form
// kennt.
func storniereWeilZurueck(ctx context.Context, q Datenbank, schadensfallID string, hoehe betrag.Cent, bearbeiterID, grund string) error {
	var leserID *string
	err := q.QueryRow(ctx, `
		UPDATE schadensfaelle
		SET ist_bezahlt = true, storniert_am = NOW(), storniert_von = $1,
		    stornierungsgrund = $2, aktualisiert_am = NOW()
		WHERE id = $3 AND storniert_am IS NULL
		RETURNING schueler_id::text
	`, bearbeiterID, grund, schadensfallID).Scan(&leserID)
	// Keine Zeile: Die Forderung ist zwischen Lesen und Schreiben verschwunden oder wurde
	// anderswo storniert. Die Theke meldete sonst „Forderung storniert" über eine Forderung,
	// die weiter offen steht.
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("forderung %s ließ sich nicht stornieren — sie ist nicht mehr offen", schadensfallID)
	}
	if err != nil {
		return fmt.Errorf("forderung stornieren: %w", err)
	}
	// Dieselben Schlüssel wie StorniereForderung; die Kennung des Lesers ist der Griff der Tilgung.
	details := map[string]any{"betrag": hoehe, "grund": grund, "anlass": "rueckgabe"}
	if leserID != nil {
		details["schueler_id"] = *leserID
	}
	kontext := "Forderung storniert: das Buch ist zurück"
	return schreibeProtokollEintrag(ctx, q, protokollEintrag{
		Tabelle: "schadensfaelle", Aktion: "STORNIERUNG", DatensatzID: schadensfallID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: details,
	})
}

// merkeRueckgabeNachUebergabe setzt den Merker am übergebenen Bescheid. Die Forderung bleibt,
// wie sie ist: Über sie entscheidet nicht mehr die Schule.
func merkeRueckgabeNachUebergabe(ctx context.Context, q Datenbank, f rueckkehrForderung, bearbeiterID string) error {
	tag, err := q.Exec(ctx,
		`UPDATE schadensersatz_bescheide SET rueckgabe_nach_uebergabe = true WHERE id = $1`,
		f.bescheidID)
	if err != nil {
		return fmt.Errorf("rückgabe nach übergabe vermerken: %w", err)
	}
	// 0 Zeilen: Der Bescheid ist weg. Den Hinweis an der Theke trotzdem auszugeben hieße,
	// das Sekretariat auf einen Merker zu verweisen, den niemand sehen wird.
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("bescheid %s ließ sich nicht als Rückgabe nach Übergabe vermerken", f.bescheidID)
	}
	kontext := "Rückgabe nach Übergabe — die Schulaufsicht ist unverzüglich zu informieren"
	return schreibeProtokollEintrag(ctx, q, protokollEintrag{
		Tabelle: "schadensersatz_bescheide", Aktion: "UPDATE", DatensatzID: f.bescheidID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{"referenznummer": f.referenznummer, "schadensfall_id": f.id},
	})
}

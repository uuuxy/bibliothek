package repository

import (
	"context"
	"fmt"
	"time"
)

// Das abgeschriebene Buch kommt zurück (#597, Etappe 2).
//
// Eine Forderung „nicht zurückgegeben" ist eine Wette darauf, dass das Buch weg bleibt.
// Legt das Kind es später doch auf die Theke, endet die Wette — und zwar verschieden, je
// nachdem, wo der Bescheid gerade liegt:
//
//   - Noch bei der Schule (offen): Die Schule storniert die Forderung selbst. Damit
//     fallen Sperre und Löschblockade, die an der offenen Forderung hängen.
//   - Schon bei der Schulaufsicht (übergeben): Die Schule storniert NICHTS. Der Anspruch
//     wird dort durchgesetzt; sie ist „unverzüglich zu informieren" (Leitfaden LMF
//     Nr. 12.7). Dafür trägt der Bescheid `rueckgabe_nach_uebergabe` — das Sekretariat
//     sieht ihn in der Liste, die Theke bekommt den Satz sofort.
//
// Bezahlte Forderungen bleiben unberührt: Wer bezahlt hat und das Buch später bringt,
// ist ein Erstattungsfall und keine Stornierung — den entscheidet ein Mensch.

// rueckkehrForderung ist eine offene Forderung des Exemplars samt ihrem Brief.
type rueckkehrForderung struct {
	id             string
	betrag         float64
	bescheidStatus string
	bescheidID     string
	referenznummer string
}

// RueckkehrBefund sagt, was das Wiederauftauchen eines Exemplars ausgelöst hat.
type RueckkehrBefund struct {
	// StornierteForderungen: Anzahl der Forderungen, die die Schule selbst beendet hat.
	StornierteForderungen int
	// StornierterBetrag: ihre Summe, für die Meldung an der Theke.
	StornierterBetrag float64
	// AufsichtInformieren: Referenznummern der Bescheide, die schon übergeben waren.
	AufsichtInformieren []string
}

// VerbucheRueckkehr behandelt die offenen „nicht zurückgegeben"-Forderungen eines
// Exemplars, das wieder aufgetaucht ist.
//
// Die Transaktionsgrenze liegt beim Aufrufer (wie MarkiereVerlustAlsGefunden): Das
// Zurückholen des Exemplars und das Ende der Forderung gehören in EINE Transaktion —
// sonst steht das Buch wieder im Regal und die Forderung bleibt, oder umgekehrt.
func VerbucheRueckkehr(ctx context.Context, q DBQueryer, exemplarID, bearbeiterID string) (RueckkehrBefund, error) {
	var befund RueckkehrBefund

	// FOR UPDATE OF f: Zwei Arbeitsplätze, die denselben Barcode gleichzeitig scannen,
	// stornieren sonst beide — der zweite sieht die Forderung noch offen.
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

	grund := "Rückgabe am " + time.Now().Format("02.01.2006")
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

// storniereWeilZurueck beendet die Forderung wie StornierungGebuehr: ist_bezahlt mit
// gesetzt (alle Offen-Filter hängen daran), storniert_am unterscheidet sie von einer
// echten Zahlung. Dieselbe Audit-Aktion STORNIERUNG, damit die Revision EINE Form kennt.
func storniereWeilZurueck(ctx context.Context, q DBQueryer, schadensfallID string, betrag float64, bearbeiterID, grund string) error {
	tag, err := q.Exec(ctx, `
		UPDATE schadensfaelle
		SET ist_bezahlt = true, storniert_am = NOW(), storniert_von = $1,
		    stornierungsgrund = $2, aktualisiert_am = NOW()
		WHERE id = $3 AND storniert_am IS NULL
	`, bearbeiterID, grund, schadensfallID)
	if err != nil {
		return fmt.Errorf("forderung stornieren: %w", err)
	}
	// 0 Zeilen: Die Forderung ist zwischen Lesen und Schreiben verschwunden oder wurde
	// anderswo storniert. Ohne diese Prüfung meldete die Theke „Forderung storniert" über
	// eine Forderung, die weiter offen steht (Phantom-Erfolg-Sweep 31.08.2026).
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("forderung %s ließ sich nicht stornieren — sie ist nicht mehr offen", schadensfallID)
	}
	kontext := "Forderung storniert: das Buch ist zurück"
	return schreibeAuditLog(ctx, q, auditEntry{
		Tabelle: "schadensfaelle", Aktion: "STORNIERUNG", DatensatzID: schadensfallID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{"betrag": betrag, "grund": grund, "anlass": "rueckgabe"},
	})
}

// merkeRueckgabeNachUebergabe setzt den Merker am übergebenen Bescheid. Die Forderung
// bleibt, wie sie ist — über sie entscheidet nicht mehr die Schule.
func merkeRueckgabeNachUebergabe(ctx context.Context, q DBQueryer, f rueckkehrForderung, bearbeiterID string) error {
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
	return schreibeAuditLog(ctx, q, auditEntry{
		Tabelle: "schadensersatz_bescheide", Aktion: "UPDATE", DatensatzID: f.bescheidID,
		BearbeiterID: &bearbeiterID, Akteur: "USER", Kontext: &kontext,
		Details: map[string]any{"referenznummer": f.referenznummer, "schadensfall_id": f.id},
	})
}

package service

import (
	"context"
	"fmt"

	"bibliothek/repository"
)

// Die Sperren einer neuen Ausleihe — EIN Prüfweg für Buch und Gerät (docs/OFFEN.md 4.4,
// entschieden am 24.09.2026). Bis dahin gab es zwei, und sie widersprachen sich: Das Buch
// prüfte nur Schüler und ließ jede Sperre übergehen, das Gerät prüfte jeden Leser und ließ
// keine übergehen.
//
// Die Regel richtet sich danach, WER gesperrt hat:
//
//   - Eine Sperre am Leser — von Hand (is_manually_blocked) oder die, die das Programm den
//     Ehemaligen setzt (ist_gesperrt: Versetzung, LUSD-Import) — lässt an der Theke nur die
//     Rückgabe zu. Aufgehoben wird sie in der Akte (PATCH /api/admin/students/{id}/lock).
//     So hält es Littera: „Eine Rückgabe von Medien ist auch bei gesperrten Lesern möglich!
//     Für eine Ausleihe muss jedoch die Sperre zunächst in den Leserstammdaten aufgehoben
//     werden!"
//   - Eine offene Forderung und die Überfällig-Automatik sind Hinweise des Programms. Wer
//     Schülerdaten ändern darf, übergeht sie (override_block wirkt nur mit edit_students,
//     api/action.go); das Protokoll hält es fest. Gezählt wird jede offene Forderung, gleich
//     aus welchem Topf — ein Konto je Leser wie in Littera.
//   - Beim Lernmittel gilt nur die Sperre von Hand, die Entscheidung eines Menschen. Die
//     Schule am 22.09.2026: „Für die Lernmittel darf es keinerlei automatische ‚Sperrung'
//     geben, auch nicht eine Sperrung, die bestimmte Personen aufheben können."
//   - Ein Kollege wird nie gesperrt (entschieden am 16.09.2026, bestätigt am 24.09.2026). Er
//     hat keine Frist und bekommt keine Forderung; eine Sperre an seinem Konto zählt nicht.
//   - Ein anonymisierter Datensatz ist keine Person mehr: Er bekommt nichts, auch kein
//     Lernmittel, und die Sperre lässt sich nicht aufheben.

// sperrLage ist das Ergebnis der Prüfung, wenn die Ausleihe entstehen darf.
type sperrLage struct {
	// uebergangen nennt die übergangenen Hinweise. Ins Protokoll kommen sie erst, wenn die
	// Ausleihe entstanden ist (protokolliereUebergangen).
	uebergangen []string
	// einst sind die Einstellungen, die die Überfällig-Automatik gelesen hat; der
	// Geräte-Pfad rechnet daraus die Frist. nil, wenn die Automatik nicht lief.
	einst *SystemEinstellungen
}

// pruefeAusleihSperren prüft die Sperren für eine NEUE Ausleihe an leser. Eine Rückgabe
// prüft sie nie: Wer gesperrt ist, muss zurückgeben können. q trägt die Abfragen — am Scan
// der Pool, beim Nachbuchen die Transaktion des Eintrags. uebergehen ist override_block.
func pruefeAusleihSperren(ctx context.Context, q repository.DBQueryer, leser *repository.Student, lernmittel, uebergehen bool) (sperrLage, error) {
	if istKollegium(leser) {
		return sperrLage{}, nil
	}
	if err := pruefeSperreAmLeser(leser, lernmittel); err != nil {
		return sperrLage{}, err
	}
	if lernmittel {
		return sperrLage{}, nil
	}
	return pruefeHinweise(ctx, q, leser.ID, uebergehen)
}

// istKollegium: Lehrkraft oder LiV. Leer heißt Schüler — so liest es die Sicht `schueler`
// (repository.Student.Art); dieselbe Regel wie leserArt.js im Browser.
func istKollegium(leser *repository.Student) bool {
	return leser.Art != "" && leser.Art != "schueler"
}

// pruefeSperreAmLeser prüft die zwei Schalter am Leser. Ihr Merkmal (SperreAmLeser) sagt der
// Theke, dass sie nichts übergehen, sondern die Sperre aufheben lässt. Der Grund steht im
// SperrGrundFehler; wer view_students nicht hat, bekommt ihn nicht (api.ohneSperrgrund).
func pruefeSperreAmLeser(leser *repository.Student, lernmittel bool) error {
	if leser.IstAnonymisiert {
		// Ohne Merkmal: Aufheben kann man diese Sperre nicht (api/student_lock.go lehnt ab).
		return fmt.Errorf("%w: der Datensatz ist anonymisiert", ErrBlocked)
	}
	if leser.IsManuallyBlocked {
		return &SperrGrundFehler{
			Kern:  SperreAmLeser(fmt.Errorf("%w: Manuelle Sperre", ErrBlocked)),
			Grund: sperrGrund(leser, "ohne Grund"),
		}
	}
	if leser.IstGesperrt && !lernmittel {
		// block_reason ist bei dieser Sperre garantiert gefüllt (chk_schueler_block_reason);
		// der Ersatztext greift nur bei Altbeständen.
		return &SperrGrundFehler{
			Kern:  SperreAmLeser(ErrBlocked),
			Grund: sperrGrund(leser, "Grund nicht erfasst"),
		}
	}
	return nil
}

// sperrGrund ist der Grund am Leser oder, wenn keiner da ist, ersatz.
func sperrGrund(leser *repository.Student, ersatz string) string {
	if leser.BlockReason != nil && *leser.BlockReason != "" {
		return *leser.BlockReason
	}
	return ersatz
}

// pruefeHinweise prüft die zwei Hinweise des Programms: unbezahlte Forderungen und die
// Überfällig-Automatik (ab MaxOverdueItems Medien, die länger als MaxOverdueDays überfällig
// sind). Mit uebergehen gehen sie durch und stehen in sperrLage.uebergangen.
func pruefeHinweise(ctx context.Context, q repository.DBQueryer, leserID string, uebergehen bool) (sperrLage, error) {
	var lage sperrLage

	offen, err := zaehleOffeneSchaeden(ctx, q, leserID)
	if err != nil {
		return sperrLage{}, err
	}
	if offen > 0 {
		if !uebergehen {
			return sperrLage{}, UebergehbareSperre(fmt.Errorf("%w: %d unbezahlte(r) Schadensfall/-fälle offen", ErrBlocked, offen))
		}
		lage.uebergangen = append(lage.uebergangen, fmt.Sprintf("Ausleihsperre manuell ignoriert (unbezahlte Schäden: %d)", offen))
	}

	einst, err := ladeSystemEinstellungen(ctx, q)
	if err != nil {
		return sperrLage{}, err
	}
	lage.einst = einst
	ueberfaellig, err := zaehleUeberfaelligeMedien(ctx, q, leserID, einst.MaxOverdueDays)
	if err != nil {
		return sperrLage{}, err
	}
	if ueberfaellig >= einst.MaxOverdueItems {
		if !uebergehen {
			return sperrLage{}, UebergehbareSperre(fmt.Errorf("%w: %d überfällige Medien vorhanden (Sperr-Automatik)", ErrBlocked, ueberfaellig))
		}
		lage.uebergangen = append(lage.uebergangen, fmt.Sprintf("Ausleihsperre manuell ignoriert (überfällig: %d Medien)", ueberfaellig))
	}
	return lage, nil
}

// protokolliereUebergangen schreibt je übergangenen Hinweis einen Eintrag (OVERRIDE_BLOCK).
// Aufgerufen wird sie, wenn die Ausleihe entstanden ist, nicht schon bei der Prüfung: Am
// Gerät unterbricht die Zubehör-Liste zwischen beidem, und beim Buch können danach noch
// Ausleihlimit und Vormerkung abweisen — dann ist nichts übergangen worden.
func protokolliereUebergangen(ctx context.Context, audit repository.AuditRepository, staffID, leserID string, uebergangen []string) {
	for _, grund := range uebergangen {
		logAuditErr(overrideBlockAction, audit.LogAdminAktion(ctx, staffID, "OVERRIDE_BLOCK", "", map[string]any{
			"schueler_id": leserID,
			"reason":      grund,
		}))
	}
}

// zaehleOffeneSchaeden zählt die unbezahlten, nicht stornierten Schadensfälle eines Lesers
// — aus jedem Topf. storniert_am setzt ist_bezahlt=true, daher genügt die Flag-Prüfung.
func zaehleOffeneSchaeden(ctx context.Context, q repository.DBQueryer, schuelerID string) (int, error) {
	var n int
	err := q.QueryRow(ctx,
		`SELECT COUNT(*) FROM schadensfaelle WHERE schueler_id = $1 AND ist_bezahlt = false`,
		schuelerID).Scan(&n)
	return n, err
}

// zaehleUeberfaelligeMedien zählt die überfälligen (älter als maxOverdueDays), noch nicht
// zurückgegebenen BUCH-Medien eines Lesers (Dauerleihen und Geräte ausgenommen).
func zaehleUeberfaelligeMedien(ctx context.Context, q repository.DBQueryer, schuelerID string, maxOverdueDays int) (int, error) {
	var n int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ausleihen
		WHERE schueler_id = $1
		  AND rueckgabe_am IS NULL
		  AND rueckgabe_frist < CURRENT_TIMESTAMP - (INTERVAL '1 day' * $2)
		  AND ist_handapparat = false
		  AND geraet_id IS NULL
	`, schuelerID, maxOverdueDays).Scan(&n)
	return n, err
}

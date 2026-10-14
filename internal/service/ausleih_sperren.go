package service

import (
	"context"
	"fmt"

	"bibliothek/pkg/leserart"
	"bibliothek/repository"
)

// Die Sperren einer neuen Ausleihe: ein Prüfweg für Buch, Gerät und Nachbuchen, damit alle drei
// derselben Regel folgen. Sie richtet sich danach, wer gesperrt hat:
//
//   - Eine Sperre am Leser, von Hand (is_manually_blocked) oder vom Programm für die Ehemaligen
//     (ist_gesperrt: Versetzung, LUSD-Import), lässt an der Theke nur die Rückgabe zu.
//     Aufgehoben wird sie in der Akte (PATCH /api/admin/students/{id}/lock). So hält es
//     Littera: „Eine Rückgabe von Medien ist auch bei gesperrten Lesern möglich! Für eine
//     Ausleihe muss jedoch die Sperre zunächst in den Leserstammdaten aufgehoben werden!"
//   - Eine offene Forderung und die Überfällig-Automatik sind Hinweise des Programms. Wer
//     Schülerdaten ändern darf, übergeht sie (override_block wirkt nur mit edit_students,
//     api/theke/eingabe.go); das Protokoll hält es fest. Gezählt wird jede offene Forderung,
//     gleich aus welchem Topf: ein Konto je Leser wie in Littera.
//   - Beim Lernmittel gilt nur die Sperre von Hand, die Entscheidung eines Menschen. Die
//     Schule verlangt: „Für die Lernmittel darf es keinerlei automatische ‚Sperrung' geben,
//     auch nicht eine Sperrung, die bestimmte Personen aufheben können."
//   - Ein Kollege wird nie gesperrt. Er hat keine Frist und bekommt keine Forderung; eine
//     Sperre an seinem Konto zählt nicht.
//   - Ein anonymisierter Datensatz ist keine Person mehr: Er bekommt nichts, auch kein
//     Lernmittel, und die Sperre lässt sich nicht aufheben.

// sperrLage ist das Ergebnis der Prüfung, wenn die Ausleihe entstehen darf.
type sperrLage struct {
	// uebergangen nennt die übergangenen Hinweise. Ins Protokoll kommen sie erst, wenn die
	// Ausleihe entstanden ist (protokolliereUebergangen).
	uebergangen []string
	// einst sind die Einstellungen, die die Überfällig-Automatik gelesen hat; der
	// Geräte-Pfad rechnet daraus die Frist. nil, wenn die Automatik nicht lief.
	einst *ausleihEinstellungen
}

// pruefeAusleihSperren prüft die Sperren für eine neue Ausleihe an leser. Eine Rückgabe
// prüft sie nie: Wer gesperrt ist, muss zurückgeben können. q trägt die Abfragen — am Scan
// der Pool, beim Nachbuchen die Transaktion des Eintrags. uebergehen ist override_block.
func pruefeAusleihSperren(ctx context.Context, q repository.Datenbank, leser *repository.Leser, lernmittel, uebergehen bool) (sperrLage, error) {
	if err := PruefeSperreAmLeser(leser, lernmittel); err != nil {
		return sperrLage{}, err
	}
	if istKollegium(leser) || lernmittel {
		return sperrLage{}, nil
	}
	return pruefeHinweise(ctx, q, leser.ID, uebergehen)
}

// PruefeSperreAmLeser ist der Teil der Regel, den auch die Verlängerung fragt
// (Einzelverlängerung und Frist von Hand, api/theke/ausleihe.go): Hält eine Sperre am Leser
// diese Ausleihe an? Beim Schulbuch nur die von Hand, beim Kollegen keine; der anonymisierte
// Datensatz bekommt nichts. Der Satz der Schule zum Lernmittel gilt auch für die Ehemaligen und
// bei der Verlängerung; die Hinweise (Forderung, überfällig) halten eine Verlängerung nicht an.
func PruefeSperreAmLeser(leser *repository.Leser, lernmittel bool) error {
	if istKollegium(leser) {
		return nil
	}
	return pruefeSperrschalter(leser, lernmittel)
}

// istKollegium sagt, ob der Leser zum Kollegium gehört: jede Art außer Schüler, leer heißt
// Schüler (leserart.IstKollegium).
func istKollegium(leser *repository.Leser) bool {
	return leserart.IstKollegium(leser.Art)
}

// pruefeSperrschalter prüft die zwei Schalter am Leser. Ihr Merkmal (SperreAmLeser) sagt der
// Theke, dass sie nichts übergehen, sondern die Sperre aufheben lässt. Der Grund steht im
// SperrgrundFehler; wer view_students nicht hat, bekommt ihn nicht (ohneSperrgrund in
// api/theke/eingabe.go).
func pruefeSperrschalter(leser *repository.Leser, lernmittel bool) error {
	if leser.IstAnonymisiert {
		// Ohne Merkmal: Aufheben kann man diese Sperre nicht (api/leser/student_lock.go lehnt ab).
		return fmt.Errorf("%w: der Datensatz ist anonymisiert", ErrGesperrt)
	}
	if leser.IstVonHandGesperrt {
		return &SperrgrundFehler{
			Kern:  SperreAmLeser(fmt.Errorf("%w: Manuelle Sperre", ErrGesperrt)),
			Grund: sperrgrund(leser, "ohne Grund"),
		}
	}
	if leser.IstGesperrt && !lernmittel {
		// block_reason ist bei dieser Sperre garantiert gefüllt (chk_schueler_block_reason);
		// der Ersatztext greift nur bei Altbeständen.
		return &SperrgrundFehler{
			Kern:  SperreAmLeser(ErrGesperrt),
			Grund: sperrgrund(leser, "Grund nicht erfasst"),
		}
	}
	return nil
}

// sperrgrund ist der Grund am Leser oder, wenn keiner da ist, ersatz.
func sperrgrund(leser *repository.Leser, ersatz string) string {
	if leser.Sperrgrund != nil && *leser.Sperrgrund != "" {
		return *leser.Sperrgrund
	}
	return ersatz
}

// pruefeHinweise prüft die zwei Hinweise des Programms: unbezahlte Forderungen und die
// Überfällig-Automatik (ab UeberfaelligSperreAbMedien Medien, die länger als
// UeberfaelligKarenzTage überfällig sind). Mit uebergehen gehen sie durch und stehen in
// sperrLage.uebergangen.
func pruefeHinweise(ctx context.Context, q repository.Datenbank, leserID string, uebergehen bool) (sperrLage, error) {
	var lage sperrLage

	offen, _, err := repository.ZaehleOffeneSchaeden(ctx, q, leserID)
	if err != nil {
		return sperrLage{}, err
	}
	if offen > 0 {
		if !uebergehen {
			return sperrLage{}, UebergehbareSperre(fmt.Errorf("%w: %d unbezahlte(r) Schadensfall/-fälle offen", ErrGesperrt, offen))
		}
		lage.uebergangen = append(lage.uebergangen, fmt.Sprintf("Ausleihsperre manuell ignoriert (unbezahlte Schäden: %d)", offen))
	}

	einst, err := ladeAusleihEinstellungen(ctx, q)
	if err != nil {
		return sperrLage{}, err
	}
	lage.einst = einst
	ueberfaellig, err := repository.ZaehleUeberfaelligeAusleihen(ctx, q, leserID, einst.UeberfaelligKarenzTage)
	if err != nil {
		return sperrLage{}, err
	}
	if ueberfaellig >= einst.UeberfaelligSperreAbMedien {
		if !uebergehen {
			return sperrLage{}, UebergehbareSperre(fmt.Errorf("%w: %d überfällige Medien vorhanden (Sperr-Automatik)", ErrGesperrt, ueberfaellig))
		}
		lage.uebergangen = append(lage.uebergangen, fmt.Sprintf("Ausleihsperre manuell ignoriert (überfällig: %d Medien)", ueberfaellig))
	}
	return lage, nil
}

// aktionUebergehen nennt im Log den Eintrag, der ein Übergehen festhalten sollte.
const aktionUebergehen = "override-block"

// protokolliereUebergangen schreibt je übergangenen Hinweis einen Eintrag (OVERRIDE_BLOCK).
// Aufgerufen wird sie, wenn die Ausleihe entstanden ist, nicht schon bei der Prüfung: Am
// Gerät unterbricht die Zubehör-Liste zwischen beidem, und beim Buch können danach noch
// Ausleihlimit und Vormerkung abweisen — dann ist nichts übergangen worden.
func protokolliereUebergangen(ctx context.Context, protokoll repository.ProtokollAbfragen, bearbeiterID, leserID string, uebergangen []string) {
	for _, grund := range uebergangen {
		meldeProtokollFehler(aktionUebergehen, protokoll.ProtokolliereAdminAktion(ctx, bearbeiterID, "OVERRIDE_BLOCK", "", map[string]any{
			"schueler_id": leserID,
			"reason":      grund,
		}))
	}
}

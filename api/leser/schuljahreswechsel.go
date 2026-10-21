package leser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/db"
	"bibliothek/pkg/clientip"
	"bibliothek/pkg/schulzeit"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5"
)

// schuljahreswechselAntwort sind die Zahlen des Schuljahreswechsels. Bei Probelauf wurde
// nichts geschrieben; die Zahlen sind die genaue Vorschau (dieselben Abfragen, die
// Transaktion wird zurückgerollt).
type schuljahreswechselAntwort struct {
	Versetzt   int  `json:"promoted_count"`
	Archiviert int  `json:"archived_count"`
	Probelauf  bool `json:"dry_run"`
	// Klassenlehrer-Zuordnungen, die mit versetzt oder als Abschlussklasse entfernt wurden.
	// Bliebe die Zuordnung beim Jahreswechsel auf dem alten Klassennamen stehen, ginge die
	// Mahnliste der neuen 6a an niemanden, ohne Fehlermeldung.
	ZuordnungenVersetzt int `json:"mapping_versetzt"`
	ZuordnungenEntfernt int `json:"mapping_entfernt"`
	// Zuordnungen, deren Zielname schon belegt war (z. B. '9a' und '09a' laufen
	// beide auf '10a'): Sie bleiben unverändert stehen und werden hier genannt,
	// statt den ganzen Lauf scheitern zu lassen.
	Zuordnungskonflikte []string `json:"mapping_konflikte,omitempty"`
	// Klassen, die mit keinem Jahrgang von 1 bis 13 beginnen („14a", eine Buchnummer): Ihre
	// Schüler bleiben, wo sie sind, und die Vorschau nennt sie.
	KlassenAusserhalb []string `json:"klassen_ausserhalb,omitempty"`
}

// schuljahreswechselAnfrage verlangt eine ausdrückliche Bestätigung im Rumpf: eine Sicherung
// am Server gegen versehentliche oder automatische Aufrufe (etwa eine Wiederholung); bei einem
// unumkehrbaren Lauf dieser Tragweite reicht eine Bestätigung in der Oberfläche nicht.
// Probelauf führt dieselbe Berechnung in einer Transaktion aus und rollt sie zurück: eine
// genaue Vorschau ohne Wirkung, auch ohne Bestätigung.
type schuljahreswechselAnfrage struct {
	Bestaetigt bool `json:"confirm"`
	Probelauf  bool `json:"dry_run"`
}

// Schuljahreswechsel führt den automatischen Schuljahreswechsel durch.
func (t *Tueren) Schuljahreswechsel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		claims, ok := auth.ClaimsAusKontext(ctx)
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}

		anfrage, ok := leseSchuljahreswechselAnfrage(w, r)
		if !ok {
			return
		}

		antwort, ok := t.fuehreSchuljahreswechselAus(ctx, w, r, anfrage, claims.KontoID)
		if !ok {
			return
		}
		antwort.Probelauf = anfrage.Probelauf
		tuer.AntworteJSON(w, http.StatusOK, antwort)
	}
}

// fuehreSchuljahreswechselAus wickelt den Lauf der Versetzung in einer einzigen strikten
// Transaktion ab: Entweder wird jeder Schüler versetzt oder archiviert, oder bei jedem Fehler
// keiner (db.Verwerfe greift auf jedem Fehler- und Panic-Pfad). Im Probelauf wird bewusst
// nicht committet (dieselbe Berechnung, ohne Wirkung). ok=false: die Fehlerantwort wurde
// bereits geschrieben.
func (t *Tueren) fuehreSchuljahreswechselAus(ctx context.Context, w http.ResponseWriter, r *http.Request, anfrage schuljahreswechselAnfrage, kontoID string) (antwort schuljahreswechselAntwort, ok bool) {
	tx, err := t.DB.Pool.Begin(ctx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return antwort, false
	}
	defer db.Verwerfe(ctx, tx)

	if !anfrage.Probelauf {
		// Harte Serialisierung gegen den Doppellauf. Der COUNT-Check unten allein ist
		// TOCTOU: Zwei gleichzeitige Klicks (oder zwei Admins) lesen beide 0, bevor einer
		// committet — und versetzen die ganze Schule ein zweites Mal (+2 Stufen,
		// irreversibel). Der Advisory-Lock (transaktionsgebunden, gibt beim Commit/
		// Rollback automatisch frei) zwingt den zweiten Lauf zu warten, bis der erste
		// fertig ist; danach sieht dessen COUNT den Protokolleintrag und antwortet 409.
		if err := repository.SperreSchuljahreswechsel(ctx, tx); err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return antwort, false
		}
		if !pruefeDoppellaufSchutz(ctx, tx, w) {
			return antwort, false
		}
	}

	if antwort.KlassenAusserhalb, err = repository.LadeKlassenAusserhalbDerVersetzung(ctx, tx); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return antwort, false
	}
	if antwort.Versetzt, antwort.Archiviert, err = repository.VersetzeSchueler(ctx, tx, schulzeit.Zone().String()); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return antwort, false
	}

	// Die Klassenlehrer-Zuordnung wandert mit der Kohorte (dieselbe Transaktion,
	// dieselbe Rechenregel). Die Klassen-Bücherlisten (class_books) bleiben
	// bewusst unangetastet: Sie hängen am Stufen-Namen ("die 8a liest Markl 2"
	// gilt für jede künftige 8a), nicht an den Menschen; zwei Bedeutungen von
	// "Klasse", die der Jahreswechsel nicht verwechseln darf.
	if err := versetzeKlassenlehrerZuordnung(ctx, tx, &antwort); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return antwort, false
	}

	if !anfrage.Probelauf {
		if !t.finalisiereSchuljahreswechsel(ctx, tx, w, r, kontoID, antwort.Versetzt, antwort.Archiviert) {
			return antwort, false
		}
	}
	return antwort, true
}

// versetzeKlassenlehrerZuordnung zählt die Klassennamen der Lehrer-Zuordnung mit
// derselben Regel hoch wie repository.VersetzeSchueler die Schüler. Absteigend nach
// Stufe, damit '9a' erst nach dem Wegzug von '10a' auf den freien Namen rücken
// kann; Abschlussklassen-Zeilen werden entfernt (die Kohorte verlässt die
// Schule). Ebenso die Zeilen der Jahrgänge 6 und 10: Nach
// der 6 und nach der 10 bildet die Schule die Klassen neu — aus der Förderstufe
// werden die Zweige, aus der 10 die Oberstufe —, und die neuen Klassen bekommen
// neue Klassenleitungen. Hochgezählt landete die Zuordnung auf „07F1" oder
// „11G1", Klassen, die es an der Schule nicht gibt, mit der alten Lehrkraft.
// Der Konflikt-Zweig (Zielname belegt → Zeile bleibt stehen und wird
// gemeldet) ist seit dem Klassen-Vokabular (Migration 079) Rückfallebene: Die
// klassische Ursache — '9a' und '09a' nebeneinander, beide → '10a' — kann durch
// die Kanonisierung nicht mehr entstehen. Er bleibt, damit ein Namenskonflikt
// den Schuljahreswechsel auch künftig nie scheitern lässt.
func versetzeKlassenlehrerZuordnung(ctx context.Context, tx pgx.Tx, antwort *schuljahreswechselAntwort) error {
	zeilen, err := repository.LadeKlassenleitungVersetzungen(ctx, tx)
	if err != nil {
		return err
	}

	for _, z := range zeilen {
		if z.Entfaellt {
			if _, err := repository.LoescheKlassenleitung(ctx, tx, z.Alt); err != nil {
				return err
			}
			antwort.ZuordnungenEntfernt++
			continue
		}
		belegt, err := repository.IstKlassenleitungVorhanden(ctx, tx, z.Neu)
		if err != nil {
			return err
		}
		if belegt {
			antwort.Zuordnungskonflikte = append(antwort.Zuordnungskonflikte, z.Alt+" -> "+z.Neu)
			continue
		}
		if err := repository.BenenneKlassenleitungUm(ctx, tx, z.Neu, z.Alt); err != nil {
			return err
		}
		antwort.ZuordnungenVersetzt++
	}
	return nil
}

// leseSchuljahreswechselAnfrage liest den Rumpf der Anfrage und verlangt die ausdrückliche
// Bestätigung (außer im Probelauf). ok=false: die Fehlerantwort wurde bereits geschrieben.
func leseSchuljahreswechselAnfrage(w http.ResponseWriter, r *http.Request) (schuljahreswechselAnfrage, bool) {
	var anfrage schuljahreswechselAnfrage
	if err := json.NewDecoder(r.Body).Decode(&anfrage); err != nil {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, apierrors.MeldungAnfrageUnlesbar, err)
		return anfrage, false
	}
	if !anfrage.Probelauf && !anfrage.Bestaetigt {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Bitte den Schuljahreswechsel bestätigen", nil)
		return anfrage, false
	}
	return anfrage, true
}

// pruefeDoppellaufSchutz verhindert einen zweiten Schuljahreswechsel innerhalb von
// 12 Stunden (ein zweiter Lauf würde alle Schüler +2 versetzen). ok=false: die
// Fehlerantwort (500 oder 409) wurde bereits geschrieben.
//
// Das Fenster ist bewusst großzügig: Der Schuljahreswechsel läuft einmal im Jahr. Ein
// Fenster von zehn Minuten ließe genau den wahrscheinlichsten Fehler durch: Der Admin bemerkt
// etwas, korrigiert und klickt nach einer Viertelstunde erneut, und die ganze Schule würde ein
// zweites Mal versetzt. Ein Wiederholungslauf nach einem fehlgeschlagenen Wechsel bleibt
// möglich: Der Eintrag im Protokoll wird atomar mit dem Lauf committet
// (finalisiereSchuljahreswechsel), ein abgebrochener Lauf hinterlässt also keinen Eintrag und
// keine Sperre.
func pruefeDoppellaufSchutz(ctx context.Context, tx pgx.Tx, w http.ResponseWriter) bool {
	juengsteLaeufe, err := repository.ZaehleJuengsteSchuljahreswechsel(ctx, tx)
	if err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	if juengsteLaeufe > 0 {
		apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Der Schuljahreswechsel lief vor wenigen Minuten. Ein zweiter Lauf würde alle Schüler doppelt versetzen.", nil)
		return false
	}
	return true
}

// finalisiereSchuljahreswechsel schreibt den Eintrag im Protokoll atomar mit dem Lauf (er
// trägt zugleich den Doppellauf-Schutz) und committet die Transaktion. ok=false: die
// Fehlerantwort wurde bereits geschrieben.
func (t *Tueren) finalisiereSchuljahreswechsel(ctx context.Context, tx pgx.Tx, w http.ResponseWriter, r *http.Request, kontoID string, versetzt, archiviert int) bool {
	if err := repository.SchreibeAdminProtokoll(ctx, tx, kontoID, repository.AktionSchuljahreswechsel, clientip.AusAnfrage(r),
		fmt.Sprintf(`{"versetzt": %d, "abgaenger": %d}`, versetzt, archiviert)); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}

	if err := tx.Commit(ctx); err != nil {
		apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
		return false
	}
	log.Printf("Schuljahreswechsel durchgeführt (Benutzer %s): %d Schüler versetzt, %d neue Abgänger", kontoID, versetzt, archiviert)
	return true
}

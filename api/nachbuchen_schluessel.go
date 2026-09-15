package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"bibliothek/internal/service"
	"bibliothek/repository"

	"github.com/google/uuid"
)

// Der Idempotenz-Schlüssel an der Nachbuch-Tür (Rasterdurchgang 15.09.2026, OFFEN.md 5.15).
//
// Der Theken-Rechner benutzt für einen Scan EINEN Schlüssel: zuerst beim Online-Versand, und wenn
// dessen Antwort nicht ankommt, beim Nachbuchen. Die Tür liest darum, WAS unter dem Schlüssel
// steht, nicht nur, OB etwas steht:
//
//   - nichts: ein gewöhnlicher Offline-Scan — reservieren, buchen, Ergebnis ablegen;
//   - eine Reservierung ohne Antwort: die Online-Buchung läuft noch — „wiederholen";
//   - ein abgelegtes Ergebnis der Tür: eine wiederholte Portion — dieselbe Antwort, nichts neu;
//   - eine Online-Ausleihe oder -Rückgabe: vollständig gebucht — „bereits_gebucht";
//   - eine Online-Fremdrückgabe zu einer Ausleihe: die Ausleihe fehlt noch — übernehmen und unter
//     der Ausnahme des Wächters buchen;
//   - alles andere (Fehlerantwort, Ausweis- oder Suchantwort): online wurde nichts gebucht —
//     übernehmen und wie einen gewöhnlichen Scan buchen.
//
// Übernehmen heißt: Die gespeicherte Antwort wird zur Reservierung dieser Anfrage
// (repository.UebernimmIdempotenzAntwort). Ein zweiter Aufruf mit derselben Portion findet dann
// die Reservierung und bucht nicht daneben. Scheitert das Buchen am Server, wird die Online-Antwort
// zurückgelegt, sonst verlöre die nächste Runde, dass die Fremdrückgabe schon gebucht ist. Stirbt
// der Server dazwischen, übernimmt nach repository.IdempotenzReservierungsfrist die nächste
// Anfrage die verwaiste Reservierung und bucht wie einen gewöhnlichen Scan: Der Wächter meldet
// dann „veraltet" — laut, nicht still.
//
// Dieselbe Tabelle wie der Online-Versand, derselbe 24-Stunden-Ablauf (jobs/cron.go). Ein
// abgelegtes Ergebnis steht als {"nachbuchen": …} da; eine Online-Antwort trägt dieses Feld nie.

// nachbuchAblage ist die Form eines abgelegten Ergebnisses der Tür in idempotency_keys.
type nachbuchAblage struct {
	Nachbuchen *NachbuchenErgebnis `json:"nachbuchen"`
}

// nachbuchSchluesselLage sagt, was mit einem Eintrag geschieht: Entweder steht die Antwort schon
// fest, oder der Schlüssel gehört dieser Anfrage und es wird gebucht.
type nachbuchSchluesselLage struct {
	antwort           *NachbuchenErgebnis
	fremdrueckgabeVon *string                       // Ausnahme des Wächters, nur beim Buchen
	online            *repository.IdempotenzAntwort // übernommene Online-Antwort, beim Serverfehler zurücklegen
}

// ergreifeNachbuchSchluessel liest den Schlüssel und entscheidet nach der Liste im Kopf.
func (s *Server) ergreifeNachbuchSchluessel(ctx context.Context, e NachbuchenEintrag) (nachbuchSchluesselLage, error) {
	reserviert, antwort, err := repository.ReserviereIdempotenzSchluessel(ctx, s.DB.Pool, e.Schluessel)
	if err != nil {
		return nachbuchSchluesselLage{}, err
	}
	if reserviert {
		return nachbuchSchluesselLage{}, nil
	}
	if antwort.InArbeit() {
		return nachbuchSchluesselLage{antwort: nachbuchNochInArbeit(e)}, nil
	}
	var ablage nachbuchAblage
	if json.Unmarshal(antwort.Daten, &ablage) == nil && ablage.Nachbuchen != nil {
		return nachbuchSchluesselLage{antwort: ablage.Nachbuchen}, nil
	}

	var fremdrueckgabeVon *string
	if antwort.Status >= http.StatusOK && antwort.Status < http.StatusMultipleChoices {
		var gebucht ActionResponse
		if json.Unmarshal(antwort.Daten, &gebucht) == nil && (gebucht.Type == "ausleihe" || gebucht.Type == "rueckgabe") {
			if !nurFremdrueckgabeVorAusleihe(gebucht, e.Absicht) {
				return nachbuchSchluesselLage{antwort: &NachbuchenErgebnis{
					Schluessel: e.Schluessel, Ergebnis: nachbuchBereitsGebucht, Daten: &gebucht,
				}}, nil
			}
			fremdrueckgabeVon = gebucht.LoanID
		}
	}

	uebernommen, err := repository.UebernimmIdempotenzAntwort(ctx, s.DB.Pool, e.Schluessel, *antwort)
	if err != nil {
		return nachbuchSchluesselLage{}, err
	}
	if !uebernommen { // ein gleichzeitiger Aufruf war schneller
		return nachbuchSchluesselLage{antwort: nachbuchNochInArbeit(e)}, nil
	}
	return nachbuchSchluesselLage{fremdrueckgabeVon: fremdrueckgabeVon, online: antwort}, nil
}

// nurFremdrueckgabeVorAusleihe: Der Online-Versand hat nur die Fremdrückgabe gebucht, und der
// Eintrag will ausleihen — die Ausleihe fehlt noch. Ohne gültige Ausleih-Kennung gibt es keine
// Ausnahme; der Eintrag gilt dann als vollständig gebucht.
func nurFremdrueckgabeVorAusleihe(gebucht ActionResponse, absicht string) bool {
	if gebucht.Type != "rueckgabe" || !gebucht.Fremdrueckgabe || absicht != service.NachbuchAbsichtAusleihe || gebucht.LoanID == nil {
		return false
	}
	_, err := uuid.Parse(*gebucht.LoanID)
	return err == nil
}

func nachbuchNochInArbeit(e NachbuchenEintrag) *NachbuchenErgebnis {
	return &NachbuchenErgebnis{Schluessel: e.Schluessel, Ergebnis: nachbuchWiederholen, Grund: "wird gerade gebucht"}
}

// legeNachbuchErgebnisAb schreibt das Ergebnis in die Reservierung — mit WithoutCancel wie
// saveToCache: Bricht der Rechner nach dem Buchen ab, soll die Wiederholung das Ergebnis finden.
func (s *Server) legeNachbuchErgebnisAb(ctx context.Context, erg NachbuchenErgebnis) {
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), idempotenzSpeicherfrist)
	defer abbruch()
	daten, err := json.Marshal(nachbuchAblage{Nachbuchen: &erg})
	if err != nil {
		log.Printf("nachbuchen: Ergebnis zu %s nicht serialisierbar: %v", erg.Schluessel, err)
		return
	}
	if err := repository.SpeichereIdempotenzAntwort(ctx, s.DB.Pool, erg.Schluessel, daten, http.StatusOK); err != nil {
		log.Printf("nachbuchen: Ergebnis zu %s nicht abgelegt: %v", erg.Schluessel, err)
	}
}

// gibNachbuchSchluesselZurueck räumt nach einem Serverfehler: Eine übernommene Online-Antwort
// kommt zurück, eine eigene Reservierung wird freigegeben.
func (s *Server) gibNachbuchSchluesselZurueck(ctx context.Context, schluessel string, lage nachbuchSchluesselLage) {
	ctx, abbruch := context.WithTimeout(context.WithoutCancel(ctx), idempotenzSpeicherfrist)
	defer abbruch()
	if lage.online != nil {
		if err := repository.StelleIdempotenzAntwortWiederHer(ctx, s.DB.Pool, schluessel, *lage.online); err != nil {
			log.Printf("nachbuchen: Online-Antwort zu %s nicht zurückgelegt: %v", schluessel, err)
		}
		return
	}
	if _, err := repository.GibIdempotenzSchluesselFrei(ctx, s.DB.Pool, schluessel); err != nil {
		log.Printf("nachbuchen: Schlüssel %s nicht freigegeben: %v", schluessel, err)
	}
}

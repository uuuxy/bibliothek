package leser

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"bibliothek/internal/api/tuer"
	"bibliothek/internal/apierrors"
	"bibliothek/internal/pkg/code39"
	"bibliothek/internal/repository"
)

// Leserliste liefert die Leser, wahlweise gefiltert nach Klasse, Jahrgang, Suchbegriff, Status
// und Art.
// lmfAbfragen liefert die Klassen für den Jahrgangsfilter. Als Parameter, nicht über t.DB:
// Die Türen dieses Pakets bekommen ihre Abfragen von außen, und ein Griff nach t.DB.Pool
// sprengte jeden Test, der die Türen ohne Datenbank baut (TestLeserlisteLeereListeIstArray).
func (t *Tueren) Leserliste(leserAbfragen *repository.LeserAbfragen, lmfAbfragen *repository.LMFAbfragen) http.HandlerFunc {
	return apierrors.MitFehlerantwort(func(w http.ResponseWriter, r *http.Request) error {
		// Klasse oder Jahrgang — die Klasse gewinnt, wenn beides kommt (sie ist die
		// genauere Angabe). Die Übersetzung steht in schueler_jahrgang_filter.go.
		klassen, ohneTreffer, filterFehler := klassenFilter(r.Context(), lmfAbfragen,
			r.URL.Query().Get("klasse"), r.URL.Query().Get("jahrgang"))
		if filterFehler != nil {
			return apierrors.Internal("Klassen des Jahrgangs konnten nicht gelesen werden", filterFehler)
		}
		if ohneTreffer {
			// Zu diesem Jahrgang gibt es keine Klasse. Eine leere Liste ist die richtige
			// Antwort; ohne diesen Zweig stünde hier die ganze Kartei.
			tuer.AntworteJSON(w, http.StatusOK, []repository.LeserListenZeile{})
			return nil
		}
		// q sucht auf dem Server. Ohne das filterte die Schülerdatei im Browser über die
		// gelieferten (gekappten) Zeilen und fand niemanden dahinter.
		suche := strings.TrimSpace(r.URL.Query().Get("q"))

		sortierung, sortierFehler := leserSortierung(r)
		if sortierFehler != nil {
			return sortierFehler
		}

		// status=ehemalige: der Reiter „Ehemalige" — dieselbe Liste mit
		// umgekehrtem Vorzeichen (ist_abgaenger), kein eigener Endpunkt.
		//
		// art=alle: die Leserdatei. Muss ausdrücklich angefragt werden, denn an dieser
		// Tür hängen drei Ansichten mit verschiedenen Absichten — die Leserdatei, der
		// Reiter „Ehemalige" und die Schülersuche des Vormerkungs-Reiters. Für die
		// beiden letzten wäre ein Kollege in der Liste falsch: Eine Vormerkung schickt
		// eine Nachricht an die Elternadresse, und die hat er nicht. Deshalb bleibt die
		// Vorgabe „nur Schüler", und wer alle meint, sagt es.
		ansicht := leserAnsicht{status: r.URL.Query().Get("status"), art: r.URL.Query().Get("art"),
			klassen: klassen, sortierung: sortierung}
		leser, err := ladeLeserliste(r.Context(), leserAbfragen, ansicht, suche)
		if err != nil {
			return apierrors.Internal("Fehler beim Abrufen der Leserliste", err)
		}

		tuer.AntworteListe(w, leser, repository.LeserListeGrenze(suche))
		return nil
	})
}

// leserAnsicht sagt, welche der drei Ansichten an dieser Tür fragt und wie sie filtert und ordnet.
type leserAnsicht struct {
	status, art string
	klassen     []string
	sortierung  repository.LeserSortierung
}

// ladeLeserliste lädt die Liste der Ansicht zum Suchtext. Ein alter Ausweis trägt im Strichcode
// ein Prüfzeichen von Code 39 („A-100037“ für A-10003): Findet der Scan niemanden, gilt er ohne
// das Zeichen, wie an der Theke (service.OmniboxDienst.VerarbeiteEingabe), und nur als zweiter
// Versuch.
func ladeLeserliste(ctx context.Context, leserAbfragen *repository.LeserAbfragen, ansicht leserAnsicht, suche string) ([]repository.LeserListenZeile, error) {
	lade := func(suche string) ([]repository.LeserListenZeile, error) {
		switch {
		case ansicht.status == "ehemalige":
			return leserAbfragen.LadeEhemaligenListe(ctx, suche, ansicht.sortierung)
		case ansicht.art == "alle":
			return leserAbfragen.LadeLeserListe(ctx, ansicht.klassen, suche, ansicht.sortierung)
		default:
			return leserAbfragen.LadeSchuelerListe(ctx, ansicht.klassen, suche, ansicht.sortierung)
		}
	}
	leser, err := lade(suche)
	if err != nil || len(leser) > 0 {
		return leser, err
	}
	if kern, hatPruefzeichen := code39.OhnePruefzeichen(suche); hatPruefzeichen {
		return lade(kern)
	}
	return leser, nil
}

// leserSortierung liest Spalte und Richtung aus der Anfrage.
//
// Ein unbekannter Spaltenname ist ein 400 und keine stille Vorgabe: Die Liste käme sortiert
// nach irgendetwas zurück, die Oberfläche zeigte ihren Pfeil an der geklickten Spalte, und
// niemand würde je erfahren, dass beides nicht zusammenpasst. Die Meldung nennt die
// erlaubten Werte.
func leserSortierung(r *http.Request) (repository.LeserSortierung, *apierrors.Fehler) {
	spalte := repository.LeserSortierspalte(strings.TrimSpace(r.URL.Query().Get("sortierung")))
	richtung := strings.TrimSpace(r.URL.Query().Get("richtung"))

	s := repository.LeserSortierung{Spalte: spalte, Absteigend: richtung == "ab"}
	if !s.IstErlaubt() {
		erlaubt := make([]string, 0, len(repository.LeserSortierspalten))
		for _, e := range repository.LeserSortierspalten {
			erlaubt = append(erlaubt, string(e))
		}
		return repository.LeserSortierung{}, apierrors.BadRequest(fmt.Sprintf(
			"Die Sortierung „%s“ gibt es nicht. Möglich sind %s oder leer für die Reihenfolge der Kartei.",
			spalte, strings.Join(erlaubt, ", ")), errors.New("unbekannte sortierspalte"))
	}
	// „ab" oder leer/„auf" — ein dritter Wert ist ein Tippfehler des Aufrufers und
	// bekommt dieselbe Antwort wie eine unbekannte Spalte.
	if richtung != "" && richtung != "auf" && richtung != "ab" {
		return repository.LeserSortierung{}, apierrors.BadRequest(fmt.Sprintf(
			"Die Richtung „%s“ gibt es nicht. Möglich sind auf und ab.", richtung), errors.New("unbekannte sortierrichtung"))
	}
	return s, nil
}

package leser

import (
	"errors"
	"fmt"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

// Papierkorb liefert die Leser im Papierkorb, die jüngsten Löschungen zuerst, höchstens 1000.
func (t *Tueren) Papierkorb() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// LIMIT gegen unbegrenztes Wachstum: Leser im Papierkorb sammeln sich an
		// (die 180-Tage-Anonymisierung leert die Zeile, entfernt sie aber nicht). Der
		// Papierkorb ist zum Wiederherstellen der jüngsten Löschungen da; 1000 ist weit
		// über dem realistischen Bestand einer manuell befüllten Papierkorb-Ansicht.
		const papierkorbGrenze = 1000
		zeilen, err := repository.LadePapierkorb(ctx, t.DB.Pool, papierkorbGrenze)
		if err != nil {
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		// anonymized_at reist mit: Nach der Anonymisierung weist LeserWiederherstellen das
		// Wiederherstellen mit 409 ab, und ohne das Feld böte die Oberfläche einen Knopf an,
		// der nur scheitern kann.
		leser := make([]map[string]any, 0, len(zeilen))
		for _, z := range zeilen {
			leser = append(leser, map[string]any{
				"id":             z.ID,
				"barcode_id":     z.Barcode,
				"vorname":        z.Vorname,
				"nachname":       z.Nachname,
				"klasse":         z.Klasse,
				"abgaenger_jahr": z.AbgaengerJahr,
				"ist_gesperrt":   z.Gesperrt,
				"deleted_at":     z.GeloeschtAm,
				"anonymized_at":  z.AnonymisiertAm,
			})
		}

		tuer.AntworteJSON(w, http.StatusOK, leser)
	}
}

// LeserWiederherstellen holt einen Leser aus dem Papierkorb zurück.
func (t *Tueren) LeserWiederherstellen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Lesers fehlt", nil)
			return
		}

		ctx := r.Context()

		// Anonymisierte Papierkorb-Zeilen (anonymized_at gesetzt) sind kein Schüler mehr,
		// sondern ein Pseudonym ohne Namen — „wiederherstellen" ergäbe einen gesperrten
		// „Anonym"-Datensatz in der aktiven Liste. 409 statt Wiederherstellen.
		if anonymisiert, err := repository.IstPapierkorbLeserAnonymisiert(ctx, t.DB.Pool, id); err == nil && anonymisiert {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, "Dieser Datensatz ist bereits anonymisiert (DSGVO) und lässt sich nicht wiederherstellen", nil)
			return
		}

		// Wiederherstellen: deleted_at zurücknehmen und die Lösch-Sperre aufheben.
		// LegeLeserInPapierkorb setzt ist_gesperrt=true mit block_reason='Systematisch gelöscht';
		// ohne das Aufheben bliebe der wiederhergestellte Schüler dauerhaft gesperrt
		// (Zombie-Sperre) und könnte nichts ausleihen. Eine Sperre aus anderem Grund bleibt
		// bestehen.
		wiederhergestellt, err := repository.StelleLeserWiederHer(ctx, t.DB.Pool, id)
		if err != nil {
			// Drei Teilindizes gelten nur für aktive Zeilen: Namensindex (Migration 108,
			// in der Normalform suchnorm), LUSD-ID und Ausweis-Barcode. Solange die Zeile
			// im Papierkorb liegt, ist ihr Platz frei — ein Import oder eine Handanlage
			// kann ihn besetzen. Dann ist das Wiederherstellen keine Störung, sondern
			// eine Lage, die jemand auflösen muss: 409 mit dem Grund und dem Weg heraus.
			if meldung, ok := wiederherstellungsKollision(err); ok {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, meldung, err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		if wiederhergestellt == 0 {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, "Schüler nicht im Papierkorb gefunden", nil)
			return
		}

		// Das Wiederherstellen ins Protokoll, wenn jemand angemeldet ist.
		if claims, ok := auth.ClaimsAusKontext(ctx); ok {
			t.SchreibeAdminProtokoll(ctx, claims.KontoID, "RESTORE_STUDENT", clientip.AusAnfrage(r), `{"schueler_id":"`+id+`"}`)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Schüler erfolgreich wiederhergestellt",
		})
	}
}

// wiederherstellungsKollision übersetzt die Eindeutigkeits-Verletzung beim Wiederherstellen
// in einen Satz, mit dem die Bibliothek etwas anfangen kann. Ohne ihn stünde dort „Ein Eintrag
// mit denselben Angaben besteht bereits", eine Sackgasse, die weder den Zwilling noch den Weg
// heraus nennt.
//
// Jeder der drei Indizes ist ein Teilindex auf deleted_at IS NULL; sie greifen also genau
// in dem Moment, in dem die Zeile wieder aktiv würde.
func wiederherstellungsKollision(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return "", false
	}
	switch pgErr.ConstraintName {
	case "unique_schueler_name_gebdatum":
		return "Wiederherstellen nicht möglich: Ein aktiver Datensatz trägt bereits denselben Namen und dasselbe Geburtsdatum. Beide über „Schüler zusammenführen“ vereinen, statt zwei Zeilen für denselben Menschen zu führen.", true
	case "uniq_schueler_barcode_active":
		return "Wiederherstellen nicht möglich: Ein aktiver Datensatz trägt bereits denselben Ausweis-Barcode. Erst den Barcode des aktiven Datensatzes ändern.", true
	case "uniq_ausweis_ueber_personen":
		return "Wiederherstellen nicht möglich: Eine Lehrkraft trägt inzwischen denselben Ausweis-Barcode. Erst einer der beiden Personen eine andere Nummer geben.", true
	case "uniq_schueler_lusd_id_active":
		return "Wiederherstellen nicht möglich: Ein aktiver Datensatz trägt bereits dieselbe LUSD-ID. Beide über „Schüler zusammenführen“ vereinen.", true
	case repository.ConstraintNummerUeberBuchUndAusweis:
		return "Wiederherstellen nicht möglich: Die Ausweisnummer ist inzwischen der Barcode eines Buchs. Erst dem Buch eine andere Nummer geben.", true
	}
	return "", false
}

// LeserEndgueltigLoeschen entfernt einen Leser, der im Papierkorb liegt, endgültig
// (repository.LoescheLeserEndgueltig); offene Ausleihen und unbezahlte Forderungen
// verhindern es.
func (t *Tueren) LeserEndgueltigLoeschen(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusBadRequest, "Die Kennung des Lesers fehlt", nil)
			return
		}

		claims, ok := auth.ClaimsAusKontext(r.Context())
		if !ok {
			apierrors.AntworteFehlerMitMeldung(w, http.StatusUnauthorized, auth.MeldungNichtAngemeldet, nil)
			return
		}

		ctx := r.Context()
		if err := protokollAbfragen.LoescheLeserEndgueltig(ctx, id, claims.KontoID); err != nil {
			// Blockade (offene Ausleihen / unbezahlte Schäden / nicht im Papierkorb) ist ein
			// Konflikt, alles andere nicht: Bekäme jeder Fehler die 409, auch ein
			// Verbindungsabbruch, suchte die Bibliothek einen offenen Vorgang, den es nicht gibt,
			// und der echte Fehler bliebe verdeckt.
			//
			// Die Form (verschachtelte ifs statt eines switch, „nicht gefunden" zuerst) ist die
			// im Haus übliche und keine Geschmacksfrage: Der Fehler-Kollaps-Detektor liest die
			// if-Bedingung und die erste Anweisung des Rumpfs; einen switch darunter sieht er nicht.
			if errors.Is(err, repository.ErrLeserNichtGefunden) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusNotFound, meldungLeserNichtGefunden, err)
				return
			}
			var blockiert *repository.LoeschenBlockiertFehler
			if errors.As(err, &blockiert) {
				apierrors.AntworteFehlerMitMeldung(w, http.StatusConflict, loeschenBlockiertMeldung(blockiert), err)
				return
			}
			apierrors.AntworteFehler(w, http.StatusInternalServerError, err)
			return
		}

		t.SchreibeAdminProtokoll(ctx, claims.KontoID, "PURGE_STUDENT", clientip.AusAnfrage(r), `{"schueler_id":"`+id+`"}`)

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Schüler endgültig und DSGVO-konform gelöscht",
		})
	}
}

// loeschenBlockiertMeldung nennt, was dem endgültigen Löschen im Weg steht, bei Ausleihen und
// Forderungen mit ihrer Zahl.
func loeschenBlockiertMeldung(e *repository.LoeschenBlockiertFehler) string {
	switch {
	case e.NichtImPapierkorb:
		return "Endgültiges Löschen blockiert: Der Datensatz liegt nicht im Papierkorb. Erst löschen, dann endgültig entfernen."
	case e.OffeneAusleihen == 1:
		return "Endgültiges Löschen blockiert: 1 offene Ausleihe"
	case e.OffeneAusleihen > 1:
		return fmt.Sprintf("Endgültiges Löschen blockiert: %d offene Ausleihen", e.OffeneAusleihen)
	case e.UnbezahlteSchadensfaelle == 1:
		return "Endgültiges Löschen blockiert: 1 unbezahlter Schadensfall"
	}
	return fmt.Sprintf("Endgültiges Löschen blockiert: %d unbezahlte Schadensfälle", e.UnbezahlteSchadensfaelle)
}

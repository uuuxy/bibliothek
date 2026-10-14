package leser

import (
	"errors"
	"net/http"

	"bibliothek/api/tuer"
	"bibliothek/apierrors"
	"bibliothek/auth"
	"bibliothek/pkg/clientip"
	"bibliothek/repository"

	"github.com/jackc/pgx/v5/pgconn"
)

// Papierkorb liefert eine Liste aller weichgelöschten Schüler für den Papierkorb.
func (t *Tueren) Papierkorb() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// LIMIT gegen unbegrenztes Wachstum: soft-gelöschte Schüler sammeln sich an
		// (die 180-Tage-Anonymisierung leert die Zeile, entfernt sie aber nicht). Der
		// Papierkorb ist zum Wiederherstellen der JÜNGSTEN Löschungen da; 1000 ist weit
		// über dem realistischen Bestand einer manuell befüllten Papierkorb-Ansicht.
		const papierkorbGrenze = 1000
		zeilen, err := repository.LadePapierkorb(ctx, t.DB.Pool, papierkorbGrenze)
		if err != nil {
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
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

// LeserWiederherstellen stellt einen weichgelöschten Schüler wieder her.
func (t *Tueren) LeserWiederherstellen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		ctx := r.Context()

		// Anonymisierte Papierkorb-Zeilen (anonymized_at gesetzt) sind kein Schüler mehr,
		// sondern ein Pseudonym ohne Namen — „wiederherstellen" ergäbe einen gesperrten
		// „Anonym"-Datensatz in der aktiven Liste (Prüfung 22.08.2026). 409 statt Restore.
		if anonymisiert, err := repository.IstPapierkorbLeserAnonymisiert(ctx, t.DB.Pool, id); err == nil && anonymisiert {
			apierrors.SendHTTPError(w, http.StatusConflict,
				errors.New("dieser Datensatz ist bereits anonymisiert (DSGVO) und kann nicht wiederhergestellt werden"))
			return
		}

		// Restore: deleted_at zurücknehmen UND die Lösch-Sperre aufheben. LegeLeserInPapierkorb
		// setzt ist_gesperrt=true mit block_reason='Systematisch gelöscht'; ohne das
		// Aufheben bliebe der wiederhergestellte Schüler dauerhaft gesperrt (Zombie-Sperre)
		// und könnte nichts ausleihen. Eine Sperre aus ANDEREM Grund bleibt bestehen.
		wiederhergestellt, err := repository.StelleLeserWiederHer(ctx, t.DB.Pool, id)
		if err != nil {
			// Drei Teilindizes gelten nur für AKTIVE Zeilen: Namensindex (Migration 108,
			// in der Normalform suchnorm), LUSD-ID und Ausweis-Barcode. Solange die Zeile
			// im Papierkorb liegt, ist ihr Platz frei — ein Import oder eine Handanlage
			// kann ihn besetzen. Dann ist das Wiederherstellen keine Störung, sondern
			// eine Lage, die jemand auflösen muss: 409 mit dem Grund und dem Weg heraus.
			if meldung, ok := wiederherstellungsKollision(err); ok {
				apierrors.SendHTTPError(w, http.StatusConflict, errors.New(meldung))
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		if wiederhergestellt == 0 {
			apierrors.SendHTTPError(w, http.StatusNotFound, errors.New("schüler nicht im Papierkorb gefunden"))
			return
		}

		// Optional: Audit-Log für Restore anlegen
		if claims, ok := auth.GetClaims(ctx); ok {
			t.SchreibeAdminProtokoll(ctx, claims.UserID, "RESTORE_STUDENT", clientip.AusAnfrage(r), `{"schueler_id":"`+id+`"}`)
		}

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Schüler erfolgreich wiederhergestellt",
		})
	}
}

// wiederherstellungsKollision übersetzt die Eindeutigkeits-Verletzung beim Wiederherstellen in einen
// Satz, mit dem die Bibliothek etwas anfangen kann. Ohne ihn stand dort „Ein interner
// Datenbankfehler ist aufgetreten" — eine Sackgasse: Sie nennt weder den Zwilling noch
// den Weg heraus (Bestands-Durchgang 10.09.2026).
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

// LeserEndgueltigLoeschen entfernt einen im Papierkorb liegenden Schüler endgültig und
// DSGVO-konform (Ausleihhistorie/Audit anonymisiert, bezahlte Schäden gelöscht,
// Datensatz entfernt). Offene Ausleihen/unbezahlte Schäden blockieren die Löschung.
func (t *Tueren) LeserEndgueltigLoeschen(protokollAbfragen repository.ProtokollAbfragen) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			apierrors.SendHTTPError(w, http.StatusBadRequest, errors.New("fehlende Schüler-ID"))
			return
		}

		claims, ok := auth.GetClaims(r.Context())
		if !ok {
			apierrors.SendHTTPError(w, http.StatusUnauthorized, errors.New("fehlende Session-Information"))
			return
		}

		ctx := r.Context()
		if err := protokollAbfragen.LoescheLeserEndgueltig(ctx, id, claims.UserID); err != nil {
			// Blockade (offene Ausleihen / unbezahlte Schäden / nicht im Papierkorb) ist ein
			// Konflikt. Alles andere NICHT: Bis zum 17.09.2026 bekam jeder Fehler die 409 —
			// auch ein Verbindungsabbruch. Das schickt die Bibliothek los, einen offenen
			// Vorgang zu suchen, den es nicht gibt, und verdeckt den echten Fehler.
			//
			// Die Form (verschachtelte ifs statt eines switch) ist die im Haus übliche und
			// keine Geschmacksfrage: Der Fehler-Kollaps-Detektor liest die if-Bedingung und
			// das ERSTE Statement des Rumpfs. Ein switch darunter sieht er nicht — er hat
			// diesen Handler beim Umbau am 17.09.2026 zu Recht angemahnt.
			if errors.Is(err, repository.ErrLoeschenBlockiert) {
				apierrors.SendHTTPError(w, http.StatusConflict, err)
				return
			}
			if errors.Is(err, repository.ErrLeserNichtGefunden) {
				apierrors.SendHTTPError(w, http.StatusNotFound, err)
				return
			}
			apierrors.SendHTTPError(w, http.StatusInternalServerError, err)
			return
		}

		t.SchreibeAdminProtokoll(ctx, claims.UserID, "PURGE_STUDENT", clientip.AusAnfrage(r), `{"schueler_id":"`+id+`"}`)

		tuer.AntworteJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Schüler endgültig und DSGVO-konform gelöscht",
		})
	}
}

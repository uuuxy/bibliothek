package inventur

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// Idempotenz des Listenimports.
//
// Der Import ist ADDITIV: Jede Zeile legt ihre Stückzahl als frische Exemplare an
// (legeImportExemplareAn). Wer dieselbe Liste zweimal einspielt, hat doppelten Bestand —
// und zweimal einspielen ist genau das, was nach einer verlorenen Antwort passiert: Der
// Server hat committet, der Browser sah Timeout oder Netzfehler, der Mensch drückt
// nochmal. Der Schaden ist still: nirgends steht, dass 400 Exemplare zu viel da sind.
//
// Der Aufrufer benennt seinen Lauf mit X-Idempotency-Key (eine UUID je Dateiauswahl,
// ListenImportWidget.svelte). Der Server merkt sich den Schlüssel in idempotency_keys —
// derselben Tabelle wie die Theken-Aktionen (api/action.go, Migration 028, Aufräumen nach
// 24 h in jobs/cron.go). Drei Zustände:
//
//   - Schlüssel neu: reservieren (Marker 409 „läuft"), importieren, Ergebnis eintragen.
//   - Schlüssel reserviert, Lauf noch offen: 409 mit der Bitte zu warten. Der zweite Klick
//     startet keinen zweiten Import.
//   - Schlüssel abgeschlossen: die gespeicherte Antwort — der Klick holt das Ergebnis ab.
//
// Gemerkt wird nur, was GESCHRIEBEN wurde. Ein Lauf ohne ein einziges importiertes Buch
// (Abbruch, Netzfehler bei allen Lookups) gibt den Schlüssel wieder frei, damit ein
// erneuter Versuch mit derselben Dateiauswahl echt läuft statt die Fehlmeldung von eben
// zu wiederholen. Ohne Schlüssel läuft der Import wie bisher — der Header ist freiwillig,
// damit Skripte und alte Aufrufer nicht brechen.

const importSchluesselKopf = "X-Idempotency-Key"

const importLaeuftMeldung = "Dieser Import läuft noch auf dem Server. Bitte warten und dann erneut auf " +
	"„Liste importieren“ drücken – das holt das Ergebnis ab, ohne die Liste ein zweites Mal einzuspielen."

// importLaeuftMarker ist die Antwort, die ein reservierter, noch offener Schlüssel liefert.
// Einmal beim Start gebaut — ein Marshal-Fehler an einem festen Literal wäre ein
// Programmierfehler, kein Laufzeitfall.
var importLaeuftMarker = mussJSON(map[string]string{"error": importLaeuftMeldung})

func mussJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic("import_idempotenz: " + err.Error())
	}
	return b
}

// gecachteAntwort ist, was der zweite Aufruf mit demselben Schlüssel bekommt.
type gecachteAntwort struct {
	Status int
	Daten  json.RawMessage
}

// importSchluessel liest den Kopf. Leer ist erlaubt; ein Wert, der keine UUID ist, wird
// mit 400 abgewiesen, bevor irgendetwas passiert — die Spalte ist vom Typ UUID, und ein
// Fehler dort käme sonst erst nach dem Import.
func importSchluessel(writer http.ResponseWriter, request *http.Request) (string, bool) {
	wert := request.Header.Get(importSchluesselKopf)
	if wert == "" {
		return "", true
	}
	if _, err := uuid.Parse(wert); err != nil {
		writeError(writer, http.StatusBadRequest, importSchluesselKopf+" muss eine UUID sein")
		return "", false
	}
	return wert, true
}

// reserviereImportLauf trägt den Schlüssel mit dem Marker „läuft" ein. frisch=true heißt:
// dieser Aufruf darf importieren. Sonst kommt zurück, was gespeichert ist — der Marker
// oder das Ergebnis des ersten Laufs.
func (handler *APIHandler) reserviereImportLauf(ctx context.Context, schluessel string) (frisch bool, alt *gecachteAntwort, err error) {
	tag, err := handler.repo.db.Exec(ctx, `
		INSERT INTO idempotency_keys (idempotency_key, response_data, status_code)
		VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`, schluessel, importLaeuftMarker, http.StatusConflict)
	if err != nil {
		return false, nil, err
	}
	if tag.RowsAffected() == 1 {
		return true, nil, nil
	}
	var a gecachteAntwort
	if err := handler.repo.db.QueryRow(ctx, `
		SELECT response_data, status_code FROM idempotency_keys WHERE idempotency_key = $1`,
		schluessel).Scan(&a.Daten, &a.Status); err != nil {
		return false, nil, err
	}
	return false, &a, nil
}

// schliesseImportLauf ersetzt den Marker durch das Ergebnis — oder gibt den Schlüssel
// frei, wenn nichts geschrieben wurde. Läuft ohne die Abbruchsignale des Requests: Genau
// der Fall „Browser weg, Server fertig" ist der, für den das Gedächtnis da ist.
func (handler *APIHandler) schliesseImportLauf(ctx context.Context, schluessel string, geschrieben bool, status int, antwort map[string]any) {
	ctx = context.WithoutCancel(ctx)
	if !geschrieben {
		tag, err := handler.repo.db.Exec(ctx,
			`DELETE FROM idempotency_keys WHERE idempotency_key = $1`, schluessel)
		if err != nil || tag.RowsAffected() == 0 {
			log.Printf("listenimport idempotenz: schlüssel %s nicht freigegeben: %v", schluessel, err)
		}
		return
	}
	daten, err := json.Marshal(antwort)
	if err != nil {
		log.Printf("listenimport idempotenz: antwort nicht serialisierbar: %v", err)
		return
	}
	// 0 Zeilen hieße: Der Marker ist weg (Aufräumjob nach 24 h mitten im Lauf) — dann
	// gäbe es beim nächsten Klick keinen Schutz. Das gehört ins Log, nicht ins Stille.
	tag, err := handler.repo.db.Exec(ctx, `
		UPDATE idempotency_keys SET response_data = $2, status_code = $3
		 WHERE idempotency_key = $1`, schluessel, daten, status)
	if err != nil || tag.RowsAffected() == 0 {
		log.Printf("listenimport idempotenz: ergebnis zu %s nicht gespeichert: %v", schluessel, err)
	}
}
